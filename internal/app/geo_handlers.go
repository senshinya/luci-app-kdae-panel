package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tuoro/kdae-panel/internal/geodata"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// GeoService 是 geo 数据维护能力的消费者侧接口。
type GeoService interface {
	Status(ctx context.Context) geodata.Status
	Download(ctx context.Context, source upstream.GeoSource) (upstream.GeoData, error)
	Apply(ctx context.Context, data upstream.GeoData) (geodata.Status, error)
}

type GeoSourceService interface {
	CustomSources() []upstream.CustomGeoSource
	CreateCustomSource(source upstream.CustomGeoSource) (upstream.CustomGeoSource, error)
	UpdateCustomSource(id string, source upstream.CustomGeoSource) (upstream.CustomGeoSource, error)
	DeleteCustomSource(id string) error
}

type geoRequest struct {
	Source string `json:"source"`
}

// geoUpdateTimeout 覆盖下载与落盘的总时长。
// geo 有几十兆，而这条路径常常正走在被 dae 接管的链路上，给得比接口超时宽裕。
const geoUpdateTimeout = 10 * time.Minute

// geoUpdater 把手动触发与定时触发收在同一个任务追踪器下：
// 并发防护只有一份，定时轮次的进度与失败原因也会出现在 geo 卡片里。
type geoUpdater struct {
	service    GeoService
	operations *sync.Mutex
	logger     *slog.Logger
	jobs       *installJobs
}

func newGeoUpdater(service GeoService, operations *sync.Mutex, logger *slog.Logger) *geoUpdater {
	return &geoUpdater{
		service:    service,
		operations: operations,
		logger:     logger,
		jobs:       &installJobs{job: Job{Phase: PhaseIdle}},
	}
}

// start 供手动触发：异步执行，apply 阶段等待操作锁——用户正等着结果，宁可排队。
func (u *geoUpdater) start(source upstream.GeoSource) bool {
	if !u.jobs.begin(PhaseDownloading, string(source), "", "geo 数据") {
		return false
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), geoUpdateTimeout)
		defer cancel()
		_ = u.run(ctx, source, func() error {
			u.operations.Lock()
			return nil
		})
	}()
	return true
}

// runScheduled 供定时任务：同步执行，apply 阶段拿不到锁就跳过本轮——
// 调度器会在几分钟后重试，而不是排在一个可能长达数分钟的安装操作后面。
// 来源沿用上次成功更新的那一个（从未更新过则用默认来源）：定时任务
// 静默改换规则集会改变 geosite: 规则的含义，绝不能发生。
func (u *geoUpdater) runScheduled(ctx context.Context) error {
	source := u.service.Status(ctx).DefaultSource
	if !u.jobs.begin(PhaseDownloading, string(source), "", "geo 数据") {
		return errors.New("已有 geo 更新任务正在执行，本轮已跳过")
	}
	return u.run(ctx, source, func() error {
		if !u.operations.TryLock() {
			return errors.New("另一个控制操作正在执行，本轮已跳过")
		}
		return nil
	})
}

// run 执行一次完整更新，锁的获取策略由调用方注入。
func (u *geoUpdater) run(ctx context.Context, source upstream.GeoSource, acquire func() error) error {
	// 下载与校验不触碰任何共享状态，因此不占控制锁：
	// 二十多兆的下载不该把配置保存和订阅定时刷新一起堵住。
	data, err := u.service.Download(ctx, source)
	if err != nil {
		u.logger.Warn("下载 geo 数据失败", "error", err)
		u.jobs.finish(err)
		return err
	}

	u.jobs.advance(PhaseApplying)
	if err := acquire(); err != nil {
		u.jobs.finish(err)
		return err
	}
	defer u.operations.Unlock()
	if _, err := u.service.Apply(ctx, data); err != nil {
		u.logger.Warn("更新 geo 数据失败", "error", err)
		u.jobs.finish(err)
		return err
	}
	u.jobs.finish(nil)
	return nil
}

