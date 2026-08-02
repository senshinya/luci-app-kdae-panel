package app

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tuoro/kdae-panel/internal/daeconn"
	"github.com/tuoro/kdae-panel/internal/host"
)

// connectionsLogWindow 是每次对账拉取的日志行数，取 host 后端的上限：
// 窗口越大，能从环形缓冲里抢救回来的连接事件越多。
const connectionsLogWindow = host.MaxLogLines

// connectionsMaxEntries 是单次响应的记录数上限，与存储容量同数量级。
const connectionsMaxEntries = 2000

type connectionsSummary struct {
	LiveTCP      int `json:"liveTcp"`
	TCPSockets   int `json:"tcpSockets"`
	UDPSockets   int `json:"udpSockets"`
	WindowEvents int `json:"windowEvents"`
}

type connectionsResponse struct {
	SnapshotAt time.Time          `json:"snapshotAt"`
	SnapshotOK bool               `json:"snapshotOk"`
	LogLevel   string             `json:"logLevel,omitempty"`
	Dropped    int                `json:"dropped,omitempty"`
	// Truncated 表示有记录未被逐条列出（超过 limit，或入站腿超过快照上限）。
	// summary 里的计数不受影响，仍是完整的。
	Truncated bool               `json:"truncated,omitempty"`
	Summary   connectionsSummary `json:"summary"`
	Entries   []daeconn.Record   `json:"entries"`
}

// registerConnectionRoutes 注册连接活动端点。数据全部来自公开接口：
// 服务日志（元数据）与 /proc/net（存活证据），不触碰 dae 内部状态。
func registerConnectionRoutes(router *http.ServeMux, hostService HostService, configuration ConfigurationService) {
	store := daeconn.NewStore()
	router.HandleFunc("GET /api/v1/connections", func(writer http.ResponseWriter, request *http.Request) {
		if hostService == nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "host_service_unavailable", "主机服务管理尚未初始化")
			return
		}
		limit := connectionsLogWindow
		if rawLimit := request.URL.Query().Get("limit"); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed <= 0 {
				writeAPIError(writer, http.StatusBadRequest, "invalid_connection_limit", "连接条数必须是正整数")
				return
			}
			if parsed > connectionsMaxEntries {
				parsed = connectionsMaxEntries
			}
			limit = parsed
		}
		// 日志窗口始终拉满：limit 只裁剪响应，不该缩小合并进存储的事件范围。
		entries, err := hostService.Logs(request.Context(), connectionsLogWindow)
		if err != nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "logs_unavailable", err.Error())
			return
		}
		events, dropped := daeconn.ParseEntries(entries)

		var snapshot daeconn.Snapshot
		snapshotOK := false
		if status, statusErr := hostService.Status(request.Context()); statusErr == nil {
			if taken, snapErr := daeconn.TakeSnapshot(status.MainPID); snapErr == nil {
				snapshot, snapshotOK = taken, true
			}
		}
		records := store.Reconcile(events, snapshot, snapshotOK)

		summary := connectionsSummary{TCPSockets: snapshot.TCPSockets, UDPSockets: snapshot.UDPSockets}
		for _, record := range records {
			switch record.Status {
			case daeconn.StatusLive, daeconn.StatusOrphan:
				summary.LiveTCP++
			}
			if record.Status != daeconn.StatusOrphan {
				summary.WindowEvents++
			}
		}

		logLevel := ""
		if configuration != nil {
			if document, readErr := configuration.Read(request.Context()); readErr == nil {
				logLevel = daeconn.LogLevelFromConfig(document.Content)
				if logLevel == "" {
					logLevel = "info" // dae 的默认级别，配置未写时生效
				}
			}
		}

		snapshotAt := snapshot.TakenAt
		if snapshotAt.IsZero() {
			snapshotAt = time.Now().UTC()
		}
		listed, truncated := truncateConnectionRecords(records, limit)
		writeJSON(writer, http.StatusOK, connectionsResponse{
			SnapshotAt: snapshotAt,
			SnapshotOK: snapshotOK,
			LogLevel:   logLevel,
			Dropped:    dropped,
			Truncated:  truncated || snapshot.Truncated,
			Summary:    summary,
			Entries:    listed,
		})
	})
}

// truncateConnectionRecords 把响应裁剪到 limit 条，存活与孤儿记录优先占用
// 配额：它们是"存活中"视图的全部内容，按"最近 N 条"一刀切会让那个视图残缺。
// 但优先不等于无限——存活记录的数量由局域网流量决定，不设上限就等于把响应
// 大小交给外部控制。records 已按时间倒序，裁剪保持原有顺序。
// 返回 truncated 表示有记录未被列出，此时 summary 里的计数仍然是完整的。
func truncateConnectionRecords(records []daeconn.Record, limit int) ([]daeconn.Record, bool) {
	if len(records) <= limit {
		return records, false
	}
	alive := 0
	for _, record := range records {
		if isAliveRecord(record) {
			alive++
		}
	}
	// 存活记录本身超额时，只保留最新的 limit 条，其余一并让位。
	aliveQuota := min(alive, limit)
	otherQuota := limit - aliveQuota
	kept := make([]daeconn.Record, 0, limit)
	for _, record := range records {
		if isAliveRecord(record) {
			if aliveQuota > 0 {
				kept = append(kept, record)
				aliveQuota--
			}
			continue
		}
		if otherQuota > 0 {
			kept = append(kept, record)
			otherQuota--
		}
	}
	return kept, true
}

func isAliveRecord(record daeconn.Record) bool {
	return record.Status == daeconn.StatusLive || record.Status == daeconn.StatusOrphan
}
