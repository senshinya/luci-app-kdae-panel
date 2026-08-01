package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/auth"
	"github.com/tuoro/kdae-panel/internal/configstore"
	"github.com/tuoro/kdae-panel/internal/dae"
	"github.com/tuoro/kdae-panel/internal/daeinstall"
	"github.com/tuoro/kdae-panel/internal/geodata"
	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/netprobe"
	"github.com/tuoro/kdae-panel/internal/panelupdate"
	"github.com/tuoro/kdae-panel/internal/schedule"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

type stubDaeService struct {
	report  dae.Report
	outline dae.Outline
	err     error
}

type stubConfigurationService struct {
	document configstore.Document
	saveErr  error
}

func (s stubConfigurationService) Read(_ context.Context) (configstore.Document, error) {
	return s.document, nil
}

func (s stubConfigurationService) Validate(_ context.Context, _ string) error {
	return nil
}

func (s stubConfigurationService) Save(_ context.Context, _, _ string, _ bool) (configstore.SaveResult, error) {
	return configstore.SaveResult{}, s.saveErr
}

func (s stubConfigurationService) ListBackups(_ context.Context) ([]configstore.Backup, error) {
	return []configstore.Backup{}, nil
}

func (s stubConfigurationService) CreateBackup(_ context.Context, name, note string) (configstore.Backup, error) {
	return configstore.Backup{Name: name, Note: note}, nil
}

func (s stubConfigurationService) UpdateBackup(_ context.Context, id, name, note string) (configstore.Backup, error) {
	return configstore.Backup{ID: id, Name: name, Note: note}, nil
}

func (s stubConfigurationService) DeleteBackup(_ context.Context, _ string) error {
	return nil
}

func (s stubConfigurationService) PreviewBackup(_ context.Context, _ string) (configstore.BackupPreview, error) {
	return configstore.BackupPreview{Valid: true, CurrentHash: s.document.Hash}, nil
}

func (s stubConfigurationService) Restore(_ context.Context, _, _ string, _ bool) (configstore.SaveResult, error) {
	return configstore.SaveResult{}, nil
}

func (s stubDaeService) Inspect(_ context.Context) dae.Report {
	return s.report
}

func (s stubDaeService) Outline(_ context.Context) (dae.Outline, error) {
	return s.outline, s.err
}

func (s stubDaeService) Validate(_ context.Context, _ string) error {
	return s.err
}

func (s stubDaeService) Reload(_ context.Context) error {
	return s.err
}

func (s stubDaeService) Suspend(_ context.Context, _ bool) error {
	return s.err
}

func (s stubDaeService) Sysdump(_ context.Context) (dae.Sysdump, error) {
	return dae.Sysdump{Filename: "dae-sysdump.test.tar.gz", Content: []byte("test dump")}, s.err
}

type stubHostService struct {
	status     host.Status
	interfaces []host.NetworkInterface
	actions    []host.Action
	err        error
	logs       []host.LogEntry
	logsErr    error
}

type stubAuthenticationService struct {
	initialized bool
	session     auth.Session
	err         error
	setupCalls  int
}

func (s *stubAuthenticationService) Initialized(_ context.Context) (bool, error) {
	return s.initialized, s.err
}

func (s *stubAuthenticationService) Setup(_ context.Context, _, _ string) (auth.Session, error) {
	s.setupCalls++
	return s.session, s.err
}

func (s *stubAuthenticationService) Login(_ context.Context, _, _ string) (auth.Session, error) {
	return s.session, s.err
}

func (s *stubAuthenticationService) GetSession(_ context.Context, token string) (auth.Session, error) {
	if s.err != nil || token != s.session.Token {
		return auth.Session{}, auth.ErrInvalidSession
	}
	return s.session, nil
}

func (s *stubAuthenticationService) Logout(_ context.Context, _ string) error {
	return s.err
}

func (s *stubAuthenticationService) ChangePassword(_ context.Context, _ int64, _, _ string) (auth.Session, error) {
	return s.session, s.err
}

func (s *stubHostService) Status(_ context.Context) (host.Status, error) {
	return s.status, s.err
}

func (s *stubHostService) Action(_ context.Context, action host.Action) error {
	s.actions = append(s.actions, action)
	return s.err
}

func (s *stubHostService) Logs(_ context.Context, _ int) ([]host.LogEntry, error) {
	return append([]host.LogEntry(nil), s.logs...), s.logsErr
}

func (s *stubHostService) Interfaces(_ context.Context) ([]host.NetworkInterface, error) {
	return s.interfaces, s.err
}

