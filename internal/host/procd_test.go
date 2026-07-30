package host

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/command"
)

// errExitStatus 模拟命令以非零码退出。
var errExitStatus = errors.New("exit status 1")

// scriptedRunner 按 "命令 参数..." 精确匹配返回预设结果，
// 匹配不到就让测试失败——静默返回零值会把"命令根本没被调用"伪装成成功。
type scriptedRunner struct {
	t        *testing.T
	replies  map[string]command.Result
	failures map[string]error
	calls    []string
}

func (r *scriptedRunner) Run(_ context.Context, name string, args ...string) (command.Result, error) {
	key := name
	for _, arg := range args {
		key += " " + arg
	}
	r.calls = append(r.calls, key)
	if err, ok := r.failures[key]; ok {
		return r.replies[key], err
	}
	result, ok := r.replies[key]
	if !ok {
		r.t.Fatalf("未预期的命令调用 %q", key)
	}
	return result, nil
}

const ubusRunning = `{"dae":{"instances":{"instance1":{"running":true,"pid":4321,` +
	`"command":["/usr/bin/dae","run","-c","/etc/dae/config.dae"]}}}}`

const ubusStopped = `{"dae":{"instances":{"instance1":{"running":false,"pid":0,` +
	`"command":["/usr/bin/dae","run","-c","/etc/dae/config.dae"]}}}}`

// fakeProc 造出一个 /proc/<pid> 供 Status 读取内存、CPU 与环境变量。
func fakeProc(t *testing.T, pid int, rssKB uint64, utime, stime uint64, environ string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, strconv.Itoa(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("创建 fake proc 目录: %v", err)
	}
	status := "Name:\tdae\nVmRSS:\t" + strconv.FormatUint(rssKB, 10) + " kB\n"
	writeFile(t, filepath.Join(dir, "status"), status)
	// /proc/<pid>/stat 的 utime 是第 14 个字段、stime 是第 15 个。
	fields := make([]string, 52)
	for index := range fields {
		fields[index] = "0"
	}
	fields[1] = "(dae)"
	fields[13] = strconv.FormatUint(utime, 10)
	fields[14] = strconv.FormatUint(stime, 10)
	stat := ""
	for index, field := range fields {
		if index > 0 {
			stat += " "
		}
		stat += field
	}
	writeFile(t, filepath.Join(dir, "stat"), stat)
	writeFile(t, filepath.Join(dir, "environ"), environ)
	previous := procRoot
	procRoot = root
	t.Cleanup(func() { procRoot = previous })
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入 %s: %v", path, err)
	}
}

// initScriptDir 造出一个假的 /etc/init.d 并让本次测试指向它。
func initScriptDir(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		writeFile(t, filepath.Join(dir, name), "#!/bin/sh /etc/rc.common\n")
	}
	previous := initDirectory
	initDirectory = dir
	t.Cleanup(func() { initDirectory = previous })
	return dir
}

func newTestProcdManager(t *testing.T, runner command.Runner) *procdManager {
	t.Helper()
	manager, err := newProcdManager(Options{
		ServiceName: "dae",
		DaeBinary:   "/usr/bin/dae",
		Runner:      runner,
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("构造 procd 管理器: %v", err)
	}
	return manager
}

func TestProcdStatusRunning(t *testing.T) {
	dir := initScriptDir(t, "dae")
	fakeProc(t, 4321, 2048, 30, 20, "PATH=/usr/bin\x00DAE_LOCATION_ASSET=/etc/dae\x00")
	runner := &scriptedRunner{t: t, replies: map[string]command.Result{
		`ubus call service list {"name":"dae"}`: {Stdout: ubusRunning},
		filepath.Join(dir, "dae") + " enabled":  {},
	}}
	manager := newTestProcdManager(t, runner)

	status, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("Status 返回错误: %v", err)
	}
	if status.ActiveState != "active" || status.SubState != "running" {
		t.Fatalf("状态 = %s/%s，期望 active/running", status.ActiveState, status.SubState)
	}
	if status.MainPID != 4321 {
		t.Fatalf("MainPID = %d，期望 4321", status.MainPID)
	}
	if status.ExecStartPath != "/usr/bin/dae" {
		t.Fatalf("ExecStartPath = %q，期望 /usr/bin/dae", status.ExecStartPath)
	}
	if status.UnitFileState != "enabled" {
		t.Fatalf("UnitFileState = %q，期望 enabled", status.UnitFileState)
	}
	if status.UnitPath != filepath.Join(dir, "dae") {
		t.Fatalf("UnitPath = %q，期望 %q", status.UnitPath, filepath.Join(dir, "dae"))
	}
	if status.MemoryBytes != 2048*1024 {
		t.Fatalf("MemoryBytes = %d，期望 %d", status.MemoryBytes, 2048*1024)
	}
	if status.CPUUsageNanoseconds != 50*clockTickNanoseconds {
		t.Fatalf("CPU = %d，期望 %d", status.CPUUsageNanoseconds, 50*clockTickNanoseconds)
	}
	if status.Environment["DAE_LOCATION_ASSET"] != "/etc/dae" {
		t.Fatalf("Environment = %v，期望含 DAE_LOCATION_ASSET=/etc/dae", status.Environment)
	}
}