func registerGeoRoutes(router *http.ServeMux, updater *geoUpdater, sources GeoSourceService) {
	if updater == nil {
		// 生产构造器始终注入 Geo 服务；这个分支只为依赖注入不完整的测试或
		// 定制构建保留，避免端点落成难以理解的 404。
		unavailable := func(writer http.ResponseWriter, _ *http.Request) {
			writeAPIError(writer, http.StatusServiceUnavailable, "geo_update_disabled",
				"Geo 数据服务未初始化，请检查面板启动日志")
		}
		for _, pattern := range []string{
			"GET /api/v1/dae/geo", "POST /api/v1/dae/geo",
			"GET /api/v1/dae/geo/sources", "POST /api/v1/dae/geo/sources",
			"PUT /api/v1/dae/geo/sources/{id}", "DELETE /api/v1/dae/geo/sources/{id}",
		} {
			router.HandleFunc(pattern, unavailable)
		}
		return
	}

	router.HandleFunc("GET /api/v1/dae/geo", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]any{
			"status": updater.service.Status(request.Context()),
			"job":    updater.jobs.snapshot(),
		})
	})

	router.HandleFunc("POST /api/v1/dae/geo", func(writer http.ResponseWriter, request *http.Request) {
		// 来源可省略，此时沿用状态里给出的默认值（上次用过的那个）。
		payload := geoRequest{}
		if !decodeOptionalJSONBody(writer, request, &payload) {
			return
		}
		source := updater.service.Status(request.Context()).DefaultSource
		if payload.Source != "" {
			parsed, err := upstream.ParseGeoSource(payload.Source)
			if err != nil {
				writeAPIError(writer, http.StatusBadRequest, "invalid_geo_source", err.Error())
				return
			}
			if strings.HasPrefix(string(parsed), "custom:") && !geoSourceAvailable(updater.service.Status(request.Context()), parsed) {
				writeAPIError(writer, http.StatusBadRequest, "invalid_geo_source", "自定义 geo 数据来源不存在或已被删除")
				return
			}
			source = parsed
		}
		if !updater.start(source) {
			writeAPIError(writer, http.StatusConflict, "geo_update_in_progress", "已有 geo 更新任务正在执行")
			return
		}
		writeJSON(writer, http.StatusAccepted, map[string]any{"job": updater.jobs.snapshot()})
	})

	registerGeoSourceRoutes(router, updater, sources)
}

func geoSourceAvailable(status geodata.Status, source upstream.GeoSource) bool {
	for _, candidate := range status.Sources {
		if candidate.Source == source {
			return true
		}
	}
	return false
}

func registerGeoSourceRoutes(router *http.ServeMux, updater *geoUpdater, sources GeoSourceService) {
	unavailable := func(writer http.ResponseWriter) bool {
		if sources != nil {
			return false
		}
		writeAPIError(writer, http.StatusServiceUnavailable, "geo_sources_unavailable", "自定义 geo 来源存储未初始化")
		return true
	}
	busy := func(writer http.ResponseWriter) bool {
		job := updater.jobs.snapshot()
		if job.Phase != PhaseDownloading && job.Phase != PhaseApplying {
			return false
		}
		writeAPIError(writer, http.StatusConflict, "geo_update_in_progress", "geo 更新进行中，暂时不能修改来源")
		return true
	}

	router.HandleFunc("GET /api/v1/dae/geo/sources", func(writer http.ResponseWriter, _ *http.Request) {
		if unavailable(writer) {
			return
		}
		customSources := sources.CustomSources()
		if customSources == nil {
			customSources = []upstream.CustomGeoSource{}
		}
		writeJSON(writer, http.StatusOK, map[string]any{"sources": customSources})
	})
	router.HandleFunc("POST /api/v1/dae/geo/sources", func(writer http.ResponseWriter, request *http.Request) {
		if unavailable(writer) || busy(writer) {
			return
		}
		var payload upstream.CustomGeoSource
		if !decodeBody(writer, request, &payload, 64<<10, false) {
			return
		}
		created, err := sources.CreateCustomSource(payload)
		if err != nil {
			writeAPIError(writer, http.StatusBadRequest, "invalid_geo_source", err.Error())
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	router.HandleFunc("PUT /api/v1/dae/geo/sources/{id}", func(writer http.ResponseWriter, request *http.Request) {
		if unavailable(writer) || busy(writer) {
			return
		}
		var payload upstream.CustomGeoSource
		if !decodeBody(writer, request, &payload, 64<<10, false) {
			return
		}
		updated, err := sources.UpdateCustomSource(request.PathValue("id"), payload)
		if err != nil {
			writeGeoSourceError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, updated)
	})
	router.HandleFunc("DELETE /api/v1/dae/geo/sources/{id}", func(writer http.ResponseWriter, request *http.Request) {
		if unavailable(writer) || busy(writer) {
			return
		}
		id := request.PathValue("id")
		status := updater.service.Status(request.Context())
		if status.Managed != nil && status.Managed.Source == upstream.GeoSource("custom:"+id) {
			writeAPIError(writer, http.StatusConflict, "geo_source_in_use",
				"该来源是当前 Geo 数据的记录来源；请先用另一个来源更新成功后再删除")
			return
		}
		if err := sources.DeleteCustomSource(id); err != nil {
			writeGeoSourceError(writer, err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})
}

func writeGeoSourceError(writer http.ResponseWriter, err error) {
	if errors.Is(err, upstream.ErrCustomGeoSourceNotFound) {
		writeAPIError(writer, http.StatusNotFound, "geo_source_not_found", err.Error())
		return
	}
	writeAPIError(writer, http.StatusBadRequest, "invalid_geo_source", err.Error())
}