func TestHealth(t *testing.T) {
	application, err := NewWithDae(
		Config{Version: "test-version"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		stubDaeService{},
	)
	if err != nil {
		t.Fatalf("初始化应用失败: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		Backend string `json:"backend"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if response.Status != "ok" || response.Version != "test-version" {
		t.Fatalf("响应内容异常: %+v", response)
	}
	// 后端选错的症状（服务控制全部失败）离原因很远，健康检查必须直接说出结论。
	if response.Backend != "systemd" && response.Backend != "procd" {
		t.Fatalf("backend = %q，期望 systemd 或 procd", response.Backend)
	}
}

func TestUnknownAPIUsesJSONNotFound(t *testing.T) {
	application, err := NewWithDae(
		Config{Version: "test-version"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		stubDaeService{},
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("Content-Type = %q", contentType)
	}
}

func TestDaeCapabilities(t *testing.T) {
	service := stubDaeService{report: dae.Report{
		Available:  true,
		Version:    "test-dae",
		DetectedAt: time.Unix(1, 0).UTC(),
		Commands:   map[string]bool{"reload": true},
	}}
	application, err := NewWithDae(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		service,
	)
	if err != nil {
		t.Fatalf("初始化应用失败: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/dae/capabilities", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d", recorder.Code, http.StatusOK)
	}
	var response dae.Report
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !response.Available || response.Version != "test-dae" || !response.Commands["reload"] {
		t.Fatalf("响应内容异常: %+v", response)
	}
}

func TestConfigurationConflictResponse(t *testing.T) {
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{
			Dae:           stubDaeService{},
			Configuration: stubConfigurationService{saveErr: configstore.ErrConflict},
		},
	)
	if err != nil {
		t.Fatalf("初始化应用失败: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/config",
		strings.NewReader(`{"content":"test","expectedHash":"stale"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("状态码 = %d，期望 %d，响应 = %s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if response.Error.Code != "configuration_conflict" {
		t.Fatalf("错误码 = %q", response.Error.Code)
	}
}

func TestConfigurationBackupMetadataRoutes(t *testing.T) {
	dir := t.TempDir()
	entryPath := filepath.Join(dir, "config.dae")
	if err := os.WriteFile(entryPath, []byte("global {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	configuration, err := configstore.NewManager(entryPath, filepath.Join(dir, "backups"), stubDaeService{})
	if err != nil {
		t.Fatal(err)
	}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Configuration: configuration},
	)
	if err != nil {
		t.Fatal(err)
	}

	create := httptest.NewRequest(http.MethodPost, "/api/v1/config/backups", strings.NewReader(`{"name":"稳定配置","note":"切换前"}`))
	create.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	application.Handler().ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("创建存档状态码 = %d，响应 = %s", created.Code, created.Body.String())
	}
	var backup configstore.Backup
	if err := json.NewDecoder(created.Body).Decode(&backup); err != nil {
		t.Fatal(err)
	}
	if backup.Name != "稳定配置" || backup.Note != "切换前" || backup.ID == "" {
		t.Fatalf("创建存档响应异常: %+v", backup)
	}

	update := httptest.NewRequest(http.MethodPut, "/api/v1/config/backups/"+url.PathEscape(backup.ID), strings.NewReader(`{"name":"日常配置","note":"已验证"}`))
	update.Header.Set("Content-Type", "application/json")
	updated := httptest.NewRecorder()
	application.Handler().ServeHTTP(updated, update)
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), "日常配置") {
		t.Fatalf("编辑存档响应异常: status=%d body=%s", updated.Code, updated.Body.String())
	}

	previewed := httptest.NewRecorder()
	application.Handler().ServeHTTP(previewed, httptest.NewRequest(
		http.MethodGet, "/api/v1/config/backups/"+url.PathEscape(backup.ID)+"/preview", nil))
	if previewed.Code != http.StatusOK {
		t.Fatalf("预览存档状态码 = %d，响应 = %s", previewed.Code, previewed.Body.String())
	}
	var preview configstore.BackupPreview
	if err := json.NewDecoder(previewed.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	if !preview.Valid || !preview.Same || preview.CurrentHash == "" || preview.Backup.ID != backup.ID {
		t.Fatalf("预览存档响应异常: %+v", preview)
	}

	deleted := httptest.NewRecorder()
	application.Handler().ServeHTTP(deleted, httptest.NewRequest(
		http.MethodDelete, "/api/v1/config/backups/"+url.PathEscape(backup.ID), nil))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("删除存档状态码 = %d，响应 = %s", deleted.Code, deleted.Body.String())
	}
	listed := httptest.NewRecorder()
	application.Handler().ServeHTTP(listed, httptest.NewRequest(http.MethodGet, "/api/v1/config/backups", nil))
	if listed.Code != http.StatusOK || listed.Body.String() != "[]\n" {
		t.Fatalf("删除后列表异常: status=%d body=%s", listed.Code, listed.Body.String())
	}
}

func TestServiceRestartAction(t *testing.T) {
	hostService := &stubHostService{}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Host: hostService},
	)
	if err != nil {
		t.Fatalf("初始化应用失败: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/service/actions/restart", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	if len(hostService.actions) != 1 || hostService.actions[0] != host.ActionRestart {
		t.Fatalf("服务动作异常: %v", hostService.actions)
	}
}

func TestServiceStartAndStopPersistBootState(t *testing.T) {
	hostService := &stubHostService{}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Host: hostService},
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, action := range []string{"start", "stop"} {
		response := httptest.NewRecorder()
		application.Handler().ServeHTTP(response, httptest.NewRequest(
			http.MethodPost, "/api/v1/service/actions/"+action, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s 状态码 = %d，响应 = %s", action, response.Code, response.Body.String())
		}
	}
	want := []host.Action{host.ActionEnableNow, host.ActionDisableNow}
	if !reflect.DeepEqual(hostService.actions, want) {
		t.Fatalf("服务动作 = %v，期望 %v", hostService.actions, want)
	}
}

func TestAdoptRunningServiceBootState(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	active := &stubHostService{status: host.Status{ActiveState: "active", UnitFileState: "disabled"}}
	adoptRunningServiceBootState(active, logger)
	if want := []host.Action{host.ActionEnable}; !reflect.DeepEqual(active.actions, want) {
		t.Fatalf("运行中的旧服务迁移动作 = %v，期望 %v", active.actions, want)
	}

	for _, status := range []host.Status{
		{ActiveState: "inactive", UnitFileState: "disabled"},
		{ActiveState: "active", UnitFileState: "enabled"},
	} {
		service := &stubHostService{status: status}
		adoptRunningServiceBootState(service, logger)
		if len(service.actions) != 0 {
			t.Fatalf("状态 %+v 不应被迁移，实际动作 %v", status, service.actions)
		}
	}
}

func TestServiceSuspendStateClearsAfterReload(t *testing.T) {
	hostService := &stubHostService{status: host.Status{
		Name:        "dae.service",
		ActiveState: "active",
		SubState:    "running",
		MainPID:     42,
	}}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Host: hostService},
	)
	if err != nil {
		t.Fatal(err)
	}

	suspend := httptest.NewRecorder()
	application.Handler().ServeHTTP(suspend,
		httptest.NewRequest(http.MethodPost, "/api/v1/service/actions/suspend", nil))
	if suspend.Code != http.StatusOK || !strings.Contains(suspend.Body.String(), "dae 已暂停") {
		t.Fatalf("暂停响应异常: status=%d body=%s", suspend.Code, suspend.Body.String())
	}

	assertSuspended := func(want bool) {
		t.Helper()
		response := httptest.NewRecorder()
		application.Handler().ServeHTTP(response,
			httptest.NewRequest(http.MethodGet, "/api/v1/service", nil))
		var payload struct {
			Suspended bool `json:"suspended"`
		}
		if response.Code != http.StatusOK || json.NewDecoder(response.Body).Decode(&payload) != nil {
			t.Fatalf("读取服务状态失败: status=%d body=%s", response.Code, response.Body.String())
		}
		if payload.Suspended != want {
			t.Fatalf("suspended = %v，期望 %v", payload.Suspended, want)
		}
	}
	assertSuspended(true)

	// dae 被面板外部停止时，进程内的旧标记不能把未运行服务误报成暂停。
	hostService.status.ActiveState = "inactive"
	assertSuspended(false)
	hostService.status.ActiveState = "active"
	assertSuspended(false)
	application.Handler().ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/api/v1/service/actions/suspend", nil))
	assertSuspended(true)

	reload := httptest.NewRecorder()
	application.Handler().ServeHTTP(reload,
		httptest.NewRequest(http.MethodPost, "/api/v1/service/actions/reload", nil))
	if reload.Code != http.StatusOK {
		t.Fatalf("重载响应异常: status=%d body=%s", reload.Code, reload.Body.String())
	}
	assertSuspended(false)

	// 面板外发生的 systemd 重启不会经过动作端点，PID 变化也必须让旧状态失效。
	application.Handler().ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/api/v1/service/actions/suspend", nil))
	hostService.status.MainPID = 43
	assertSuspended(false)
}

func TestServiceReloadDefersWhenDaeIsNotRunning(t *testing.T) {
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{
			Dae:  stubDaeService{err: configstore.ErrReloadDeferred},
			Host: &stubHostService{},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response,
		httptest.NewRequest(http.MethodPost, "/api/v1/service/actions/reload", nil))
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"deferred":true`) ||
		!strings.Contains(response.Body.String(), "下次启动") {
		t.Fatalf("延后重载响应异常: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestServiceStartFailureExplainsMissingGeoClassification(t *testing.T) {
	hostService := &stubHostService{
		err:  errors.New("执行 systemd start 失败"),
		logs: []host.LogEntry{{Message: "country code twitter not found in /etc/dae/geoip.dat"}},
	}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Host: hostService},
	)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/service/actions/start", nil))
	if recorder.Code != http.StatusBadGateway || !strings.Contains(recorder.Body.String(), "geoip:twitter") {
		t.Fatalf("启动失败应指出 Geo 分类：%d %s", recorder.Code, recorder.Body.String())
	}
}

func TestHostInterfaces(t *testing.T) {
	want := []host.NetworkInterface{
		{Name: "dae0", Addresses: []string{"10.0.0.1/24"}},
		{Name: "ens2", Addresses: []string{"192.168.50.23/24", "fe80::1/64"}},
		{Name: "lo", Addresses: []string{"127.0.0.1/8", "::1/128"}},
	}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Host: &stubHostService{interfaces: want}},
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/host/interfaces", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	var got []host.NetworkInterface
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("接口列表 = %+v，期望 %+v", got, want)
	}
}

func TestHostInterfacesReportEnumerationFailure(t *testing.T) {
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Host: &stubHostService{err: errors.New("枚举失败")}},
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/host/interfaces", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "host_interfaces_unavailable") {
		t.Fatalf("响应异常: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestHostInterfacesRequireHostService(t *testing.T) {
	application, err := NewWithDae(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		stubDaeService{},
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/host/interfaces", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "host_service_unavailable") {
		t.Fatalf("响应异常: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestControlOperationRejectsConcurrentAction(t *testing.T) {
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Host: &stubHostService{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	application.operations.Lock()
	defer application.operations.Unlock()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/service/actions/restart", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("并发控制操作状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
}

func TestSysdumpDownload(t *testing.T) {
	application, err := NewWithDae(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		stubDaeService{},
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/diagnostics/sysdump", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || recorder.Body.String() != "test dump" {
		t.Fatalf("sysdump 响应异常: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "application/gzip" {
		t.Fatalf("Content-Type = %q", recorder.Header().Get("Content-Type"))
	}
	if !strings.Contains(recorder.Header().Get("Content-Disposition"), "dae-sysdump.test.tar.gz") {
		t.Fatalf("Content-Disposition = %q", recorder.Header().Get("Content-Disposition"))
	}
}

type stubProbeService struct {
	results []netprobe.Result
	err     error
	targets []netprobe.Target
}

func (s *stubProbeService) Probe(_ context.Context, targets []netprobe.Target) ([]netprobe.Result, error) {
	s.targets = targets
	return s.results, s.err
}

func TestLatencyProbeEndpoint(t *testing.T) {
	prober := &stubProbeService{results: []netprobe.Result{{Host: "example.com", Port: 443, Reachable: true, LatencyMs: 12.5}}}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Probe: prober},
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/net/latency", strings.NewReader(`{"targets":[{"host":"example.com","port":443}]}`))
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Results []netprobe.Result `json:"results"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(response.Results) != 1 || !response.Results[0].Reachable || response.Results[0].LatencyMs != 12.5 {
		t.Fatalf("响应内容异常: %+v", response.Results)
	}
	if len(prober.targets) != 1 || prober.targets[0].Host != "example.com" {
		t.Fatalf("探测目标异常: %+v", prober.targets)
	}

	invalid := httptest.NewRecorder()
	prober.err = errors.New("探测目标不能为空")
	application.Handler().ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/api/v1/net/latency", strings.NewReader(`{"targets":[]}`)))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("无效请求状态码 = %d，响应 = %s", invalid.Code, invalid.Body.String())
	}
}

func TestLatencyProbeLogsTargets(t *testing.T) {
	var logs strings.Builder
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(&logs, nil)),
		Dependencies{Dae: stubDaeService{}, Probe: &stubProbeService{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/net/latency", strings.NewReader(`{"targets":[{"host":"hk.example.com","port":443}]}`))
	application.Handler().ServeHTTP(httptest.NewRecorder(), request)
	if !strings.Contains(logs.String(), "hk.example.com:443") {
		t.Fatalf("探测目标应记入审计日志，实际日志 = %s", logs.String())
	}
}

type stubScheduleService struct {
	status  schedule.Status
	err     error
	updates []schedule.Settings
}

func (s *stubScheduleService) Status() schedule.Status {
	return s.status
}

func (s *stubScheduleService) Update(settings schedule.Settings) (schedule.Status, error) {
	s.updates = append(s.updates, settings)
	if s.err != nil {
		return schedule.Status{}, s.err
	}
	return schedule.Status{Settings: settings}, nil
}

func TestScheduleEndpoints(t *testing.T) {
	service := &stubScheduleService{status: schedule.Status{Settings: schedule.Settings{Enabled: true, IntervalMinutes: 720}}}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Schedule: service},
	)
	if err != nil {
		t.Fatal(err)
	}

	read := httptest.NewRecorder()
	application.Handler().ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/schedule/reload", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("读取状态码 = %d，响应 = %s", read.Code, read.Body.String())
	}
	var status schedule.Status
	if err := json.NewDecoder(read.Body).Decode(&status); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !status.Enabled || status.IntervalMinutes != 720 {
		t.Fatalf("状态内容异常: %+v", status)
	}

	write := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/schedule/reload", strings.NewReader(`{"enabled":false,"intervalMinutes":60}`))
	request.Header.Set("Content-Type", "application/json")
	application.Handler().ServeHTTP(write, request)
	if write.Code != http.StatusOK {
		t.Fatalf("更新状态码 = %d，响应 = %s", write.Code, write.Body.String())
	}
	if len(service.updates) != 1 || service.updates[0].IntervalMinutes != 60 || service.updates[0].Enabled {
		t.Fatalf("更新参数异常: %+v", service.updates)
	}

	service.err = &schedule.InvalidSettingsError{Cause: errors.New("刷新间隔必须在 5 到 10080 分钟之间")}
	invalid := httptest.NewRecorder()
	invalidRequest := httptest.NewRequest(http.MethodPut, "/api/v1/schedule/reload", strings.NewReader(`{"enabled":true,"intervalMinutes":1}`))
	application.Handler().ServeHTTP(invalid, invalidRequest)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("非法设置状态码 = %d，响应 = %s", invalid.Code, invalid.Body.String())
	}

	// 写盘失败不是用户输入问题，必须区分为 500
	service.err = errors.New("写入设置: 权限不足")
	failed := httptest.NewRecorder()
	failedRequest := httptest.NewRequest(http.MethodPut, "/api/v1/schedule/reload", strings.NewReader(`{"enabled":true,"intervalMinutes":60}`))
	application.Handler().ServeHTTP(failed, failedRequest)
	if failed.Code != http.StatusInternalServerError {
		t.Fatalf("写盘失败状态码 = %d，响应 = %s", failed.Code, failed.Body.String())
	}
}

func TestScheduleRunnerWiredWithOperationsLock(t *testing.T) {
	schedulePath := t.TempDir() + "/schedule.json"
	application, err := NewWithDependencies(
		Config{Version: "test-panel", SchedulePath: schedulePath},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	read := httptest.NewRecorder()
	application.Handler().ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/schedule/reload", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("真实调度器状态码 = %d，响应 = %s", read.Code, read.Body.String())
	}
	update := httptest.NewRecorder()
	application.Handler().ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/api/v1/schedule/reload", strings.NewReader(`{"enabled":true,"intervalMinutes":30}`)))
	if update.Code != http.StatusOK {
		t.Fatalf("真实调度器更新状态码 = %d，响应 = %s", update.Code, update.Body.String())
	}
	var status schedule.Status
	if err := json.NewDecoder(update.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Enabled || status.NextRunAt.IsZero() {
		t.Fatalf("更新后应排期下一轮: %+v", status)
	}
}

type stubInstallService struct {
	status   daeinstall.Status
	plan     daeinstall.Provision
	versions []daeinstall.Version
	binary   []byte
	cached   bool
	err      error
	release  chan struct{}
	// 安装在后台 goroutine 里执行，测试主协程会同时读取记录，必须加锁。
	mu        sync.Mutex
	installed []string
}

func (s *stubInstallService) record(entry string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.installed = append(s.installed, entry)
}

func (s *stubInstallService) records() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.installed...)
}

func (s *stubInstallService) Status(context.Context) daeinstall.Status {
	return s.status
}

func (s *stubInstallService) Versions(_ context.Context, source upstream.Source, _ int) ([]daeinstall.Version, error) {
	return s.versions, s.err
}

func (s *stubInstallService) Provision(context.Context) daeinstall.Provision {
	return s.plan
}

func (s *stubInstallService) Acquire(_ context.Context, _ upstream.Source, _, _ string, requireBundle bool) (upstream.Bundle, bool, error) {
	if s.release != nil {
		<-s.release
	}
	bundle := upstream.Bundle{Platform: "x86_64", Binary: s.binary}
	if requireBundle {
		bundle.Unit = []byte("dae.service")
	}
	return bundle, s.cached, s.err
}

func (s *stubInstallService) DeleteCached(source upstream.Source, ref string) error {
	s.record("delete-cache:" + string(source) + ":" + ref)
	return s.err
}

func (s *stubInstallService) Preflight(_ context.Context, _ []byte) (daeinstall.Compatibility, error) {
	s.record("preflight")
	if s.err != nil {
		return daeinstall.Compatibility{}, s.err
	}
	return daeinstall.Compatibility{
		Compatible: true, Version: "dae test", OutlineSupported: true, ConfigPresent: true,
	}, nil
}

func (s *stubInstallService) FirstInstall(_ context.Context, _ upstream.Bundle, source upstream.Source, ref, _ string) (daeinstall.Status, error) {
	s.record("first:" + string(source) + ":" + ref)
	return s.status, s.err
}

func (s *stubInstallService) Install(_ context.Context, _ []byte, source upstream.Source, ref, _, _ string) (daeinstall.Status, error) {
	s.record(string(source) + ":" + ref)
	return s.status, s.err
}

func (s *stubInstallService) Rollback(context.Context) (daeinstall.Status, error) {
	s.record("rollback")
	return s.status, s.err
}

func (s *stubInstallService) Uninstall(_ context.Context, options daeinstall.UninstallOptions) error {
	s.record(fmt.Sprintf("uninstall:%t:%t", options.PurgeConfig, options.PurgeGeo))
	return s.err
}

func newInstallApp(t *testing.T, service InstallService) *App {
	t.Helper()
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Install: service},
	)
	if err != nil {
		t.Fatal(err)
	}
	return application
}

func TestDaeInstallDisabledByDefault(t *testing.T) {
	// 未注入安装服务即代表功能关闭，所有相关端点都必须明确拒绝
	application := newInstallApp(t, nil)
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/dae/install", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/dae/versions?source=official", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/dae/install", strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)),
		httptest.NewRequest(http.MethodGet, "/api/v1/dae/compatibility", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/dae/compatibility", strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)),
		httptest.NewRequest(http.MethodDelete, "/api/v1/dae/cache", strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)),
		httptest.NewRequest(http.MethodPost, "/api/v1/dae/rollback", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/dae/uninstall", nil),
	} {
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s 状态码 = %d，期望 503", request.Method, request.URL.Path, recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), "dae_install_disabled") {
			t.Fatalf("响应应说明功能未启用: %s", recorder.Body.String())
		}
	}
}

// OpenWrt 上没有 KDAE_PANEL_ENABLE_DAE_INSTALL 这个环境变量，开关在
// /etc/config/kdae-panel 里。指引照抄 systemd 那套，用户一定白忙一场。
func TestDaeInstallDisabledPointsAtUCIOnProcd(t *testing.T) {
	application, err := NewWithDependencies(
		Config{Version: "test-panel", ServiceBackend: host.BackendProcd},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/dae/install", nil))
	body := recorder.Body.String()
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d，期望 503", recorder.Code)
	}
	if !strings.Contains(body, "enable_dae_install") || !strings.Contains(body, "/etc/config/kdae-panel") {
		t.Fatalf("procd 下应指向 UCI 开关: %s", body)
	}
	for _, forbidden := range []string{"KDAE_PANEL_ENABLE_DAE_INSTALL", "ReadWritePaths", "systemd"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("procd 下的提示不该出现 %q: %s", forbidden, body)
		}
	}
}

func TestDaeVersionsRejectsUnknownSource(t *testing.T) {
	application := newInstallApp(t, &stubInstallService{})
	for _, query := range []string{"", "?source=", "?source=../etc", "?source=evil"} {
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/dae/versions"+query, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("source=%q 状态码 = %d，期望 400", query, recorder.Code)
		}
	}
}

func TestDaeInstallRunsAsynchronously(t *testing.T) {
	// Ready 表示机器上已有 dae，因此走替换而不是首次安装
	service := &stubInstallService{binary: []byte("v2"), cached: true, status: daeinstall.Status{Ready: true}}
	application := newInstallApp(t, service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/dae/install", strings.NewReader(`{"source":"kdae","ref":"30187784287","label":"d63a0c1"}`))
	application.Handler().ServeHTTP(recorder, request)
	// 安装耗时以分钟计，必须立即返回而不是把请求挂着
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("状态码 = %d，期望 202，响应 = %s", recorder.Code, recorder.Body.String())
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(service.records()) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if installed := service.records(); len(installed) != 1 || installed[0] != "kdae:30187784287" {
		t.Fatalf("安装调用 = %v", installed)
	}
	if job := awaitJobSettled(t, application); !job.Cached {
		t.Fatalf("缓存命中应写进任务状态: %+v", job)
	}
}

func TestDaeCompatibilityRunsAsynchronously(t *testing.T) {
	service := &stubInstallService{
		binary: []byte("v2"), cached: true, status: daeinstall.Status{Ready: true},
	}
	application := newInstallApp(t, service)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		"/api/v1/dae/compatibility", strings.NewReader(`{"source":"official","ref":"v2.0.0","label":"v2.0.0"}`)))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("状态码 = %d，期望 202，响应 = %s", recorder.Code, recorder.Body.String())
	}
	job := awaitCompatibilitySettled(t, application)
	if job.Phase != PhaseDone || job.Result == nil || !job.Result.Compatible || !job.Cached {
		t.Fatalf("兼容性预检结果异常: %+v", job)
	}
	if records := service.records(); len(records) != 1 || records[0] != "preflight" {
		t.Fatalf("预检调用 = %v", records)
	}
}

func TestDaeCompatibilityRequiresExistingInstall(t *testing.T) {
	application := newInstallApp(t, &stubInstallService{status: daeinstall.Status{Ready: false}})
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		"/api/v1/dae/compatibility", strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)))
	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), "dae_not_installed") {
		t.Fatalf("首次安装预检响应 = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestDaeCachedVersionCanBeDeleted(t *testing.T) {
	service := &stubInstallService{}
	application := newInstallApp(t, service)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/dae/cache",
		strings.NewReader(`{"source":"official","ref":"v2.0.0"}`))
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("状态码 = %d，期望 204，响应 = %s", recorder.Code, recorder.Body.String())
	}
	if records := service.records(); len(records) != 1 || records[0] != "delete-cache:official:v2.0.0" {
		t.Fatalf("删除缓存调用 = %v", records)
	}
}

func TestDaeCachedVersionDeleteReportsMissing(t *testing.T) {
	service := &stubInstallService{err: daeinstall.ErrCachedVersionNotFound}
	application := newInstallApp(t, service)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/dae/cache",
		strings.NewReader(`{"source":"kdae","ref":"30187784287"}`))
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "cached_version_not_found") {
		t.Fatalf("不存在的缓存响应 = %d %s", recorder.Code, recorder.Body.String())
	}
}

// 还没有 dae 时必须走首次安装，而不是去替换一个不存在的文件。
func TestDaeInstallUsesFirstInstallWhenNotReady(t *testing.T) {
	service := &stubInstallService{
		binary: []byte("v1"),
		status: daeinstall.Status{Ready: false},
		plan:   daeinstall.Provision{Possible: true},
	}
	application := newInstallApp(t, service)

	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/dae/install",
		strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("状态码 = %d", recorder.Code)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && len(service.records()) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if installed := service.records(); len(installed) != 1 || installed[0] != "first:official:v2.0.0" {
		t.Fatalf("应走首次安装，实际调用 = %v", installed)
	}
}

// 还没有 dae 时，状态响应要带上首次安装的可行性说明。
func TestDaeInstallStatusIncludesProvision(t *testing.T) {
	service := &stubInstallService{
		status: daeinstall.Status{Ready: false},
		plan:   daeinstall.Provision{Possible: true, BinaryPath: "/usr/bin/dae", UnitPath: "/etc/systemd/system/dae.service"},
	}
	application := newInstallApp(t, service)

	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/dae/install", nil))
	var payload struct {
		Provision *daeinstall.Provision `json:"provision"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Provision == nil || !payload.Provision.Possible {
		t.Fatalf("未就绪时应附带首次安装可行性: %+v", payload.Provision)
	}
}

// 任务进行中不计算可行性：这个查询被界面每两秒轮询一次，而可行性探测要实际
// 试写目标目录，其中之一是 systemd 正在 inotify 监视的单元目录。
func TestDaeInstallStatusOmitsProvisionWhileJobRuns(t *testing.T) {
	release := make(chan struct{})
	service := &stubInstallService{
		binary:  []byte("v2"),
		release: release,
		status:  daeinstall.Status{Ready: false},
		plan:    daeinstall.Provision{Possible: true},
	}
	application := newInstallApp(t, service)
	defer close(release)

	started := httptest.NewRecorder()
	application.Handler().ServeHTTP(started, httptest.NewRequest(http.MethodPost, "/api/v1/dae/install",
		strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)))
	if started.Code != http.StatusAccepted {
		t.Fatalf("状态码 = %d", started.Code)
	}

	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/dae/install", nil))
	var payload struct {
		Provision *daeinstall.Provision `json:"provision"`
		Job       Job                   `json:"job"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Job.Phase != PhaseDownloading {
		t.Fatalf("任务应处于下载中: %+v", payload.Job)
	}
	if payload.Provision != nil {
		t.Fatalf("任务进行中不应计算可行性: %+v", payload.Provision)
	}
}

// 安装失败必须让用户看得见原因，而不是任务默默停在某个中间态。
func TestDaeInstallReportsFailure(t *testing.T) {
	service := &stubInstallService{
		binary: []byte("v2"),
		status: daeinstall.Status{Ready: true},
		err:    errors.New("新版本拒绝当前配置"),
	}
	application := newInstallApp(t, service)

	application.Handler().ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/api/v1/dae/install",
			strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)))

	job := awaitJobSettled(t, application)
	if job.Phase != PhaseFailed {
		t.Fatalf("任务应标记为失败: %+v", job)
	}
	if !strings.Contains(job.Error, "拒绝当前配置") {
		t.Fatalf("失败原因应透传给前端: %q", job.Error)
	}
	if job.EndedAt == nil {
		t.Fatal("失败的任务也应记录结束时间")
	}
}

func TestDaeRollbackRunsAsynchronously(t *testing.T) {
	service := &stubInstallService{status: daeinstall.Status{Ready: true}}
	application := newInstallApp(t, service)

	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/dae/rollback", nil))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("状态码 = %d，期望 202，响应 = %s", recorder.Code, recorder.Body.String())
	}

	if job := awaitJobSettled(t, application); job.Phase != PhaseDone {
		t.Fatalf("回滚应完成: %+v", job)
	}
	if records := service.records(); len(records) != 1 || records[0] != "rollback" {
		t.Fatalf("回滚调用 = %v", records)
	}
}

// 回滚失败同样要如实上报，不能因为它复用安装任务的状态机就报成"安装失败"。
func TestDaeRollbackReportsFailure(t *testing.T) {
	service := &stubInstallService{status: daeinstall.Status{Ready: true}, err: errors.New("没有可回滚的上一版本")}
	application := newInstallApp(t, service)

	application.Handler().ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/api/v1/dae/rollback", nil))

	job := awaitJobSettled(t, application)
	if job.Phase != PhaseFailed || !strings.Contains(job.Error, "没有可回滚") {
		t.Fatalf("回滚失败应如实上报: %+v", job)
	}
}

func TestDaeUninstallRunsAsynchronously(t *testing.T) {
	service := &stubInstallService{status: daeinstall.Status{Ready: true}}
	application := newInstallApp(t, service)

	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/dae/uninstall", nil))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("状态码 = %d，期望 202，响应 = %s", recorder.Code, recorder.Body.String())
	}

	if job := awaitJobSettled(t, application); job.Phase != PhaseDone || job.Label != "卸载 dae" {
		t.Fatalf("卸载任务应完成: %+v", job)
	}
	if records := service.records(); len(records) != 1 || records[0] != "uninstall:false:false" {
		t.Fatalf("卸载调用 = %v", records)
	}
}

func TestDaeUninstallPassesExplicitDataChoice(t *testing.T) {
	service := &stubInstallService{status: daeinstall.Status{Ready: true}}
	application := newInstallApp(t, service)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/dae/uninstall",
		strings.NewReader(`{"purgeConfig":true,"purgeGeo":true}`))
	application.Handler().ServeHTTP(httptest.NewRecorder(), request)
	if job := awaitJobSettled(t, application); job.Phase != PhaseDone {
		t.Fatalf("卸载任务应完成: %+v", job)
	}
	if records := service.records(); len(records) != 1 || records[0] != "uninstall:true:true" {
		t.Fatalf("卸载选项没有透传: %v", records)
	}
}

func TestDaeUninstallReportsFailure(t *testing.T) {
	service := &stubInstallService{status: daeinstall.Status{Ready: true}, err: errors.New("当前 dae 没有面板安装记录")}
	application := newInstallApp(t, service)

	application.Handler().ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/api/v1/dae/uninstall", nil))

	job := awaitJobSettled(t, application)
	if job.Phase != PhaseFailed || !strings.Contains(job.Error, "没有面板安装记录") {
		t.Fatalf("卸载失败应如实上报: %+v", job)
	}
}

// awaitJobSettled 轮询状态端点直到任务不再进行中。
func awaitJobSettled(t *testing.T, application *App) Job {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/dae/install", nil))
		var payload struct {
			Job Job `json:"job"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Job.Phase != PhaseDownloading && payload.Job.Phase != PhaseApplying {
			return payload.Job
		}
		if !time.Now().Before(deadline) {
			t.Fatalf("任务迟迟没有结束: %+v", payload.Job)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func awaitCompatibilitySettled(t *testing.T, application *App) CompatibilityJob {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder,
			httptest.NewRequest(http.MethodGet, "/api/v1/dae/compatibility", nil))
		var payload struct {
			Job CompatibilityJob `json:"job"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Job.Phase != PhaseDownloading && payload.Job.Phase != PhaseApplying {
			return payload.Job
		}
		if !time.Now().Before(deadline) {
			t.Fatalf("预检任务迟迟没有结束: %+v", payload.Job)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestDaeInstallRejectsConcurrentJobs(t *testing.T) {
	release := make(chan struct{})
	service := &stubInstallService{binary: []byte("v2"), release: release}
	application := newInstallApp(t, service)
	body := func() *strings.Reader {
		return strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)
	}

	first := httptest.NewRecorder()
	application.Handler().ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/v1/dae/install", body()))
	if first.Code != http.StatusAccepted {
		t.Fatalf("首个任务状态码 = %d", first.Code)
	}

	second := httptest.NewRecorder()
	application.Handler().ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/v1/dae/install", body()))
	if second.Code != http.StatusConflict {
		t.Fatalf("并发任务状态码 = %d，期望 409", second.Code)
	}
	close(release)
}

func TestDaeCompatibilityAndInstallShareOneTaskGate(t *testing.T) {
	release := make(chan struct{})
	service := &stubInstallService{
		binary: []byte("v2"), release: release, status: daeinstall.Status{Ready: true},
	}
	application := newInstallApp(t, service)

	preflight := httptest.NewRecorder()
	application.Handler().ServeHTTP(preflight, httptest.NewRequest(http.MethodPost,
		"/api/v1/dae/compatibility", strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)))
	if preflight.Code != http.StatusAccepted {
		t.Fatalf("预检状态码 = %d，响应 = %s", preflight.Code, preflight.Body.String())
	}

	install := httptest.NewRecorder()
	application.Handler().ServeHTTP(install, httptest.NewRequest(http.MethodPost,
		"/api/v1/dae/install", strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)))
	if install.Code != http.StatusConflict || !strings.Contains(install.Body.String(), "version_task_in_progress") {
		t.Fatalf("预检期间安装响应 = %d %s", install.Code, install.Body.String())
	}

	remove := httptest.NewRecorder()
	application.Handler().ServeHTTP(remove, httptest.NewRequest(http.MethodDelete,
		"/api/v1/dae/cache", strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)))
	if remove.Code != http.StatusConflict || !strings.Contains(remove.Body.String(), "version_task_in_progress") {
		t.Fatalf("预检期间删除缓存响应 = %d %s", remove.Code, remove.Body.String())
	}
	close(release)
}

func newUpdateCheckApp(t *testing.T, version string, checker PanelReleaseChecker) *App {
	t.Helper()
	application, err := NewWithDependencies(
		Config{Version: version},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, PanelRelease: checker},
	)
	if err != nil {
		t.Fatal(err)
	}
	return application
}

// fetchPanelUpdate 返回响应里的 check 部分——自升级未启用时那是全部内容。
func fetchPanelUpdate(t *testing.T, application *App) map[string]any {
	t.Helper()
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/panel/update", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Check map[string]any `json:"check"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Check
}

func forcePanelUpdateCheck(t *testing.T, application *App) map[string]any {
	t.Helper()
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/panel/update/check", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("强制检查状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Check map[string]any `json:"check"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Check
}

type stubPanelUpdateService struct {
	status    panelupdate.Status
	mu        sync.Mutex
	applied   int
	requested string
	err       error
}

func (s *stubPanelUpdateService) Status(context.Context) panelupdate.Status { return s.status }

func (s *stubPanelUpdateService) SetEnabled(enabled bool) error {
	if s.err != nil {
		return s.err
	}
	s.mu.Lock()
	s.status.Enabled = enabled
	s.status.Updatable = enabled
	s.mu.Unlock()
	return nil
}

func (s *stubPanelUpdateService) Download(_ context.Context, version string) (upstream.PanelBinary, error) {
	s.mu.Lock()
	s.requested = version
	s.mu.Unlock()
	if s.err != nil {
		return upstream.PanelBinary{}, s.err
	}
	return upstream.PanelBinary{Version: version, Content: []byte("\x7fELF")}, nil
}

func (s *stubPanelUpdateService) Apply(context.Context, upstream.PanelBinary) error {
	s.mu.Lock()
	s.applied++
	s.mu.Unlock()
	return nil
}

func (s *stubPanelUpdateService) applyCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applied
}

// 没有升级服务的非正式嵌入部署要返回明确错误，不能落到 api_not_found。
func TestPanelSelfUpdateUnavailable(t *testing.T) {
	application := newUpdateCheckApp(t, "v0.1.0", nil)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/panel/update", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d，期望 503", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "panel_self_update_unavailable") {
		t.Fatalf("响应应说明部署不支持: %s", recorder.Body.String())
	}
}

func TestPanelSelfUpdatePreferenceCanBeChangedFromAPI(t *testing.T) {
	service := &stubPanelUpdateService{status: panelupdate.Status{Enabled: false}}
	application := newSelfUpdateApp(t, service)

	preference := httptest.NewRecorder()
	application.Handler().ServeHTTP(preference, httptest.NewRequest(http.MethodPut,
		"/api/v1/panel/update/preference", strings.NewReader(`{"enabled":true}`)))
	if preference.Code != http.StatusOK || !strings.Contains(preference.Body.String(), `"enabled":true`) {
		t.Fatalf("开启响应 = %d %s", preference.Code, preference.Body.String())
	}

	upgrade := httptest.NewRecorder()
	application.Handler().ServeHTTP(upgrade, httptest.NewRequest(http.MethodPost,
		"/api/v1/panel/update", strings.NewReader(`{"version":"v0.2.0"}`)))
	if upgrade.Code != http.StatusAccepted {
		t.Fatalf("界面开启后应能升级: %d %s", upgrade.Code, upgrade.Body.String())
	}
}

func TestPanelSelfUpdateRejectsWhilePreferenceDisabled(t *testing.T) {
	service := &stubPanelUpdateService{status: panelupdate.Status{Enabled: false}}
	application := newSelfUpdateApp(t, service)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		"/api/v1/panel/update", strings.NewReader(`{"version":"v0.2.0"}`)))
	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), "panel_self_update_disabled") {
		t.Fatalf("关闭状态响应 = %d %s", recorder.Code, recorder.Body.String())
	}
}

func newSelfUpdateApp(t *testing.T, service PanelUpdateService) *App {
	t.Helper()
	application, err := NewWithDependencies(
		Config{Version: "v0.1.0"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, PanelUpdate: service},
	)
	if err != nil {
		t.Fatal(err)
	}
	return application
}

func TestPanelSelfUpdateRunsAsynchronously(t *testing.T) {
	service := &stubPanelUpdateService{status: panelupdate.Status{Enabled: true, Updatable: true}}
	application := newSelfUpdateApp(t, service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/panel/update",
		strings.NewReader(`{"version":"v0.2.0"}`))
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && service.applyCount() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if service.applyCount() != 1 {
		t.Fatalf("应用次数 = %d，期望 1", service.applyCount())
	}
	service.mu.Lock()
	requested := service.requested
	service.mu.Unlock()
	if requested != "v0.2.0" {
		t.Fatalf("请求的版本 = %q", requested)
	}
}

// 目录不可写等情形要在启动任务前就挡住，并把原因如实带出。
func TestPanelSelfUpdateRejectsWhenNotUpdatable(t *testing.T) {
	service := &stubPanelUpdateService{status: panelupdate.Status{
		Enabled:   true,
		Updatable: false,
		Problem:   "面板无法写入 /usr/bin：只读文件系统",
	}}
	application := newSelfUpdateApp(t, service)

	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/panel/update", nil))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("状态码 = %d，期望 409", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "只读文件系统") {
		t.Fatalf("应带出具体原因: %s", recorder.Body.String())
	}
	if service.applyCount() != 0 {
		t.Fatal("不可升级时不得执行替换")
	}
}

// 版本号会被拼进下载地址，含斜杠或空白的取值必须在拼接前拦住。
func TestPanelSelfUpdateRejectsMalformedVersion(t *testing.T) {
	service := &stubPanelUpdateService{status: panelupdate.Status{Enabled: true, Updatable: true}}
	application := newSelfUpdateApp(t, service)

	for _, bad := range []string{`{"version":"../../etc"}`, `{"version":"v1 0"}`, `{"version":"1.0.0"}`} {
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder,
			httptest.NewRequest(http.MethodPost, "/api/v1/panel/update", strings.NewReader(bad)))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s 状态码 = %d，期望 400", bad, recorder.Code)
		}
	}
	if service.applyCount() != 0 {
		t.Fatal("非法版本不得触发升级")
	}
}

// 新版本检查：结果必须缓存，dev 构建不联网也不提示。
func TestPanelUpdateCheck(t *testing.T) {
	var calls atomic.Int64
	application := newUpdateCheckApp(t, "v0.1.2", func(context.Context) (string, error) {
		calls.Add(1)
		return "v0.2.0", nil
	})

	payload := fetchPanelUpdate(t, application)
	if payload["updateAvailable"] != true || payload["latest"] != "v0.2.0" {
		t.Fatalf("应提示新版本，实际 %v", payload)
	}
	// 第二次请求走缓存，检查函数不得再被调用
	fetchPanelUpdate(t, application)
	if calls.Load() != 1 {
		t.Fatalf("检查函数被调用 %d 次，期望缓存后只有 1 次", calls.Load())
	}
}

func TestPanelUpdateManualCheckBypassesCache(t *testing.T) {
	var calls atomic.Int64
	application := newUpdateCheckApp(t, "v0.1.2", func(context.Context) (string, error) {
		if calls.Add(1) == 1 {
			return "v0.2.0", nil
		}
		return "v0.3.0", nil
	})

	first := fetchPanelUpdate(t, application)
	if first["latest"] != "v0.2.0" {
		t.Fatalf("初次检查版本 = %v", first["latest"])
	}
	manual := forcePanelUpdateCheck(t, application)
	if manual["latest"] != "v0.3.0" || manual["updateAvailable"] != true {
		t.Fatalf("手动检查未绕过缓存: %v", manual)
	}
	forcePanelUpdateCheck(t, application)
	if calls.Load() != 2 {
		t.Fatalf("冷却期内重复检查调用了 %d 次，期望 2 次", calls.Load())
	}
}

func TestPanelUpdateCheckSkipsDevBuild(t *testing.T) {
	var calls atomic.Int64
	application := newUpdateCheckApp(t, "dev", func(context.Context) (string, error) {
		calls.Add(1)
		return "v9.9.9", nil
	})
	payload := fetchPanelUpdate(t, application)
	if payload["updateAvailable"] != false {
		t.Fatalf("dev 构建不应提示升级，实际 %v", payload)
	}
	if calls.Load() != 0 {
		t.Fatal("dev 构建不应发起联网检查")
	}
}

// checked 把"查过了，没有更新"与"根本没查"分开。
//
// 少了它，界面只能看 latest 与 error 都为空来猜，而这正是 dev 构建、关掉检查、
// 以及检查能力压根没注册这三种情形的共同外观——procd 上就猜错过，对一个从未
// 联网的部署报了绿色的"当前已是最新版本"。
func TestPanelUpdateCheckedDistinguishesNoCheckFromUpToDate(t *testing.T) {
	upToDate := fetchPanelUpdate(t, newUpdateCheckApp(t, "v1.0.0",
		func(context.Context) (string, error) { return "v1.0.0", nil }))
	if upToDate["checked"] != true || upToDate["updateAvailable"] != false {
		t.Fatalf("查过且已是最新时 checked 应为 true，实际 %v", upToDate)
	}

	// 检查被显式关掉：DisableUpdateCheck 为真时 NewWithDependencies 不会补上
	// 默认检查器，handler 拿到的 checker 就是 nil——与 procd 上"检查器没注册"
	// 走的是同一条分支。
	disabled, err := NewWithDependencies(
		Config{Version: "v1.0.0", DisableUpdateCheck: true},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	for name, application := range map[string]*App{
		"检查已关闭": disabled,
		"dev 构建": newUpdateCheckApp(t, "dev",
			func(context.Context) (string, error) { return "v9.9.9", nil }),
	} {
		payload := fetchPanelUpdate(t, application)
		if payload["checked"] != false {
			t.Fatalf("%s：checked 应为 false，实际 %v", name, payload)
		}
		if _, present := payload["latest"]; present {
			t.Fatalf("%s：没查过就不该有 latest，实际 %v", name, payload)
		}
	}
}

// 检查失败要如实带出错误并短缓存，而不是假装没有新版本还长期缓存失败。
func TestPanelUpdateCheckReportsError(t *testing.T) {
	application := newUpdateCheckApp(t, "v0.1.2", func(context.Context) (string, error) {
		return "", errors.New("上游不可达")
	})
	payload := fetchPanelUpdate(t, application)
	if payload["updateAvailable"] != false || payload["error"] != "上游不可达" {
		t.Fatalf("应带出检查失败原因，实际 %v", payload)
	}
}

func TestVersionBehind(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v0.1.2", "v0.2.0", true},
		{"v0.1.2", "v0.1.2", false},
		{"v0.2.0", "v0.1.9", false},
		{"v0.9.9", "v1.0.0", true},
		{"v1.0.0-rc.1", "v1.0.0", true},
		{"v1.0.0", "v1.0.0-rc.1", false},
		{"dev", "v1.0.0", false},
		{"v0.1.2", "not-a-version", false},
		// build metadata 不参与优先级比较（semver 规范）。快照构建靠这一条与它所
		// 基于的 tag 平级：领先 8 个提交不该被判成落后。
		{"v1.0.0+git8.5df15b7", "v1.0.0", false},
		{"v1.0.0+git8.5df15b7", "v1.1.0", true},
		{"v1.0.0-rc1+git3.abc1234", "v1.0.0", true},
		{"v1.0.0+", "v1.1.0", false},
	}
	for _, item := range cases {
		if got := versionBehind(item.current, item.latest); got != item.want {
			t.Fatalf("versionBehind(%q, %q) = %v，期望 %v", item.current, item.latest, got, item.want)
		}
	}
}

// 编进二进制的版本号由 scripts/panel-version.sh 算出，Makefile 与 OpenWrt 流水线
// 共用它。这里把那个脚本的产出直接喂给面板的比较器：形状与语义在同一个测试里钉死，
// 改了脚本而没改比较器（或反过来）会立刻红。
//
// 两种错法都不报错，症状全发生在用户的路由器上：
//
//   - 认不出（分支名 "main"、"123/merge"）：checked 恒为 false，设置页显示"本部署不做
//     新版本检查"，而检查能力明明注册着。
//   - 排在所基于的 tag 之前（写死的 v0.0.1-git<n>.<hash>，或 git describe 原样的
//     v1.0.0-8-g<hash>——连字符后的部分都会被当作预发布段）：横幅恒亮"面板有新版本
//     v1.0.0"，而那一版比正在跑的代码还旧。
func TestPanelBinaryVersionsAreComparable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("脚本需要 POSIX sh")
	}
	script := filepath.Join("..", "..", "scripts", "panel-version.sh")
	version := func(describe, release string) string {
		t.Helper()
		command := exec.Command("sh", script, release)
		command.Env = append(os.Environ(), "KDAE_PANEL_DESCRIBE="+describe)
		out, err := command.Output()
		if err != nil {
			t.Fatalf("panel-version.sh describe=%q release=%q: %v", describe, release, err)
		}
		return strings.TrimSpace(string(out))
	}

	for _, item := range []struct {
		describe, release, want string
	}{
		{release: "v1.2.0", want: "v1.2.0"},                                      // release：tag 原样
		{release: "1.2.0", want: "v1.2.0"},                                       // 少写的 v 补上
		{describe: "v1.0.0", want: "v1.0.0"},                                     // 正好落在 tag 上
		{describe: "v1.0.0-dirty", want: "v1.0.0+dirty"},                         // 工作区有改动
		{describe: "v1.0.0-8-g5df15b7", want: "v1.0.0+git8.5df15b7"},             // 领先 tag 8 个提交
		{describe: "v1.0.0-8-g5df15b7-dirty", want: "v1.0.0+git8.5df15b7.dirty"}, //
		{describe: "v1.0.0-rc1-3-gabc1234", want: "v1.0.0-rc1+git3.abc1234"},     // 预发布 tag 之后
	} {
		got := version(item.describe, item.release)
		if got != item.want {
			t.Fatalf("describe=%q release=%q 得到 %q，期望 %q", item.describe, item.release, got, item.want)
		}
		if _, ok := parseSemver(got); !ok {
			t.Fatalf("parseSemver(%q) 解析失败，面板会据此关掉新版本检查", got)
		}
	}

	// 领先 tag 的快照与该 tag 平级。这正是用户报的那条误报：v1.0.0 之后 8 个提交
	// 构建出来的包，被劝去"升级"到 v1.0.0。
	snapshot := version("v1.0.0-8-g5df15b7", "")
	if versionBehind(snapshot, "v1.0.0") {
		t.Fatalf("%q 被判为落后于 v1.0.0：横幅会劝用户装回比在跑的代码更旧的包", snapshot)
	}
	// 但真出了新发布仍要如实提示，否则就成了"永不提醒"。
	if !versionBehind(snapshot, "v1.1.0") {
		t.Fatalf("%q 应当被判为落后于 v1.1.0", snapshot)
	}
	// 基于旧 tag 的分支快照同样要提示。
	if old := version("v0.9.3-5-gabc1234", ""); !versionBehind(old, "v1.0.0") {
		t.Fatalf("%q 应当被判为落后于 v1.0.0", old)
	}

	// 用户看到的是接口的输出，这里就走一遍完整的 handler：快照机器上"查过了、
	// 最新发布是 v1.0.0、但没有可升的版本"三件事必须同时成立。少了 checked，
	// 界面会把它误当成"这个部署不做检查"。
	payload := fetchPanelUpdate(t, newUpdateCheckApp(t, snapshot,
		func(context.Context) (string, error) { return "v1.0.0", nil }))
	if payload["checked"] != true || payload["latest"] != "v1.0.0" || payload["updateAvailable"] != false {
		t.Fatalf("快照 %q 对 v1.0.0 的检查结果 = %v", snapshot, payload)
	}

	// 算不出一个能用的版本号时宁可让构建失败：两种症状都要等用户打开面板才看得见。
	for _, describe := range []string{
		"nightly-3-gabc1234", // tag 本身不是 semver → 面板会关掉检查
		// 转换没生效（这里用一个 sed 认不出的哈希模拟）。原样输出本身是合法
		// semver，形状断言拦不住，但那个领先段会被当成预发布排到 tag 之前。
		"v1.0.0-9-gXXXXXXX",
	} {
		broken := exec.Command("sh", script, "")
		broken.Env = append(os.Environ(), "KDAE_PANEL_DESCRIBE="+describe)
		if out, err := broken.Output(); err == nil {
			t.Fatalf("describe=%q 应当让构建失败，实际输出 %q", describe, strings.TrimSpace(string(out)))
		}
	}
}

type stubGeoService struct {
	status    geodata.Status
	err       error
	mu        sync.Mutex
	applied   int
	requested upstream.GeoSource
	custom    []upstream.CustomGeoSource
}

func (s *stubGeoService) Status(context.Context) geodata.Status { return s.status }

func (s *stubGeoService) Download(_ context.Context, source upstream.GeoSource) (upstream.GeoData, error) {
	s.mu.Lock()
	s.requested = source
	s.mu.Unlock()
	if s.err != nil {
		return upstream.GeoData{}, s.err
	}
	return upstream.GeoData{
		Release: upstream.GeoRelease{Source: source, Tag: "202607252248"},
		Files:   map[string][]byte{upstream.GeoIPName: []byte("ip"), upstream.GeoSiteName: []byte("site")},
	}, nil
}

func (s *stubGeoService) requestedSource() upstream.GeoSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requested
}

func (s *stubGeoService) Apply(context.Context, upstream.GeoData) (geodata.Status, error) {
	s.mu.Lock()
	s.applied++
	s.mu.Unlock()
	return s.status, s.err
}

func (s *stubGeoService) applyCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applied
}

func (s *stubGeoService) CustomSources() []upstream.CustomGeoSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]upstream.CustomGeoSource(nil), s.custom...)
}

func (s *stubGeoService) CreateCustomSource(source upstream.CustomGeoSource) (upstream.CustomGeoSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	source.ID = "0123456789abcdef"
	source.Source = "custom:" + upstream.GeoSource(source.ID)
	s.custom = append(s.custom, source)
	return source, nil
}

func (s *stubGeoService) UpdateCustomSource(id string, source upstream.CustomGeoSource) (upstream.CustomGeoSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.custom {
		if s.custom[index].ID == id {
			source.ID = id
			source.Source = upstream.GeoSource("custom:" + id)
			s.custom[index] = source
			return source, nil
		}
	}
	return upstream.CustomGeoSource{}, upstream.ErrCustomGeoSourceNotFound
}

func (s *stubGeoService) DeleteCustomSource(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.custom {
		if s.custom[index].ID == id {
			s.custom = append(s.custom[:index], s.custom[index+1:]...)
			return nil
		}
	}
	return upstream.ErrCustomGeoSourceNotFound
}

func newGeoApp(t *testing.T, service GeoService) *App {
	t.Helper()
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Geo: service},
	)
	if err != nil {
		t.Fatal(err)
	}
	return application
}

// 定时轮次在 apply 阶段拿不到操作锁必须跳过并如实记录，绝不排队：
// 排在一个可能长达数分钟的安装操作后面，会让调度器"失败后尽快重试"
// 的短间隔退化成长时间悬挂。
func TestScheduledGeoUpdateSkipsWhenOperationBusy(t *testing.T) {
	service := &stubGeoService{status: geodata.Status{
		DefaultSource: upstream.GeoSourceLoyalsoldier,
		Updatable:     true,
	}}
	operations := &sync.Mutex{}
	updater := newGeoUpdater(service, operations, slog.New(slog.NewTextHandler(io.Discard, nil)))

	operations.Lock()
	defer operations.Unlock()
	err := updater.runScheduled(context.Background())
	if err == nil || !strings.Contains(err.Error(), "本轮已跳过") {
		t.Fatalf("锁被占用时应跳过本轮，err = %v", err)
	}
	if service.applyCount() != 0 {
		t.Fatal("跳过的轮次不应写入任何数据")
	}
	// 跳过的原因必须进任务状态，geo 卡片才能解释"为什么没更新"
	job := updater.jobs.snapshot()
	if job.Phase != PhaseFailed || !strings.Contains(job.Error, "本轮已跳过") {
		t.Fatalf("任务状态应记录跳过原因，实际 %+v", job)
	}
}

// 定时轮次必须沿用上次的来源：静默改换规则集会改变 geosite: 规则的含义。
func TestScheduledGeoUpdateFollowsDefaultSource(t *testing.T) {
	service := &stubGeoService{status: geodata.Status{
		DefaultSource: upstream.GeoSourceV2fly,
		Updatable:     true,
	}}
	updater := newGeoUpdater(service, &sync.Mutex{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := updater.runScheduled(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := service.requestedSource(); got != upstream.GeoSourceV2fly {
		t.Fatalf("下载来源 = %q，期望沿用状态里的 %q", got, upstream.GeoSourceV2fly)
	}
	if service.applyCount() != 1 {
		t.Fatalf("应用次数 = %d，期望 1", service.applyCount())
	}
}

// 手动任务在跑时定时轮次必须让路，反之亦然——追踪器只有一份。
func TestScheduledGeoUpdateYieldsToRunningJob(t *testing.T) {
	service := &stubGeoService{status: geodata.Status{Updatable: true}}
	updater := newGeoUpdater(service, &sync.Mutex{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if !updater.jobs.begin(PhaseDownloading, "manual", "", "geo 数据") {
		t.Fatal("预置手动任务失败")
	}
	err := updater.runScheduled(context.Background())
	if err == nil || !strings.Contains(err.Error(), "已有 geo 更新任务") {
		t.Fatalf("已有任务时应拒绝，err = %v", err)
	}
	if service.applyCount() != 0 {
		t.Fatal("被拒绝的轮次不应写入任何数据")
	}
}

// geo 自动更新的设置端点随 geo 功能一起出现，路径与订阅刷新平行。
func TestGeoScheduleRoutes(t *testing.T) {
	service := &stubGeoService{status: geodata.Status{Updatable: true}}
	application, err := NewWithDependencies(
		Config{Version: "test-panel", GeoSchedulePath: filepath.Join(t.TempDir(), "geo-schedule.json")},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Geo: service},
	)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/schedule/geo", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET 状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}

	putRecorder := httptest.NewRecorder()
	putRequest := httptest.NewRequest(http.MethodPut, "/api/v1/schedule/geo",
		strings.NewReader(`{"enabled":true,"intervalMinutes":4320}`))
	application.Handler().ServeHTTP(putRecorder, putRequest)
	if putRecorder.Code != http.StatusOK {
		t.Fatalf("PUT 状态码 = %d，响应 = %s", putRecorder.Code, putRecorder.Body.String())
	}
	if !strings.Contains(putRecorder.Body.String(), `"intervalMinutes":4320`) {
		t.Fatalf("PUT 应回读生效设置: %s", putRecorder.Body.String())
	}

	// geo 功能关闭时，它的调度端点也应报告未初始化而不是 404
	disabled := newGeoApp(t, nil)
	disabledRecorder := httptest.NewRecorder()
	disabled.Handler().ServeHTTP(disabledRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/schedule/geo", nil))
	if disabledRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("功能关闭时状态码 = %d，期望 503", disabledRecorder.Code)
	}
}

// 未注入 geo 服务即代表功能关闭。必须返回可读的 geo_update_disabled，
// 而不是落到 api_not_found——前端正是按这个错误码分支处理的。
func TestGeoUpdateDisabledByDefault(t *testing.T) {
	application := newGeoApp(t, nil)
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/dae/geo", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/dae/geo", nil),
	} {
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s 状态码 = %d，期望 503", request.Method, recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), "geo_update_disabled") {
			t.Fatalf("响应应说明功能未启用: %s", recorder.Body.String())
		}
	}
}

// geo 更新与 dae 版本管理是两个独立开关：只开 geo 的部署是正常情况。
func TestGeoUpdateWorksWithoutDaeInstall(t *testing.T) {
	service := &stubGeoService{status: geodata.Status{
		Sources:       []upstream.GeoSourceInfo{{Source: upstream.GeoSourceLoyalsoldier}},
		DefaultSource: upstream.GeoSourceLoyalsoldier,
		TargetDir:     "/etc/dae",
		Updatable:     true,
	}}
	application := newGeoApp(t, service)

	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/dae/geo", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	// dae 版本管理没开，它的端点仍应各自返回自己的未启用提示
	installRecorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(installRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/dae/install", nil))
	if !strings.Contains(installRecorder.Body.String(), "dae_install_disabled") {
		t.Fatalf("dae 版本管理应报告自己未启用: %s", installRecorder.Body.String())
	}
}

func TestGeoUpdateRunsAsynchronously(t *testing.T) {
	service := &stubGeoService{status: geodata.Status{Updatable: true}}
	application := newGeoApp(t, service)

	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/dae/geo", nil))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("状态码 = %d，期望 202，响应 = %s", recorder.Code, recorder.Body.String())
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && service.applyCount() == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if service.applyCount() != 1 {
		t.Fatalf("应恰好应用一次，实际 %d 次", service.applyCount())
	}
}

// 请求体里指定的来源必须真正透传下去；两个来源的规则集不是同一套，
// 选了 v2fly 却装了 Loyalsoldier 会静默改变路由行为。
func TestGeoUpdateHonoursRequestedSource(t *testing.T) {
	service := &stubGeoService{status: geodata.Status{
		DefaultSource: upstream.GeoSourceLoyalsoldier,
		Updatable:     true,
	}}
	application := newGeoApp(t, service)

	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/dae/geo",
		strings.NewReader(`{"source":"v2fly"}`)))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("状态码 = %d，响应 = %s", recorder.Code, recorder.Body.String())
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && service.requestedSource() == "" {
		time.Sleep(5 * time.Millisecond)
	}
	if got := service.requestedSource(); got != upstream.GeoSourceV2fly {
		t.Fatalf("上游收到的来源 = %q，期望 v2fly", got)
	}
}

// 省略来源时沿用状态里给出的默认值（上次用过的那个）。
func TestGeoUpdateFallsBackToDefaultSource(t *testing.T) {
	service := &stubGeoService{status: geodata.Status{
		DefaultSource: upstream.GeoSourceV2fly,
		Updatable:     true,
	}}
	application := newGeoApp(t, service)

	application.Handler().ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/api/v1/dae/geo", nil))
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && service.requestedSource() == "" {
		time.Sleep(5 * time.Millisecond)
	}
	if got := service.requestedSource(); got != upstream.GeoSourceV2fly {
		t.Fatalf("应沿用默认来源，实际 %q", got)
	}
}

func TestGeoUpdateRejectsUnknownSource(t *testing.T) {
	service := &stubGeoService{status: geodata.Status{Updatable: true}}
	application := newGeoApp(t, service)

	for _, body := range []string{`{"source":"evil"}`, `{"source":"../etc"}`, `{"source":"v2rayA"}`} {
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/dae/geo",
			strings.NewReader(body)))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s 状态码 = %d，期望 400", body, recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), "invalid_geo_source") {
			t.Fatalf("错误码应为 invalid_geo_source: %s", recorder.Body.String())
		}
	}
	if service.requestedSource() != "" {
		t.Fatal("来源非法时不该发起任何下载")
	}
}

func TestGeoCustomSourceCRUD(t *testing.T) {
	service := &stubGeoService{status: geodata.Status{Updatable: true}}
	application := newGeoApp(t, service)
	body := `{"label":"自建规则","geoipUrl":"https://example.com/geoip.dat",` +
		`"geoipSha256Url":"https://example.com/geoip.dat.sha256sum",` +
		`"geositeUrl":"https://example.com/geosite.dat",` +
		`"geositeSha256Url":"https://example.com/geosite.dat.sha256sum"}`

	create := httptest.NewRecorder()
	application.Handler().ServeHTTP(create, httptest.NewRequest(http.MethodPost,
		"/api/v1/dae/geo/sources", strings.NewReader(body)))
	if create.Code != http.StatusCreated || !strings.Contains(create.Body.String(), "custom:0123456789abcdef") {
		t.Fatalf("创建来源失败：%d %s", create.Code, create.Body.String())
	}

	update := httptest.NewRecorder()
	application.Handler().ServeHTTP(update, httptest.NewRequest(http.MethodPut,
		"/api/v1/dae/geo/sources/0123456789abcdef",
		strings.NewReader(strings.Replace(body, "自建规则", "更新后的规则", 1))))
	if update.Code != http.StatusOK || !strings.Contains(update.Body.String(), "更新后的规则") {
		t.Fatalf("修改来源失败：%d %s", update.Code, update.Body.String())
	}

	list := httptest.NewRecorder()
	application.Handler().ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/dae/geo/sources", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "更新后的规则") {
		t.Fatalf("来源列表异常：%d %s", list.Code, list.Body.String())
	}

	remove := httptest.NewRecorder()
	application.Handler().ServeHTTP(remove, httptest.NewRequest(http.MethodDelete,
		"/api/v1/dae/geo/sources/0123456789abcdef", nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("删除来源失败：%d %s", remove.Code, remove.Body.String())
	}

	empty := httptest.NewRecorder()
	application.Handler().ServeHTTP(empty, httptest.NewRequest(http.MethodGet, "/api/v1/dae/geo/sources", nil))
	if empty.Code != http.StatusOK || !strings.Contains(empty.Body.String(), `"sources":[]`) {
		t.Fatalf("空来源列表必须编码成数组：%d %s", empty.Code, empty.Body.String())
	}
}

func TestGeoCustomSourceInUseCannotBeDeleted(t *testing.T) {
	const id = "0123456789abcdef"
	service := &stubGeoService{
		status: geodata.Status{Managed: &geodata.State{Source: upstream.GeoSource("custom:" + id)}},
		custom: []upstream.CustomGeoSource{{ID: id, Source: upstream.GeoSource("custom:" + id), Label: "使用中"}},
	}
	application := newGeoApp(t, service)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete,
		"/api/v1/dae/geo/sources/"+id, nil))
	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), "geo_source_in_use") {
		t.Fatalf("使用中的来源应拒绝删除：%d %s", recorder.Code, recorder.Body.String())
	}
}

func TestGeoUpdateReportsFailure(t *testing.T) {
	service := &stubGeoService{
		status: geodata.Status{Updatable: true},
		err:    errors.New("校验和不匹配"),
	}
	application := newGeoApp(t, service)
	application.Handler().ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/api/v1/dae/geo", nil))

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		recorder := httptest.NewRecorder()
		application.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/dae/geo", nil))
		var payload struct {
			Job Job `json:"job"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Job.Phase == PhaseFailed {
			if !strings.Contains(payload.Job.Error, "校验和") {
				t.Fatalf("失败原因应透传: %q", payload.Job.Error)
			}
			return
		}
		if payload.Job.Phase == PhaseDone {
			t.Fatal("下载失败时任务不该报告成功")
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("任务迟迟没有结束")
}

func TestDaeInstallRequiresAuthentication(t *testing.T) {
	session := auth.Session{Token: "session", CSRFToken: "csrf", ExpiresAt: time.Now().Add(time.Hour), User: auth.User{ID: 1, Username: "admin"}}
	service := &stubInstallService{}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{
			Dae:            stubDaeService{},
			Install:        service,
			Authentication: &stubAuthenticationService{initialized: true, session: session},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	anonymous := httptest.NewRecorder()
	application.Handler().ServeHTTP(anonymous, httptest.NewRequest(http.MethodPost, "/api/v1/dae/install", strings.NewReader(`{"source":"official","ref":"v2.0.0"}`)))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("未登录状态码 = %d", anonymous.Code)
	}

	withoutCSRF := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/dae/install", strings.NewReader(`{"source":"official","ref":"v2.0.0"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session.Token})
	application.Handler().ServeHTTP(withoutCSRF, request)
	if withoutCSRF.Code != http.StatusForbidden {
		t.Fatalf("缺少 CSRF 状态码 = %d", withoutCSRF.Code)
	}
	if installed := service.records(); len(installed) != 0 {
		t.Fatalf("未授权请求不应触发安装: %v", installed)
	}
}

func TestDiagnosticReportRequiresAuthentication(t *testing.T) {
	session := auth.Session{
		Token: "session", CSRFToken: "csrf", ExpiresAt: time.Now().Add(time.Hour),
		User: auth.User{ID: 1, Username: "admin"},
	}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{
			Dae:            stubDaeService{report: dae.Report{Available: true}},
			Authentication: &stubAuthenticationService{initialized: true, session: session},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	anonymous := httptest.NewRecorder()
	application.Handler().ServeHTTP(anonymous, httptest.NewRequest(http.MethodGet, "/api/v1/diagnostics/report", nil))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("未登录状态码 = %d", anonymous.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/diagnostics/report", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session.Token})
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("诊断报告状态码 = %d，响应 = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Overall string `json:"overall"`
		Counts  struct {
			OK, Warning, Error, Unknown int
		} `json:"counts"`
		Items []json.RawMessage `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	total := payload.Counts.OK + payload.Counts.Warning + payload.Counts.Error + payload.Counts.Unknown
	if payload.Overall == "" || len(payload.Items) != 9 || total != len(payload.Items) {
		t.Fatalf("诊断报告结构异常: overall=%q counts=%+v items=%d", payload.Overall, payload.Counts, len(payload.Items))
	}
}

func TestLatencyProbeRequiresAuthentication(t *testing.T) {
	session := auth.Session{Token: "session", CSRFToken: "csrf", ExpiresAt: time.Now().Add(time.Hour), User: auth.User{ID: 1, Username: "admin"}}
	prober := &stubProbeService{}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{
			Dae:            stubDaeService{},
			Probe:          prober,
			Authentication: &stubAuthenticationService{initialized: true, session: session},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	body := func() *strings.Reader {
		return strings.NewReader(`{"targets":[{"host":"example.com","port":443}]}`)
	}

	anonymous := httptest.NewRecorder()
	application.Handler().ServeHTTP(anonymous, httptest.NewRequest(http.MethodPost, "/api/v1/net/latency", body()))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("未登录状态码 = %d，响应 = %s", anonymous.Code, anonymous.Body.String())
	}

	withoutCSRFRequest := httptest.NewRequest(http.MethodPost, "/api/v1/net/latency", body())
	withoutCSRFRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session.Token})
	withoutCSRF := httptest.NewRecorder()
	application.Handler().ServeHTTP(withoutCSRF, withoutCSRFRequest)
	if withoutCSRF.Code != http.StatusForbidden {
		t.Fatalf("缺少 CSRF 状态码 = %d，响应 = %s", withoutCSRF.Code, withoutCSRF.Body.String())
	}
	if prober.targets != nil {
		t.Fatalf("未授权请求不应触发探测: %+v", prober.targets)
	}
}

func TestAuthenticationProtectsAPIAndChecksCSRF(t *testing.T) {
	session := auth.Session{
		Token:     "session-token",
		CSRFToken: "csrf-token",
		ExpiresAt: time.Now().Add(time.Hour),
		User:      auth.User{ID: 1, Username: "admin"},
	}
	authService := &stubAuthenticationService{initialized: true, session: session}
	configuration := stubConfigurationService{}
	application, err := NewWithDependencies(
		Config{Version: "test-panel"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{
			Dae:            stubDaeService{},
			Configuration:  configuration,
			Authentication: authService,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	unauthorized := httptest.NewRecorder()
	application.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/config", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("未登录状态码 = %d", unauthorized.Code)
	}

	withoutCSRFRequest := httptest.NewRequest(http.MethodPut, "/api/v1/config", strings.NewReader(`{"content":"test"}`))
	withoutCSRFRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session.Token})
	withoutCSRF := httptest.NewRecorder()
	application.Handler().ServeHTTP(withoutCSRF, withoutCSRFRequest)
	if withoutCSRF.Code != http.StatusForbidden {
		t.Fatalf("缺少 CSRF 状态码 = %d，响应 = %s", withoutCSRF.Code, withoutCSRF.Body.String())
	}

	validRequest := httptest.NewRequest(http.MethodPut, "/api/v1/config", strings.NewReader(`{"content":"test"}`))
	validRequest.Host = "panel.example"
	validRequest.Header.Set("Origin", "http://panel.example")
	validRequest.Header.Set("X-CSRF-Token", session.CSRFToken)
	validRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session.Token})
	validResponse := httptest.NewRecorder()
	application.Handler().ServeHTTP(validResponse, validRequest)
	if validResponse.Code != http.StatusOK {
		t.Fatalf("有效请求状态码 = %d，响应 = %s", validResponse.Code, validResponse.Body.String())
	}
}

func TestSetupRequiresBootstrapTokenAndClosesAfterInitialization(t *testing.T) {
	session := auth.Session{Token: "session", CSRFToken: "csrf", ExpiresAt: time.Now().Add(time.Hour), User: auth.User{ID: 1, Username: "admin"}}
	authService := &stubAuthenticationService{session: session}
	setupURLFile := filepath.Join(t.TempDir(), "setup-url")
	if err := os.WriteFile(setupURLFile, []byte("one-time-link\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	application, err := NewWithDependencies(
		Config{Version: "test", BootstrapToken: "bootstrap-secret", SetupURLFile: setupURLFile},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		Dependencies{Dae: stubDaeService{}, Authentication: authService},
	)
	if err != nil {
		t.Fatal(err)
	}

	rejected := httptest.NewRecorder()
	rejectedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/bootstrap", strings.NewReader(`{"token":"wrong"}`))
	application.Handler().ServeHTTP(rejected, rejectedRequest)
	if rejected.Code != http.StatusForbidden || authService.setupCalls != 0 {
		t.Fatalf("错误 token 响应: status=%d calls=%d", rejected.Code, authService.setupCalls)
	}

	unauthorized := httptest.NewRecorder()
	unauthorizedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup", strings.NewReader(`{"username":"admin","password":"a secure test password"}`))
	application.Handler().ServeHTTP(unauthorized, unauthorizedRequest)
	if unauthorized.Code != http.StatusForbidden || authService.setupCalls != 0 {
		t.Fatalf("缺少初始化授权响应: status=%d calls=%d", unauthorized.Code, authService.setupCalls)
	}

	authorized := httptest.NewRecorder()
	authorizedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/bootstrap", strings.NewReader(`{"token":"bootstrap-secret"}`))
	application.Handler().ServeHTTP(authorized, authorizedRequest)
	if authorized.Code != http.StatusNoContent {
		t.Fatalf("初始化链接授权失败: status=%d body=%s", authorized.Code, authorized.Body)
	}
	var setupCookie *http.Cookie
	for _, cookie := range authorized.Result().Cookies() {
		if cookie.Name == setupAuthorizationCookieName {
			setupCookie = cookie
			break
		}
	}
	if setupCookie == nil || setupCookie.HttpOnly != true || setupCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("初始化授权 Cookie 异常: %+v", setupCookie)
	}

	accepted := httptest.NewRecorder()
	acceptedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup", strings.NewReader(`{"username":"admin","password":"a secure test password"}`))
	acceptedRequest.AddCookie(setupCookie)
	application.Handler().ServeHTTP(accepted, acceptedRequest)
	if accepted.Code != http.StatusCreated || authService.setupCalls != 1 {
		t.Fatalf("正确 token 响应: status=%d body=%s calls=%d", accepted.Code, accepted.Body, authService.setupCalls)
	}
	if _, err := os.Stat(setupURLFile); !os.IsNotExist(err) {
		t.Fatalf("管理员创建后首次访问链接文件仍然存在: %v", err)
	}

	authService.initialized = true
	closed := httptest.NewRecorder()
	closedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup", strings.NewReader(`{"username":"admin","password":"short"}`))
	application.Handler().ServeHTTP(closed, closedRequest)
	if closed.Code != http.StatusConflict || authService.setupCalls != 1 {
		t.Fatalf("已初始化响应: status=%d calls=%d", closed.Code, authService.setupCalls)
	}
}

func TestSetupAuthorizationExpiresAndRejectsTampering(t *testing.T) {
	expiresAt := time.Date(2026, 7, 23, 12, 10, 0, 0, time.UTC)
	value := newSetupAuthorization("bootstrap-secret", expiresAt)

	validRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
	validRequest.AddCookie(&http.Cookie{Name: setupAuthorizationCookieName, Value: value})
	if !validSetupAuthorization(validRequest, "bootstrap-secret", expiresAt.Add(-time.Second)) {
		t.Fatal("未过期的初始化授权应当有效")
	}
	if validSetupAuthorization(validRequest, "bootstrap-secret", expiresAt) {
		t.Fatal("到期的初始化授权应当失效")
	}

	tamperedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
	tamperedRequest.AddCookie(&http.Cookie{Name: setupAuthorizationCookieName, Value: value + "x"})
	if validSetupAuthorization(tamperedRequest, "bootstrap-secret", expiresAt.Add(-time.Second)) {
		t.Fatal("被篡改的初始化授权不应通过")
	}
}

func TestBootstrapSetupURLUsesLoopbackForWildcardListen(t *testing.T) {
	got := bootstrapSetupURL("0.0.0.0:2023", "secret")
	want := "http://127.0.0.1:2023/setup#bootstrap=secret"
	if got != want {
		t.Fatalf("初始化链接 = %q，期望 %q", got, want)
	}
}

func TestDefaultConfigListensOnLAN(t *testing.T) {
	config := DefaultConfig()
	if got := config.ListenAddress; got != "0.0.0.0:2023" {
		t.Fatalf("默认监听地址 = %q，期望同时接受本机和局域网连接", got)
	}
	if !config.EnableDaeInstall {
		t.Fatal("dae 版本管理应默认开启")
	}
	if !config.EnableSelfUpdate {
		t.Fatal("面板一键升级应默认开启，并允许在界面关闭")
	}
}

// 链接的 fragment 里带着 bootstrap token，公网 IPv4 必须留在名单外：路由器上那种
// 地址在 WAN 侧，把它渲染成可点链接等于劝管理员把 token 沿明文 HTTP 发到互联网上。
func TestBootstrapSetupURLsIncludeLANIPv4AddressesOnly(t *testing.T) {
	addresses := []net.Addr{
		&net.IPNet{IP: net.ParseIP("192.168.50.8"), Mask: net.CIDRMask(24, 32)},
		&net.IPNet{IP: net.ParseIP("10.20.30.40"), Mask: net.CIDRMask(8, 32)},
		&net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)},
		&net.IPNet{IP: net.ParseIP("203.0.113.7"), Mask: net.CIDRMask(24, 32)},
	}
	got := bootstrapSetupURLsForAddresses("0.0.0.0:2023", "secret", addresses)
	want := []string{
		"http://10.20.30.40:2023/setup#bootstrap=secret",
		"http://192.168.50.8:2023/setup#bootstrap=secret",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("初始化链接 = %q，期望 %q", got, want)
	}
}

func TestBootstrapSetupURLsFallBackToLoopbackWithoutLAN(t *testing.T) {
	addresses := []net.Addr{
		&net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)},
		&net.IPNet{IP: net.ParseIP("203.0.113.7"), Mask: net.CIDRMask(24, 32)},
		&net.IPNet{IP: net.ParseIP("224.0.0.1"), Mask: net.CIDRMask(24, 32)},
	}
	got := bootstrapSetupURLsForAddresses("0.0.0.0:2023", "secret", addresses)
	want := "http://127.0.0.1:2023/setup#bootstrap=secret"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("无内网地址时的初始化链接 = %q，期望 %q", got, want)
	}
}

func TestBootstrapSetupURLIPv6WildcardFallsBackToIPv6Loopback(t *testing.T) {
	got := bootstrapSetupURL("[::]:2023", "secret")
	want := "http://[::1]:2023/setup#bootstrap=secret"
	if got != want {
		t.Fatalf("IPv6 wildcard 初始化链接 = %q，期望 %q", got, want)
	}
}

func TestBootstrapSetupURLsIncludeUsableIPv6AndCGNATAddresses(t *testing.T) {
	addresses := []net.Addr{
		&net.IPNet{IP: net.ParseIP("100.64.1.2"), Mask: net.CIDRMask(10, 32)},
		&net.IPNet{IP: net.ParseIP("fd00::5"), Mask: net.CIDRMask(64, 128)},
		&net.IPNet{IP: net.ParseIP("2001:db8::5"), Mask: net.CIDRMask(64, 128)},
		&net.IPNet{IP: net.ParseIP("fe80::1"), Mask: net.CIDRMask(64, 128)},
		&net.IPNet{IP: net.ParseIP("::1"), Mask: net.CIDRMask(128, 128)},
	}
	got := bootstrapSetupURLsForAddresses("[::]:2023", "secret", addresses)
	want := []string{
		"http://100.64.1.2:2023/setup#bootstrap=secret",
		"http://[2001:db8::5]:2023/setup#bootstrap=secret",
		"http://[fd00::5]:2023/setup#bootstrap=secret",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("IPv6 wildcard 初始化链接 = %q，期望 %q", got, want)
	}
}

func TestBootstrapSetupURLsDeduplicateAddresses(t *testing.T) {
	addresses := []net.Addr{
		&net.IPNet{IP: net.ParseIP("192.168.1.1"), Mask: net.CIDRMask(24, 32)},
		&net.IPAddr{IP: net.ParseIP("192.168.1.1")},
	}
	got := bootstrapSetupURLsForAddresses("0.0.0.0:2023", "secret", addresses)
	if len(got) != 1 || got[0] != "http://192.168.1.1:2023/setup#bootstrap=secret" {
		t.Fatalf("重复地址应只生成一条链接，实际 %q", got)
	}
}

func TestBootstrapSetupURLEscapesConfiguredToken(t *testing.T) {
	token := "token with & + #"
	parsed, err := url.Parse(bootstrapSetupURL("127.0.0.1:2023", token))
	if err != nil {
		t.Fatal(err)
	}
	values, err := url.ParseQuery(parsed.EscapedFragment())
	if err != nil {
		t.Fatal(err)
	}
	if values.Get("bootstrap") != token {
		t.Fatalf("初始化链接 token = %q，期望 %q", values.Get("bootstrap"), token)
	}
}

func TestSyncSetupURLFileWritesPrivateFileAndRemovesStaleFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime", "setup-url")
	urls := []string{
		"http://10.0.0.2:2023/setup#bootstrap=first",
		"http://10.0.0.3:2023/setup#bootstrap=second",
	}
	if err := syncSetupURLFile(path, urls); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), strings.Join(urls, "\n")+"\n"; got != want {
		t.Fatalf("交接文件内容 = %q，期望 %q", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); runtime.GOOS != "windows" && got != 0o600 {
		t.Fatalf("交接文件权限 = %o，期望 600", got)
	}
	if err := syncSetupURLFile(path, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("已初始化启动后旧交接文件仍然存在: %v", err)
	}
}

func TestLoginLimiterBlocksRepeatedFailures(t *testing.T) {
	limiter := newLoginLimiter()
	now := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	for range 5 {
		limiter.Failure("address\x00127.0.0.1")
	}
	retryAfter, allowed := limiter.Allow("address\x00127.0.0.1")
	if allowed || retryAfter != 15*time.Minute {
		t.Fatalf("限速结果异常: allowed=%v retryAfter=%v", allowed, retryAfter)
	}
	limiter.Success("address\x00127.0.0.1")
	if _, allowed := limiter.Allow("address\x00127.0.0.1"); !allowed {
		t.Fatal("成功登录后应该清除限速状态")
	}
}

func TestLoginLimiterBoundsState(t *testing.T) {
	limiter := newLoginLimiter()
	limiter.now = func() time.Time { return time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC) }
	for index := 0; index < maxLoginAttempts+100; index++ {
		limiter.Failure("address\x00" + strconv.Itoa(index))
	}
	if len(limiter.attempts) != maxLoginAttempts {
		t.Fatalf("限速状态数量 = %d", len(limiter.attempts))
	}
}

func TestProxyTrustUsesForwardedClientAndScheme(t *testing.T) {
	trust, err := parseProxyTrust("127.0.0.0/8,::1/128")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://panel.example/api/v1/auth/login", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Host = "panel.example"
	request.Header.Set("X-Forwarded-For", "198.51.100.10, 127.0.0.2")
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("Origin", "https://panel.example")

	keys := loginKeys(request, trust)
	if len(keys) != 1 || keys[0] != "address\x00198.51.100.10" {
		t.Fatalf("登录限速键 = %q", keys)
	}
	if !sameOrigin(request, trust) {
		t.Fatal("可信代理的 HTTPS 同源请求应该通过")
	}
}

func TestUntrustedForwardedHeadersIgnored(t *testing.T) {
	trust, err := parseProxyTrust("127.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://panel.example/api/v1/auth/login", nil)
	request.RemoteAddr = "198.51.100.20:12345"
	request.Host = "panel.example"
	request.Header.Set("X-Forwarded-For", "203.0.113.9")
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("Origin", "https://panel.example")

	if got := trust.clientAddress(request); got != "198.51.100.20" {
		t.Fatalf("客户端地址 = %q", got)
	}
	if sameOrigin(request, trust) {
		t.Fatal("不可信来源不能伪造 HTTPS 同源请求")
	}
}

func TestSecurityHeadersHonorTrustedHTTPSProxy(t *testing.T) {
	trust, _ := parseProxyTrust("127.0.0.0/8")
	handler := securityHeaders(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}), trust)
	request := httptest.NewRequest(http.MethodGet, "http://panel.example/", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("可信 HTTPS 代理后应发送 HSTS")
	}
}

// newBackendTestConfig 造一份指向临时目录的最小可用配置。
func newBackendTestConfig(t *testing.T, backend host.Backend) Config {
	t.Helper()
	directory := t.TempDir()
	cfg := DefaultConfig()
	cfg.ServiceBackend = backend
	cfg.ListenAddress = "127.0.0.1:0"
	cfg.DatabasePath = filepath.Join(directory, "panel.db")
	cfg.BackupDir = filepath.Join(directory, "backups")
	cfg.SchedulePath = filepath.Join(directory, "schedule.json")
	cfg.InstallStatePath = filepath.Join(directory, "dae-install.json")
	cfg.GeoStatePath = filepath.Join(directory, "geo-update.json")
	cfg.GeoSchedulePath = filepath.Join(directory, "geo-schedule.json")
	cfg.PanelBackupPath = filepath.Join(directory, "kdae-panel.previous")
	cfg.DaeConfigPath = filepath.Join(directory, "config.dae")
	// 版本管理会去探测上游平台并写系统目录，与本测试无关，关掉。
	cfg.EnableDaeInstall = false
	// New() 建出的是真实应用，/api/v1/panel/update 不在认证中间件的公开
	// 路径白名单里；固定 token 是为了让测试能自己跑完一遍初始化拿到会话，
	// 而不必依赖 New() 内部随机生成、外部读不到的那个值。
	cfg.BootstrapToken = "backend-test-bootstrap-token"
	return cfg
}

// adminSessionCookie 走一遍真实的首次初始化流程（bootstrap → setup），
// 换回登录会话 Cookie。New() 建的应用给非公开的 /api/v1/* 路径都挂了认证
// 中间件，面板更新接口不在白名单内，后续请求必须带着这张 Cookie 才能穿过去。
func adminSessionCookie(t *testing.T, application *App, bootstrapToken string) *http.Cookie {
	t.Helper()
	bootstrap := httptest.NewRecorder()
	application.Handler().ServeHTTP(bootstrap, httptest.NewRequest(http.MethodPost, "/api/v1/auth/bootstrap",
		strings.NewReader(`{"token":"`+bootstrapToken+`"}`)))
	if bootstrap.Code != http.StatusNoContent {
		t.Fatalf("初始化链接授权失败: status=%d body=%s", bootstrap.Code, bootstrap.Body)
	}
	var setupCookie *http.Cookie
	for _, cookie := range bootstrap.Result().Cookies() {
		if cookie.Name == setupAuthorizationCookieName {
			setupCookie = cookie
		}
	}
	if setupCookie == nil {
		t.Fatal("缺少初始化授权 Cookie")
	}

	setup := httptest.NewRecorder()
	setupRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup",
		strings.NewReader(`{"username":"admin","password":"a secure test password"}`))
	setupRequest.AddCookie(setupCookie)
	application.Handler().ServeHTTP(setup, setupRequest)
	if setup.Code != http.StatusCreated {
		t.Fatalf("创建管理员失败: status=%d body=%s", setup.Code, setup.Body)
	}
	for _, cookie := range setup.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			return cookie
		}
	}
	t.Fatal("缺少登录会话 Cookie")
	return nil
}

// panelUpdateStatusPresent 请求面板更新接口，回报响应里有没有 status 字段。
// status 缺失即代表自升级能力根本没注册（handler 的 nil 分支）。
func panelUpdateStatusPresent(t *testing.T, application *App, session *http.Cookie) bool {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/panel/update", nil)
	request.AddCookie(session)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d", recorder.Code, http.StatusOK)
	}
	var payload map[string]json.RawMessage
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	_, present := payload["status"]
	return present
}

// procd 部署一定不能提供自升级：它会从上游 tuoro/kdae-panel 取回一个不含
// procd 后端的二进制并替换自己，开启即自毁。这条断言防止以后有人"顺手"打开。
func TestProcdBackendDoesNotRegisterSelfUpdate(t *testing.T) {
	cfg := newBackendTestConfig(t, host.BackendProcd)
	application, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化应用: %v", err)
	}
	defer func() { _ = application.Close() }()
	session := adminSessionCookie(t, application, cfg.BootstrapToken)

	if panelUpdateStatusPresent(t, application, session) {
		t.Fatal("procd 后端下不应注册面板自升级能力")
	}
}

// panelUpdateCheck 取出响应里的 check 部分（需要会话的 New() 应用专用）。
func panelUpdateCheck(t *testing.T, application *App, session *http.Cookie) map[string]any {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/panel/update", nil)
	request.AddCookie(session)
	recorder := httptest.NewRecorder()
	application.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d", recorder.Code, http.StatusOK)
	}
	var payload struct {
		Check map[string]any `json:"check"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return payload.Check
}

// 升级不给，检查要给：procd 装的是 ipk，"有没有新版本"仍然值得知道。
// 但必须问对仓库——上游 tuoro/kdae-panel 发的是 systemd 用的裸二进制，
// 拿它的 tag 报出来的新版本既装不上也不该装。
func TestProcdBackendChecksPackageRepository(t *testing.T) {
	cfg := newBackendTestConfig(t, host.BackendProcd)
	// 非 semver 版本号让检查短路，测试不联网。releasesUrl 在检查之前就填好，
	// 断言的正是"面板认为该去哪个仓库问"这件事本身。
	cfg.Version = "dev"
	application, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化应用: %v", err)
	}
	defer func() { _ = application.Close() }()
	session := adminSessionCookie(t, application, cfg.BootstrapToken)

	check := panelUpdateCheck(t, application, session)
	want := upstream.ReleasesURL(upstream.PackageRepoOwner, upstream.PackageRepoName)
	if got, _ := check["releasesUrl"].(string); got != want {
		t.Fatalf("releasesUrl = %q，期望 %q", got, want)
	}
	if panelUpdateStatusPresent(t, application, session) {
		t.Fatal("检查放开不等于自升级放开：procd 下仍不该注册自升级能力")
	}
}

// 对照组：systemd 仍然问上游仓库。
func TestSystemdBackendChecksUpstreamRepository(t *testing.T) {
	cfg := newBackendTestConfig(t, host.BackendSystemd)
	cfg.Version = "dev"
	application, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化应用: %v", err)
	}
	defer func() { _ = application.Close() }()
	session := adminSessionCookie(t, application, cfg.BootstrapToken)

	check := panelUpdateCheck(t, application, session)
	want := upstream.ReleasesURL(upstream.PanelRepoOwner, upstream.PanelRepoName)
	if got, _ := check["releasesUrl"].(string); got != want {
		t.Fatalf("releasesUrl = %q，期望 %q", got, want)
	}
}

// 对照组：systemd 后端的行为必须原样保留，否则上面那条断言可能只是
// "配置造错了导致哪个后端都注册不上"。
func TestSystemdBackendStillRegistersSelfUpdate(t *testing.T) {
	cfg := newBackendTestConfig(t, host.BackendSystemd)
	application, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化应用: %v", err)
	}
	defer func() { _ = application.Close() }()
	session := adminSessionCookie(t, application, cfg.BootstrapToken)

	if !panelUpdateStatusPresent(t, application, session) {
		t.Fatal("systemd 后端下自升级能力应当照旧注册")
	}
}