// dae 停止时 procd 仍保留实例定义，但面板必须报 inactive，
// 同时 ExecStartPath 不能为空——调用方靠它判断这台机器上有没有 dae。
func TestProcdStatusStoppedKeepsExecStartPath(t *testing.T) {
	dir := initScriptDir(t, "dae")
	runner := &scriptedRunner{
		t: t,
		replies: map[string]command.Result{
			`ubus call service list {"name":"dae"}`: {Stdout: ubusStopped},
			filepath.Join(dir, "dae") + " enabled":  {ExitCode: 1},
		},
		failures: map[string]error{
			filepath.Join(dir, "dae") + " enabled": errExitStatus,
		},
	}
	manager := newTestProcdManager(t, runner)

	status, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("Status 返回错误: %v", err)
	}
	if status.ActiveState != "inactive" {
		t.Fatalf("ActiveState = %q，期望 inactive", status.ActiveState)
	}
	if status.ExecStartPath != "/usr/bin/dae" {
		t.Fatalf("ExecStartPath = %q，期望 /usr/bin/dae", status.ExecStartPath)
	}
	if status.UnitFileState != "disabled" {
		t.Fatalf("UnitFileState = %q，期望 disabled", status.UnitFileState)
	}
}

// ubus 查不到服务时不能报错，也不能把 ExecStartPath 留空：
// 回退到面板配置的路径，它与 init 脚本同源于 UCI。
func TestProcdStatusFallsBackWhenUbusEmpty(t *testing.T) {
	dir := initScriptDir(t, "dae")
	runner := &scriptedRunner{t: t, replies: map[string]command.Result{
		`ubus call service list {"name":"dae"}`: {Stdout: "{}"},
		filepath.Join(dir, "dae") + " enabled":  {},
	}}
	manager := newTestProcdManager(t, runner)

	status, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("Status 返回错误: %v", err)
	}
	if status.ExecStartPath != "/usr/bin/dae" {
		t.Fatalf("ExecStartPath = %q，期望回退到 /usr/bin/dae", status.ExecStartPath)
	}
	if status.LoadState != "loaded" {
		t.Fatalf("LoadState = %q，期望 loaded", status.LoadState)
	}
}

// init 脚本不在，说明包坏了或没装；必须如实报 not-found，
// 不能因为二进制碰巧在就假装服务已就绪。
func TestProcdStatusReportsMissingInitScript(t *testing.T) {
	dir := initScriptDir(t)
	runner := &scriptedRunner{t: t, replies: map[string]command.Result{
		`ubus call service list {"name":"dae"}`: {Stdout: "{}"},
	}}
	manager := newTestProcdManager(t, runner)

	status, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("Status 返回错误: %v", err)
	}
	if status.LoadState != "not-found" {
		t.Fatalf("LoadState = %q，期望 not-found", status.LoadState)
	}
	if status.UnitFileState != "" {
		t.Fatalf("UnitFileState = %q，期望空", status.UnitFileState)
	}
	if status.UnitPath != filepath.Join(dir, "dae") {
		t.Fatalf("UnitPath = %q，期望 %q", status.UnitPath, filepath.Join(dir, "dae"))
	}
}

func TestProcdActionRunsInitScript(t *testing.T) {
	dir := initScriptDir(t, "dae")
	runner := &scriptedRunner{t: t, replies: map[string]command.Result{
		filepath.Join(dir, "dae") + " restart": {},
	}}
	manager := newTestProcdManager(t, runner)

	if err := manager.Action(context.Background(), ActionRestart); err != nil {
		t.Fatalf("Action 返回错误: %v", err)
	}
	if len(runner.calls) != 1 || runner.calls[0] != filepath.Join(dir, "dae")+" restart" {
		t.Fatalf("调用记录 = %v", runner.calls)
	}
}

