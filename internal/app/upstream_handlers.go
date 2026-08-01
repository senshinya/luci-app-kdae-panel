package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/tuoro/kdae-panel/internal/daeinstall"
	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// InstallService 是 dae 版本管理能力的消费者侧接口。
type InstallService interface {
	Status(ctx context.Context) daeinstall.Status
	Provision(ctx context.Context) daeinstall.Provision
	Versions(ctx context.Context, source upstream.Source, limit int) ([]daeinstall.Version, error)
	Acquire(ctx context.Context, source upstream.Source, ref, label string, requireBundle bool) (upstream.Bundle, bool, error)
	DeleteCached(source upstream.Source, ref string) error
	Preflight(ctx context.Context, binary []byte) (daeinstall.Compatibility, error)
	Install(ctx context.Context, binary []byte, source upstream.Source, ref, label, platform string) (daeinstall.Status, error)
	FirstInstall(ctx context.Context, bundle upstream.Bundle, source upstream.Source, ref, label string) (daeinstall.Status, error)
	Rollback(ctx context.Context) (daeinstall.Status, error)
	Uninstall(ctx context.Context, options daeinstall.UninstallOptions) error
}

type CompatibilityJob struct {
	Phase     JobPhase                  `json:"phase"`
	Source    string                    `json:"source,omitempty"`
	Ref       string                    `json:"ref,omitempty"`
	Label     string                    `json:"label,omitempty"`
	Cached    bool                      `json:"cached,omitempty"`
	Result    *daeinstall.Compatibility `json:"result,omitempty"`
	StartedAt *time.Time                `json:"startedAt,omitempty"`
	EndedAt   *time.Time                `json:"endedAt,omitempty"`
	Error     string                    `json:"error,omitempty"`
}

type compatibilityJobs struct {
	mu  sync.Mutex
	job CompatibilityJob
}

// versionTaskGate 把下载、预检、切换、回滚、卸载与缓存删除视作同一类任务。
// 这些动作会共享版本缓存或安装状态，必须在“检查并占用”这一步就原子互斥，
// 不能先分别读取两套任务状态再启动，否则两个并发请求可能同时通过检查。
type versionTaskGate struct {
	mu   sync.Mutex
	busy bool
}

func (g *versionTaskGate) begin() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.busy {
		return false
	}
	g.busy = true
	return true
}

func (g *versionTaskGate) end() {
	g.mu.Lock()
	g.busy = false
	g.mu.Unlock()
}

func (j *compatibilityJobs) snapshot() CompatibilityJob {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.job
}

func (j *compatibilityJobs) begin(source, ref, label string) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.job.Phase == PhaseDownloading || j.job.Phase == PhaseApplying {
		return false
	}
	startedAt := time.Now().UTC()
	j.job = CompatibilityJob{Phase: PhaseDownloading, Source: source, Ref: ref, Label: label, StartedAt: &startedAt}
	return true
}

func (j *compatibilityJobs) checking(cached bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.job.Phase, j.job.Cached = PhaseApplying, cached
}

func (j *compatibilityJobs) finish(result *daeinstall.Compatibility, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	endedAt := time.Now().UTC()
	j.job.EndedAt = &endedAt
	j.job.Result = result
	if err != nil {
		j.job.Phase, j.job.Error = PhaseFailed, err.Error()
		return
	}
	j.job.Phase, j.job.Error = PhaseDone, ""
}

type installRequest struct {
	Source string `json:"source"`
	Ref    string `json:"ref"`
	Label  string `json:"label,omitempty"`
}

type cacheRequest struct {
	Source string `json:"source"`
	Ref    string `json:"ref"`
}

// JobPhase 描述安装任务当前进行到哪一步。
type JobPhase string

const (
	PhaseIdle        JobPhase = "idle"
	PhaseDownloading JobPhase = "downloading"
	PhaseApplying    JobPhase = "applying"
	PhaseDone        JobPhase = "done"
	PhaseFailed      JobPhase = "failed"
)

