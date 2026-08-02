package app

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/tuoro/kdae-panel/internal/daeconn"
	"github.com/tuoro/kdae-panel/internal/host"
)

const connectionsMaxEntries = 2000

type connectionsSummary struct {
	OutboundTCP  int `json:"outboundTcp"`
	UDPSockets   int `json:"udpSockets"`
	WindowEvents int `json:"windowEvents"`
}

type connectionEndpoint struct {
	Address string `json:"address"`
	Count   int    `json:"count"`
}

type connectionsResponse struct {
	SnapshotAt time.Time            `json:"snapshotAt"`
	SnapshotOK bool                 `json:"snapshotOk"`
	LogsOK     bool                 `json:"logsOk"`
	Dropped    int                  `json:"dropped,omitempty"`
	Truncated  bool                 `json:"truncated,omitempty"`
	Summary    connectionsSummary   `json:"summary"`
	Endpoints  []connectionEndpoint `json:"endpoints"`
	Entries    []daeconn.Event      `json:"entries"`
}

type connectionTracker struct {
	host        HostService
	snapshotter daeconn.Snapshotter
	store       *daeconn.Store
}

func registerConnectionRoutes(router *http.ServeMux, hostService HostService, snapshotter daeconn.Snapshotter) {
	if snapshotter == nil {
		snapshotter = daeconn.NewProcSnapshotter()
	}
	tracker := &connectionTracker{host: hostService, snapshotter: snapshotter, store: daeconn.NewStore()}
	router.HandleFunc("GET /api/v1/connections", tracker.handle)
}

// handle 分别采集历史流水和实时出站端点。任一来源临时不可用时保留另一边，
// 由 LogsOK / SnapshotOK 明确告诉前端，避免一处故障让整页失效。
func (tracker *connectionTracker) handle(writer http.ResponseWriter, request *http.Request) {
	if tracker.host == nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "host_service_unavailable", "主机服务管理尚未初始化")
		return
	}
	limit, err := connectionLimit(request)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_connection_limit", err.Error())
		return
	}

	logs, logErr := tracker.host.Logs(request.Context(), host.MaxLogLines)
	lines := make([]daeconn.LogLine, len(logs))
	for index, entry := range logs {
		lines[index] = daeconn.LogLine{Timestamp: entry.Timestamp, Message: entry.Message}
	}
	events, dropped := daeconn.Parse(lines)
	merged, storeTruncated := tracker.store.Merge(events)

	var snapshot daeconn.Snapshot
	snapshotOK := false
	if status, statusErr := tracker.host.Status(request.Context()); statusErr == nil {
		if taken, snapshotErr := tracker.snapshotter.Snapshot(request.Context(), status.MainPID); snapshotErr == nil {
			snapshot, snapshotOK = taken, true
		}
	}

	listed, responseTruncated := merged, false
	if len(listed) > limit {
		listed, responseTruncated = listed[:limit], true
	}
	snapshotAt := snapshot.TakenAt
	if snapshotAt.IsZero() {
		snapshotAt = time.Now().UTC()
	}
	writeJSON(writer, http.StatusOK, connectionsResponse{
		SnapshotAt: snapshotAt,
		SnapshotOK: snapshotOK,
		LogsOK:     logErr == nil,
		Dropped:    dropped,
		Truncated:  storeTruncated || snapshot.Truncated || responseTruncated,
		Summary: connectionsSummary{
			OutboundTCP:  snapshot.OutboundTCP,
			UDPSockets:   snapshot.UDPSockets,
			WindowEvents: len(merged),
		},
		Endpoints: sortedConnectionEndpoints(snapshot.Endpoints),
		Entries:   listed,
	})
}

func connectionLimit(request *http.Request) (int, error) {
	raw := request.URL.Query().Get("limit")
	if raw == "" {
		return host.MaxLogLines, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > connectionsMaxEntries {
		return 0, errors.New("连接条数必须是 1 到 2000 之间的整数")
	}
	return limit, nil
}

func sortedConnectionEndpoints(counts map[string]int) []connectionEndpoint {
	endpoints := make([]connectionEndpoint, 0, len(counts))
	for address, count := range counts {
		endpoints = append(endpoints, connectionEndpoint{Address: address, Count: count})
	}
	sort.Slice(endpoints, func(left, right int) bool {
		if endpoints[left].Count != endpoints[right].Count {
			return endpoints[left].Count > endpoints[right].Count
		}
		return endpoints[left].Address < endpoints[right].Address
	})
	return endpoints
}
