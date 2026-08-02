package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/auth"
	"github.com/tuoro/kdae-panel/internal/configstore"
	"github.com/tuoro/kdae-panel/internal/daeconn"
	"github.com/tuoro/kdae-panel/internal/host"
)

type stubConnectionSnapshotter struct {
	snapshot daeconn.Snapshot
	err      error
	pid      int
}

func (snapshotter *stubConnectionSnapshotter) Snapshot(_ context.Context, pid int) (daeconn.Snapshot, error) {
	snapshotter.pid = pid
	return snapshotter.snapshot, snapshotter.err
}

func TestConnectionsEndpoint(t *testing.T) {
	timestamp := time.Now().UTC().Add(-time.Minute)
	hostService := &stubHostService{
		status: host.Status{MainPID: 42, ActiveState: "active"},
		logs: []host.LogEntry{
			{
				Timestamp: timestamp,
				Message:   `level=info msg="192.0.2.2:1234 <-> example.com:443" ip=203.0.113.1:443 sniffed=Example.COM. network=tcp4 outbound=proxy dialer=tokyo mac=02:00:00:00:00:01`,
			},
			{
				Timestamp: timestamp.Add(-time.Minute),
				Message:   `level=info msg="192.0.2.1:1235 <-> example.com:443" ip=203.0.113.1:443 network=tcp4 outbound=proxy dialer=tokyo mac=02:00:00:00:00:01`,
			},
		},
	}
	snapshotter := &stubConnectionSnapshotter{snapshot: daeconn.Snapshot{
		TakenAt:        timestamp.Add(time.Second),
		OutboundTCP:    2,
		UDPSockets:     1,
		SampledTCPPeak: 4,
		SampledUDPPeak: 2,
		Endpoints:      map[string]int{"203.0.113.9:443": 1, "203.0.113.8:443": 3},
	}}
	application, err := NewWithDependencies(Config{Version: "test"}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{
		Dae:           stubDaeService{},
		Host:          hostService,
		Configuration: stubConfigurationService{document: configstore.Document{Content: "global { log_level: warn }"}},
		Connections:   snapshotter,
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/connections?limit=100&window=15", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	var response connectionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if snapshotter.pid != 42 || !response.SnapshotOK || !response.ServiceRunning ||
		response.SocketWindowSeconds != int(daeconn.RecentSampleWindow/time.Second) || !response.LogsOK || response.LogLevel != "warn" ||
		response.Summary.OutboundTCP != 2 || response.Summary.UDPSockets != 1 ||
		response.Summary.SampledTCPPeak != 4 || response.Summary.SampledUDPPeak != 2 || response.Summary.WindowEvents != 2 ||
		response.Summary.WindowClients != 1 || response.Summary.WindowTargets != 1 {
		t.Fatalf("响应概况异常: %+v, pid=%d", response, snapshotter.pid)
	}
	if len(response.Endpoints) != 2 || response.Endpoints[0].Address != "203.0.113.8:443" || response.Endpoints[0].Count != 3 {
		t.Fatalf("端点分布异常: %+v", response.Endpoints)
	}
	if len(response.Entries) != 2 || response.Entries[0].Outbound != "proxy" {
		t.Fatalf("响应记录异常: %+v", response.Entries)
	}
	if len(response.Facets.Targets) != 1 || response.Facets.Targets[0].Label != "example.com" || response.Facets.Targets[0].Count != 2 {
		t.Fatalf("目标分布异常: %+v", response.Facets.Targets)
	}
	if len(response.Facets.Clients) != 1 || response.Facets.Clients[0].Label != "192.0.2.2" ||
		response.Facets.Clients[0].Note != "02:00:00:00:00:01" || response.Facets.Clients[0].Count != 2 {
		t.Fatalf("客户端未按 MAC 合并或没有保留最新 IP: %+v", response.Facets.Clients)
	}
	if len(response.Facets.Nodes) != 1 || response.Facets.Nodes[0].Label != "tokyo" || response.Facets.Nodes[0].Count != 2 ||
		len(response.Facets.Groups) != 1 || response.Facets.Groups[0].Label != "proxy" || response.Facets.Groups[0].Count != 2 {
		t.Fatalf("路由分布异常: nodes=%+v groups=%+v", response.Facets.Nodes, response.Facets.Groups)
	}
}

func TestConnectionsEndpointDistinguishesStoppedServiceFromEmptySnapshot(t *testing.T) {
	snapshotter := &stubConnectionSnapshotter{snapshot: daeconn.Snapshot{Endpoints: map[string]int{}}}
	application, err := NewWithDependencies(Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{
		Dae: stubDaeService{}, Host: &stubHostService{status: host.Status{}}, Connections: snapshotter,
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil))
	var response connectionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || response.ServiceRunning || !response.SnapshotOK || snapshotter.pid != 0 {
		t.Fatalf("停止状态被误报为实时零连接: status=%d response=%+v pid=%d", recorder.Code, response, snapshotter.pid)
	}
}

func TestConnectionsEndpointRejectsInvalidLimit(t *testing.T) {
	application, err := NewWithDependencies(Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{
		Dae:  stubDaeService{},
		Host: &stubHostService{},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"0", "2001", "invalid"} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/connections?limit="+value, nil)
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("limit=%q 状态码 = %d，响应 = %s", value, recorder.Code, recorder.Body.String())
		}
	}
	for _, value := range []string{"0", "1441", "999999999999999999", "invalid"} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/connections?window="+value, nil)
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("window=%q 状态码 = %d，响应 = %s", value, recorder.Code, recorder.Body.String())
		}
	}
}

