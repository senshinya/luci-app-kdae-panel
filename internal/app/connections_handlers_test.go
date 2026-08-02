package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/auth"
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
	timestamp := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	hostService := &stubHostService{
		status: host.Status{MainPID: 42, ActiveState: "active"},
		logs: []host.LogEntry{{
			Timestamp: timestamp,
			Message:   `level=info msg="192.0.2.1:1234 <-> example.com:443" ip=203.0.113.1:443 network=tcp4 outbound=proxy dialer=tokyo`,
		}},
	}
	snapshotter := &stubConnectionSnapshotter{snapshot: daeconn.Snapshot{
		TakenAt:     timestamp.Add(time.Second),
		OutboundTCP: 2,
		UDPSockets:  1,
		Endpoints:   map[string]int{"203.0.113.9:443": 1, "203.0.113.8:443": 3},
	}}
	application, err := NewWithDependencies(Config{Version: "test"}, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{
		Dae:         stubDaeService{},
		Host:        hostService,
		Connections: snapshotter,
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/connections?limit=100", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	var response connectionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if snapshotter.pid != 42 || !response.SnapshotOK || !response.LogsOK || response.Summary.OutboundTCP != 2 || response.Summary.UDPSockets != 1 || response.Summary.WindowEvents != 1 {
		t.Fatalf("响应概况异常: %+v, pid=%d", response, snapshotter.pid)
	}
	if len(response.Endpoints) != 2 || response.Endpoints[0].Address != "203.0.113.8:443" || response.Endpoints[0].Count != 3 {
		t.Fatalf("端点分布异常: %+v", response.Endpoints)
	}
	if len(response.Entries) != 1 || response.Entries[0].Outbound != "proxy" {
		t.Fatalf("响应记录异常: %+v", response.Entries)
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
