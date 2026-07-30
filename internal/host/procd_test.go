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
