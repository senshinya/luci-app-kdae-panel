package app

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/tuoro/kdae-panel/internal/daediag"
	"github.com/tuoro/kdae-panel/internal/host"
)

type serviceActionRequest struct {
	Abort bool `json:"abort"`
}

func registerServiceRoutes(router *http.ServeMux, daeService DaeService, hostService HostService, operations *sync.Mutex) {
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
		writeJSON(writer, http.StatusOK, status)
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
		startedAt := time.Now().UTC()
		switch action {
		case string(host.ActionStart), string(host.ActionStop), string(host.ActionRestart):
			if hostService == nil {
				writeAPIError(writer, http.StatusServiceUnavailable, "host_service_unavailable", "主机服务管理尚未初始化")
				return
			}
			err = hostService.Action(request.Context(), host.Action(action))
		case "reload":
			err = daeService.Reload(request.Context())
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
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok", "action": action})
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
