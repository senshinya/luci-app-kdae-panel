package app

import (
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/tuoro/kdae-panel/internal/configstore"
	"github.com/tuoro/kdae-panel/internal/daediag"
	"github.com/tuoro/kdae-panel/internal/host"
)

type serviceActionRequest struct {
	Abort bool `json:"abort"`
}

// serviceControlState 记录面板最近一次 suspend 动作。
// dae 没有查询暂停状态的 CLI 接口，因此只能在面板侧记录；启动、停止、重启和重载
// 都会清除它。这个状态是运行时状态，面板重启后归零，避免把一次旧暂停误报成当前状态。
type serviceControlState struct {
	mu        sync.Mutex
	suspended bool
	mainPID   int
}

func (s *serviceControlState) markSuspended(mainPID int) {
	s.mu.Lock()
	s.suspended = true
	s.mainPID = mainPID
	s.mu.Unlock()
}

func (s *serviceControlState) clearSuspended() {
	s.mu.Lock()
	s.suspended = false
	s.mainPID = 0
	s.mu.Unlock()
}

func (s *serviceControlState) isSuspended(status host.Status) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	// systemd 在面板外停止或重启 dae 时，也不能继续显示一次过期的暂停状态。
	if s.suspended && (status.ActiveState != "active" ||
		(s.mainPID > 0 && status.MainPID > 0 && s.mainPID != status.MainPID)) {
		s.suspended = false
		s.mainPID = 0
	}
	return s.suspended
}

type serviceStatusResponse struct {
	host.Status
	Suspended bool `json:"suspended"`
}

func registerServiceRoutes(router *http.ServeMux, daeService DaeService, hostService HostService, operations *sync.Mutex) {
	controlState := &serviceControlState{}
	router.HandleFunc("GET /api/v1/host/interfaces", func(writer http.ResponseWriter, request *http.Request) {
		if hostService == nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "host_service_unavailable", "主机服务管理尚未初始化")
			return
		}
		interfaces, err := hostService.Interfaces(request.Context())
		if err != nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "host_interfaces_unavailable", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, interfaces)
	})

	router.HandleFunc("GET /api/v1/service", func(writer http.ResponseWriter, request *http.Request) {
		if hostService == nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "host_service_unavailable", "主机服务管理尚未初始化")
			return
		}
		status, err := hostService.Status(request.Context())
		if err != nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "service_status_unavailable", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, serviceStatusResponse{
			Status:    status,
			Suspended: controlState.isSuspended(status),
		})
	})

	router.HandleFunc("POST /api/v1/service/actions/{action}", func(writer http.ResponseWriter, request *http.Request) {
		action := request.PathValue("action")
		var payload serviceActionRequest
		if !decodeOptionalJSONBody(writer, request, &payload) {
			return
		}
		if !acquireOperation(writer, operations) {
			return
		}
		defer operations.Unlock()

		var err error
		deferred := false
		startedAt := time.Now().UTC()
		switch action {
		case string(host.ActionStart), string(host.ActionStop), string(host.ActionRestart):
			if hostService == nil {
				writeAPIError(writer, http.StatusServiceUnavailable, "host_service_unavailable", "主机服务管理尚未初始化")
				return
			}
			hostAction := host.Action(action)
			// 用户从面板启动或停止时，同时更新 systemd 的开机状态；版本切换走安装器
			// 内部的普通 start/stop/restart，不会被这里改变。
			switch hostAction {
			case host.ActionStart:
				hostAction = host.ActionEnableNow
			case host.ActionStop:
				hostAction = host.ActionDisableNow
			}
			err = hostService.Action(request.Context(), hostAction)
		case "reload":
			err = daeService.Reload(request.Context())
			if errors.Is(err, configstore.ErrReloadDeferred) {
				err, deferred = nil, true
			}
		case "suspend":
			err = daeService.Suspend(request.Context(), payload.Abort)
		default:
			writeAPIError(writer, http.StatusBadRequest, "unsupported_service_action", "不支持的服务动作: "+action)
			return
		}
		if err != nil {
			if hostService != nil && (action == string(host.ActionStart) || action == string(host.ActionRestart)) {
				if entries, logErr := hostService.Logs(request.Context(), 80); logErr == nil {
					err = daediag.ExplainGeoFailure(err, entries, startedAt)
				}
			}
			writeAPIError(writer, http.StatusBadGateway, "service_action_failed", err.Error())
			return
		}
		switch action {
		case string(host.ActionStart), string(host.ActionStop), string(host.ActionRestart), "reload":
			controlState.clearSuspended()
		case "suspend":
			mainPID := 0
			if hostService != nil {
				if status, statusErr := hostService.Status(request.Context()); statusErr == nil {
					mainPID = status.MainPID
				}
			}
			controlState.markSuspended(mainPID)
		}
		response := map[string]any{
			"status":    "ok",
			"action":    action,
			"suspended": action == "suspend",
			"deferred":  deferred,
		}
		if action == "suspend" {
			response["message"] = "dae 已暂停，代理流量处理已停止；点击无损重载即可恢复"
		} else if action == string(host.ActionStart) {
			response["message"] = "dae 已启动，并已设为随系统启动"
		} else if action == string(host.ActionStop) {
			response["message"] = "dae 已停止，并已取消随系统启动"
		} else if deferred {
			response["message"] = "dae 当前未运行，无需重载；下次启动会读取磁盘配置"
		}
		writeJSON(writer, http.StatusOK, response)
	})

	router.HandleFunc("GET /api/v1/logs", func(writer http.ResponseWriter, request *http.Request) {
		if hostService == nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "host_service_unavailable", "主机服务管理尚未初始化")
			return
		}
		limit := 200
		if rawLimit := request.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed <= 0 {
				writeAPIError(writer, http.StatusBadRequest, "invalid_log_limit", "日志条数必须是正整数")
				return
			}
			limit = parsed
		}
		entries, err := hostService.Logs(request.Context(), limit)
		if err != nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "logs_unavailable", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, entries)
	})

	router.HandleFunc("GET /api/v1/diagnostics/sysdump", func(writer http.ResponseWriter, request *http.Request) {
		archive, err := daeService.Sysdump(request.Context())
		if err != nil {
			writeAPIError(writer, http.StatusBadGateway, "sysdump_failed", err.Error())
			return
		}
		writer.Header().Set("Content-Type", "application/gzip")
		writer.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(archive.Filename))
		writer.Header().Set("Content-Length", strconv.Itoa(len(archive.Content)))
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(archive.Content)
	})
}

// decodeOptionalJSONBody 解码可以整个省略的小请求体。
func decodeOptionalJSONBody(writer http.ResponseWriter, request *http.Request, destination any) bool {
	return decodeBody(writer, request, destination, 64<<10, true)
}
