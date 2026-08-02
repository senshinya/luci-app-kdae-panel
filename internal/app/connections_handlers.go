package app

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
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

// connectionsGroup 是一组计数。Key 的含义由所在字段决定：客户端、域名、节点或出站组。
type connectionsGroup struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
	// Note 是补充说明，目前只有客户端分组用它带上 MAC。
	Note string `json:"note,omitempty"`
}

type connectionsResponse struct {
	SnapshotAt time.Time          `json:"snapshotAt"`
	SnapshotOK bool               `json:"snapshotOk"`
	LogLevel   string             `json:"logLevel,omitempty"`
	Dropped    int                `json:"dropped,omitempty"`
	Truncated  bool               `json:"truncated,omitempty"`
	Summary    connectionsSummary `json:"summary"`
	// 以下四组都是窗口内新建连接的分组，来自日志。
	// 不按 dae 当前出站 socket 的远端分组：那张表的键是代理服务器地址，
	// 有几个节点就只有几行，说的和 summary.outboundSockets 是同一件事。
	Clients []connectionsGroup `json:"clients"`
	Domains []connectionsGroup `json:"domains"`
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

		clients := countClients(merged)
		domains := countBy(merged, destinationName)
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
			Clients:    clients,
			Domains:    domains,
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

// countClients 按客户端地址统计，并带上该地址最近一次出现的 MAC。
// 局域网里 IP 会变而 MAC 不变，带上它才能认出是哪台设备。
func countClients(events []daeconn.Event) []connectionsGroup {
	counts := map[string]int{}
	macs := map[string]string{}
	for _, event := range events {
		address := hostOnly(event.Src)
		if address == "" {
			continue
		}
		counts[address]++
		// 事件已按时间倒序，第一次见到的就是最近一次。
		if _, known := macs[address]; !known && event.Mac != "" && event.Mac != "00:00:00:00:00:00" {
			macs[address] = event.Mac
		}
	}
	groups := sortGroups(counts)
	for index := range groups {
		groups[index].Note = macs[groups[index].Key]
	}
	return groups
}

// destinationName 取一条连接的可读目的地：优先嗅探到的域名，没有就用拨号目标的主机部分。
func destinationName(event daeconn.Event) string {
	if event.Sniffed != "" {
		return event.Sniffed
	}
	return hostOnly(event.Target)
}

// hostOnly 去掉 "host:port" 的端口部分，兼容 "[v6]:port" 形态。
func hostOnly(value string) string {
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "[") {
		if end := strings.LastIndex(value, "]"); end > 0 {
			return value[1:end]
		}
		return value
	}
	if index := strings.LastIndex(value, ":"); index > 0 {
		return value[:index]
	}
	return value
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
