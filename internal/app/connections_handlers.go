package app

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/tuoro/kdae-panel/internal/daeconn"
	"github.com/tuoro/kdae-panel/internal/host"
)

// connectionsLogWindow 是每次拉取的日志行数，取 host 后端的上限：
// 窗口越大，能从环形缓冲里抢救回来的连接事件越多。
const connectionsLogWindow = host.MaxLogLines

// connectionsMaxEntries 是单次响应的流水条数上限，与存储容量同数量级。
const connectionsMaxEntries = 2000

// connectionsMaxGroups 限制分组结果条目数，防止异常配置把响应撑大。
const connectionsMaxGroups = 100

type connectionsSummary struct {
	// OutboundSockets 是 dae 此刻持有的 ESTABLISHED 出站连接数，实时值。
	OutboundSockets int `json:"outboundSockets"`
	UDPSockets      int `json:"udpSockets"`
	// WindowEvents 是窗口内累积的连接建立事件数，历史值。
	WindowEvents int `json:"windowEvents"`
	ActiveNodes  int `json:"activeNodes"`
}

// connectionsGroup 是一组计数。Key 的含义由所在字段决定：出站端点、节点或出站组。
type connectionsGroup struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type connectionsResponse struct {
	SnapshotAt time.Time          `json:"snapshotAt"`
	SnapshotOK bool               `json:"snapshotOk"`
	LogLevel   string             `json:"logLevel,omitempty"`
	Dropped    int                `json:"dropped,omitempty"`
	Truncated  bool               `json:"truncated,omitempty"`
	Summary    connectionsSummary `json:"summary"`
	// Endpoints 是 dae 当前出站连接按远端 ip:port 的分组，来自 socket，实时。
	Endpoints []connectionsGroup `json:"endpoints"`
	// Nodes、Groups 是窗口内新建连接按节点、按出站组的分组，来自日志，历史。
	Nodes   []connectionsGroup `json:"nodes"`
	Groups  []connectionsGroup `json:"groups"`
	Entries []daeconn.Event    `json:"entries"`
}

// registerConnectionRoutes 注册连接活动端点。数据全部来自公开来源：
// 服务日志给出每条连接的元数据，/proc/net 给出 dae 当前的出站连接分布。
// 不提供逐条存活判定——dae 的 eBPF 数据面不暴露客户端侧连接状态。
func registerConnectionRoutes(router *http.ServeMux, hostService HostService, configuration ConfigurationService) {
	store := daeconn.NewStore()
	router.HandleFunc("GET /api/v1/connections", func(writer http.ResponseWriter, request *http.Request) {
		if hostService == nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "host_service_unavailable", "主机服务管理尚未初始化")
			return
		}
		limit := connectionsMaxEntries
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
		entries, err := hostService.Logs(request.Context(), connectionsLogWindow)
		if err != nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "logs_unavailable", err.Error())
			return
		}
		events, dropped := daeconn.ParseEntries(entries)
		merged := store.Merge(events)

		var snapshot daeconn.Snapshot
		snapshotOK := false
		if status, statusErr := hostService.Status(request.Context()); statusErr == nil {
			if taken, snapErr := daeconn.TakeSnapshot(status.MainPID); snapErr == nil {
				snapshot, snapshotOK = taken, true
			}
		}

		nodes := countBy(merged, func(event daeconn.Event) string { return event.Dialer })
		groups := countBy(merged, func(event daeconn.Event) string { return event.Outbound })
		summary := connectionsSummary{
			OutboundSockets: snapshot.TCPSockets,
			UDPSockets:      snapshot.UDPSockets,
			WindowEvents:    len(merged),
			ActiveNodes:     len(nodes),
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
		listed := merged
		truncated := snapshot.Truncated
		if len(listed) > limit {
			listed, truncated = listed[:limit], true
		}
		writeJSON(writer, http.StatusOK, connectionsResponse{
			SnapshotAt: snapshotAt,
			SnapshotOK: snapshotOK,
			LogLevel:   logLevel,
			Dropped:    dropped,
			Truncated:  truncated,
			Summary:    summary,
			Endpoints:  sortGroups(snapshot.Outbound),
			Nodes:      nodes,
			Groups:     groups,
			Entries:    listed,
		})
	})
}

// countBy 按 key 统计事件数，空 key 跳过。结果按数量倒序，同数按名称。
func countBy(events []daeconn.Event, key func(daeconn.Event) string) []connectionsGroup {
	counts := map[string]int{}
	for _, event := range events {
		if name := key(event); name != "" {
			counts[name]++
		}
	}
	return sortGroups(counts)
}

func sortGroups(counts map[string]int) []connectionsGroup {
	groups := make([]connectionsGroup, 0, len(counts))
	for key, count := range counts {
		groups = append(groups, connectionsGroup{Key: key, Count: count})
	}
	sort.Slice(groups, func(left, right int) bool {
		if groups[left].Count != groups[right].Count {
			return groups[left].Count > groups[right].Count
		}
		return groups[left].Key < groups[right].Key
	})
	if len(groups) > connectionsMaxGroups {
		groups = groups[:connectionsMaxGroups]
	}
	return groups
}
