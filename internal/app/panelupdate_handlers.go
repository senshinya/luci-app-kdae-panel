package app

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tuoro/kdae-panel/internal/panelupdate"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// PanelReleaseChecker 查询面板自身的最新发布 tag。
type PanelReleaseChecker func(ctx context.Context) (string, error)

// PanelUpdateService 是面板自升级能力的消费者侧接口。
type PanelUpdateService interface {
	Status(ctx context.Context) panelupdate.Status
	SetEnabled(enabled bool) error
	Download(ctx context.Context, version string) (upstream.PanelBinary, error)
	Apply(ctx context.Context, binary upstream.PanelBinary) error
}

type panelUpdateRequest struct {
	Version string `json:"version"`
}

type panelUpdatePreferenceRequest struct {
	Enabled *bool `json:"enabled"`
}

type panelUpdate struct {
	Current string `json:"current"`
	Latest  string `json:"latest,omitempty"`
	// Checked 区分"查过了，没有更新"与"根本没查"。
	//
	// 少了它，dev 构建与关掉检查的部署都只是 latest 为空、error 为空，
	// 与"已是最新"在结构上不可分辨，界面只能猜——procd 上就猜错过，
	// 对一个从未联网的部署报了绿色的"当前已是最新版本"。
	Checked         bool      `json:"checked"`
	UpdateAvailable bool      `json:"updateAvailable"`
	CheckedAt       time.Time `json:"checkedAt"`
	Error           string    `json:"error,omitempty"`
	// ReleasesURL 是本次检查所针对的仓库的发布页。
	// 由后端给出：检查的是哪个仓库只有后端知道，前端写死早晚会指错地方。
	ReleasesURL string `json:"releasesUrl,omitempty"`
}

const (
	// 这是给人看的提醒，不追求新鲜度；但绝不能把 GitHub 接口打成限流。
	panelUpdateCacheOK   = 6 * time.Hour
	panelUpdateCacheFail = 15 * time.Minute
	// 手动检查用于发布后立即确认，短时间内重复点击不应再次请求 GitHub。
	panelUpdateForceCooldown = time.Minute
	// 自升级要下载几兆的发布包，给得比接口超时宽裕。
	panelUpdateTimeout = 10 * time.Minute
)

// registerPanelUpdateRoutes 提供面板自身的版本检查与一键自升级。
//
// checker 为 nil 表示检查被关闭（KDAE_PANEL_DISABLE_UPDATE_CHECK），
// service 只在测试或不完整的嵌入部署中允许为空；正式程序始终创建管理器，
// 启用状态由它自己持久化，界面因此随时能开关而不需要 SSH。
func registerPanelUpdateRoutes(router *http.ServeMux, current string, checker PanelReleaseChecker,
	service PanelUpdateService, releasesURL string, operations *sync.Mutex, logger *slog.Logger) {
	var mu sync.Mutex
	var cached panelUpdate
	var expiresAt time.Time
	var lastForcedCheck time.Time
	jobs := &installJobs{job: Job{Phase: PhaseIdle}}

	check := func(ctx context.Context, force bool) panelUpdate {
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		if !force && now.Before(expiresAt) {
			return cached
		}
		if force && !lastForcedCheck.IsZero() && now.Sub(lastForcedCheck) < panelUpdateForceCooldown {
			return cached
		}
		if force {
			lastForcedCheck = now
		}
		result := panelUpdate{Current: current, CheckedAt: now.UTC(), ReleasesURL: releasesURL}
		ttl := panelUpdateCacheOK
		// dev 构建没有可比的版本号：不联网、不提示，而不是拿 dev 和 tag 硬比
		if _, ok := parseSemver(current); checker != nil && ok {
			result.Checked = true
			latest, err := checker(ctx)
			if err != nil {
				result.Error = err.Error()
				ttl = panelUpdateCacheFail
			} else {
				result.Latest = latest
				result.UpdateAvailable = versionBehind(current, latest)
			}
		}
		cached = result
		expiresAt = now.Add(ttl)
		return cached
	}

	payload := func(ctx context.Context, force bool) map[string]any {
		result := map[string]any{"check": check(ctx, force)}
		if service != nil {
			result["status"] = service.Status(ctx)
			result["job"] = jobs.snapshot()
		}
		return result
	}

	router.HandleFunc("GET /api/v1/panel/update", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, payload(request.Context(), false))
	})

	router.HandleFunc("POST /api/v1/panel/update/check", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, payload(request.Context(), true))
	})

	router.HandleFunc("PUT /api/v1/panel/update/preference", func(writer http.ResponseWriter, request *http.Request) {
		if service == nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "panel_self_update_unavailable",
				"当前部署不支持面板自升级")
			return
		}
		var payload panelUpdatePreferenceRequest
		if !decodeSmallJSONBody(writer, request, &payload) {
			return
		}
		if payload.Enabled == nil {
			writeAPIError(writer, http.StatusBadRequest, "invalid_self_update_preference", "必须指定 enabled")
			return
		}
		if err := service.SetEnabled(*payload.Enabled); err != nil {
			writeAPIError(writer, http.StatusInternalServerError, "self_update_preference_failed", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"status": service.Status(request.Context())})
	})

	router.HandleFunc("POST /api/v1/panel/update", func(writer http.ResponseWriter, request *http.Request) {
		if service == nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "panel_self_update_unavailable",
				"当前部署不支持面板自升级")
			return
		}
		payload := panelUpdateRequest{}
		if !decodeOptionalJSONBody(writer, request, &payload) {
			return
		}
		if payload.Version != "" && !validVersionTag(payload.Version) {
			writeAPIError(writer, http.StatusBadRequest, "invalid_panel_version",
				"版本号需与发布 tag 一致，形如 v0.2.0")
			return
		}
		status := service.Status(request.Context())
		if !status.Enabled {
			writeAPIError(writer, http.StatusConflict, "panel_self_update_disabled",
				"面板一键升级已关闭，可在设置页或更新提示中直接启用")
			return
		}
		if !status.Updatable {
			writeAPIError(writer, http.StatusConflict, "panel_self_update_unavailable", status.Problem)
			return
		}
		if !jobs.begin(PhaseDownloading, "panel", payload.Version, "面板自身") {
			writeAPIError(writer, http.StatusConflict, "panel_self_update_in_progress", "已有升级任务正在执行")
			return
		}
		go runPanelUpdate(jobs, service, operations, logger, payload.Version)
		writeJSON(writer, http.StatusAccepted, map[string]any{"job": jobs.snapshot()})
	})
}

