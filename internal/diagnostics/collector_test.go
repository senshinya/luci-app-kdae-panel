package diagnostics

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/configstore"
	"github.com/tuoro/kdae-panel/internal/dae"
	"github.com/tuoro/kdae-panel/internal/geodata"
	"github.com/tuoro/kdae-panel/internal/host"
)

type fakeDae struct{ report dae.Report }

func (f fakeDae) Inspect(context.Context) dae.Report { return f.report }

type fakeConfiguration struct {
	document configstore.Document
	readErr  error
	validErr error
}

func (f fakeConfiguration) Read(context.Context) (configstore.Document, error) {
	return f.document, f.readErr
}

func (f fakeConfiguration) Validate(context.Context, string) error { return f.validErr }

type fakeHost struct {
	status     host.Status
	statusErr  error
	logs       []host.LogEntry
	logsErr    error
	interfaces []host.NetworkInterface
	ifaceErr   error
}

func (f fakeHost) Status(context.Context) (host.Status, error) { return f.status, f.statusErr }
func (f fakeHost) Logs(context.Context, int) ([]host.LogEntry, error) {
	return f.logs, f.logsErr
}
func (f fakeHost) Interfaces(context.Context) ([]host.NetworkInterface, error) {
	return f.interfaces, f.ifaceErr
}

type fakeGeo struct{ status geodata.Status }

func (f fakeGeo) Status(context.Context) geodata.Status { return f.status }

type fakeSystem struct{ snapshot SystemSnapshot }

func (f fakeSystem) Snapshot(context.Context) SystemSnapshot { return f.snapshot }

func healthyCollector(now time.Time) *Collector {
	return New(Options{
		Dae: fakeDae{report: dae.Report{Available: true, Binary: "/usr/bin/dae", Version: "dae v2"}},
		Configuration: fakeConfiguration{document: configstore.Document{
			Path: "/etc/dae/config.dae", Content: "global {}", Hash: strings.Repeat("a", 64), Size: 9,
		}},
		Host: fakeHost{
			status:     host.Status{ActiveState: "active", SubState: "running", MainPID: 42, UnitFileState: "enabled"},
			interfaces: []host.NetworkInterface{{Name: "lo"}, {Name: "eth0", Addresses: []string{"192.0.2.2/24"}}},
		},
		Geo: fakeGeo{status: geodata.Status{Files: []geodata.File{
			{Name: "geoip.dat", Present: true, Path: "/etc/dae/geoip.dat", Size: 1},
			{Name: "geosite.dat", Present: true, Path: "/etc/dae/geosite.dat", Size: 1},
		}}},
		System: fakeSystem{snapshot: SystemSnapshot{
			OS: "linux", Architecture: "amd64", Kernel: "6.12.0", BTFPresent: true, BPFFSMounted: true,
			DefaultRoutes: []string{"IPv4 default via 192.0.2.1 dev eth0"},
		}},
		Now: func() time.Time { return now },
	})
}

func TestHealthyReportIsAllOK(t *testing.T) {
	report := healthyCollector(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)).Report(context.Background())
	if report.Overall != LevelOK || report.Counts.OK != len(report.Items) || len(report.Items) != 9 {
		t.Fatalf("健康报告异常: %+v", report)
	}
	for _, item := range report.Items {
		if item.Level != LevelOK || item.Summary == "" {
			t.Fatalf("检查项异常: %+v", item)
		}
	}
}

func TestReportKeepsIndependentFailuresActionable(t *testing.T) {
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	collector := healthyCollector(now)
	collector.configuration = fakeConfiguration{
		document: configstore.Document{Path: "/etc/dae/config.dae", Content: "bad", Hash: "abcd", Size: 3},
		validErr: errors.New("unknown field tls_fragment"),
	}
	collector.geo = fakeGeo{status: geodata.Status{Files: []geodata.File{
		{Name: "geoip.dat"}, {Name: "geosite.dat"},
	}}}
	collector.host = fakeHost{
		status:     host.Status{ActiveState: "failed", SubState: "failed", ExecMainStatus: 1},
		interfaces: []host.NetworkInterface{{Name: "eth0"}},
		logs:       []host.LogEntry{{Timestamp: now.Add(-time.Minute), Level: "error", Message: "failed to load config"}},
	}
	report := collector.Report(context.Background())
	if report.Overall != LevelError || report.Counts.Error < 3 || report.Counts.Warning < 1 {
		t.Fatalf("故障严重度汇总异常: %+v", report.Counts)
	}
	assertItemContains(t, report.Items, "configuration", "tls_fragment")
	assertItemContains(t, report.Items, "geo", "geoip.dat")
	assertItemContains(t, report.Items, "recent-logs", "failed to load config")
}

func TestUnavailableSubsystemDoesNotAbortReport(t *testing.T) {
	collector := healthyCollector(time.Now())
	collector.host = fakeHost{statusErr: errors.New("systemd unavailable"), logsErr: errors.New("journal unavailable"), ifaceErr: errors.New("net unavailable")}
	report := collector.Report(context.Background())
	if len(report.Items) != 9 || report.Counts.Unknown < 3 {
		t.Fatalf("子系统失败不应中断整份报告: %+v", report)
	}
}

func TestRoutineReloadWarningsAreNotReportedAsFaults(t *testing.T) {
	now := time.Now()
	collector := healthyCollector(now)
	collector.host = fakeHost{
		status:     host.Status{ActiveState: "active", SubState: "running", MainPID: 42, UnitFileState: "enabled"},
		interfaces: []host.NetworkInterface{{Name: "eth0"}},
		logs: []host.LogEntry{
			{Timestamp: now, Level: "warning", Message: `level=warning msg="[Reload] Finished"`},
			{Timestamp: now, Level: "warning", Message: `level=warning msg="[Reload] Retired old control plane"`},
		},
	}
	report := collector.Report(context.Background())
	for _, item := range report.Items {
		if item.ID == "recent-logs" && item.Level != LevelOK {
			t.Fatalf("正常重载生命周期不应触发警告: %+v", item)
		}
	}
}

func assertItemContains(t *testing.T, items []Item, id, value string) {
	t.Helper()
	for _, item := range items {
		if item.ID != id {
			continue
		}
		joined := item.Summary + " " + strings.Join(item.Details, " ") + " " + item.Suggestion
		if !strings.Contains(joined, value) {
			t.Fatalf("检查项 %s 不含 %q: %+v", id, value, item)
		}
		return
	}
	t.Fatalf("未找到检查项 %s", id)
}

func TestRouteAndMountParsers(t *testing.T) {
	dir := t.TempDir()
	mounts := filepath.Join(dir, "mountinfo")
	if err := os.WriteFile(mounts, []byte("33 22 0:29 / /sys/fs/bpf rw,nosuid - bpf bpf rw\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if mounted, err := bpffsMounted(mounts); err != nil || !mounted {
		t.Fatalf("bpffs 解析失败: mounted=%t err=%v", mounted, err)
	}
	routes := filepath.Join(dir, "route")
	content := "Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT\n" +
		"eth0 00000000 0102A8C0 0003 0 0 100 00000000 0 0 0\n"
	if err := os.WriteFile(routes, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := ipv4DefaultRoutes(routes)
	if err != nil || len(values) != 1 || !strings.Contains(values[0], "192.168.2.1") {
		t.Fatalf("默认路由解析失败: %v %v", values, err)
	}
}
