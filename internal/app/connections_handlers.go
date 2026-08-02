package app

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tuoro/kdae-panel/internal/daeconfig"
	"github.com/tuoro/kdae-panel/internal/daeconn"
	"github.com/tuoro/kdae-panel/internal/host"
)

const (
	connectionsMaxEntries    = 2000
	connectionsDefaultWindow = 15 * time.Minute
	connectionsMaxWindow     = 24 * time.Hour
	connectionFacetLimit     = 200
)

type connectionsSummary struct {
	OutboundTCP   int `json:"outboundTcp"`
	UDPSockets    int `json:"udpSockets"`
	WindowEvents  int `json:"windowEvents"`
	WindowClients int `json:"windowClients"`
	WindowTargets int `json:"windowTargets"`
}

type connectionEndpoint struct {
	Address string `json:"address"`
	Count   int    `json:"count"`
}

type connectionFacet struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Count int    `json:"count"`
	Note  string `json:"note,omitempty"`
}

type connectionFacets struct {
	Targets []connectionFacet `json:"targets"`
	Clients []connectionFacet `json:"clients"`
	Nodes   []connectionFacet `json:"nodes"`
	Groups  []connectionFacet `json:"groups"`
}

type connectionsResponse struct {
	SnapshotAt   time.Time            `json:"snapshotAt"`
	SnapshotOK   bool                 `json:"snapshotOk"`
	LogsOK       bool                 `json:"logsOk"`
	LogLevel     string               `json:"logLevel,omitempty"`
	Dropped      int                  `json:"dropped,omitempty"`
	Truncated    bool                 `json:"truncated,omitempty"`
	FacetLimited bool                 `json:"facetLimited,omitempty"`
	Summary      connectionsSummary   `json:"summary"`
	Facets       connectionFacets     `json:"facets"`
	Endpoints    []connectionEndpoint `json:"endpoints"`
	Entries      []daeconn.Event      `json:"entries"`
}

type connectionTracker struct {
	host          HostService
	configuration ConfigurationService
	snapshotter   daeconn.Snapshotter
	store         *daeconn.Store
}

func registerConnectionRoutes(
	router *http.ServeMux,
	hostService HostService,
	configuration ConfigurationService,
	snapshotter daeconn.Snapshotter,
) {
	if snapshotter == nil {
		snapshotter = daeconn.NewProcSnapshotter()
	}
	tracker := &connectionTracker{
		host: hostService, configuration: configuration, snapshotter: snapshotter, store: daeconn.NewStore(),
	}
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
	window, err := connectionWindow(request)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_connection_window", err.Error())
		return
	}

	logs, logErr := tracker.host.Logs(request.Context(), host.MaxLogLines)
	lines := make([]daeconn.LogLine, len(logs))
	for index, entry := range logs {
		lines[index] = daeconn.LogLine{Timestamp: entry.Timestamp, Message: entry.RawLine()}
	}
	events, dropped := daeconn.Parse(lines)
	merged, storeTruncated := tracker.store.Merge(events)
	now := time.Now().UTC()
	windowed := connectionEventsSince(merged, now.Add(-window))
	facets, clientCount, targetCount, facetLimited := buildConnectionFacets(windowed)

	var snapshot daeconn.Snapshot
	snapshotOK := false
	if status, statusErr := tracker.host.Status(request.Context()); statusErr == nil {
		if taken, snapshotErr := tracker.snapshotter.Snapshot(request.Context(), status.MainPID); snapshotErr == nil {
			snapshot, snapshotOK = taken, true
		}
	}

	listed, responseTruncated := windowed, false
	if len(listed) > limit {
		listed, responseTruncated = listed[:limit], true
	}
	snapshotAt := snapshot.TakenAt
	if snapshotAt.IsZero() {
		snapshotAt = now
	}
	logLevel := ""
	if tracker.configuration != nil {
		if document, configErr := tracker.configuration.Read(request.Context()); configErr == nil {
			logLevel = daeconfig.LogLevel(document.Content)
		}
	}
	writeJSON(writer, http.StatusOK, connectionsResponse{
		SnapshotAt:   snapshotAt,
		SnapshotOK:   snapshotOK,
		LogsOK:       logErr == nil,
		LogLevel:     logLevel,
		Dropped:      dropped,
		Truncated:    storeTruncated || snapshot.Truncated || responseTruncated,
		FacetLimited: facetLimited,
		Summary: connectionsSummary{
			OutboundTCP:   snapshot.OutboundTCP,
			UDPSockets:    snapshot.UDPSockets,
			WindowEvents:  len(windowed),
			WindowClients: clientCount,
			WindowTargets: targetCount,
		},
		Facets:    facets,
		Endpoints: sortedConnectionEndpoints(snapshot.Endpoints),
		Entries:   listed,
	})
}