// validVersionTag 与一键部署脚本对 KDAE_PANEL_VERSION 的校验保持一致：
// 拼进下载地址之前拦住含斜杠、空白等会改写路径或让请求离奇失败的取值。
func validVersionTag(value string) bool {
	if len(value) < 2 || value[0] != 'v' {
		return false
	}
	for _, char := range value[1:] {
		switch {
		case char >= '0' && char <= '9', char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z':
		case char == '.', char == '_', char == '-':
		default:
			return false
		}
	}
	return true
}

func runPanelUpdate(jobs *installJobs, service PanelUpdateService, operations *sync.Mutex,
	logger *slog.Logger, version string) {
	ctx, cancel := context.WithTimeout(context.Background(), panelUpdateTimeout)
	defer cancel()

	// 下载与校验不触碰任何共享状态，因此不占控制锁。
	binary, err := service.Download(ctx, version)
	if err != nil {
		logger.Warn("下载面板新版本失败", "error", err)
		jobs.finish(err)
		return
	}

	jobs.advance(PhaseApplying)
	// 替换自身之前先拿控制锁：正在进行的配置保存或 dae 安装不该被一次
	// 自我重启从中间打断。
	operations.Lock()
	defer operations.Unlock()
	if err := service.Apply(ctx, binary); err != nil {
		logger.Warn("升级面板失败", "error", err)
		jobs.finish(err)
		return
	}
	// 走到这里进程即将被 systemd 停掉。把任务标记为完成只是尽力而为——
	// 状态本就随进程消失，界面靠"健康接口报出新版本"确认升级成功。
	jobs.finish(nil)
}

type semver struct {
	parts      [3]int
	prerelease bool
}

// parseSemver 解析 vX.Y.Z、vX.Y.Z-pre 与 vX.Y.Z+build。解析不了（如 "dev"）返回
// false，调用方据此放弃比较而不是猜。
//
// build metadata（+ 之后的部分）按 semver 规范不参与优先级比较，这条不是可有可无的
// 细节：领先 tag 的构建正是靠它与所基于的 tag 平级（scripts/panel-version.sh 产出
// v1.0.0+git8.<hash>）。把提交数写成预发布段会掉到 tag 之前，快照机器上因此恒亮
// 一条"面板有新版本"，指向的却是比在跑的代码更旧的发布。
func parseSemver(value string) (semver, bool) {
	value = strings.TrimPrefix(value, "v")
	value, build, hasBuild := strings.Cut(value, "+")
	if hasBuild && build == "" {
		return semver{}, false
	}
	base, pre, hasPre := strings.Cut(value, "-")
	fields := strings.Split(base, ".")
	if len(fields) != 3 || (hasPre && pre == "") {
		return semver{}, false
	}
	var parsed semver
	parsed.prerelease = hasPre
	for index, field := range fields {
		number, err := strconv.Atoi(field)
		if err != nil || number < 0 || (len(field) > 1 && strings.HasPrefix(field, "0")) {
			return semver{}, false
		}
		parsed.parts[index] = number
	}
	return parsed, true
}

// versionBehind 判断 current 是否落后于 latest。任一方解析失败都按"不落后"
// 处理：提醒宁缺毋滥，解析不了的版本号不该弹出升级横幅。
func versionBehind(current, latest string) bool {
	currentVersion, ok := parseSemver(current)
	if !ok {
		return false
	}
	latestVersion, ok := parseSemver(latest)
	if !ok {
		return false
	}
	for index := range currentVersion.parts {
		if currentVersion.parts[index] != latestVersion.parts[index] {
			return currentVersion.parts[index] < latestVersion.parts[index]
		}
	}
	// 基数相同：当前是预发布而最新是正式版，语义上落后（v1.0.0-rc.1 < v1.0.0）
	return currentVersion.prerelease && !latestVersion.prerelease
}