// Job 是一次安装或回滚的进度。
// 安装要下载几十兆并重启服务，耗时以分钟计，远超 HTTP 写超时，
// 因此做成异步任务，由前端轮询，而不是把请求挂在那里。
// 时间字段用指针：omitempty 对 time.Time 无效，
// 否则空闲任务会向前端吐出 0001-01-01 并被当成真实时间渲染。
type Job struct {
	Phase     JobPhase   `json:"phase"`
	Source    string     `json:"source,omitempty"`
	Ref       string     `json:"ref,omitempty"`
	Label     string     `json:"label,omitempty"`
	Cached    bool       `json:"cached,omitempty"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
	Error     string     `json:"error,omitempty"`
}

type installJobs struct {
	mu  sync.Mutex
	job Job
}

func (j *installJobs) snapshot() Job {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.job
}

func (j *installJobs) begin(phase JobPhase, source, ref, label string) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.job.Phase == PhaseDownloading || j.job.Phase == PhaseApplying {
		return false
	}
	startedAt := time.Now().UTC()
	j.job = Job{Phase: phase, Source: source, Ref: ref, Label: label, StartedAt: &startedAt}
	return true
}

func (j *installJobs) advance(phase JobPhase) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.job.Phase = phase
}

func (j *installJobs) markCached(cached bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.job.Cached = cached
}

func (j *installJobs) finish(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	endedAt := time.Now().UTC()
	j.job.EndedAt = &endedAt
	if err != nil {
		j.job.Phase = PhaseFailed
		j.job.Error = err.Error()
		return
	}
	j.job.Phase = PhaseDone
	j.job.Error = ""
}

func registerUpstreamRoutes(router *http.ServeMux, service InstallService, operations *sync.Mutex,
	logger *slog.Logger, backend host.Backend) {
	if service == nil {
		// 功能可显式关闭；没有 Install 依赖时让端点明确告诉客户端，
		// 而不是静默缺路由。开关在两套部署里是两个东西：OpenWrt 上没有
		// KDAE_PANEL_ENABLE_DAE_INSTALL 这个环境变量，照着它操作一定无效。
		message := "dae 版本管理未启用，请设置 KDAE_PANEL_ENABLE_DAE_INSTALL=true；" +
			"自定义安装路径还需加入服务单元的 ReadWritePaths"
		if backend == host.BackendProcd {
			message = "dae 版本管理未启用，请在 /etc/config/kdae-panel 里把 enable_dae_install 设为 1，" +
				"再执行 /etc/init.d/kdae-panel restart"
		}
		unavailable := func(writer http.ResponseWriter, _ *http.Request) {
			writeAPIError(writer, http.StatusServiceUnavailable, "dae_install_disabled", message)
		}
		for _, pattern := range []string{
			"GET /api/v1/dae/install", "POST /api/v1/dae/install",
			"GET /api/v1/dae/versions", "POST /api/v1/dae/rollback",
			"DELETE /api/v1/dae/cache", "POST /api/v1/dae/uninstall",
			"GET /api/v1/dae/compatibility", "POST /api/v1/dae/compatibility",
		} {
			router.HandleFunc(pattern, unavailable)
		}
		return
	}

	jobs := &installJobs{job: Job{Phase: PhaseIdle}}
	compatibility := &compatibilityJobs{job: CompatibilityJob{Phase: PhaseIdle}}
	versionTasks := &versionTaskGate{}

	router.HandleFunc("GET /api/v1/dae/install", func(writer http.ResponseWriter, request *http.Request) {
		status := service.Status(request.Context())
		job := jobs.snapshot()
		payload := map[string]any{"status": status, "job": job}
		// 还没有 dae 时附上首次安装的可行性，让界面能直接说清缺什么。
		// 任务进行中不计算：这个查询会被界面每两秒轮询一次，而可行性探测要
		// 实际试写目标目录，其中之一是 systemd 正在监视的单元目录。
		if !status.Ready && job.Phase != PhaseDownloading && job.Phase != PhaseApplying {
			payload["provision"] = service.Provision(request.Context())
		}
		writeJSON(writer, http.StatusOK, payload)
	})

	router.HandleFunc("GET /api/v1/dae/versions", func(writer http.ResponseWriter, request *http.Request) {
		source, err := upstream.ParseSource(request.URL.Query().Get("source"))
		if err != nil {
			writeAPIError(writer, http.StatusBadRequest, "invalid_upstream_source", err.Error())
			return
		}
		limit := 30
		if raw := request.URL.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 || parsed > 100 {
				writeAPIError(writer, http.StatusBadRequest, "invalid_limit", "版本数量必须是 1 到 100 之间的整数")
				return
			}
			limit = parsed
		}
		versions, err := service.Versions(request.Context(), source, limit)
		if err != nil {
			writeAPIError(writer, http.StatusBadGateway, "upstream_unavailable", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"versions": versions})
	})

	router.HandleFunc("POST /api/v1/dae/install", func(writer http.ResponseWriter, request *http.Request) {
		var payload installRequest
		if !decodeSmallJSONBody(writer, request, &payload) {
			return
		}
		source, err := upstream.ParseSource(payload.Source)
		if err != nil {
			writeAPIError(writer, http.StatusBadRequest, "invalid_upstream_source", err.Error())
			return
		}
		if payload.Ref == "" {
			writeAPIError(writer, http.StatusBadRequest, "invalid_version", "必须指定要安装的版本")
			return
		}
		if !versionTasks.begin() {
			writeAPIError(writer, http.StatusConflict, "version_task_in_progress", "已有版本管理任务正在执行")
			return
		}
		if !jobs.begin(PhaseDownloading, payload.Source, payload.Ref, payload.Label) {
			versionTasks.end()
			writeAPIError(writer, http.StatusConflict, "install_in_progress", "已有安装任务正在执行")
			return
		}
		go runInstall(jobs, versionTasks, service, operations, logger, source, payload.Ref, payload.Label)
		writeJSON(writer, http.StatusAccepted, map[string]any{"job": jobs.snapshot()})
	})

	router.HandleFunc("GET /api/v1/dae/compatibility", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]any{"job": compatibility.snapshot()})
	})
	router.HandleFunc("POST /api/v1/dae/compatibility", func(writer http.ResponseWriter, request *http.Request) {
		var payload installRequest
		if !decodeSmallJSONBody(writer, request, &payload) {
			return
		}
		source, err := upstream.ParseSource(payload.Source)
		if err != nil {
			writeAPIError(writer, http.StatusBadRequest, "invalid_upstream_source", err.Error())
			return
		}
		if payload.Ref == "" {
			writeAPIError(writer, http.StatusBadRequest, "invalid_version", "必须指定要预检的版本")
			return
		}
		if !service.Status(request.Context()).Ready {
			writeAPIError(writer, http.StatusConflict, "dae_not_installed", "首次安装会在安装事务内完成校验，无需单独预检")
			return
		}
		if !versionTasks.begin() {
			writeAPIError(writer, http.StatusConflict, "version_task_in_progress", "已有版本管理任务正在执行")
			return
		}
		if !compatibility.begin(payload.Source, payload.Ref, payload.Label) {
			versionTasks.end()
			writeAPIError(writer, http.StatusConflict, "compatibility_in_progress", "已有兼容性预检正在执行")
			return
		}
		go runCompatibility(compatibility, versionTasks, service, logger, source, payload.Ref, payload.Label)
		writeJSON(writer, http.StatusAccepted, map[string]any{"job": compatibility.snapshot()})
	})

	router.HandleFunc("DELETE /api/v1/dae/cache", func(writer http.ResponseWriter, request *http.Request) {
		var payload cacheRequest
		if !decodeSmallJSONBody(writer, request, &payload) {
			return
		}
		source, err := upstream.ParseSource(payload.Source)
		if err != nil {
			writeAPIError(writer, http.StatusBadRequest, "invalid_upstream_source", err.Error())
			return
		}
		if payload.Ref == "" {
			writeAPIError(writer, http.StatusBadRequest, "invalid_version", "必须指定要删除的本地版本")
			return
		}
		if !versionTasks.begin() {
			writeAPIError(writer, http.StatusConflict, "version_task_in_progress", "已有版本管理任务正在执行")
			return
		}
		defer versionTasks.end()
		if err := service.DeleteCached(source, payload.Ref); err != nil {
			if errors.Is(err, daeinstall.ErrCachedVersionNotFound) {
				writeAPIError(writer, http.StatusNotFound, "cached_version_not_found", err.Error())
				return
			}
			writeAPIError(writer, http.StatusInternalServerError, "cache_delete_failed", err.Error())
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})

	router.HandleFunc("POST /api/v1/dae/rollback", func(writer http.ResponseWriter, request *http.Request) {
		if !versionTasks.begin() {
			writeAPIError(writer, http.StatusConflict, "version_task_in_progress", "已有版本管理任务正在执行")
			return
		}
		if !jobs.begin(PhaseApplying, "", "", "回滚") {
			versionTasks.end()
			writeAPIError(writer, http.StatusConflict, "install_in_progress", "已有安装任务正在执行")
			return
		}
		go runRollback(jobs, versionTasks, service, operations, logger)
		writeJSON(writer, http.StatusAccepted, map[string]any{"job": jobs.snapshot()})
	})

	router.HandleFunc("POST /api/v1/dae/uninstall", func(writer http.ResponseWriter, request *http.Request) {
		var options daeinstall.UninstallOptions
		// 无请求体等同于安全零值，兼容旧客户端；有请求体仍只接受一个小 JSON 对象。
		if !decodeOptionalJSONBody(writer, request, &options) {
			return
		}
		if !versionTasks.begin() {
			writeAPIError(writer, http.StatusConflict, "version_task_in_progress", "已有版本管理任务正在执行")
			return
		}
		if !jobs.begin(PhaseApplying, "", "", "卸载 dae") {
			versionTasks.end()
			writeAPIError(writer, http.StatusConflict, "install_in_progress", "已有版本管理任务正在执行")
			return
		}
		go runUninstall(jobs, versionTasks, service, operations, logger, options)
		writeJSON(writer, http.StatusAccepted, map[string]any{"job": jobs.snapshot()})
	})
}

func runCompatibility(jobs *compatibilityJobs, versionTasks *versionTaskGate, service InstallService, logger *slog.Logger,
	source upstream.Source, ref, label string) {
	defer versionTasks.end()
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	logger.Info("开始预检 dae 版本兼容性", "source", source, "ref", ref)
	bundle, cached, err := service.Acquire(ctx, source, ref, label, false)
	if err != nil {
		jobs.finish(nil, err)
		return
	}
	jobs.checking(cached)
	result, err := service.Preflight(ctx, bundle.Binary)
	if err != nil {
		jobs.finish(nil, err)
		return
	}
	jobs.finish(&result, nil)
}

// installTimeout 覆盖下载与替换的总时长。任务在后台跑，不受 HTTP 写超时约束。
const installTimeout = 15 * time.Minute

func runInstall(jobs *installJobs, versionTasks *versionTaskGate, service InstallService,
	operations *sync.Mutex, logger *slog.Logger,
	source upstream.Source, ref, label string) {
	defer versionTasks.end()
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()

	logger.Info("开始安装 dae 版本", "source", source, "ref", ref)
	// 首次安装需要完整发布包；已有 dae 时只需二进制，可以直接命中本地缓存。
	// 读取缓存或下载都不占控制锁，几十兆 I/O 不该堵住配置保存与订阅刷新。
	requireBundle := !service.Status(ctx).Ready
	bundle, cached, err := service.Acquire(ctx, source, ref, label, requireBundle)
	if err != nil {
		logger.Warn("下载 dae 版本失败", "source", source, "ref", ref, "error", err)
		jobs.finish(err)
		return
	}
	jobs.markCached(cached)

	jobs.advance(PhaseApplying)
	operations.Lock()
	defer operations.Unlock()
	// 已有 dae 就替换二进制；还没有就连同单元、种子配置与 geo 数据一起装。
	if !service.Status(ctx).Ready {
		if len(bundle.Unit) == 0 {
			jobs.finish(errors.New("读取本地版本后发现 dae 已被外部卸载；首次安装需要完整发布包，请重试"))
			return
		}
		_, err = service.FirstInstall(ctx, bundle, source, ref, label)
	} else {
		_, err = service.Install(ctx, bundle.Binary, source, ref, label, bundle.Platform)
	}
	if err != nil {
		logger.Warn("安装 dae 版本失败", "source", source, "ref", ref, "error", err)
		jobs.finish(err)
		return
	}
	logger.Info("已安装 dae 版本", "source", source, "ref", ref)
	jobs.finish(nil)
}

func runRollback(jobs *installJobs, versionTasks *versionTaskGate, service InstallService,
	operations *sync.Mutex, logger *slog.Logger) {
	defer versionTasks.end()
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()

	operations.Lock()
	defer operations.Unlock()
	if _, err := service.Rollback(ctx); err != nil {
		logger.Warn("回滚 dae 版本失败", "error", err)
		jobs.finish(err)
		return
	}
	logger.Info("已回滚 dae 版本")
	jobs.finish(nil)
}

func runUninstall(jobs *installJobs, versionTasks *versionTaskGate, service InstallService,
	operations *sync.Mutex, logger *slog.Logger,
	options daeinstall.UninstallOptions) {
	defer versionTasks.end()
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()

	operations.Lock()
	defer operations.Unlock()
	if err := service.Uninstall(ctx, options); err != nil {
		logger.Warn("卸载 dae 失败", "error", err)
		jobs.finish(err)
		return
	}
	logger.Info("已卸载 dae", "purge_config", options.PurgeConfig, "purge_geo", options.PurgeGeo)
	jobs.finish(nil)
}
