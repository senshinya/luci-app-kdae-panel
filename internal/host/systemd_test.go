package host

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/command"
)

func TestInterfacesAreSorted(t *testing.T) {
	manager := interfaceLister{}
	interfaces, err := manager.Interfaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(interfaces) == 0 {
		t.Fatal("系统至少应暴露一个网络接口")
	}
	for index, item := range interfaces {
		if item.Name == "" {
			t.Fatal("接口名不能为空")
		}
		if index > 0 && interfaces[index-1].Name > item.Name {
			t.Fatalf("接口未按名称排序: %+v", interfaces)
		}
		if !sort.StringsAreSorted(item.Addresses) {
			t.Fatalf("%s 的地址未排序: %v", item.Name, item.Addresses)
		}
	}
}

func TestInterfacesHonorCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	manager := interfaceLister{}
	if _, err := manager.Interfaces(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("错误 = %v，期望 context.Canceled", err)
	}
}

type fakeRunner struct {
	results map[string]command.Result
	errors  map[string]error
	calls   []string
}

func (r *fakeRunner) Run(_ context.Context, name string, args ...string) (command.Result, error) {
	key := name
	for _, arg := range args {
		key += " " + arg
	}
	r.calls = append(r.calls, key)
	return r.results[key], r.errors[key]
}

func TestStatus(t *testing.T) {
	key := "systemctl show dae --no-page --property=Id,Description,LoadState,ActiveState,SubState,UnitFileState,MainPID,ExecMainStatus,ActiveEnterTimestamp,ExecMainStartTimestamp,MemoryCurrent,CPUUsageNSec,TasksCurrent,NRestarts,FragmentPath,ExecStart,Environment"
	runner := &fakeRunner{results: map[string]command.Result{
		key: {Stdout: "Id=dae.service\nDescription=dae Service\nLoadState=loaded\nActiveState=active\nSubState=running\nMainPID=123\nMemoryCurrent=4096\nCPUUsageNSec=8000\nTasksCurrent=7\nNRestarts=2\nExecStart={ path=/usr/local/bin/dae ; argv[]=/usr/local/bin/dae run --disable-timestamp ; ignore_errors=no }\nEnvironment=DAE_LOCATION_ASSET=/opt/geo LANG=C\n"},
	}, errors: map[string]error{}}
	manager, err := New(Options{
		Backend:     BackendSystemd,
		ServiceName: "dae",
		Systemctl:   "systemctl",
		Journalctl:  "journalctl",
		Runner:      runner,
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	status, err := manager.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Name != "dae.service" || status.ActiveState != "active" || status.MainPID != 123 || status.MemoryBytes != 4096 || status.Restarts != 2 {
		t.Fatalf("状态解析异常: %+v", status)
	}
	// 安装新版本时要替换的正是这个路径，解析错会导致"装了但没换"
	if status.ExecStartPath != "/usr/local/bin/dae" {
		t.Fatalf("ExecStart 路径解析 = %q", status.ExecStartPath)
	}
	// DAE_LOCATION_ASSET 决定 dae 从哪里读 geo，漏了它会把更新写到不生效的地方
	if status.Environment["DAE_LOCATION_ASSET"] != "/opt/geo" {
		t.Fatalf("环境变量解析 = %+v", status.Environment)
	}
}

func TestParseEnvironment(t *testing.T) {
	got := parseEnvironment("DAE_LOCATION_ASSET=/etc/dae LANG=C EMPTY=")
	if got["DAE_LOCATION_ASSET"] != "/etc/dae" || got["LANG"] != "C" {
		t.Fatalf("解析结果 = %+v", got)
	}
	if value, ok := got["EMPTY"]; !ok || value != "" {
		t.Fatalf("空值也应保留键: %+v", got)
	}
	if parseEnvironment("") != nil {
		t.Fatal("空输入应返回 nil")
	}
	if parseEnvironment("没有等号") != nil {
		t.Fatal("解析不出任何键值时应返回 nil")
	}
}

func TestParseExecStartPath(t *testing.T) {
	cases := map[string]string{
		"{ path=/usr/local/bin/dae ; argv[]=/usr/local/bin/dae run ; ignore_errors=no }": "/usr/local/bin/dae",
		"{ path=/usr/bin/dae ; argv[]=/usr/bin/dae run -c /etc/dae/config.dae }":         "/usr/bin/dae",
		"":                            "",
		"{ argv[]=/usr/bin/dae run }": "",
	}
	for input, want := range cases {
		if got := parseExecStartPath(input); got != want {
			t.Fatalf("parseExecStartPath(%q) = %q，期望 %q", input, got, want)
		}
	}
}

func TestActionAllowlist(t *testing.T) {
	runner := &fakeRunner{results: map[string]command.Result{
		"systemctl restart dae": {},
		"systemctl enable dae":  {},
		"systemctl disable dae": {},
	}, errors: map[string]error{}}
	manager, _ := New(Options{
		Backend:     BackendSystemd,
		ServiceName: "dae",
		Systemctl:   "systemctl",
		Journalctl:  "journalctl",
		Runner:      runner,
		Timeout:     time.Second,
	})
	if err := manager.Action(context.Background(), ActionRestart); err != nil {
		t.Fatal(err)
	}
	if err := manager.Action(context.Background(), ActionEnable); err != nil {
		t.Fatal(err)
	}
	if err := manager.Action(context.Background(), ActionDisable); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []Action{"mask", "isolate", "poweroff"} {
		if err := manager.Action(context.Background(), forbidden); err == nil {
			t.Fatalf("未允许的动作 %q 应该被拒绝", forbidden)
		}
	}
	want := []string{"systemctl restart dae", "systemctl enable dae", "systemctl disable dae"}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("命令调用 = %v，期望 %v", runner.calls, want)
	}
}

// daemon-reload 是全局动作，不带服务名——带上服务名会让 systemctl 报错。
func TestDaemonReloadTakesNoServiceName(t *testing.T) {
	runner := &fakeRunner{results: map[string]command.Result{
		"systemctl daemon-reload": {},
	}, errors: map[string]error{}}
	manager, _ := New(Options{
		Backend:     BackendSystemd,
		ServiceName: "dae",
		Systemctl:   "systemctl",
		Journalctl:  "journalctl",
		Runner:      runner,
		Timeout:     time.Second,
	})

	if err := manager.Action(context.Background(), ActionDaemonReload); err != nil {
		t.Fatal(err)
	}
	want := []string{"systemctl daemon-reload"}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("命令调用 = %v，期望 %v", runner.calls, want)
	}
}