// procd 每次执行 init 脚本都会重读定义，没有等价的全局重载动作。
// 必须静默成功：dae 的首次安装与卸载事务都会调用它，报错会让整条链路失败。
func TestProcdDaemonReloadIsNoop(t *testing.T) {
	initScriptDir(t, "dae")
	runner := &scriptedRunner{t: t, replies: map[string]command.Result{}}
	manager := newTestProcdManager(t, runner)

	if err := manager.Action(context.Background(), ActionDaemonReload); err != nil {
		t.Fatalf("daemon-reload 应当静默成功，实际: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("daemon-reload 不应执行任何命令，实际 %v", runner.calls)
	}
}

// setsid 不能省：重启命令是面板的子进程，procd 停掉面板实例时会连它一起杀掉，
// 于是命令先于重启本身死亡，面板永远升级不完。
func TestProcdRestartSelfDetachesWithSetsid(t *testing.T) {
	expected := "/bin/sh -c setsid " + PanelInitScript + " restart >/dev/null 2>&1 &"
	runner := &scriptedRunner{t: t, replies: map[string]command.Result{expected: {}}}
	manager := newTestProcdManager(t, runner)

	if err := manager.RestartSelf(context.Background()); err != nil {
		t.Fatalf("RestartSelf 返回错误: %v", err)
	}
	if len(runner.calls) != 1 || runner.calls[0] != expected {
		t.Fatalf("调用记录 = %v，期望 %q", runner.calls, expected)
	}
}

func TestProcdLogsParsesUboxFormat(t *testing.T) {
	initScriptDir(t, "dae")
	output := "Fri Jul 31 01:02:03 2026 daemon.warn dae[4321]: level=warn msg=\"节点不可达\" dialer=n1\n" +
		"Fri Jul 31 01:02:04 2026 daemon.info dae[4321]: level=info msg=\"已重载配置\"\n"
	runner := &scriptedRunner{t: t, replies: map[string]command.Result{
		"logread -e dae": {Stdout: output},
	}}
	manager := newTestProcdManager(t, runner)

	entries, err := manager.Logs(context.Background(), 100)
	if err != nil {
		t.Fatalf("Logs 返回错误: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("条目数 = %d，期望 2", len(entries))
	}
	if entries[0].Level != "warn" || entries[0].Priority != 4 {
		t.Fatalf("首条级别 = %s/%d，期望 warn/4", entries[0].Level, entries[0].Priority)
	}
	if entries[0].Message != "节点不可达" {
		t.Fatalf("首条消息 = %q，期望 节点不可达", entries[0].Message)
	}
	if entries[0].PID != "4321" || entries[0].Unit != "dae" {
		t.Fatalf("首条 PID/Unit = %s/%s", entries[0].PID, entries[0].Unit)
	}
	if entries[0].Timestamp.IsZero() {
		t.Fatal("首条时间戳为零值，ubox 格式应当解析成功")
	}
	if entries[1].Level != "info" || entries[1].Message != "已重载配置" {
		t.Fatalf("次条 = %s/%q", entries[1].Level, entries[1].Message)
	}
}

// busybox 的 logread 没有 facility.level 也没有年份；解析不出就退回整行，
// 绝不能丢日志——用户看日志正是因为出了问题。
func TestProcdLogsFallsBackToRawLine(t *testing.T) {
	initScriptDir(t, "dae")
	runner := &scriptedRunner{t: t, replies: map[string]command.Result{
		"logread -e dae": {Stdout: "Jul 31 01:02:03 router dae[7]: 裸消息\n"},
	}}
	manager := newTestProcdManager(t, runner)

	entries, err := manager.Logs(context.Background(), 100)
	if err != nil {
		t.Fatalf("Logs 返回错误: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("条目数 = %d，期望 1", len(entries))
	}
	if entries[0].Message != "裸消息" {
		t.Fatalf("消息 = %q，期望 裸消息", entries[0].Message)
	}
	if entries[0].Level != "info" || entries[0].Priority != 6 {
		t.Fatalf("级别 = %s/%d，期望默认 info/6", entries[0].Level, entries[0].Priority)
	}
}

// limit 是"最新 N 条"，因此要截尾部而不是头部。
func TestProcdLogsKeepsNewestWithinLimit(t *testing.T) {
	initScriptDir(t, "dae")
	output := ""
	for index := 1; index <= 5; index++ {
		output += "Fri Jul 31 01:02:0" + strconv.Itoa(index) +
			" 2026 daemon.info dae[7]: level=info msg=\"第" + strconv.Itoa(index) + "条\"\n"
	}
	runner := &scriptedRunner{t: t, replies: map[string]command.Result{
		"logread -e dae": {Stdout: output},
	}}
	manager := newTestProcdManager(t, runner)

	entries, err := manager.Logs(context.Background(), 2)
	if err != nil {
		t.Fatalf("Logs 返回错误: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("条目数 = %d，期望 2", len(entries))
	}
	if entries[0].Message != "第4条" || entries[1].Message != "第5条" {
		t.Fatalf("保留的是 %q / %q，期望最后两条", entries[0].Message, entries[1].Message)
	}
}

func TestNewReturnsProcdBackend(t *testing.T) {
	manager, err := New(Options{Backend: BackendProcd, ServiceName: "dae", DaeBinary: "/usr/bin/dae"})
	if err != nil {
		t.Fatalf("New 返回错误: %v", err)
	}
	if _, ok := manager.(*procdManager); !ok {
		t.Fatalf("类型 = %T，期望 *procdManager", manager)
	}
}

func TestBackendResolveRejectsUnknown(t *testing.T) {
	if _, err := Backend("upstart").Resolve(); err == nil {
		t.Fatal("未知后端应当报错")
	}
}
