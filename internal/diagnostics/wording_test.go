package diagnostics

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/configstore"
	"github.com/tuoro/kdae-panel/internal/dae"
	"github.com/tuoro/kdae-panel/internal/geodata"
	"github.com/tuoro/kdae-panel/internal/host"
)

// systemdOnlyTokens 是只在 systemd 部署上存在的东西。它们出现在 procd 的诊断报告
// 里，就等于让一个正卡在故障里的用户去敲一条 OpenWrt 上根本没有的命令、去改一个
// 不存在的文件。诊断页比别处更不能这样——会点开它的人恰恰最没有余力分辨。
var systemdOnlyTokens = []string{"systemd", "systemctl", "journald", "dae.service", "ReadWritePaths", "单元文件"}

func backendCollector(backend host.Backend, host_ fakeHost, geo fakeGeo, system fakeSystem, now time.Time) *Collector {
	return New(Options{
		Dae: fakeDae{report: dae.Report{Available: true, Binary: "/usr/bin/dae", Version: "dae v2"}},
		Configuration: fakeConfiguration{document: configstore.Document{
			Path: "/etc/dae/config.dae", Content: "global {}", Hash: strings.Repeat("a", 64), Size: 9,
		}},
		Host: host_, Geo: geo, System: system,
		Now:     func() time.Time { return now },
		Backend: backend,
	})
}

// 两个场景加起来把所有分后端的措辞都走一遍：状态读不出来、日志读不出来、geo 目录
// 不可维护、缺 BTF，以及"显示运行中却拿不到主进程号"。
func wordingScenarios(now time.Time) []struct {
	name   string
	host   fakeHost
	geo    fakeGeo
	system fakeSystem
} {
	return []struct {
		name   string
		host   fakeHost
		geo    fakeGeo
		system fakeSystem
	}{
		{
			name: "读不到服务状态与日志",
			host: fakeHost{
				statusErr: errors.New("backend exploded"),
				logsErr:   errors.New("log source exploded"),
			},
			geo: fakeGeo{status: geodata.Status{Problem: "数据目录不可写"}},
			system: fakeSystem{snapshot: SystemSnapshot{
				OS: "linux", Architecture: "amd64", Kernel: "6.12.0",
			}},
		},
		{
			name: "运行中但拿不到主进程号",
			host: fakeHost{
				status: host.Status{
					ActiveState: "active", SubState: "running", MainPID: 0,
					UnitFileState: "enabled", UnitPath: "/etc/init.d/dae",
				},
				interfaces: []host.NetworkInterface{{Name: "eth0", Addresses: []string{"192.0.2.2/24"}}},
			},
			geo: fakeGeo{status: geodata.Status{Files: []geodata.File{
				{Name: "geoip.dat", Present: true, Path: "/etc/dae/geoip.dat", Size: 1},
				{Name: "geosite.dat", Present: true, Path: "/etc/dae/geosite.dat", Size: 1},
			}}},
			system: fakeSystem{snapshot: SystemSnapshot{
				OS: "linux", Architecture: "amd64", Kernel: "6.12.0", BTFPresent: true, BPFFSMounted: true,
				DefaultRoutes: []string{"IPv4 default via 192.0.2.1 dev eth0"},
			}},
		},
	}
}

func TestProcdReportNeverNamesSystemdOnlyThings(t *testing.T) {
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	for _, scenario := range wordingScenarios(now) {
		procd := reportText(backendCollector(host.BackendProcd, scenario.host, scenario.geo, scenario.system, now).
			Report(context.Background()))
		for _, token := range systemdOnlyTokens {
			if strings.Contains(procd, token) {
				t.Fatalf("%s：procd 报告出现 systemd 专有的 %q\n%s", scenario.name, token, procd)
			}
		}

		// 反向断言不是凑数：如果这些措辞在 systemd 下也已经消失，上面那条就成了永真
		// 的空断言，以后有人把 systemd 文案换回去也不会有人发现。
		systemd := reportText(backendCollector(host.BackendSystemd, scenario.host, scenario.geo, scenario.system, now).
			Report(context.Background()))
		hit := false
		for _, token := range systemdOnlyTokens {
			if strings.Contains(systemd, token) {
				hit = true
				break
			}
		}
		if !hit {
			t.Fatalf("%s：systemd 报告一个专有名词都没出现，这条用例已经测不到东西了\n%s", scenario.name, systemd)
		}
	}
}

// 分叉只许发生在名字和建议上。判定条件、级别、汇总这些"结论"两个后端必须完全一致，
// 否则同一台机器换个 init 系统就会读到不同的诊断结果。
func TestBothBackendsReachTheSameVerdicts(t *testing.T) {
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	for _, scenario := range wordingScenarios(now) {
		procd := backendCollector(host.BackendProcd, scenario.host, scenario.geo, scenario.system, now).
			Report(context.Background())
		systemd := backendCollector(host.BackendSystemd, scenario.host, scenario.geo, scenario.system, now).
			Report(context.Background())
		if procd.Overall != systemd.Overall || procd.Counts != systemd.Counts {
			t.Fatalf("%s：两个后端的汇总不一致 procd=%+v/%+v systemd=%+v/%+v",
				scenario.name, procd.Overall, procd.Counts, systemd.Overall, systemd.Counts)
		}
		if len(procd.Items) != len(systemd.Items) {
			t.Fatalf("%s：检查项数量不一致 procd=%d systemd=%d", scenario.name, len(procd.Items), len(systemd.Items))
		}
		for index := range procd.Items {
			left, right := procd.Items[index], systemd.Items[index]
			if left.ID != right.ID || left.Category != right.Category || left.Level != right.Level {
				t.Fatalf("%s：第 %d 项判定不一致 procd=%+v systemd=%+v", scenario.name, index, left, right)
			}
		}
	}
}

// 后端未知时退回 systemd 措辞：那是上游的原生部署，也是所有文案的原始写法。
func TestUnknownBackendKeepsSystemdWording(t *testing.T) {
	for _, backend := range []host.Backend{"", host.BackendAuto, host.BackendSystemd} {
		if got := wordingFor(backend); got != systemdWording {
			t.Fatalf("后端 %q 应按 systemd 措辞: %+v", string(backend), got)
		}
	}
	if got := wordingFor(host.BackendProcd); got != procdWording {
		t.Fatalf("procd 应使用 procd 措辞: %+v", got)
	}
}

func reportText(report Report) string {
	var builder strings.Builder
	for _, item := range report.Items {
		builder.WriteString(item.Title)
		builder.WriteString(" ")
		builder.WriteString(item.Summary)
		builder.WriteString(" ")
		builder.WriteString(strings.Join(item.Details, " "))
		builder.WriteString(" ")
		builder.WriteString(item.Suggestion)
		builder.WriteString("\n")
	}
	return builder.String()
}