func TestLogs(t *testing.T) {
	runner := &fakeRunner{results: map[string]command.Result{
		"journalctl --unit dae --no-pager --output json --lines 2": {Stdout: "{\"__REALTIME_TIMESTAMP\":\"1000000\",\"PRIORITY\":\"6\",\"MESSAGE\":\"started\",\"_SYSTEMD_UNIT\":\"dae.service\",\"_PID\":\"9\"}\n"},
	}, errors: map[string]error{}}
	manager, _ := New(Options{
		Backend:     BackendSystemd,
		ServiceName: "dae",
		Systemctl:   "systemctl",
		Journalctl:  "journalctl",
		Runner:      runner,
		Timeout:     time.Second,
	})

	entries, err := manager.Logs(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Message != "started" || entries[0].Level != "info" || !entries[0].Timestamp.Equal(time.Unix(1, 0).UTC()) {
		t.Fatalf("日志解析异常: %+v", entries)
	}
}

func TestLogsRejectInvalidPriorityValue(t *testing.T) {
	entries, err := parseJournal("{\"PRIORITY\":\"invalid\",\"MESSAGE\":\"test\"}\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Priority != -1 || entries[0].Level != "unknown" {
		t.Fatalf("非法优先级解析结果 = %+v", entries)
	}
}

func TestCommandError(t *testing.T) {
	runner := &fakeRunner{
		results: map[string]command.Result{"systemctl start dae": {Stderr: "permission denied"}},
		errors:  map[string]error{"systemctl start dae": errors.New("exit status 1")},
	}
	manager, _ := New(Options{
		Backend:     BackendSystemd,
		ServiceName: "dae",
		Systemctl:   "systemctl",
		Journalctl:  "journalctl",
		Runner:      runner,
		Timeout:     time.Second,
	})
	if err := manager.Action(context.Background(), ActionStart); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("错误信息异常: %v", err)
	}
}