func TestConnectionsEndpointAppliesWindowBeforeFacetsAndLimit(t *testing.T) {
	now := time.Now().UTC()
	hostService := &stubHostService{logs: []host.LogEntry{
		{Timestamp: now.Add(-time.Minute), Message: `level=info msg="192.0.2.1:1 <-> recent.example:443" ip=203.0.113.1:443 network=tcp4 outbound=proxy`},
		{Timestamp: now.Add(-20 * time.Minute), Message: `level=info msg="192.0.2.2:2 <-> old.example:443" ip=203.0.113.2:443 network=tcp4 outbound=direct`},
	}}
	application, err := NewWithDependencies(Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{
		Dae: stubDaeService{}, Host: hostService,
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/connections?window=5&limit=1", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	var response connectionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Summary.WindowEvents != 1 || len(response.Entries) != 1 || len(response.Facets.Targets) != 1 ||
		response.Facets.Targets[0].Label != "recent.example" || len(response.Facets.Groups) != 1 || response.Facets.Groups[0].Label != "proxy" {
		t.Fatalf("时间窗没有在分布和列表之前生效: %+v", response)
	}
}

func TestConnectionHostAndMACNormalization(t *testing.T) {
	for input, expected := range map[string]string{
		"example.com:443":  "example.com",
		"[2001:db8::1]:53": "2001:db8::1",
		"2001:db8::1":      "2001:db8::1",
		"[2001:db8::2]":    "2001:db8::2",
	} {
		if actual := connectionHost(input); actual != expected {
			t.Errorf("connectionHost(%q) = %q, want %q", input, actual, expected)
		}
	}
	if actual := connectionMAC("02-00-00-00-00-01"); actual != "02:00:00:00:00:01" {
		t.Fatalf("正常 MAC = %q", actual)
	}
	for _, invalid := range []string{"", "00:00:00:00:00:00", "ff:ff:ff:ff:ff:ff", "invalid"} {
		if actual := connectionMAC(invalid); actual != "" {
			t.Errorf("无效 MAC %q = %q", invalid, actual)
		}
	}
}

func TestBuildConnectionFacetsLimitsOnlyPayload(t *testing.T) {
	events := make([]daeconn.Event, connectionFacetLimit+1)
	for index := range events {
		events[index] = daeconn.Event{Target: fmt.Sprintf("target-%03d.example:443", index)}
	}
	facets, clientCount, targetCount, limited := buildConnectionFacets(events)
	if !limited || len(facets.Targets) != connectionFacetLimit || targetCount != connectionFacetLimit+1 || clientCount != 0 {
		t.Fatalf("分布上限误伤摘要计数: limited=%v targets=%d/%d clients=%d", limited, len(facets.Targets), targetCount, clientCount)
	}
}

func TestConnectionsEndpointDegradesWhenSnapshotFails(t *testing.T) {
	hostService := &stubHostService{
		status: host.Status{MainPID: 42},
		logs:   []host.LogEntry{{Message: `level=info msg="192.0.2.1:1 <-> example.com:443" ip=203.0.113.1:443 network=tcp4 outbound=proxy`}},
	}
	application, err := NewWithDependencies(Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{
		Dae:         stubDaeService{},
		Host:        hostService,
		Connections: &stubConnectionSnapshotter{err: errors.New("procfs 不可读")},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	var response connectionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.SnapshotOK || !response.LogsOK || len(response.Entries) != 1 || len(response.Endpoints) != 0 {
		t.Fatalf("快照失败影响了日志流水或伪造了端点: %+v", response)
	}
}

func TestConnectionsEndpointKeepsSnapshotWhenLogsFail(t *testing.T) {
	timestamp := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	application, err := NewWithDependencies(Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{
		Dae:  stubDaeService{},
		Host: &stubHostService{status: host.Status{MainPID: 42}, logsErr: errors.New("journald 不可读")},
		Connections: &stubConnectionSnapshotter{snapshot: daeconn.Snapshot{
			TakenAt: timestamp, OutboundTCP: 4, Endpoints: map[string]int{"203.0.113.8:443": 4},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	var response connectionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.LogsOK || !response.SnapshotOK || response.Summary.OutboundTCP != 4 || len(response.Endpoints) != 1 {
		t.Fatalf("日志失败时未保留 socket 快照: %+v", response)
	}
}

func TestConnectionsEndpointRequiresHostService(t *testing.T) {
	application, err := NewWithDependencies(Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{Dae: stubDaeService{}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
}

func TestConnectionsEndpointRequiresAuthentication(t *testing.T) {
	session := auth.Session{
		Token: "session", CSRFToken: "csrf", ExpiresAt: time.Now().Add(time.Hour),
		User: auth.User{ID: 1, Username: "admin"},
	}
	application, err := NewWithDependencies(Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{
		Dae:            stubDaeService{},
		Host:           &stubHostService{},
		Connections:    &stubConnectionSnapshotter{},
		Authentication: &stubAuthenticationService{initialized: true, session: session},
	})
	if err != nil {
		t.Fatal(err)
	}

	anonymous := httptest.NewRecorder()
	application.Handler().ServeHTTP(anonymous, httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("未登录状态码 = %d", anonymous.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session.Token})
	authorized := httptest.NewRecorder()
	application.Handler().ServeHTTP(authorized, request)
	if authorized.Code != http.StatusOK {
		t.Fatalf("已登录状态码 = %d，响应 = %s", authorized.Code, authorized.Body.String())
	}
}