func connectionWindow(request *http.Request) (time.Duration, error) {
	raw := request.URL.Query().Get("window")
	if raw == "" {
		return connectionsDefaultWindow, nil
	}
	minutes, err := strconv.Atoi(raw)
	if err != nil || minutes < 1 || minutes > int(connectionsMaxWindow/time.Minute) {
		return 0, errors.New("连接时间窗必须是 1 到 1440 之间的分钟数")
	}
	return time.Duration(minutes) * time.Minute, nil
}

func connectionEventsSince(events []daeconn.Event, cutoff time.Time) []daeconn.Event {
	filtered := make([]daeconn.Event, 0, len(events))
	for _, event := range events {
		if !event.Timestamp.Before(cutoff) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func buildConnectionFacets(events []daeconn.Event) (connectionFacets, int, int, bool) {
	targets := countConnectionFacets(events, targetFacet)
	clients := countConnectionFacets(events, clientFacet)
	nodes := countConnectionFacets(events, func(event daeconn.Event) (string, string, string) {
		return event.Dialer, event.Dialer, ""
	})
	groups := countConnectionFacets(events, func(event daeconn.Event) (string, string, string) {
		return event.Outbound, event.Outbound, ""
	})
	limited := len(targets) > connectionFacetLimit || len(clients) > connectionFacetLimit ||
		len(nodes) > connectionFacetLimit || len(groups) > connectionFacetLimit
	return connectionFacets{
		Targets: limitConnectionFacets(targets),
		Clients: limitConnectionFacets(clients),
		Nodes:   limitConnectionFacets(nodes),
		Groups:  limitConnectionFacets(groups),
	}, len(clients), len(targets), limited
}

func countConnectionFacets(events []daeconn.Event, identify func(daeconn.Event) (string, string, string)) []connectionFacet {
	counts := make(map[string]*connectionFacet)
	for _, event := range events {
		id, label, note := identify(event)
		if id == "" || label == "" {
			continue
		}
		facet, exists := counts[id]
		if !exists {
			// Store 按时间倒序返回事件，因此首次出现的标签就是这个身份的最新地址。
			facet = &connectionFacet{ID: id, Label: label, Note: note}
			counts[id] = facet
		}
		facet.Count++
	}
	facets := make([]connectionFacet, 0, len(counts))
	for _, facet := range counts {
		facets = append(facets, *facet)
	}
	sort.Slice(facets, func(left, right int) bool {
		if facets[left].Count != facets[right].Count {
			return facets[left].Count > facets[right].Count
		}
		return facets[left].Label < facets[right].Label
	})
	return facets
}

func targetFacet(event daeconn.Event) (string, string, string) {
	target := event.Sniffed
	if target == "" {
		target = event.Target
	}
	target = strings.ToLower(strings.TrimSuffix(connectionHost(target), "."))
	return target, target, ""
}

func clientFacet(event daeconn.Event) (string, string, string) {
	host := connectionHost(event.Src)
	if host == "" {
		return "", "", ""
	}
	if mac := connectionMAC(event.Mac); mac != "" {
		return "mac:" + mac, host, mac
	}
	return "ip:" + strings.ToLower(host), host, ""
}

func connectionHost(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(host, "[]")
	}
	trimmed := strings.Trim(value, "[]")
	if address, err := netip.ParseAddr(trimmed); err == nil {
		return address.Unmap().String()
	}
	return value
}

func connectionMAC(value string) string {
	hardware, err := net.ParseMAC(strings.TrimSpace(value))
	if err != nil || len(hardware) == 0 || hardware[0]&1 != 0 {
		return ""
	}
	for _, octet := range hardware {
		if octet != 0 {
			return hardware.String()
		}
	}
	return ""
}

func limitConnectionFacets(facets []connectionFacet) []connectionFacet {
	if len(facets) > connectionFacetLimit {
		return facets[:connectionFacetLimit]
	}
	return facets
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
