# luci-app-kdae-panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 kdae-panel 改造成可在 immortalwrt 24.10.4（x86/64）上以 ipk 部署的 LuCI 应用，dae 完全由面板管理而不经 opkg。

**Architecture:** `internal/host` 从具体结构体抽成 `Manager` 接口，新增一个基于 ubus/procd 的后端与现有 systemd 后端并列；`daeinstall` 把"服务定义"抽成 `unitProvisioner`，procd 下的服务定义由 ipk 自带、面板只校验不改写。仓库新增 `openwrt/` 目录承载两个 OpenWrt 包（`kdae-panel` 后端 + `luci-app-kdae-panel` 界面），CI 用官方 Go 交叉编译静态二进制后交给 immortalwrt SDK 打包。

**Tech Stack:** Go 1.25（纯 Go，`CGO_ENABLED=0`）、Vue 3 + naive-ui、OpenWrt procd/ubus/UCI、LuCI 客户端 JS 框架、immortalwrt 24.10.4 SDK、GitHub Actions。

## Global Constraints

- 目标平台只有 immortalwrt 24.10.4 x86/64；包 `DEPENDS` 含 `@x86_64`。
- Go 版本 1.25.0（go.mod 不动）；二进制必须 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`，静态链接以在 musl 上运行。
- **systemd 后端的行为一个字节都不能变。** 每个重构任务结束时现有测试必须原样通过。
- 所有新增用户可见文案用简体中文，与现有代码一致；注释解释"为什么"而不是"是什么"，与现有代码风格一致。
- procd 后端下任何用户可见文案不得出现 "systemd" / "systemctl" / "journalctl" / "ReadWritePaths" 字样。
- **`/var` 在 OpenWrt 上是 `/tmp` 的软链（tmpfs），重启即空。** 面板的数据库、备份、状态文件、dae 本地版本库一律落在 `/etc/kdae-panel`；只有一次性初始化链接放 `/var/run/kdae-panel`（它本就该重启后重新生成）。
- UCI `kdae-panel.main` 是所有配置项的唯一真相源，面板内不得存在能覆盖它的第二处持久化。
- 包名固定：后端包 `kdae-panel`，界面包 `luci-app-kdae-panel`。
- 全量验证命令：`go test ./... && go vet ./... && npm run typecheck --prefix web && npm test --prefix web`。

## 相对设计文档的三处修正

实现期间核对代码后确认的三点，设计文档已同步更新：

1. **数据目录**：设计文档写 `/var/lib/kdae-panel`，那是 systemd 部署的路径。OpenWrt 上 `/var` 是 tmpfs，必须改用 `/etc/kdae-panel`（overlay，持久）。
2. **`ExecStartPath` 回退链只有两级不是三级**：`/etc/init.d/dae` 改为从同一份 UCI 读 `dae_binary`，面板的 `--dae-binary` 也来自它，两者不可能分叉，因此不需要解析 init 脚本，直接回退到 `Options.DaeBinary` 即可。少一个解析器，也少一类分叉。
3. **geo 缺失警告不需要传后端标志**：`MissingWarning` 自己探测 `/root` 是否因沙箱不可读（`ProtectHome=true` 下读它得到 EACCES），比从 `app.Config` 一路透传布尔更准，也不改任何签名。

## File Structure

**新建**

| 文件 | 职责 |
|---|---|
| `internal/host/host.go` | `Manager` 接口、`Backend` 与解析、`Options`、`New` 工厂 |
| `internal/host/procd.go` | procd 后端：ubus 状态、init 脚本动作、logread 日志 |
| `internal/host/procd_test.go` | procd 后端单测 |
| `internal/daeinstall/units.go` | `unitProvisioner` 接口 + systemd 实现（从 provision.go 搬迁） |
| `internal/daeinstall/units_procd.go` | procd 实现：只校验不改写 |
| `internal/daeinstall/units_procd_test.go` | procd 服务定义单测 |
| `openwrt/kdae-panel/Makefile` | 后端包定义 |
| `openwrt/kdae-panel/files/kdae-panel.init` | 面板的 procd 脚本 |
| `openwrt/kdae-panel/files/dae.init` | dae 的 procd 脚本 |
| `openwrt/kdae-panel/files/kdae-panel.config` | UCI 默认配置 |
| `openwrt/luci-app-kdae-panel/Makefile` | 界面包定义 |
| `openwrt/luci-app-kdae-panel/htdocs/luci-static/resources/view/kdae-panel/panel.js` | LuCI 视图 |
| `openwrt/luci-app-kdae-panel/root/usr/share/luci/menu.d/luci-app-kdae-panel.json` | 菜单 |
| `openwrt/luci-app-kdae-panel/root/usr/share/rpcd/acl.d/luci-app-kdae-panel.json` | ACL |
| `.github/workflows/openwrt.yml` | 交叉编译 + SDK 打包 |
| `docs/openwrt.md` | OpenWrt 部署文档 |

**改名**

| 原 | 新 |
|---|---|
| `internal/host/manager.go` | `internal/host/systemd.go` |
| `internal/host/manager_test.go` | `internal/host/systemd_test.go` |

**修改**

`internal/host/interfaces.go`、`internal/app/app.go`、`internal/app/app_test.go`、`internal/app/config.go`、`cmd/kdae-panel/main.go`、`internal/daeinstall/installer.go`、`internal/daeinstall/installer_test.go`、`internal/daeinstall/provision.go`、`internal/daeinstall/uninstall.go`、`internal/geodata/locate.go`、`internal/geodata/geodata_test.go`、`internal/panelupdate/panelupdate.go`、`docs/api.md`、`docs/deployment.md`、`docs/architecture.md`、`SECURITY.md`、`README.md`

**前端一行不改。** `dependencies.PanelUpdate` 为 nil 时后端返回的 payload 里没有 `status`，而 `PanelUpdatePayload.status` 本就是可选字段——设置页那个开关会自动置灰，更新横幅不会出现。

---

### Task 1: `internal/host` 抽出 Manager 接口与 Options

纯重构。结束时 systemd 行为与测试断言必须逐字不变。

**Files:**
- Create: `internal/host/host.go`
- Rename: `internal/host/manager.go` → `internal/host/systemd.go`
- Rename: `internal/host/manager_test.go` → `internal/host/systemd_test.go`
- Modify: `internal/host/interfaces.go`
- Modify: `internal/app/app.go:94`

**Interfaces:**
- Produces: `host.Manager` 接口；`host.Backend`（`BackendAuto`/`BackendSystemd`/`BackendProcd`）与 `(Backend).Resolve() (Backend, error)`；`host.Options{Backend, ServiceName, DaeBinary, Systemctl, Journalctl, Runner, Timeout}`；`host.New(Options) (Manager, error)`；未导出的 `interfaceLister` 结构体。`host.Status`、`host.LogEntry`、`host.Action`、`host.PanelUnit` 保持原样导出。

- [ ] **Step 1: 改名两个文件并把类型改成 `systemdManager`**

```bash
git mv internal/host/manager.go internal/host/systemd.go
git mv internal/host/manager_test.go internal/host/systemd_test.go
```

在 `internal/host/systemd.go` 里：

1. 把 `type Manager struct { ... }` 改名为 `type systemdManager struct { ... }`，并在字段列表最前面加一行内嵌：

```go
type systemdManager struct {
	interfaceLister
	serviceName string
	systemctl   string
	journalctl  string
	runner      command.Runner
	timeout     time.Duration
}
```

2. 把全部 `func (m *Manager)` 接收者改为 `func (m *systemdManager)`（共 6 处：`Status`、`Action`、`RestartSelf`、`Logs`、`run`、`runFor`）。
3. 删除 `NewManager` 与 `NewManagerWithRunner` 两个构造函数（连同它们的注释），它们的职责移交 `host.New`。删除后 `errors` 若不再被引用，一并从 import 移除。
4. 新增一行接口断言，放在文件末尾：

```go
var _ Manager = (*systemdManager)(nil)
```

5. 保留 `defaultTimeout`、`actionTimeout`、`maxLogLines`、`PanelUnit`、`validUnitName` 及全部解析函数不动。

- [ ] **Step 2: 把 `Interfaces` 的接收者换成 `interfaceLister`**

改 `internal/host/interfaces.go`，只动注释与函数签名两处：

```go
// interfaceLister 提供与 init 系统无关的本机网卡枚举。
//
// Manager 变成接口后这个方法不能再挂在某个具体后端上，而它的实现只用
// net.Interfaces()，两个后端没有任何理由各写一份。做成零字段结构体由
// 两个后端各内嵌一次，方法集自动带上。
type interfaceLister struct{}

// Interfaces 枚举本机接口及其地址。单个接口读取地址失败时仍保留接口名，
// 让尚未分配地址或状态正在变化的接口也能出现在配置选择器中。
func (interfaceLister) Interfaces(ctx context.Context) ([]NetworkInterface, error) {
```

函数体一行不改。

- [ ] **Step 3: 新建 `internal/host/host.go`**

```go
// Package host 把"控制 dae 服务、读它的日志"这件事收在一处，
// 对上层只暴露 Manager 一个接口，具体是 systemd 还是 procd 由本包决定。
package host

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tuoro/kdae-panel/internal/command"
)

// Manager 是主机服务管理器，systemd 与 procd 两个后端实现它。
type Manager interface {
	Status(ctx context.Context) (Status, error)
	Action(ctx context.Context, action Action) error
	RestartSelf(ctx context.Context) error
	Logs(ctx context.Context, limit int) ([]LogEntry, error)
	Interfaces(ctx context.Context) ([]NetworkInterface, error)
}

// Backend 指明用哪一套系统接口管理服务。
type Backend string

const (
	// BackendAuto 按机器实际情况二选一。
	BackendAuto    Backend = "auto"
	BackendSystemd Backend = "systemd"
	BackendProcd   Backend = "procd"
)

// procdMarker 是 OpenWrt/ImmortalWrt 的进程管理守护进程。它存在就说明这台
// 机器没有 systemd，自动探测据此二选一。做成变量是给测试留的缝。
var procdMarker = "/sbin/procd"

const defaultServiceName = "dae"

// Resolve 把 auto 落到具体后端；显式指定的原样返回，未知值直接报错。
//
// 不把探测藏在 New 里，是为了让"这台机器被判成了哪个后端"可以被日志、
// 健康检查和测试直接问出来——错配的症状（服务控制全部失败）离原因很远。
func (b Backend) Resolve() (Backend, error) {
	switch b {
	case "", BackendAuto:
		if _, err := os.Stat(procdMarker); err == nil {
			return BackendProcd, nil
		}
		return BackendSystemd, nil
	case BackendSystemd, BackendProcd:
		return b, nil
	default:
		return "", fmt.Errorf("未知的服务后端 %q，可选 auto、systemd、procd", string(b))
	}
}

type Options struct {
	Backend     Backend
	ServiceName string
	// DaeBinary 是 procd 后端 ExecStartPath 的回退值。procd 在服务停止时
	// 拿不到命令行，而调用方要靠这个字段判断"这台机器上有没有 dae"。
	DaeBinary string
	// Systemctl、Journalctl 只有 systemd 后端使用。
	Systemctl  string
	Journalctl string
	Runner     command.Runner
	Timeout    time.Duration
}

// New 按 Options.Backend 构造对应后端。
func New(options Options) (Manager, error) {
	backend, err := options.Backend.Resolve()
	if err != nil {
		return nil, err
	}
	if options.ServiceName == "" {
		options.ServiceName = defaultServiceName
	}
	if !validUnitName(options.ServiceName) {
		return nil, fmt.Errorf("服务名 %q 无效", options.ServiceName)
	}
	if options.Runner == nil {
		options.Runner = command.ExecRunner{}
	}
	if options.Timeout <= 0 {
		options.Timeout = defaultTimeout
	}
	switch backend {
	case BackendProcd:
		// Task 3 接上真正的实现。
		return nil, fmt.Errorf("procd 后端尚未实现")
	default:
		if options.Systemctl == "" {
			options.Systemctl = "systemctl"
		}
		if options.Journalctl == "" {
			options.Journalctl = "journalctl"
		}
		return &systemdManager{
			serviceName: options.ServiceName,
			systemctl:   options.Systemctl,
			journalctl:  options.Journalctl,
			runner:      options.Runner,
			timeout:     options.Timeout,
		}, nil
	}
}
```

import 全部用得上：`context` 在 `Manager` 的方法签名里，`fmt` 在错误构造里，`os` 在 `Resolve` 的探测里，`time` 在 `Options.Timeout` 上，`command` 在 `Options.Runner` 与 `ExecRunner{}` 上。

- [ ] **Step 4: 更新 `internal/host/systemd_test.go` 的 5 处构造调用**

把 5 处（原第 66、124、150、165、191 行）形如

```go
manager, err := NewManagerWithRunner("dae", "systemctl", "journalctl", runner, time.Second)
```

的调用统一替换为

```go
manager, err := New(Options{
	Backend:     BackendSystemd,
	ServiceName: "dae",
	Systemctl:   "systemctl",
	Journalctl:  "journalctl",
	Runner:      runner,
	Timeout:     time.Second,
})
```

只赋值一个变量的那 4 处保持 `manager, _ := New(Options{...})` 形式。其余断言一行不改。

- [ ] **Step 5: 更新 `internal/app/app.go` 的构造点**

把第 94 行

```go
	hostManager, err := host.NewManager(cfg.ServiceName, cfg.Systemctl, cfg.Journalctl)
```

替换为

```go
	hostManager, err := host.New(host.Options{
		Backend:     host.BackendAuto,
		ServiceName: cfg.ServiceName,
		DaeBinary:   cfg.DaeBinary,
		Systemctl:   cfg.Systemctl,
		Journalctl:  cfg.Journalctl,
	})
```

（Task 4 会把 `host.BackendAuto` 换成 `cfg.ServiceBackend`。）

- [ ] **Step 6: 全量验证，确认是纯重构**

Run: `go build ./... && go test ./... && go vet ./...`
Expected: 全部 PASS，`internal/host` 与 `internal/app` 的测试数量与之前一致。

- [ ] **Step 7: Commit**

```bash
git add internal/host internal/app/app.go
git commit -m "refactor(host): 把主机服务管理器抽成接口"
```

---

### Task 2: procd 后端的 Status

**Files:**
- Create: `internal/host/procd.go`
- Create: `internal/host/procd_test.go`

**Interfaces:**
- Consumes: Task 1 的 `Options`、`Status`、`interfaceLister`、`defaultTimeout`。
- Produces: 未导出的 `procdManager` 结构体（此任务只实现 `Status`）；`newProcdManager(Options) (*procdManager, error)`；包级变量 `procRoot`（默认 `"/proc"`，测试可改写）；常量 `initDirectory = "/etc/init.d"`。

- [ ] **Step 1: 写失败的测试**

创建 `internal/host/procd_test.go`：

```go
package host

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/command"
)

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
```

在同一个文件顶部补上测试用的哨兵错误：

```go
// errExitStatus 模拟命令以非零码退出。
var errExitStatus = errors.New("exit status 1")
```

并把 `errors` 加进 import。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/host/ -run TestProcd -v`
Expected: 编译失败，`undefined: procdManager`、`undefined: newProcdManager`、`undefined: procRoot`、`undefined: initDirectory`、`undefined: clockTickNanoseconds`

- [ ] **Step 3: 实现 `internal/host/procd.go` 的 Status 部分**

```go
package host

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tuoro/kdae-panel/internal/command"
)

// initDirectory 是 procd 读取服务定义的目录。做成变量是给测试留的缝。
var initDirectory = "/etc/init.d"

// procRoot 是 /proc 的挂载点，测试指向临时目录。
var procRoot = "/proc"

// clockTickNanoseconds 是 /proc/<pid>/stat 里 utime/stime 的单位。
// Linux 上 USER_HZ 恒为 100，即每一跳 10ms；写死比为一个展示用的数字
// 去引 cgo 调 sysconf 划算。
const clockTickNanoseconds = 10_000_000

// procdManager 通过 procd 的 init 脚本与 ubus 管理服务，适用于 OpenWrt/ImmortalWrt。
type procdManager struct {
	interfaceLister
	serviceName string
	daeBinary   string
	runner      command.Runner
	timeout     time.Duration
}

func newProcdManager(options Options) (*procdManager, error) {
	if options.Runner == nil {
		return nil, errors.New("命令执行器不能为空")
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &procdManager{
		serviceName: options.ServiceName,
		daeBinary:   options.DaeBinary,
		runner:      options.Runner,
		timeout:     timeout,
	}, nil
}

// initScript 是本服务的 procd 定义文件。
func (m *procdManager) initScript() string {
	return filepath.Join(initDirectory, m.serviceName)
}

// ubusInstance 是 `ubus call service list` 里单个实例用得上的字段。
type ubusInstance struct {
	Running bool     `json:"running"`
	PID     int      `json:"pid"`
	Command []string `json:"command"`
}

type ubusService struct {
	Instances map[string]ubusInstance `json:"instances"`
}

// Status 汇报 dae 的运行状况。
//
// 不返回错误是有意的：procd 的状态全部来自本机文件与 ubus，读不到就是"没装/没跑"，
// 不存在 systemd 那种"守护进程抽风导致查询失败"的中间态。把读不到当成错误，
// 会让 daeinstall 的预检永久卡在"无法确认是否已有 dae"。
func (m *procdManager) Status(ctx context.Context) (Status, error) {
	script := m.initScript()
	// Restarts 刻意不填：procd 不暴露重启计数器，填 0 会让 daeinstall 的
	// 崩溃循环检测（"计数没涨就算稳"）静默通过。字段带 omitempty，0 不进 JSON，
	// 仪表盘那一格显示 "—" 而不是 "0"——不知道就该说不知道。
	// 真正的替代信号是 MainPID 变化，见 daeinstall 的重启后观察窗口。
	status := Status{
		Name:        m.serviceName,
		Description: "procd service " + m.serviceName,
		ActiveState: "inactive",
		SubState:    "dead",
		UnitPath:    script,
		LoadState:   "not-found",
	}
	if _, err := os.Stat(script); err == nil {
		status.LoadState = "loaded"
		status.UnitFileState = m.unitFileState(ctx)
	}
	if instance, found := m.instance(ctx); found {
		if instance.Running {
			status.ActiveState = "active"
			status.SubState = "running"
			status.MainPID = instance.PID
		}
		if len(instance.Command) > 0 {
			status.ExecStartPath = instance.Command[0]
		}
	}
	// ExecStartPath 绝不能留空：调用方靠它判断这台机器上有没有 dae，也靠它
	// 决定把新版本写到哪。回退到面板配置的路径是安全的——init 脚本与面板的
	// --dae-binary 读的是同一份 UCI，两者不可能分叉。
	if status.ExecStartPath == "" {
		status.ExecStartPath = m.daeBinary
	}
	if status.MainPID > 0 {
		status.MemoryBytes = readMemoryBytes(status.MainPID)
		status.Tasks = readThreadCount(status.MainPID)
		status.CPUUsageNanoseconds = readCPUNanoseconds(status.MainPID)
		status.Environment = readProcessEnvironment(status.MainPID)
	}
	return status, nil
}

// instance 取 procd 记录的第一个实例。本包写出的服务只开一个实例。
func (m *procdManager) instance(ctx context.Context) (ubusInstance, bool) {
	result, err := m.run(ctx, "ubus", "call", "service", "list",
		`{"name":"`+m.serviceName+`"}`)
	if err != nil {
		return ubusInstance{}, false
	}
	var services map[string]ubusService
	if err := json.Unmarshal([]byte(result.Stdout), &services); err != nil {
		return ubusInstance{}, false
	}
	service, ok := services[m.serviceName]
	if !ok {
		return ubusInstance{}, false
	}
	// map 遍历顺序不定，但本包写出的服务只有一个实例，取到哪个都一样。
	for _, instance := range service.Instances {
		return instance, true
	}
	return ubusInstance{}, false
}

// unitFileState 把 `/etc/init.d/<name> enabled` 的退出码翻译成开机自启状态。
func (m *procdManager) unitFileState(ctx context.Context) string {
	if _, err := m.run(ctx, m.initScript(), "enabled"); err != nil {
		return "disabled"
	}
	return "enabled"
}

func (m *procdManager) run(ctx context.Context, name string, args ...string) (command.Result, error) {
	return m.runFor(ctx, m.timeout, name, args...)
}

func (m *procdManager) runFor(ctx context.Context, timeout time.Duration, name string, args ...string) (command.Result, error) {
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return m.runner.Run(commandCtx, name, args...)
}

func readMemoryBytes(pid int) uint64 {
	value := procStatusField(pid, "VmRSS:")
	value = strings.TrimSpace(strings.TrimSuffix(value, "kB"))
	kilobytes, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return kilobytes * 1024
}

// readThreadCount 供仪表盘的"任务数"格。systemd 那边取自 TasksCurrent，
// procd 没有等价物，但 /proc/<pid>/status 的 Threads 就是同一个意思。
func readThreadCount(pid int) uint64 {
	count, err := strconv.ParseUint(procStatusField(pid, "Threads:"), 10, 64)
	if err != nil {
		return 0
	}
	return count
}

func procStatusField(pid int, prefix string) string {
	content, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "status"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		if value, found := strings.CutPrefix(line, prefix); found {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func readCPUNanoseconds(pid int) uint64 {
	content, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(content))
	if len(fields) < 15 {
		return 0
	}
	utime, err := strconv.ParseUint(fields[13], 10, 64)
	if err != nil {
		return 0
	}
	stime, err := strconv.ParseUint(fields[14], 10, 64)
	if err != nil {
		return 0
	}
	return (utime + stime) * clockTickNanoseconds
}

// readProcessEnvironment 读 dae 进程实际生效的环境变量。
// geo 更新必须知道 DAE_LOCATION_ASSET——它的优先级高于所有默认搜索路径，
// 不读它就可能把新 geo 写到一个根本不生效的目录。
func readProcessEnvironment(pid int) map[string]string {
	content, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "environ"))
	if err != nil || len(content) == 0 {
		return nil
	}
	environment := map[string]string{}
	for _, entry := range strings.Split(string(content), "\x00") {
		name, value, found := strings.Cut(entry, "=")
		if !found || name == "" {
			continue
		}
		environment[name] = value
	}
	if len(environment) == 0 {
		return nil
	}
	return environment
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/host/ -run TestProcd -v`
Expected: 4 个测试全部 PASS

- [ ] **Step 5: Commit**

```bash
git add internal/host/procd.go internal/host/procd_test.go
git commit -m "feat(host): procd 后端的服务状态查询"
```

---

### Task 3: procd 后端的 Action / RestartSelf / Logs 并接入 `host.New`

**Files:**
- Modify: `internal/host/procd.go`
- Modify: `internal/host/procd_test.go`
- Modify: `internal/host/host.go`

**Interfaces:**
- Consumes: Task 2 的 `procdManager`、`initDirectory`、`m.run`/`m.runFor`。
- Produces: `procdManager` 完整实现 `Manager`；导出常量 `PanelInitScript = "/etc/init.d/kdae-panel"`；`host.New` 在 `BackendProcd` 下返回 procd 管理器。

- [ ] **Step 1: 写失败的测试**

追加到 `internal/host/procd_test.go`：

```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/host/ -run 'TestProcd|TestNewReturns|TestBackendResolve' -v`
Expected: 编译失败，`undefined: PanelInitScript`；`TestNewReturnsProcdBackend` 因 "procd 后端尚未实现" 而 FAIL

- [ ] **Step 3: 在 `internal/host/procd.go` 追加实现**

```go
// PanelInitScript 是面板自身的 procd 初始化脚本。
const PanelInitScript = "/etc/init.d/kdae-panel"

func (m *procdManager) Action(ctx context.Context, action Action) error {
	switch action {
	case ActionStart, ActionStop, ActionRestart, ActionEnable, ActionDisable:
	case ActionDaemonReload:
		// procd 每次执行 init 脚本都重新读取服务定义，没有需要"让它重新认识
		// 单元文件"这一步。静默成功而不是报错：首次安装与卸载事务都会调用它。
		return nil
	default:
		return fmt.Errorf("不支持的服务动作 %q", action)
	}
	result, err := m.runFor(ctx, actionTimeout, m.initScript(), string(action))
	if err != nil {
		return fmt.Errorf("执行 %s %s: %s", m.initScript(), action, command.Describe(err, result))
	}
	return nil
}

// RestartSelf 请求 procd 重启面板自身。
//
// setsid 不能省。这条命令是面板的子进程，与面板同属一个会话；procd 停掉面板
// 实例时会把它一并杀掉，于是重启命令先于重启本身死亡，面板停在旧版本上而
// 调用方看到的是"命令被信号终止"。setsid 让它脱离面板的会话与进程组，
// 后台化则让本进程立刻拿回控制权去回复 HTTP 请求。
func (m *procdManager) RestartSelf(ctx context.Context) error {
	script := "setsid " + PanelInitScript + " restart >/dev/null 2>&1 &"
	result, err := m.runFor(ctx, actionTimeout, "/bin/sh", "-c", script)
	if err != nil {
		return fmt.Errorf("请求重启面板: %s", command.Describe(err, result))
	}
	return nil
}

// Logs 读系统日志。本包写出的 init 脚本把服务的 stdout/stderr 交给 procd，
// procd 再转投 syslog，因此 logread 就是全部日志的所在。
func (m *procdManager) Logs(ctx context.Context, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > maxLogLines {
		limit = maxLogLines
	}
	result, err := m.run(ctx, "logread", "-e", m.serviceName)
	if err != nil {
		return nil, fmt.Errorf("读取 logread 日志: %s", command.Describe(err, result))
	}
	entries := parseLogread(result.Stdout, m.serviceName)
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, nil
}

func parseLogread(output, serviceName string) []LogEntry {
	entries := make([]LogEntry, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if entry, ok := parseLogreadLine(line, serviceName); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

// parseLogreadLine 解析一行 logread 输出。两种前缀格式都要兼容：
//
//	Fri Jul 31 01:02:03 2026 daemon.info dae[4321]: level=info msg="…"   (ubox)
//	Jul 31 01:02:03 router dae[7]: 裸消息                                  (busybox)
//
// 唯一可靠的锚点是 "<服务名>["，因此从它切开：之前是时间戳与 facility.level，
// 之后是 pid 和消息体。解析不出的部分一律退化而不丢弃整行——用户看日志
// 正是因为出了问题，这时候少一行比格式难看严重得多。
func parseLogreadLine(line, serviceName string) (LogEntry, bool) {
	tag := serviceName + "["
	index := strings.Index(line, tag)
	if index < 0 {
		return LogEntry{}, false
	}
	entry := LogEntry{Unit: serviceName, Level: "info", Priority: 6}
	prefix := strings.TrimSpace(line[:index])
	entry.Timestamp = parseLogreadTimestamp(prefix)
	if level, ok := logreadLevel(prefix); ok {
		entry.Level = level
		entry.Priority = levelPriority(level)
	}
	pid, message, found := strings.Cut(line[index+len(tag):], "]")
	if !found {
		return entry, true
	}
	entry.PID = pid
	entry.Message = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(message), ":"))
	// dae 自己输出 logfmt。把 level/msg 提到结构化字段上，日志页的级别筛选
	// 才对 dae 的日志同样有效，而不是只对 procd 的封装层有效。
	if level, text, ok := parseLogfmt(entry.Message); ok {
		entry.Level = level
		entry.Priority = levelPriority(level)
		entry.Message = text
	}
	return entry, true
}

// logreadTimestampLayouts 覆盖 ubox 与 busybox 两种 logread 的时间格式。
var logreadTimestampLayouts = []string{
	"Mon Jan _2 15:04:05 2006",
	"2006-01-02 15:04:05",
	"Jan _2 15:04:05",
}

// parseLogreadTimestamp 解析行首时间戳，解析不出返回零值。
//
// 刻意不退回 time.Now()：那会把一批来历不明的旧日志全部标成"刚刚"，
// 比空时间戳更容易误导。零值与 systemd 后端解析失败时的行为一致。
func parseLogreadTimestamp(prefix string) time.Time {
	fields := strings.Fields(prefix)
	for count := len(fields); count > 0; count-- {
		candidate := strings.Join(fields[:count], " ")
		for _, layout := range logreadTimestampLayouts {
			if parsed, err := time.Parse(layout, candidate); err == nil {
				return parsed.UTC()
			}
		}
	}
	return time.Time{}
}

// logreadLevel 从前缀里找 "facility.level" 形式的 token 并取出 level。
func logreadLevel(prefix string) (string, bool) {
	fields := strings.Fields(prefix)
	for index := len(fields) - 1; index >= 0; index-- {
		_, level, found := strings.Cut(fields[index], ".")
		if !found {
			continue
		}
		if levelPriority(level) >= 0 {
			return level, true
		}
	}
	return "", false
}

// parseLogfmt 从 dae 的 `level=… msg="…"` 里取出级别与正文。
// 两者缺一就当作不是 logfmt，保留原始整行。
func parseLogfmt(message string) (string, string, bool) {
	var level, text string
	var foundLevel, foundText bool
	rest := strings.TrimSpace(message)
	for rest != "" {
		name, remainder, found := strings.Cut(rest, "=")
		if !found {
			break
		}
		name = strings.TrimSpace(name)
		if strings.ContainsAny(name, " \t") {
			break
		}
		var value string
		if strings.HasPrefix(remainder, `"`) {
			quoted, tail, closed := strings.Cut(remainder[1:], `"`)
			if !closed {
				break
			}
			value, rest = quoted, strings.TrimSpace(tail)
		} else {
			value, rest, _ = strings.Cut(remainder, " ")
			rest = strings.TrimSpace(rest)
		}
		switch name {
		case "level":
			level, foundLevel = value, true
		case "msg":
			text, foundText = value, true
		}
	}
	if !foundLevel || !foundText {
		return "", "", false
	}
	if levelPriority(level) < 0 {
		return "", "", false
	}
	return level, text, true
}

// levelPriority 把日志级别名映射到 syslog 优先级；未知级别返回 -1。
// 同时收 syslog 的写法（err、crit）与 dae 的写法（error、fatal）。
func levelPriority(level string) int {
	switch level {
	case "emerg":
		return 0
	case "alert":
		return 1
	case "crit", "critical", "fatal":
		return 2
	case "err", "error":
		return 3
	case "warning", "warn":
		return 4
	case "notice":
		return 5
	case "info":
		return 6
	case "debug":
		return 7
	default:
		return -1
	}
}

var _ Manager = (*procdManager)(nil)
```

把 `fmt` 加进 `internal/host/procd.go` 的 import。

- [ ] **Step 4: 把 procd 接进 `host.New`**

改 `internal/host/host.go` 的 switch 分支：

```go
	switch backend {
	case BackendProcd:
		return newProcdManager(options)
	default:
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/host/ -v`
Expected: 全部 PASS，含 Task 1 保留的 systemd 测试

- [ ] **Step 6: Commit**

```bash
git add internal/host
git commit -m "feat(host): procd 后端的服务控制与日志读取"
```

---

### Task 4: 后端选择接入配置、命令行与健康检查

**Files:**
- Modify: `internal/app/config.go`
- Modify: `internal/app/app.go`
- Modify: `cmd/kdae-panel/main.go`
- Modify: `internal/app/app_test.go`

**Interfaces:**
- Consumes: Task 1 的 `host.Backend`、`(Backend).Resolve()`、`host.Options`。
- Produces: `app.Config.ServiceBackend host.Backend`；`GET /api/v1/health` 响应新增 `"backend"` 字段。

- [ ] **Step 1: 写失败的测试**

在 `internal/app/app_test.go` 的健康检查测试里，把匿名响应结构体扩一个字段并加断言：

```go
	var response struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		Backend string `json:"backend"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	// 后端选错的症状（服务控制全部失败）离原因很远，健康检查必须直接说出结论。
	if response.Backend != "systemd" && response.Backend != "procd" {
		t.Fatalf("backend = %q，期望 systemd 或 procd", response.Backend)
	}
```

其余原有断言保持不变。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/app/ -run TestHealth -v`
Expected: FAIL，`backend = "" `

- [ ] **Step 3: 加配置项**

`internal/app/config.go`：在 `Journalctl string` 之后插入字段

```go
	// ServiceBackend 选择用 systemd 还是 procd 管理 dae。
	// 留空即 auto：存在 /sbin/procd 就用 procd，否则 systemd。
	ServiceBackend host.Backend
```

在文件顶部 import 加 `"github.com/tuoro/kdae-panel/internal/host"`。

`DefaultConfig()` 里加一行 `ServiceBackend: host.BackendAuto,`。`withDefaults()` 里加：

```go
	if c.ServiceBackend == "" {
		c.ServiceBackend = defaults.ServiceBackend
	}
```

- [ ] **Step 4: 接进 app.New 与健康检查**

`internal/app/app.go` 把 Task 1 留下的 `Backend: host.BackendAuto` 改成 `Backend: cfg.ServiceBackend`。

紧接着 `host.New(...)` 的错误处理之后，加一行启动日志：

```go
	resolvedBackend, err := cfg.ServiceBackend.Resolve()
	if err != nil {
		return nil, fmt.Errorf("解析服务后端: %w", err)
	}
	logger.Info("已选定服务后端", "backend", string(resolvedBackend))
```

健康检查在 `NewWithDependencies` 里注册，拿不到上面那个局部变量，因此就地再解析一次。`Resolve` 只做一次 `os.Stat`，健康检查的调用频率下开销可忽略；解析失败时 `New` 早已报错退出，这里的兜底只为不 panic：

```go
	router.HandleFunc("GET /api/v1/health", func(writer http.ResponseWriter, request *http.Request) {
		backend, err := cfg.ServiceBackend.Resolve()
		if err != nil {
			backend = cfg.ServiceBackend
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"status":  "ok",
			"version": cfg.Version,
			"backend": string(backend),
		})
	})
```

- [ ] **Step 5: 加命令行与环境变量**

`cmd/kdae-panel/main.go`：在 `journalctl := flag.String(...)` 之后加

```go
	serviceBackend := flag.String("service-backend", envOr("KDAE_PANEL_SERVICE_BACKEND", string(cfg.ServiceBackend)), "服务后端：auto、systemd 或 procd")
```

在 `cfg.Journalctl = *journalctl` 之后加

```go
	cfg.ServiceBackend = host.Backend(*serviceBackend)
```

import 加 `"github.com/tuoro/kdae-panel/internal/host"`。

- [ ] **Step 6: procd 下不注册面板自升级，并强制关闭更新检查**

这一步是修一个"开了就砖"的开关，不是加功能。`internal/upstream/panel.go` 里写死了

```go
	PanelRepoOwner = "tuoro"
	PanelRepoName  = "kdae-panel"
```

自升级从**上游仓库**取二进制，而上游那份**不含 procd 后端**。用户一旦开启并升级，面板会以 root 把自己替换成一个只会调 `systemctl` 的二进制，重启后彻底不可用。同理"新版本检查"拿的是上游的 tag，与本软件包的版本线毫无关系。默认关掉不够——这个开关就不该存在于 procd 部署里。

`internal/app/app.go`，把无条件构造 `panelupdate` 的那一段

```go
	updater, err := panelupdate.New(panelupdate.Options{
		Version:    cfg.Version,
		BackupPath: cfg.PanelBackupPath,
		Enabled:    cfg.EnableSelfUpdate,
		Fetcher:    upstream.NewPanelFetcher(),
		Service:    hostManager,
		Logger:     logger,
	})
	if err != nil {
		_ = authStore.Close()
		return nil, fmt.Errorf("初始化面板自升级: %w", err)
	}
	dependencies.PanelUpdate = updater
```

改为

```go
	// procd 部署由 opkg 管理升级，这里根本不构造自升级能力。
	//
	// 不是"默认关"而是"不存在"：PanelFetcher 指向的是上游 tuoro/kdae-panel，
	// 那里的发布二进制不含 procd 后端。开启后升级一次，面板就会以 root 把自己
	// 替换成一个只会调 systemctl 的程序，重启即不可用。一个开了就砖的开关，
	// 靠默认值和文案是拦不住的。新版本检查同理：上游的 tag 与本软件包的版本线
	// 不是一回事，提示只会误导。
	if resolvedBackend == host.BackendProcd {
		cfg.DisableUpdateCheck = true
	} else {
		updater, err := panelupdate.New(panelupdate.Options{
			Version:    cfg.Version,
			BackupPath: cfg.PanelBackupPath,
			Enabled:    cfg.EnableSelfUpdate,
			Fetcher:    upstream.NewPanelFetcher(),
			Service:    hostManager,
			Logger:     logger,
		})
		if err != nil {
			_ = authStore.Close()
			return nil, fmt.Errorf("初始化面板自升级: %w", err)
		}
		dependencies.PanelUpdate = updater
	}
```

`dependencies.PanelUpdate` 为 nil 时，`registerPanelUpdateRoutes` 已有的分支会让写操作返回 `503 panel_self_update_unavailable`；前端 `PanelUpdatePayload.status` 本就是可选字段，设置页那个开关会自动置灰。**前后端都不需要额外改动。**

- [ ] **Step 7: 写测试锁住这个行为**

在 `internal/app/app_test.go` 追加：

```go
// procd 部署不能提供自升级：它会从上游仓库取回一个不含 procd 后端的二进制
// 并替换自己。这条断言防止以后有人"顺手"把它打开。
func TestProcdBackendDisablesSelfUpdate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ServiceBackend = host.BackendProcd
	if cfg.EnableSelfUpdate != true {
		t.Fatal("前提变了：DefaultConfig 不再默认开启自升级，本测试需要重写")
	}
	// 只验证配置层的判定，不构造完整应用（procd 后端会去碰 /etc/init.d）。
	backend, err := cfg.ServiceBackend.Resolve()
	if err != nil {
		t.Fatalf("解析后端: %v", err)
	}
	if backend != host.BackendProcd {
		t.Fatalf("后端 = %s，期望 procd", backend)
	}
}
```

这条测试只锁住后端解析。真正的"不注册"由 Step 6 的代码分支保证，其行为在真机验证清单里核对（面板设置页的一键升级开关应为置灰）。

- [ ] **Step 8: 把新字段写进 API 文档**

`docs/api.md` 第 30 行把

```markdown
| `GET` | `/health` | 面板健康状态和版本 |
```

改为

```markdown
| `GET` | `/health` | 面板健康状态、版本与服务后端（`backend` 为 `systemd` 或 `procd`） |
```

并在该表格下方补一句：

```markdown
`backend` 说明面板正在用哪一套接口管理 dae：存在 `/sbin/procd` 时自动选 `procd`，否则 `systemd`；
可用 `KDAE_PANEL_SERVICE_BACKEND` 强制。后端选错的症状是服务控制全部失败，而那个现场离原因很远，
因此把结论直接暴露在健康检查里。
```

- [ ] **Step 9: 运行测试确认通过**

Run: `go test ./internal/app/ ./cmd/... -v && go build ./...`
Expected: PASS

- [ ] **Step 10: 手工验证 CLI 拒绝非法值**

Run: `go run ./cmd/kdae-panel --service-backend=upstart --listen 127.0.0.1:65535`
Expected: 立即退出，错误信息含 `未知的服务后端 "upstart"，可选 auto、systemd、procd`

- [ ] **Step 11: Commit**

```bash
git add internal/app cmd/kdae-panel docs/api.md
git commit -m "feat(app): 服务后端可配置并在健康检查中暴露"
```

---

### Task 5: `daeinstall` 抽出 unitProvisioner，systemd 实现搬家

纯重构。结束时 `internal/daeinstall` 现有测试必须原样通过。

**Files:**
- Create: `internal/daeinstall/units.go`
- Modify: `internal/daeinstall/provision.go`
- Modify: `internal/daeinstall/installer.go`
- Modify: `internal/daeinstall/uninstall.go`

**Interfaces:**
- Consumes: `host.Status`、`upstream.Bundle`、`atomicfile.Writable`。
- Produces: 未导出接口 `unitProvisioner{Path() string; WritableDirs() []string; RemovablePaths() []string; Detect(context.Context, host.Status) unitDetection; Plan(upstream.Bundle) (string, bool, error); Commit(context.Context, string, bool) error}`；结构体 `unitDetection{Installed bool; Blocker string; Notes []string}`；`systemdUnits` 实现；`Installer.units unitProvisioner` 字段。

- [ ] **Step 1: 新建 `internal/daeinstall/units.go`**

```go
package daeinstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// unitDetection 是"这台机器上已经有 dae 了吗"的判定结果。
type unitDetection struct {
	Installed bool
	// Blocker 非空表示不能首次安装，它直接作为拒绝理由呈现给用户。
	Blocker string
	// Notes 是不阻断安装、但用户应当知道的情况。
	Notes []string
}

// unitProvisioner 抽掉"服务定义"在不同 init 系统上的差异。
//
// systemd 下服务定义是 /etc/systemd/system/dae.service，由面板从发布包渲染并
// 写入，卸载时删除；procd 下它是 ipk 自带的 /etc/init.d/dae，面板只校验、
// 从不改写，也从不删除——那个文件属于软件包，不属于某一次 dae 安装。
type unitProvisioner interface {
	// Path 是服务定义文件的位置，回报给界面。
	Path() string
	// WritableDirs 是首次安装需要写入、因而必须提前预检的目录。
	WritableDirs() []string
	// RemovablePaths 是卸载时应当一并删除的服务定义文件。
	RemovablePaths() []string
	// VerifyRemovable 在卸载前确认服务定义确实归面板管理。
	// binaryPath 是本次将要删除的可执行文件。
	VerifyRemovable(status host.Status, binaryPath string) error
	// Detect 判定机器上是否已有 dae 服务。
	Detect(ctx context.Context, status host.Status) unitDetection
	// Plan 在动任何文件之前决定服务定义要不要写、写什么。
	// inPlace 为真表示已经就位，Commit 不必写盘。
	Plan(bundle upstream.Bundle) (content string, inPlace bool, err error)
	// Commit 落地服务定义并让 init 系统认识它。
	Commit(ctx context.Context, content string, inPlace bool) error
}

// systemdUnits 管理 systemd 的服务单元。
//
// 刻意不自带目录字段：`Installer.unitDir` 与 `unitDirectory()` / `serviceUnit()`
// 原地不动，现有测试有十余处直接读写它们，把这份状态挪个位置只会制造一批
// 与本次改造无关的测试改动。
type systemdUnits struct {
	installer *Installer
}

func (u *systemdUnits) Path() string {
	return filepath.Join(u.installer.unitDirectory(), u.installer.serviceUnit())
}

func (u *systemdUnits) WritableDirs() []string {
	return []string{u.installer.unitDirectory()}
}

func (u *systemdUnits) RemovablePaths() []string {
	return []string{u.Path()}
}

// VerifyRemovable 确认要删的单元正是面板写下的那一个。
//
// 三道关缺一不可：单元必须位于面板管理的标准路径（别人放在 /usr/lib/systemd
// 下的单元不归面板删）、必须是普通文件、它的 ExecStart 必须与本次要删的
// 可执行文件一致。少了最后一条，一个指向别处的同名单元会被连坐删除。
func (u *systemdUnits) VerifyRemovable(status host.Status, binaryPath string) error {
	// enabled-runtime 是 systemd 独有的临时启用态，面板无法无损恢复它。
	if status.UnitFileState == "enabled-runtime" {
		return errors.New("dae 使用临时启用状态 enabled-runtime，面板无法无损恢复该状态，请先执行 systemctl disable dae")
	}
	if status.UnitPath == "" {
		return errors.New("没有找到 dae 的服务单元")
	}
	unitPath, err := filepath.Abs(status.UnitPath)
	if err != nil {
		return fmt.Errorf("解析 dae 服务单元路径: %w", err)
	}
	expectedUnit, err := filepath.Abs(u.Path())
	if err != nil {
		return fmt.Errorf("解析面板服务单元路径: %w", err)
	}
	if unitPath != expectedUnit {
		return fmt.Errorf(
			"dae 服务单元位于 %s，不是面板管理的标准路径 %s；请用原安装方式卸载", unitPath, expectedUnit)
	}
	if err := regularFile(unitPath, "dae 服务单元"); err != nil {
		return err
	}
	unit, err := os.ReadFile(unitPath)
	if err != nil {
		return fmt.Errorf("读取 dae 服务单元: %w", err)
	}
	if exec := execStartBinary(unitExecStart(string(unit))); exec != binaryPath {
		return fmt.Errorf("服务单元实际启动 %s，与服务状态报告的 %s 不一致，拒绝卸载", exec, binaryPath)
	}
	return nil
}

func (u *systemdUnits) Detect(_ context.Context, status host.Status) unitDetection {
	detection := unitDetection{}
	if status.ExecStartPath == "" {
		return detection
	}
	if _, err := os.Stat(status.ExecStartPath); err == nil {
		detection.Installed = true
		detection.Blocker = fmt.Sprintf(
			"已存在 dae 服务（启动 %s），请使用版本切换而不是首次安装", status.ExecStartPath)
		return detection
	}
	// 单元在、可执行文件不在。升级路径会说"目标不存在"，首次安装若也以
	// "已有服务"为由拒绝，面板就再没有任何办法修好这台机器。只要单元指向的
	// 正是面板要写的位置，就按首次安装把它补齐。
	if filepath.Clean(status.ExecStartPath) != u.installer.binaryPath {
		detection.Installed = true
		detection.Blocker = fmt.Sprintf(
			"服务单元指向的 %s 不存在，而面板配置的 dae 路径是 %s；"+
				"请把 KDAE_PANEL_DAE_BINARY 改成前者后重试",
			status.ExecStartPath, u.installer.binaryPath)
		return detection
	}
	detection.Notes = append(detection.Notes, fmt.Sprintf(
		"服务单元已存在，但它启动的 %s 不见了，本次安装会补齐这个文件", u.installer.binaryPath))
	return detection
}

// Plan 渲染出最终要落盘的单元，并判定它是否已经就位——但不写盘。
//
// 拆成"先算后写"是为了让冲突在事务的最前面暴露：这是唯一一处可能因为机器上
// 已有用户自建单元而中止的检查，必须赶在二进制被替换之前完成。
//
// 已存在的单元一律不覆盖，除非它与本次将要写入的内容逐字节相同——那说明它正是
// 上一轮安装留下的。少了这个例外，一旦 daemon-reload 失败，重试就会被自己写下
// 的单元永久挡住：systemd 还不认识它，所以预检仍认为没装，而写入又拒绝覆盖。
func (u *systemdUnits) Plan(bundle upstream.Bundle) (string, bool, error) {
	if len(bundle.Unit) == 0 {
		return "", false, errors.New("发布包内没有 dae.service，无法创建服务单元")
	}
	rendered, err := u.render(string(bundle.Unit))
	if err != nil {
		return "", false, err
	}
	path := u.Path()
	switch existing, err := os.ReadFile(path); {
	case err == nil && string(existing) == rendered:
		return rendered, true, nil
	case err == nil:
		// 内容不同，但它启动的已经是面板要装的那个文件——官方安装器写的单元、
		// 用户自己调过的单元都属于这种。它能把新装的二进制起起来，就没有理由
		// 为了统一格式去覆盖别人的文件。
		if execStartBinary(unitExecStart(string(existing))) == u.installer.binaryPath {
			return string(existing), true, nil
		}
		return "", false, fmt.Errorf("%s 已存在且启动的不是 %s，面板不覆盖既有服务单元",
			path, u.installer.binaryPath)
	case !os.IsNotExist(err):
		return "", false, err
	}
	return rendered, false, nil
}

func (u *systemdUnits) Commit(ctx context.Context, content string, inPlace bool) error {
	if !inPlace {
		if err := writeFileSynced(u.Path(), []byte(content), unitMode); err != nil {
			return fmt.Errorf("写入服务单元: %w", err)
		}
	}
	if err := u.installer.service.Action(ctx, host.ActionDaemonReload); err != nil {
		return fmt.Errorf("重新加载 systemd 配置: %w", err)
	}
	return nil
}

// render 生成最终落盘的单元内容，并确认改写确实生效。
//
// 替换靠的是上游单元里那两个字面量默认值。上游若换了默认路径，替换会悄无声息
// 地不生效，写出一个指向别处的单元——那样 dae 起不来，而错误现场离真正的原因
// 很远。宁可在这里直接拒绝，把原因说清楚。
func (u *systemdUnits) render(unit string) (string, error) {
	rendered := retargetUnit(unit, u.installer.binaryPath, u.installer.configPath)
	execStart := unitExecStart(rendered)
	if execStart == "" {
		return "", errors.New("发布包内的 dae.service 没有 ExecStart，无法安装")
	}
	if !strings.HasPrefix(execStart, u.installer.binaryPath+" ") && execStart != u.installer.binaryPath {
		return "", fmt.Errorf(
			"发布包内的 dae.service 启动的是 %q，面板无法把它改写为 %s；"+
				"上游可能变更了默认路径，请手动创建服务单元", execStart, u.installer.binaryPath)
	}
	return rendered, nil
}

var _ unitProvisioner = (*systemdUnits)(nil)
```

- [ ] **Step 2: 从 `provision.go` 删掉搬走的部分**

只删这两个方法（它们现在是 `systemdUnits.Plan` 与 `systemdUnits.render`）：`(i *Installer) planUnit(...)`、`(i *Installer) render(...)`。

**其余一概保留在 `provision.go`**，包括 `defaultUnitDirectory` 常量、`(i *Installer) unitDirectory()`、`(i *Installer) serviceUnit()`——`provision_test.go` 与 `uninstall_test.go` 有十余处直接读写 `installer.unitDir` 与调用 `installer.serviceUnit()`，把它们挪走会制造一批与本次改造无关的测试改动，而本任务的验收标准正是"测试一行不改"。同样保留：`SeedConfig`、`configMode`/`geoMode`/`unitMode`、`Provision` 结构体与方法、`FirstInstall`、`writeGeoAssets`、`writeSeedConfig`、`execStartBinary`、`backupExistingBinary`、`unitExecStart`、`retargetUnit`。

- [ ] **Step 3: 让 `Provision` 走 unitProvisioner**

把 `Provision` 方法改成：

```go
func (i *Installer) Provision(ctx context.Context) Provision {
	result := Provision{
		BinaryPath: i.binaryPath,
		ConfigPath: i.configPath,
		UnitPath:   i.units.Path(),
	}
	// 状态查不出来，就不能断言"这台机器上没有 dae"。把查询失败当成绿灯，
	// 会让一次状态查询抽风变成一次无备份的覆盖安装。
	status, err := i.service.Status(ctx)
	if err != nil {
		result.Blockers = append(result.Blockers, fmt.Sprintf(
			"无法读取 %s 的状态，因而不能确认这台机器上是否已有 dae，已拒绝首次安装：%v",
			i.serviceUnit(), err))
		return result
	}
	detection := i.units.Detect(ctx, status)
	result.Notes = append(result.Notes, detection.Notes...)
	if detection.Blocker != "" {
		result.Installed = detection.Installed
		result.Blockers = append(result.Blockers, detection.Blocker)
		return result
	}
	if _, err := upstream.DetectPlatform(); err != nil {
		result.Blockers = append(result.Blockers, err.Error())
		return result
	}

	directories := []string{filepath.Dir(i.binaryPath), filepath.Dir(i.configPath)}
	directories = append(directories, i.units.WritableDirs()...)
	for _, directory := range directories {
		if err := atomicfile.Writable(directory); err != nil {
			result.Blockers = append(result.Blockers, fmt.Sprintf(
				"面板无法写入 %s：%v（systemd 部署需在服务单元的 ReadWritePaths 中列出该目录）",
				directory, err))
		}
	}
	if _, err := os.Stat(i.configPath); err == nil {
		result.Notes = append(result.Notes, fmt.Sprintf("%s 已存在，将保留不动", i.configPath))
	} else {
		result.Notes = append(result.Notes, fmt.Sprintf(
			"将写入不劫持任何流量的种子配置 %s，安装后需自行编写规则再启动", i.configPath))
	}
	// 服务定义里没有 dae，不代表这条路径上没有 dae。
	if _, err := os.Stat(i.binaryPath); err == nil {
		result.Notes = append(result.Notes, fmt.Sprintf(
			"%s 已存在但服务定义里没有对应的服务；安装会先备份它再替换", i.binaryPath))
	}
	result.Notes = append(result.Notes, "安装完成后不会自动启动 dae：透明代理配置不当会切断你当前的连接")
	result.Possible = len(result.Blockers) == 0
	return result
}
```

- [ ] **Step 4: 让 `FirstInstall` 走 unitProvisioner**

把 `FirstInstall` 里这两段：

```go
	unit, unitInPlace, err := i.planUnit(bundle, provision.UnitPath)
```

改为

```go
	unit, unitInPlace, err := i.units.Plan(bundle)
```

以及末尾这段：

```go
	if !unitInPlace {
		if err := writeFileSynced(provision.UnitPath, []byte(unit), unitMode); err != nil {
			return Status{}, fmt.Errorf("写入服务单元: %w", err)
		}
	}
	if err := i.service.Action(ctx, host.ActionDaemonReload); err != nil {
		return Status{}, fmt.Errorf("重新加载 systemd 配置: %w", err)
	}
```

改为

```go
	if err := i.units.Commit(ctx, unit, unitInPlace); err != nil {
		return Status{}, err
	}
```

`provision.go` 若因此不再引用 `host`，从 import 移除。

- [ ] **Step 5: 在 Installer 上挂 units 字段**

`internal/daeinstall/installer.go`：**保留** `unitDir string` 字段不动，在它下面新增一个字段

```go
	// units 抽掉不同 init 系统在"服务定义"上的差异。
	units unitProvisioner
```

在 `New` 的 `return &Installer{...}` 之前构造：

```go
	installer := &Installer{
		binaryPath:  binaryPath,
		configPath:  options.ConfigPath,
		statePath:   options.StatePath,
		backupPath:  options.StatePath + ".previous-dae",
		serviceName: options.ServiceName,
		cache:       newVersionCache(options.StatePath),
		fetcher:     options.Fetcher,
		newProbe:    newProbe,
		service:     options.Service,
		logger:      logger,
		health:      healthWindow,
		interval:    healthInterval,
	}
	installer.units = &systemdUnits{installer: installer}
	return installer, nil
```

- [ ] **Step 6: 确认测试确实一行没改**

Run: `git diff --stat -- 'internal/daeinstall/*_test.go'`
Expected: 无输出。`installer.unitDir`、`installer.serviceUnit()` 都还在原处，测试不该受影响。
若这里有输出，说明 Step 2 多删了东西——回去把它放回 `provision.go`，而不是改测试。

- [ ] **Step 7: 把 `uninstallTarget` 里的单元校验搬进 provisioner**

`internal/daeinstall/uninstall.go` 的 `uninstallTarget` 现在返回 `(host.Status, string, string, error)`，其中第三个是 unitPath，末尾约 25 行是 systemd 单元专属校验。把签名收窄为 `(host.Status, string, error)`，删掉从 `unitPath, err := filepath.Abs(status.UnitPath)` 到 `return status, target, unitPath, nil` 之前的整段（那段逻辑已原样搬进 `systemdUnits.VerifyRemovable`），末尾改为：

```go
	if err := i.units.VerifyRemovable(status, target); err != nil {
		return host.Status{}, "", err
	}
	return status, target, nil
}
```

同时删掉函数开头的 `enabled-runtime` 检查（同样已搬进 `VerifyRemovable`），并把

```go
	if status.ExecStartPath == "" || status.UnitPath == "" {
		return host.Status{}, "", "", errors.New("没有找到可卸载的 dae systemd 服务")
	}
```

改为

```go
	// UnitPath 的校验交给 units：procd 下服务定义归软件包，不该在这里挡路。
	if status.ExecStartPath == "" {
		return host.Status{}, "", errors.New("没有找到可卸载的 dae 服务")
	}
```

把其余 `return host.Status{}, "", "", ...` 全部改为三返回值形式 `return host.Status{}, "", ...`。

- [ ] **Step 8: 让 `Uninstall` 的必需路径列表由 units 决定**

`internal/daeinstall/uninstall.go`：把 `Uninstall` 开头的路径拼装改成

```go
	status, target, err := i.uninstallTarget(ctx)
	if err != nil {
		return err
	}
	dataPaths, err := i.uninstallDataPaths(status, options)
	if err != nil {
		return err
	}
	// required 里的文件必须存在，缺一个就说明状态与账本对不上，宁可中止；
	// 其余的允许缺失。procd 下服务定义属于软件包而非某次安装，因此
	// RemovablePaths 为空——卸载 dae 不该删掉 ipk 装的 /etc/init.d/dae。
	required := append([]string{target}, i.units.RemovablePaths()...)
	paths := append([]string{}, required...)
	paths = append(paths,
		i.statePath,
		i.previousStatePath(),
		i.backupPath,
		i.pendingBackupPath(),
	)
	paths = append(paths, dataPaths...)
	for index, path := range paths {
		if _, err := os.Lstat(path); err != nil {
			if index >= len(required) && os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("检查待删除文件 %s: %w", path, err)
		}
		if err := atomicfile.Writable(filepath.Dir(path)); err != nil {
			return fmt.Errorf("面板无法删除 %s：%w", path, err)
		}
	}
```

把后面 `stageRemoval(path, index >= 2)` 改为 `stageRemoval(path, index >= len(required))`。

把结尾的日志行

```go
	i.logger.Info("已卸载 dae", "binary", target, "unit", unitPath,
```

改为

```go
	i.logger.Info("已卸载 dae", "binary", target, "unit", i.units.Path(),
```

- [ ] **Step 9: 全量验证，确认是纯重构**

Run: `go build ./... && go test ./... && go vet ./...`
Expected: 全部 PASS，`internal/daeinstall` 的测试数量与之前一致。

已核对过的两处：`uninstall_test.go:206` 断言 "不是面板管理的标准路径"，`VerifyRemovable` 原样保留了这句，不受影响；没有任何测试断言 "没有找到可卸载的 dae systemd 服务"，Step 7 改这句是安全的。若仍有测试因文案失败，**只允许改断言里的期望文案**，不允许把实现改回带 systemd 字样。

- [ ] **Step 10: Commit**

```bash
git add internal/daeinstall
git commit -m "refactor(daeinstall): 把服务定义抽成 unitProvisioner"
```

---

### Task 6: `daeinstall` 的 procd 实现

**Files:**
- Create: `internal/daeinstall/units_procd.go`
- Create: `internal/daeinstall/units_procd_test.go`
- Modify: `internal/daeinstall/installer.go`
- Modify: `internal/app/app.go`

**Interfaces:**
- Consumes: Task 5 的 `unitProvisioner`、`unitDetection`、`Installer` 字段。
- Produces: `procdUnits` 实现；`daeinstall.Options.ServiceBackend host.Backend` 新字段。

- [ ] **Step 1: 写失败的测试**

创建 `internal/daeinstall/units_procd_test.go`：

```go
package daeinstall

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

func newProcdUnitsForTest(t *testing.T, binaryPath, initPath string) *procdUnits {
	t.Helper()
	return &procdUnits{
		installer: &Installer{binaryPath: binaryPath},
		path:      initPath,
	}
}

// init 脚本由 ipk 提供，面板只校验存在性，永远不写。
func TestProcdUnitsPlanAcceptsExistingScript(t *testing.T) {
	dir := t.TempDir()
	initPath := filepath.Join(dir, "dae")
	if err := os.WriteFile(initPath, []byte("#!/bin/sh /etc/rc.common\n"), 0o755); err != nil {
		t.Fatalf("写入 init 脚本: %v", err)
	}
	units := newProcdUnitsForTest(t, "/usr/bin/dae", initPath)

	content, inPlace, err := units.Plan(upstream.Bundle{})
	if err != nil {
		t.Fatalf("Plan 返回错误: %v", err)
	}
	if !inPlace {
		t.Fatal("inPlace 应为真：init 脚本由软件包提供，面板不写")
	}
	if content != "" {
		t.Fatalf("content = %q，期望空", content)
	}
}

// 脚本不在说明软件包坏了或被删。必须在动任何文件之前拒绝，
// 而不是装完二进制才发现服务起不来。
func TestProcdUnitsPlanRejectsMissingScript(t *testing.T) {
	units := newProcdUnitsForTest(t, "/usr/bin/dae", filepath.Join(t.TempDir(), "dae"))

	if _, _, err := units.Plan(upstream.Bundle{}); err == nil {
		t.Fatal("init 脚本缺失时 Plan 应当报错")
	} else if !strings.Contains(err.Error(), "kdae-panel") {
		t.Fatalf("错误信息 = %q，应当指引重装软件包", err.Error())
	}
}

// Commit 什么都不做，尤其不能调 daemon-reload 之外的任何写操作。
func TestProcdUnitsCommitIsNoop(t *testing.T) {
	dir := t.TempDir()
	initPath := filepath.Join(dir, "dae")
	units := newProcdUnitsForTest(t, "/usr/bin/dae", initPath)

	if err := units.Commit(context.Background(), "", true); err != nil {
		t.Fatalf("Commit 返回错误: %v", err)
	}
	if _, err := os.Stat(initPath); !os.IsNotExist(err) {
		t.Fatal("Commit 不应创建任何文件")
	}
}

// procd 下卸载 dae 不该删掉 ipk 装的 init 脚本。
func TestProcdUnitsRemovesNothing(t *testing.T) {
	units := newProcdUnitsForTest(t, "/usr/bin/dae", "/etc/init.d/dae")

	if paths := units.RemovablePaths(); len(paths) != 0 {
		t.Fatalf("RemovablePaths = %v，期望空", paths)
	}
	if dirs := units.WritableDirs(); len(dirs) != 0 {
		t.Fatalf("WritableDirs = %v，期望空：init 脚本目录面板从不写", dirs)
	}
	// 卸载 dae 不该因为"单元校验不过"而被拦下——procd 下根本没有要校验的单元。
	if err := units.VerifyRemovable(host.Status{}, "/usr/bin/dae"); err != nil {
		t.Fatalf("VerifyRemovable 返回错误: %v", err)
	}
}

// 二进制在即已安装。不看 ExecStartPath：procd 下服务停止时它来自回退链，
// 回退值恒等于面板配置的路径，据此判断会永远为真。
func TestProcdUnitsDetectUsesBinaryPresence(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "dae")
	units := newProcdUnitsForTest(t, binary, filepath.Join(dir, "init"))

	detection := units.Detect(context.Background(), host.Status{ExecStartPath: binary})
	if detection.Installed {
		t.Fatal("二进制不存在时不应判为已安装")
	}
	if detection.Blocker != "" {
		t.Fatalf("Blocker = %q，期望空", detection.Blocker)
	}

	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatalf("写入假二进制: %v", err)
	}
	detection = units.Detect(context.Background(), host.Status{ExecStartPath: binary})
	if !detection.Installed {
		t.Fatal("二进制存在时应判为已安装")
	}
	if !strings.Contains(detection.Blocker, "版本切换") {
		t.Fatalf("Blocker = %q，应当引导用户走版本切换", detection.Blocker)
	}
}

// procd 部署的用户看不懂 systemd 词汇，文案里不能出现它们。
func TestProcdUnitsMessagesAvoidSystemdVocabulary(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "dae")
	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatalf("写入假二进制: %v", err)
	}
	units := newProcdUnitsForTest(t, binary, filepath.Join(dir, "init"))

	_, _, planErr := units.Plan(upstream.Bundle{})
	detection := units.Detect(context.Background(), host.Status{})
	texts := []string{planErr.Error(), detection.Blocker}
	for _, forbidden := range []string{"systemd", "systemctl", "journalctl", "ReadWritePaths", ".service"} {
		for _, text := range texts {
			if strings.Contains(text, forbidden) {
				t.Fatalf("文案 %q 含 systemd 词汇 %q", text, forbidden)
			}
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/daeinstall/ -run TestProcdUnits -v`
Expected: 编译失败，`undefined: procdUnits`

- [ ] **Step 3: 实现 `internal/daeinstall/units_procd.go`**

```go
package daeinstall

import (
	"context"
	"fmt"
	"os"

	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// defaultProcdInitPath 是 kdae-panel 软件包安装的 dae 启动脚本。
const defaultProcdInitPath = "/etc/init.d/dae"

// procdUnits 管理 procd 的服务定义。
//
// 与 systemd 分支最大的不同：这个文件属于 kdae-panel 软件包，不属于任何一次
// dae 安装。面板只校验它在不在、从不写它，卸载 dae 时也不删它——删了之后
// 用户就再也没法从面板重新装回 dae，而修复手段是重装整个软件包。
type procdUnits struct {
	installer *Installer
	// path 是 init 脚本的位置，留空即用默认，测试会覆盖它。
	path string
}

func (u *procdUnits) Path() string {
	if u.path != "" {
		return u.path
	}
	return defaultProcdInitPath
}

// WritableDirs 为空：面板从不写 init 脚本所在的目录。
func (u *procdUnits) WritableDirs() []string { return nil }

// RemovablePaths 为空：init 脚本归软件包所有。
func (u *procdUnits) RemovablePaths() []string { return nil }

// VerifyRemovable 无事可做：没有要删的服务定义，也就没有"删对了没有"可校验。
// dae 可执行文件本身的归属校验在 uninstallTarget 里按摘要账本完成，与后端无关。
func (u *procdUnits) VerifyRemovable(host.Status, string) error { return nil }

// Detect 以磁盘上有没有 dae 可执行文件为准。
//
// 刻意不看 status.ExecStartPath：procd 在服务停止时拿不到命令行，那个字段
// 会回退成面板自己配置的路径，据此判断"已安装"就恒为真，首次安装从此不可达。
func (u *procdUnits) Detect(_ context.Context, _ host.Status) unitDetection {
	detection := unitDetection{}
	info, err := os.Stat(u.installer.binaryPath)
	if err != nil {
		return detection
	}
	if !info.Mode().IsRegular() {
		detection.Installed = true
		detection.Blocker = fmt.Sprintf("%s 已存在且不是普通文件，面板拒绝替换它", u.installer.binaryPath)
		return detection
	}
	detection.Installed = true
	detection.Blocker = fmt.Sprintf(
		"已存在 dae（%s），请使用版本切换而不是首次安装", u.installer.binaryPath)
	return detection
}

// Plan 只校验 init 脚本在不在。
//
// 必须赶在二进制被替换之前查：脚本缺失时装上 dae 也起不来，而那时候
// 错误现场离真正的原因（软件包被破坏）已经很远了。
func (u *procdUnits) Plan(upstream.Bundle) (string, bool, error) {
	path := u.Path()
	info, err := os.Stat(path)
	if err != nil {
		return "", false, fmt.Errorf(
			"找不到 dae 的启动脚本 %s：%v；它由 kdae-panel 软件包提供，请重新安装该软件包", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("%s 不是普通文件，面板拒绝据此安装 dae", path)
	}
	return "", true, nil
}

// Commit 什么都不做：脚本已经在位，procd 每次执行它都会重读定义。
func (u *procdUnits) Commit(context.Context, string, bool) error { return nil }

var _ unitProvisioner = (*procdUnits)(nil)
```

- [ ] **Step 4: 按后端选择 provisioner**

`internal/daeinstall/installer.go` 的 `Options` 加字段：

```go
	// ServiceBackend 决定服务定义由哪套 init 系统承载，留空即 systemd。
	ServiceBackend host.Backend
```

`New` 里把

```go
	installer.units = &systemdUnits{installer: installer}
```

改为

```go
	backend, err := options.ServiceBackend.Resolve()
	if err != nil {
		return nil, err
	}
	if backend == host.BackendProcd {
		installer.units = &procdUnits{installer: installer}
	} else {
		installer.units = &systemdUnits{installer: installer}
	}
```

- [ ] **Step 5: 在 app.New 里传后端**

`internal/app/app.go` 的 `daeinstall.New(daeinstall.Options{...})` 加一行：

```go
			ServiceBackend: cfg.ServiceBackend,
```

- [ ] **Step 6: 写崩溃循环检测的失败测试**

安装新 dae 版本后有一段观察窗口，靠 `status.Restarts` 是否增长来发现"两次采样之间已经崩过一轮又被拉起来"的情况（`installer.go` 的重启后观察循环）。**procd 不暴露重启计数器，这个安全网在 OpenWrt 上完全失效**——新版本在 respawn 循环里反复崩溃，面板会判定安装成功。

等价信号是主进程号：respawn 必然换 pid，pid 变了就等于中间挂过。这对 systemd 同样成立，因此是纯增强而非分支。

给 `internal/daeinstall/installer_test.go` 的 `fakeService` 增加两个字段（紧挨现有的 `restartsGrow` / `restartsAt`）：

```go
	// pidChurn 为真时每次状态查询都换一个 pid，模拟"两次采样之间服务已经
	// 崩过一轮又被拉起来"，而 ActiveState 全程 active。procd 不暴露重启
	// 计数器，这是那边唯一能发现崩溃循环的信号。
	pidChurn bool
	pidAt    int
```

在 `fakeService.Status` 里，参照 `restartsGrow` 的写法填 `MainPID`：

```go
	if s.pidChurn {
		s.pidAt++
	} else if s.pidAt == 0 {
		s.pidAt = 4321
	}
	status.MainPID = s.pidAt
```

（`status` 是该方法里已有的返回值变量；按文件实际写法接上即可。）

新增测试：

```go
// procd 不暴露重启计数器，崩溃循环只能靠主进程号变化发现。
// 没有这一条，respawn 循环里反复崩溃的新版本会被判定为安装成功。
func TestInstallRejectsPIDChurnDuringHealthWindow(t *testing.T) {
	service := &fakeService{activeState: "active", pidChurn: true}
	installer, _ := newTestInstaller(t, &fakeFetcher{}, service)

	err := installer.waitHealthy(context.Background())
	if err == nil {
		t.Fatal("观察窗口内 pid 变化应当判定为不稳定")
	}
	if !strings.Contains(err.Error(), "进程号") {
		t.Fatalf("错误信息 = %q，应当说明是进程号变化", err.Error())
	}
}
```

`waitHealthy` 是承载重启后观察循环的那个方法；按文件里的实际方法名调用，必要时先把 `i.service.Action(ctx, host.ActionRestart)` 那一步剥离出来以便直接测观察循环。

- [ ] **Step 7: 运行测试确认失败**

Run: `go test ./internal/daeinstall/ -run TestInstallRejectsPIDChurn -v`
Expected: FAIL，"观察窗口内 pid 变化应当判定为不稳定"

- [ ] **Step 8: 让观察窗口同时盯住进程号**

`internal/daeinstall/installer.go` 的重启后观察循环，把基线与判定都扩到 pid：

```go
	deadline := time.Now().Add(i.health)
	var baseline uint64
	var baselinePID int
	sampled := false
	for {
		select {
		case <-time.After(i.interval):
		case <-restartCtx.Done():
			return restartCtx.Err()
		}
		status, err := i.service.Status(restartCtx)
		if err != nil {
			return fmt.Errorf("重启后无法读取服务状态: %w", err)
		}
		// 崩溃重启循环里 ActiveState 会在 activating/failed 之间跳，
		// 任何一次不是 active 都判定失败，而不是等到窗口结束再看最后一眼。
		if status.ActiveState != "active" {
			return fmt.Errorf("重启后服务状态为 %s/%s", status.ActiveState, status.SubState)
		}
		// 只看 ActiveState 会漏掉采样间隔内跑完的崩溃-重启循环：
		// 两次采样都是 active，中间其实已经挂掉并被重新拉起来过。
		if !sampled {
			baseline, baselinePID, sampled = status.Restarts, status.MainPID, true
			// 第一次采样只建立基线，不能因为调度延迟已经越过 deadline 就直接成功。
			// 至少再采一次，才能判断观察期间有没有发生崩溃重启。
			continue
		}
		// systemd 的 NRestarts 单调递增，是最直接的证据。
		if status.Restarts > baseline {
			return fmt.Errorf("重启后服务在观察窗口内又重启了 %d 次，新版本很可能起不稳",
				status.Restarts-baseline)
		}
		// procd 不暴露重启计数器，NRestarts 恒为 0，上面那条永远不成立。
		// 但重新拉起必然换主进程号，pid 变了就等于中间挂过一次。
		if baselinePID != 0 && status.MainPID != 0 && status.MainPID != baselinePID {
			return fmt.Errorf("重启后服务的主进程号从 %d 变成 %d，说明它在观察窗口内挂掉并被重新拉起，新版本很可能起不稳",
				baselinePID, status.MainPID)
		}
		if !time.Now().Before(deadline) {
			return nil
		}
	}
```

- [ ] **Step 9: 运行测试确认通过**

Run: `go test ./internal/daeinstall/ ./internal/app/ -v`
Expected: 全部 PASS，含原有的 `restartsGrow` 用例（systemd 路径的检测不受影响）

- [ ] **Step 10: 全量验证**

Run: `go build ./... && go test ./... && go vet ./...`
Expected: 全部 PASS

- [ ] **Step 11: Commit**

```bash
git add internal/daeinstall internal/app/app.go
git commit -m "feat(daeinstall): procd 下的服务定义只校验不改写，崩溃循环改盯进程号"
```

---

### Task 7: geo 缺失警告与写权限提示不再假定 systemd

**Files:**
- Modify: `internal/geodata/locate.go`
- Modify: `internal/geodata/geodata_test.go`
- Modify: `internal/panelupdate/panelupdate.go`

**Interfaces:**
- Consumes: 无新依赖。
- Produces: `geodata.SandboxHiddenDir` 从 `const` 改为 `var`（测试可覆盖）；`MissingWarning` 签名不变。

- [ ] **Step 1: 写失败的测试**

追加到 `internal/geodata/geodata_test.go`：

```go
// 面板能读到 /root 时，"未找到"就是未找到，不该再拿沙箱当挡箭牌。
// procd 部署没有 ProtectHome，用户看到含 ProtectHome 的措辞只会困惑。
func TestMissingWarningIsDirectWhenHomeVisible(t *testing.T) {
	visible := t.TempDir()
	previous := SandboxHiddenDir
	SandboxHiddenDir = filepath.Join(visible, "dae")
	t.Cleanup(func() { SandboxHiddenDir = previous })

	warning := MissingWarning([]string{t.TempDir()})
	if warning == "" {
		t.Fatal("geo 文件缺失时应当给出警告")
	}
	for _, forbidden := range []string{"ProtectHome", "systemd", "面板单元"} {
		if strings.Contains(warning, forbidden) {
			t.Fatalf("警告 %q 含沙箱措辞 %q", warning, forbidden)
		}
	}
}

// 面板读不到 /root（ProtectHome=true）时必须留有余地：
// 文件可能就在那里而 dae 读得好好的，说死"未找到"会把正常系统报成故障。
func TestMissingWarningHedgesWhenHomeHidden(t *testing.T) {
	hidden := t.TempDir()
	unreadable := filepath.Join(hidden, "root")
	if err := os.Mkdir(unreadable, 0o000); err != nil {
		t.Fatalf("创建不可读目录: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o755) })
	if os.Geteuid() == 0 {
		t.Skip("root 能读任何目录，无法模拟沙箱遮挡")
	}
	previous := SandboxHiddenDir
	SandboxHiddenDir = filepath.Join(unreadable, ".local", "share", "dae")
	t.Cleanup(func() { SandboxHiddenDir = previous })

	warning := MissingWarning([]string{t.TempDir()})
	if !strings.Contains(warning, SandboxHiddenDir) {
		t.Fatalf("警告 %q 应当提到面板看不到的那个目录", warning)
	}
}
```

确保测试文件 import 含 `os`、`path/filepath`、`strings`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/geodata/ -run TestMissingWarning -v`
Expected: 编译失败，`cannot assign to SandboxHiddenDir (neither addressable nor a map index expression)`

- [ ] **Step 3: 改 `internal/geodata/locate.go`**

把常量改成变量并重写警告：

```go
// SandboxHiddenDir 是 dae 搜索顺序里面板可能看不到的那一位。
//
// dae 以 root 运行时会读 $HOME/.local/share/dae。systemd 单元设了
// ProtectHome=true 时，/root 被换成一个空且不可访问的目录，面板也没有
// CAP_DAC_OVERRIDE 可以绕；procd 部署则没有这层遮挡。仍然把它列进搜索顺序，
// 是因为 dae 确实读这里。做成变量是给测试留的缝。
var SandboxHiddenDir = "/root/.local/share/dae"

// MissingWarning 在面板可见的目录里都找不到 geo 数据时提醒，找得到就返回空。
//
// 必须提醒：dae 只在路由规则用到 geosite/geoip 时才读它们，但一旦用到而文件
// 不在，dae 会直接启动失败，且 dae validate 完全察觉不到——它只读配置文件。
//
// 措辞取决于面板能不能读到 SandboxHiddenDir。读不到时留余地：文件可能就在
// 那里而 dae 读得好好的，说死"未找到"会把一个正常运行的系统报成故障。
// 读得到时就该直说——对着 procd 部署的用户念 ProtectHome 只会让人困惑。
func MissingWarning(searchPath []string) string {
	for _, file := range locate(searchPath, Names) {
		if file.Present {
			continue
		}
		if sandboxHidesHome() {
			return fmt.Sprintf("在面板可见的目录里未找到 geoip.dat / geosite.dat；"+
				"%s 受面板沙箱限制读不到，文件若在那里 dae 仍能读到。"+
				"确实缺失且路由规则用到 geosite/geoip 时，dae 将无法启动", SandboxHiddenDir)
		}
		return "未找到 geoip.dat / geosite.dat；路由规则用到 geosite/geoip 时 dae 将无法启动"
	}
	return ""
}

// sandboxHidesHome 判断 SandboxHiddenDir 是否因为沙箱而对本进程不可见。
//
// 判据是读它的上级目录得到权限错误——ProtectHome=true 正是这个症状。
// 目录不存在不算遮挡：那说明 dae 本来就没在那里放东西。
func sandboxHidesHome() bool {
	_, err := os.ReadDir(filepath.Dir(SandboxHiddenDir))
	return errors.Is(err, fs.ErrPermission)
}
```

import 加 `"errors"` 与 `"io/fs"`。

注意 `systemDirs` 在包初始化时就把 `SandboxHiddenDir` 的值抄了一份，改这个变量不会改动 `systemDirs[0]`。这对本任务无影响（`MissingWarning` 只直接读该变量），但测试必须用 `t.Cleanup` 还原它，否则会影响同包内其他用例。

- [ ] **Step 4: 改 daeinstall 的"尚未安装"文案**

`internal/daeinstall/installer.go` 第 258 行附近把

```go
		return "", false, errors.New("dae 尚未作为 systemd 服务安装，面板只能升级或切换已有的 dae")
```

改为

```go
		return "", false, errors.New("机器上找不到 dae 的服务定义，面板只能升级或切换已有的 dae")
```

`internal/daeinstall/installer_test.go` 第 272 行断言的正是旧文案，同步改掉：

```go
	if err == nil || !strings.Contains(err.Error(), "找不到 dae 的服务定义") {
```

这是本计划里**唯一**一处需要改动既有测试断言的地方，改的是期望文案而不是行为。同文件第 269 行与 `provision_test.go` 第 51 行的同类字样在注释里，改不改都行。

- [ ] **Step 5: 改 panelupdate 的写权限提示**

`internal/panelupdate/panelupdate.go` 里把

```go
		status.Problem = fmt.Sprintf(
			"面板无法写入 %s：%v；自升级需要在 kdae-panel.service 的 ReadWritePaths 中加入该目录",
			filepath.Dir(m.binaryPath), err)
```

改为

```go
		status.Problem = fmt.Sprintf(
			"面板无法写入 %s：%v（systemd 部署需在服务单元的 ReadWritePaths 中列出该目录）",
			filepath.Dir(m.binaryPath), err)
```

- [ ] **Step 6: 运行测试确认通过**

Run: `go test ./internal/geodata/ ./internal/panelupdate/ ./internal/daeinstall/ -v`
Expected: 全部 PASS

- [ ] **Step 7: Commit**

```bash
git add internal/geodata internal/panelupdate/panelupdate.go internal/daeinstall/installer.go
git commit -m "fix: 提示文案不再假定部署在 systemd 上"
```

---

### Task 8: `kdae-panel` OpenWrt 包

**Files:**
- Create: `openwrt/kdae-panel/Makefile`
- Create: `openwrt/kdae-panel/files/kdae-panel.init`
- Create: `openwrt/kdae-panel/files/dae.init`
- Create: `openwrt/kdae-panel/files/kdae-panel.config`

**Interfaces:**
- Consumes: Task 4 的 `--service-backend`（此处不传，靠自动探测即可）、面板现有全部 flag。
- Produces: UCI 配置节 `kdae-panel.main`，选项 `enabled` `listen_addr` `listen_port` `data_dir` `dae_binary` `dae_config` `service_name` `enable_dae_install` `enable_geo_update` `trusted_proxies` `session_ttl` `secure_cookie`；构建变量 `KDAE_PANEL_BIN`。

- [ ] **Step 1: 写 UCI 默认配置**

创建 `openwrt/kdae-panel/files/kdae-panel.config`：

```
config kdae-panel 'main'
	option enabled '1'
	option listen_addr '0.0.0.0'
	option listen_port '2023'
	# /var 在 OpenWrt 上是 /tmp 的软链，重启即空。数据库、备份、状态文件与
	# dae 本地版本库必须落在 overlay 上，因此默认放 /etc/kdae-panel。
	option data_dir '/etc/kdae-panel'
	option dae_binary '/usr/bin/dae'
	option dae_config '/etc/dae/config.dae'
	option service_name 'dae'
	option enable_dae_install '1'
	# 上游默认关闭它，理由是"给部署新增一条常态化的联网取字节 → 以 root 写系统目录
	# 的路径"。这里默认打开：dae 的 geo 也归面板管，而上面的 enable_dae_install=1
	# 已经引入同一条路径且权限更大，再关掉 geo 只是逼用户手工放文件。
	# 反过来，若把 enable_dae_install 改成 0，应当一并把这一项也改成 0。
	option enable_geo_update '1'
	# 没有 enable_self_update / disable_update_check：procd 后端下面板压根不注册
	# 自升级能力，也不做版本检查（两者都指向上游 tuoro/kdae-panel，那份二进制
	# 不含 procd 后端，升级一次就把面板换成不可用的程序）。升级请安装新的 ipk。
	option trusted_proxies '127.0.0.0/8,::1/128'
	option session_ttl '12h'
	option secure_cookie '0'
```

- [ ] **Step 2: 写面板的 procd 脚本**

创建 `openwrt/kdae-panel/files/kdae-panel.init`：

```sh
#!/bin/sh /etc/rc.common

USE_PROCD=1
START=99
STOP=10

PANEL_BIN=/usr/bin/kdae-panel
RUN_DIR=/var/run/kdae-panel

start_service() {
	config_load kdae-panel

	local enabled listen_addr listen_port data_dir dae_binary dae_config service_name
	local enable_dae_install enable_geo_update
	local trusted_proxies session_ttl secure_cookie

	config_get_bool enabled main enabled 1
	[ "$enabled" = 1 ] || return 0

	config_get listen_addr main listen_addr '0.0.0.0'
	config_get listen_port main listen_port '2023'
	config_get data_dir main data_dir '/etc/kdae-panel'
	config_get dae_binary main dae_binary '/usr/bin/dae'
	config_get dae_config main dae_config '/etc/dae/config.dae'
	config_get service_name main service_name 'dae'
	config_get_bool enable_dae_install main enable_dae_install 1
	config_get_bool enable_geo_update main enable_geo_update 1
	config_get trusted_proxies main trusted_proxies '127.0.0.0/8,::1/128'
	config_get session_ttl main session_ttl '12h'
	config_get_bool secure_cookie main secure_cookie 0

	# 数据目录在 overlay 上，必须持久；配置里可能有订阅地址与节点凭据，
	# 因此只给 root 读写。
	mkdir -p "$data_dir/backups" "$(dirname "$dae_config")"
	chmod 0700 "$data_dir"
	# 一次性初始化链接反而应当是易失的：重启后它本就该重新生成。
	mkdir -p "$RUN_DIR"
	chmod 0700 "$RUN_DIR"

	procd_open_instance
	procd_set_param command "$PANEL_BIN" \
		--listen "$listen_addr:$listen_port" \
		--dae-binary "$dae_binary" \
		--dae-config "$dae_config" \
		--service-name "$service_name" \
		--database "$data_dir/panel.db" \
		--backup-dir "$data_dir/backups" \
		--schedule-file "$data_dir/schedule.json" \
		--install-state-file "$data_dir/dae-install.json" \
		--geo-state-file "$data_dir/geo-update.json" \
		--geo-schedule-file "$data_dir/geo-schedule.json" \
		--panel-backup-file "$data_dir/kdae-panel.previous" \
		--setup-url-file "$RUN_DIR/setup-url" \
		--enable-dae-install="$enable_dae_install" \
		--enable-geo-update="$enable_geo_update" \
		--trusted-proxies "$trusted_proxies" \
		--session-ttl "$session_ttl" \
		--secure-cookie="$secure_cookie"
	procd_set_param respawn
	procd_set_param stdout 1
	procd_set_param stderr 1
	procd_close_instance
}

service_triggers() {
	procd_add_reload_trigger "kdae-panel"
}

reload_service() {
	restart
}
```

- [ ] **Step 3: 写 dae 的 procd 脚本**

创建 `openwrt/kdae-panel/files/dae.init`：

```sh
#!/bin/sh /etc/rc.common

USE_PROCD=1
# dae 先于面板启动：面板起来时能直接读到真实状态，少一次"刚装完显示未运行"。
START=90
STOP=15

start_service() {
	config_load kdae-panel

	local dae_binary dae_config
	config_get dae_binary main dae_binary '/usr/bin/dae'
	config_get dae_config main dae_config '/etc/dae/config.dae'

	# 二进制由面板的版本管理页安装，全新机器上本来就没有。
	# 明确说出原因，而不是让 procd 反复重启一个不存在的程序。
	if [ ! -x "$dae_binary" ]; then
		echo "dae 尚未安装：请先在 kdae 面板的版本管理页完成安装" >&2
		return 1
	fi
	if [ ! -f "$dae_config" ]; then
		echo "找不到 dae 配置 $dae_config：请先在面板的配置页写入配置" >&2
		return 1
	fi
	# dae 要 pin eBPF map，bpffs 没挂载时直接启动失败。
	grep -q ' /sys/fs/bpf bpf ' /proc/mounts || mount -t bpf bpf /sys/fs/bpf 2>/dev/null

	procd_open_instance
	procd_set_param command "$dae_binary" run -c "$dae_config"
	# geo 数据与配置同目录：那是 dae 搜索顺序里优先级最高的默认位置，
	# 也是面板唯一确定可写的目录。
	procd_set_param env DAE_LOCATION_ASSET="$(dirname "$dae_config")"
	procd_set_param respawn
	procd_set_param stdout 1
	procd_set_param stderr 1
	procd_set_param limits nofile="1048576 1048576"
	procd_close_instance
}

reload_service() {
	# dae 支持无损重载。退化成 restart 会断掉所有进行中的连接，
	# 而 geo 更新与配置保存都会走到这里。
	local dae_binary
	config_load kdae-panel
	config_get dae_binary main dae_binary '/usr/bin/dae'
	"$dae_binary" reload
}
```

- [ ] **Step 4: 写包 Makefile**

创建 `openwrt/kdae-panel/Makefile`：

```make
include $(TOPDIR)/rules.mk

PKG_NAME:=kdae-panel
PKG_VERSION:=1.0.0
PKG_RELEASE:=1

PKG_LICENSE:=AGPL-3.0-only
PKG_LICENSE_FILES:=LICENSE
PKG_MAINTAINER:=senshinya

# KDAE_PANEL_BIN 指向已经用官方 Go 交叉编译好的静态二进制，由 CI 在调用
# make 之前导出。不在 SDK 里从 Go 源码编译：本项目要求 Go 1.25，而
# 24.10 feed 里的 golang 到不了这个版本。
KDAE_PANEL_BIN?=

include $(INCLUDE_DIR)/package.mk

define Package/kdae-panel
  SECTION:=net
  CATEGORY:=Network
  SUBMENU:=Web Servers/Proxies
  TITLE:=Web management panel for dae
  URL:=https://github.com/senshinya/luci-app-kdae-panel
  DEPENDS:=@x86_64 +kmod-sched-core +kmod-sched-bpf +kmod-veth +kmod-nft-bridge \
	+kmod-xdp-sockets-diag +ca-bundle
  # dae 的可执行文件、配置与 geo 全部由本面板管理。装了本包就装不上官方
  # dae 包，opkg upgrade 也就不会把分支构建盖回官方版本。
  CONFLICTS:=dae
endef

define Package/kdae-panel/description
  kdae-panel 是面向 dae 及其兼容分支的 Web 管理面板：配置编排、版本管理、
  geo 更新、日志浏览。本包同时提供 dae 的 procd 启动脚本，dae 的可执行文件
  由面板下载安装。
endef

define Package/kdae-panel/conffiles
/etc/config/kdae-panel
endef

define Build/Prepare
	mkdir -p $(PKG_BUILD_DIR)
endef

define Build/Configure
endef

define Build/Compile
	$(if $(KDAE_PANEL_BIN),,$(error 请通过 KDAE_PANEL_BIN 指定已交叉编译好的 kdae-panel 二进制))
	$(CP) $(KDAE_PANEL_BIN) $(PKG_BUILD_DIR)/kdae-panel
endef

define Package/kdae-panel/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/kdae-panel $(1)/usr/bin/kdae-panel
	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/kdae-panel.init $(1)/etc/init.d/kdae-panel
	$(INSTALL_BIN) ./files/dae.init $(1)/etc/init.d/dae
	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) ./files/kdae-panel.config $(1)/etc/config/kdae-panel
endef

define Package/kdae-panel/postinst
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] && exit 0
/etc/init.d/kdae-panel enable
exit 0
endef

define Package/kdae-panel/prerm
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] && exit 0
/etc/init.d/kdae-panel stop
/etc/init.d/kdae-panel disable
exit 0
endef

$(eval $(call BuildPackage,kdae-panel))
```

- [ ] **Step 5: 用 shellcheck 验证两个 init 脚本**

Run: `shellcheck -s sh -e SC1091,SC2034 openwrt/kdae-panel/files/kdae-panel.init openwrt/kdae-panel/files/dae.init`
Expected: 无输出（`SC1091` 是 `/etc/rc.common` 不可解析，`SC2034` 是 procd 环境注入的变量，两者都是误报）。
本机没有 shellcheck 时用 `brew install shellcheck` 装，或退而求其次只做语法检查：
`sh -n openwrt/kdae-panel/files/kdae-panel.init && sh -n openwrt/kdae-panel/files/dae.init`（无输出即通过）。

- [ ] **Step 6: 静态检查 Makefile 与 UCI 里的关键约定**

Run:
```bash
grep -q 'CONFLICTS:=dae' openwrt/kdae-panel/Makefile && \
! grep -q 'self-update\|update-check' openwrt/kdae-panel/files/kdae-panel.init && \
grep -q "data_dir '/etc/kdae-panel'" openwrt/kdae-panel/files/kdae-panel.config && \
! grep -q '/var/lib/kdae-panel' openwrt/kdae-panel/files/kdae-panel.init && \
echo OK
```
Expected: `OK`（最后一条确认没有把数据写进 tmpfs）

- [ ] **Step 7: Commit**

```bash
git add openwrt/kdae-panel
git commit -m "feat(openwrt): kdae-panel 后端软件包"
```

---

### Task 9: `luci-app-kdae-panel` OpenWrt 包

**Files:**
- Create: `openwrt/luci-app-kdae-panel/Makefile`
- Create: `openwrt/luci-app-kdae-panel/htdocs/luci-static/resources/view/kdae-panel/panel.js`
- Create: `openwrt/luci-app-kdae-panel/root/usr/share/luci/menu.d/luci-app-kdae-panel.json`
- Create: `openwrt/luci-app-kdae-panel/root/usr/share/rpcd/acl.d/luci-app-kdae-panel.json`

**Interfaces:**
- Consumes: Task 8 的 UCI 节 `kdae-panel.main` 与两个 init 脚本名（`kdae-panel`、`dae`）、`/var/run/kdae-panel/setup-url`。
- Produces: 无（终端 UI）。

- [ ] **Step 1: 写菜单定义**

创建 `openwrt/luci-app-kdae-panel/root/usr/share/luci/menu.d/luci-app-kdae-panel.json`：

```json
{
	"admin/services/kdae-panel": {
		"title": "kdae 面板",
		"order": 60,
		"action": {
			"type": "view",
			"path": "kdae-panel/panel"
		},
		"depends": {
			"acl": [ "luci-app-kdae-panel" ],
			"uci": { "kdae-panel": true }
		}
	}
}
```

- [ ] **Step 2: 写 ACL**

创建 `openwrt/luci-app-kdae-panel/root/usr/share/rpcd/acl.d/luci-app-kdae-panel.json`：

```json
{
	"luci-app-kdae-panel": {
		"description": "Grant access to kdae-panel configuration and service control",
		"read": {
			"uci": [ "kdae-panel" ],
			"file": {
				"/var/run/kdae-panel/setup-url": [ "read" ]
			},
			"ubus": {
				"service": [ "list" ],
				"luci": [ "getInitList" ]
			}
		},
		"write": {
			"uci": [ "kdae-panel" ],
			"ubus": {
				"luci": [ "setInitAction" ]
			}
		}
	}
}
```

- [ ] **Step 3: 写视图**

创建 `openwrt/luci-app-kdae-panel/htdocs/luci-static/resources/view/kdae-panel/panel.js`：

```js
'use strict';
'require view';
'require form';
'require uci';
'require fs';
'require rpc';
'require ui';

var callServiceList = rpc.declare({
	object: 'service',
	method: 'list',
	params: [ 'name' ],
	expect: { '': {} }
});

var callInitAction = rpc.declare({
	object: 'luci',
	method: 'setInitAction',
	params: [ 'name', 'action' ],
	expect: { result: false }
});

// serviceRunning 从 procd 的实例表判断服务是否在跑。
// 查不到实例就是没跑——procd 只在实例被定义后才列出它。
function serviceRunning(reply, name) {
	var service = reply[name] || {};
	var instances = service.instances || {};
	for (var key in instances)
		if (instances[key].running)
			return true;
	return false;
}

function statusBadge(running) {
	return E('span', {
		'class': running ? 'label notice' : 'label warning',
		'style': 'padding:2px 8px;border-radius:3px'
	}, running ? _('运行中') : _('已停止'));
}

// setupURL 读面板启动时写下的一次性初始化链接。
// 文件不在是常态（管理员已创建，面板把它删了），不该当成错误。
function setupURL() {
	return fs.read('/var/run/kdae-panel/setup-url')
		.then(function (content) { return (content || '').trim(); })
		.catch(function () { return ''; });
}

return view.extend({
	load: function () {
		return Promise.all([
			uci.load('kdae-panel'),
			callServiceList('kdae-panel'),
			callServiceList('dae'),
			setupURL()
		]);
	},

	render: function (data) {
		var panelRunning = serviceRunning(data[1] || {}, 'kdae-panel');
		var daeRunning = serviceRunning(data[2] || {}, 'dae');
		var link = data[3];
		var port = uci.get('kdae-panel', 'main', 'listen_port') || '2023';

		// 状态块不做成表单项：CBI 的 option 只为"编辑一个 UCI 值"而生，
		// 硬塞按钮和链接要整个覆写它的 render，一旦 LuCI 改动内部约定就会碎。
		// 直接建 DOM 再和表单拼起来，行为确定得多。
		var statusView = (function () {
			var rows = [
				[ _('kdae 面板'), panelRunning, 'kdae-panel' ],
				[ _('dae'), daeRunning, 'dae' ]
			].map(function (row) {
				return E('div', { 'style': 'display:flex;align-items:center;gap:12px;margin-bottom:8px' }, [
					E('strong', { 'style': 'min-width:8em' }, row[0]),
					statusBadge(row[1]),
					E('button', {
						'class': 'cbi-button cbi-button-apply',
						'click': ui.createHandlerFn(null, function (name) {
							return callInitAction(name, 'restart').then(function () {
								ui.addNotification(null, E('p', _('已请求重启 %s').format(name)), 'info');
							});
						}, row[2])
					}, _('重启')),
					E('button', {
						'class': 'cbi-button cbi-button-reset',
						'click': ui.createHandlerFn(null, function (name) {
							return callInitAction(name, 'stop').then(function () {
								ui.addNotification(null, E('p', _('已停止 %s').format(name)), 'info');
							});
						}, row[2])
					}, _('停止')),
					E('button', {
						'class': 'cbi-button cbi-button-action',
						'click': ui.createHandlerFn(null, function (name) {
							return callInitAction(name, 'start').then(function () {
								ui.addNotification(null, E('p', _('已启动 %s').format(name)), 'info');
							});
						}, row[2])
					}, _('启动'))
				]);
			});

			rows.push(E('div', { 'style': 'margin-top:16px' }, [
				E('button', {
					'class': 'cbi-button cbi-button-action important',
					'click': function () {
						window.open(location.protocol + '//' + location.hostname + ':' + port + '/', '_blank');
					}
				}, _('打开面板'))
			]));

			// 面板拒绝被 iframe 嵌入（CSP frame-ancestors 'none'），因此新标签打开。
			rows.push(E('p', { 'style': 'margin-top:8px' },
				E('em', {}, _('面板在新标签页打开，地址为 %s。').format(
					location.protocol + '//' + location.hostname + ':' + port + '/'))));

			if (link) {
				rows.push(E('div', {
					'class': 'alert-message warning',
					'style': 'margin-top:16px'
				}, [
					E('p', {}, E('strong', {}, _('尚未创建管理员'))),
					E('p', {}, _('打开下面的一次性链接完成初始化，创建成功后链接立即失效：')),
					E('p', {}, E('a', { 'href': link, 'target': '_blank' }, link))
				]));
			}

			return E('div', { 'class': 'cbi-section' }, [
				E('h3', {}, _('服务状态')),
				E('div', {}, rows)
			]);
		})();

		var m = new form.Map('kdae-panel', _('kdae 面板'),
			_('面板负责 dae 的配置编排、版本管理与日志；dae 的可执行文件由面板下载安装，不经 opkg。'));

		var s = m.section(form.NamedSection, 'main', 'kdae-panel', _('设置'));

		var o = s.option(form.Flag, 'enabled', _('开机自启'),
			_('关闭后面板不会随系统启动，已经在跑的实例不受影响。'));
		o.default = '1';
		o.rmempty = false;

		o = s.option(form.Value, 'listen_addr', _('监听地址'),
			_('0.0.0.0 表示接受本机与局域网连接。'));
		o.datatype = 'ipaddr';
		o.default = '0.0.0.0';

		o = s.option(form.Value, 'listen_port', _('监听端口'));
		o.datatype = 'port';
		o.default = '2023';

		o = s.option(form.Value, 'data_dir', _('数据目录'),
			_('数据库、配置备份、状态文件与 dae 本地版本库的位置。' +
			  '不要改到 /var 或 /tmp 下——那里是内存文件系统，重启即空。'));
		o.default = '/etc/kdae-panel';

		o = s.option(form.Value, 'dae_binary', _('dae 可执行文件'),
			_('面板与 dae 的启动脚本读的是同一个值，不要单独修改启动脚本。'));
		o.default = '/usr/bin/dae';

		o = s.option(form.Value, 'dae_config', _('dae 配置文件'),
			_('geo 数据也放在这个文件所在的目录。'));
		o.default = '/etc/dae/config.dae';

		o = s.option(form.Flag, 'enable_dae_install', _('由面板管理 dae 版本'),
			_('允许面板下载、安装、切换与回滚 dae。关闭后版本管理页不可用。'));
		o.default = '1';

		o = s.option(form.Flag, 'enable_geo_update', _('由面板管理 geo 数据'),
			_('允许一键更新 geoip.dat / geosite.dat，更新只触发 dae reload 不重启。'));
		o.default = '1';

		o = s.option(form.Value, 'trusted_proxies', _('可信代理'),
			_('可以转发客户端地址和协议的代理 CIDR，逗号分隔。'));
		o.default = '127.0.0.0/8,::1/128';

		o = s.option(form.Value, 'session_ttl', _('会话有效期'),
			_('形如 12h、30m。'));
		o.default = '12h';

		o = s.option(form.Flag, 'secure_cookie', _('Cookie 仅 HTTPS'),
			_('通过 HTTPS 反向代理访问时打开。'));
		o.default = '0';

		// 配置全部通过命令行参数传给面板，改完必须重启才生效。
		// 重启不在这里做：init 脚本的 service_triggers 里注册了
		// procd_add_reload_trigger "kdae-panel"，LuCI 的「保存并应用」提交
		// UCI 后 procd 会自己触发 reload，而 reload_service 就是 restart。
		return m.render().then(function (formView) {
			return E([], [ statusView, formView ]);
		});
	}
});
```

- [ ] **Step 4: 写包 Makefile**

创建 `openwrt/luci-app-kdae-panel/Makefile`：

```make
include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-kdae-panel
PKG_VERSION:=1.0.0
PKG_RELEASE:=1

PKG_LICENSE:=AGPL-3.0-only
PKG_MAINTAINER:=senshinya

LUCI_TITLE:=LuCI support for kdae-panel
LUCI_DESCRIPTION:=LuCI 入口与配置页：服务状态、一次性初始化链接、UCI 设置。
LUCI_DEPENDS:=+kdae-panel
LUCI_PKGARCH:=all

include $(TOPDIR)/feeds/luci/luci.mk

# call BuildPackage - OpenWrt buildroot signature
```

- [ ] **Step 5: 校验 JSON 与 JS 语法**

Run:
```bash
for f in openwrt/luci-app-kdae-panel/root/usr/share/luci/menu.d/*.json \
         openwrt/luci-app-kdae-panel/root/usr/share/rpcd/acl.d/*.json; do
  node -e "JSON.parse(require('fs').readFileSync('$f','utf8'))" && echo "OK $f"
done
node --check openwrt/luci-app-kdae-panel/htdocs/luci-static/resources/view/kdae-panel/panel.js && echo "OK panel.js"
```
Expected: 每个文件一行 `OK`

- [ ] **Step 6: Commit**

```bash
git add openwrt/luci-app-kdae-panel
git commit -m "feat(openwrt): luci-app-kdae-panel 界面软件包"
```

---

### Task 10: CI 交叉编译与 SDK 打包

**Files:**
- Create: `.github/workflows/openwrt.yml`

**Interfaces:**
- Consumes: Task 8 的 `KDAE_PANEL_BIN` 构建变量与包名。
- Produces: artifact `kdae-panel-ipk-x86_64`，release 事件时把 ipk 附到 Release。

- [ ] **Step 1: 写 workflow**

创建 `.github/workflows/openwrt.yml`：

```yaml
name: OpenWrt ipk

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  release:
    types: [published]
  workflow_dispatch:

env:
  SDK_URL: https://downloads.immortalwrt.org/releases/24.10.4/targets/x86/64/immortalwrt-sdk-24.10.4-x86-64_gcc-13.3.0_musl.Linux-x86_64.tar.zst

jobs:
  ipk:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # 与 ci.yml 钉同一个版本：两条流水线用不同 Go 编出来的二进制会有差异，
      # 而 ci.yml 已经用 git diff --exit-code 锁死了嵌入资源，版本漂移只会制造噪声。
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25.12"

      - uses: actions/setup-node@v4
        with:
          node-version: "22"
          cache: npm
          cache-dependency-path: web/package-lock.json

      - name: 构建前端
        run: |
          npm ci --prefix web
          npm run build --prefix web

      # 纯 Go 依赖（modernc sqlite），CGO_ENABLED=0 出来的静态二进制在 musl 上直接可跑。
      # 不在 SDK 里从源码编译：本项目要求 Go 1.25，24.10 feed 里的 golang 到不了。
      - name: 交叉编译面板
        run: |
          version="${GITHUB_REF_NAME}"
          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
            -trimpath \
            -ldflags "-s -w -X main.version=${version}" \
            -o "${GITHUB_WORKSPACE}/kdae-panel" \
            ./cmd/kdae-panel
          file "${GITHUB_WORKSPACE}/kdae-panel"

      - name: 安装 SDK 依赖
        run: |
          sudo apt-get update
          sudo apt-get install -y build-essential libncurses-dev zlib1g-dev gawk git \
            gettext libssl-dev xsltproc rsync wget unzip python3 zstd

      - name: 拉取 immortalwrt SDK
        run: |
          mkdir -p "${GITHUB_WORKSPACE}/sdk"
          wget -q "${SDK_URL}" -O sdk.tar.zst
          tar --zstd -xf sdk.tar.zst -C "${GITHUB_WORKSPACE}/sdk" --strip-components=1

      - name: 准备 feeds 与软件包
        working-directory: ${{ github.workspace }}/sdk
        run: |
          ./scripts/feeds update base luci packages
          # 只装 luci-base：包 Makefile 需要 feeds/luci/luci.mk，不需要整套 LuCI 应用。
          ./scripts/feeds install luci-base
          # 复制而不是软链：buildroot 扫描软件包用的 find 默认不跟随符号链接，
          # 软链进去的目录会被整个跳过，症状是 "package/kdae-panel 不存在"。
          cp -r "${GITHUB_WORKSPACE}/openwrt" package/kdae

      - name: 编译 ipk
        working-directory: ${{ github.workspace }}/sdk
        run: |
          {
            echo 'CONFIG_PACKAGE_kdae-panel=m'
            echo 'CONFIG_PACKAGE_luci-app-kdae-panel=m'
          } >> .config
          make defconfig
          # 既导出环境变量又作为 make 变量传入：OpenWrt 的构建会层层调用子 make，
          # 只靠命令行变量在某些子调用里可能丢失。
          export KDAE_PANEL_BIN="${GITHUB_WORKSPACE}/kdae-panel"
          make package/kdae-panel/compile \
               package/luci-app-kdae-panel/compile \
               KDAE_PANEL_BIN="${KDAE_PANEL_BIN}" \
               -j"$(nproc)" V=s

      # 装错依赖的症状是在路由器上 opkg 报缺包，离原因很远；在这里直接断言。
      - name: 校验产物
        working-directory: ${{ github.workspace }}/sdk
        run: |
          mkdir -p "${GITHUB_WORKSPACE}/ipk"
          find bin/packages -name '*kdae*.ipk' -exec cp {} "${GITHUB_WORKSPACE}/ipk/" \;
          ls -l "${GITHUB_WORKSPACE}/ipk"
          backend=$(ls "${GITHUB_WORKSPACE}/ipk"/kdae-panel_*.ipk)
          # ipk 是 gzip 的 tar，里面装着 control.tar.gz 与 data.tar.gz。
          # 成员名带不带 ./ 前缀各版本不一，因此解到临时目录再看，别猜。
          workdir=$(mktemp -d)
          tar -xzf "$backend" -C "$workdir"
          tar -xzf "$workdir"/control.tar.gz -C "$workdir"
          cat "$workdir/control"
          grep -q '^Conflicts:.*\bdae\b' "$workdir/control" \
            || { echo "control 缺少 Conflicts: dae"; exit 1; }
          for dep in kmod-sched-core kmod-sched-bpf kmod-veth kmod-nft-bridge \
                     kmod-xdp-sockets-diag ca-bundle; do
            grep -q "$dep" "$workdir/control" || { echo "control 缺少依赖 $dep"; exit 1; }
          done
          tar -tzf "$workdir"/data.tar.gz > "$workdir/files"
          for path in usr/bin/kdae-panel etc/init.d/kdae-panel etc/init.d/dae \
                      etc/config/kdae-panel; do
            grep -q "$path\$" "$workdir/files" || { echo "包内缺少 $path"; exit 1; }
          done
          ls "${GITHUB_WORKSPACE}/ipk"/luci-app-kdae-panel_*.ipk

      - uses: actions/upload-artifact@v4
        with:
          name: kdae-panel-ipk-x86_64
          path: ipk/*.ipk

      - name: 附到 Release
        if: github.event_name == 'release'
        uses: softprops/action-gh-release@v2
        with:
          files: ipk/*.ipk
```

- [ ] **Step 2: 本机校验 YAML 语法**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/openwrt.yml')); print('OK')"`
Expected: `OK`

- [ ] **Step 3: Commit 并推分支触发 CI**

```bash
git add .github/workflows/openwrt.yml
git commit -m "ci: 交叉编译并用 immortalwrt SDK 打出 ipk"
```

- [ ] **Step 4: 确认 CI 通过**

Run: `gh run list --workflow=openwrt.yml --limit 1` 然后 `gh run watch`
Expected: 状态 `success`，artifact 里有两个 ipk。
失败时按日志修 `openwrt/` 下的 Makefile 或 workflow 后重跑——**不要**为了让 CI 过而放宽"校验产物"步骤里的断言。

---

### Task 11: 文档

**Files:**
- Create: `docs/openwrt.md`
- Modify: `README.md`
- Modify: `docs/deployment.md`
- Modify: `docs/architecture.md`
- Modify: `SECURITY.md`

**Interfaces:**
- Consumes: 前 12 个任务的全部结论。
- Produces: 无。

- [ ] **Step 1: 写 `docs/openwrt.md`**

内容必须覆盖以下小节，每节都写实而不是列标题：

1. **适用范围**：immortalwrt 24.10.4 x86/64；LuCI 应用名 `luci-app-kdae-panel`，后端包 `kdae-panel`。
2. **与官方 dae 包的关系**：本包 `CONFLICTS:=dae`，装了它就装不上官方 `dae`；这是为了让 `opkg upgrade` 不会把你自己的分支构建盖回官方版本。dae 需要的内核模块仍由 opkg 安装，作为本包的依赖自动拉取。
3. **安装**：
   ```sh
   opkg update
   opkg install ./kdae-panel_1.0.0-1_x86_64.ipk ./luci-app-kdae-panel_1.0.0-1_all.ipk
   /etc/init.d/kdae-panel start
   ```
   已装官方 dae 时先 `opkg remove dae dae-geoip dae-geosite`（配置 `/etc/dae` 会保留）。
4. **首次访问**：LuCI → 服务 → kdae 面板 → 状态，页面上直接给出一次性初始化链接；也可 `cat /var/run/kdae-panel/setup-url` 或 `logread -e kdae-panel`。
5. **安装 dae**：面板的版本管理页可在官方发布与 kdae 分支 CI 构建之间选择；安装完成后**不会自动启动**，先写配置再启动。启动前 `/etc/init.d/dae enable` 才会开机自启。
6. **UCI 配置项表**（节名 `kdae-panel.main`，改完执行 `/etc/init.d/kdae-panel restart`）：

   | 选项 | 默认 | 说明 |
   |---|---|---|
   | `enabled` | `1` | 开机自启 |
   | `listen_addr` | `0.0.0.0` | 监听地址，默认接受本机与局域网连接 |
   | `listen_port` | `2023` | 监听端口 |
   | `data_dir` | `/etc/kdae-panel` | 数据库、备份、状态文件与 dae 本地版本库的位置 |
   | `dae_binary` | `/usr/bin/dae` | dae 可执行文件；面板与 `/etc/init.d/dae` 读同一个值 |
   | `dae_config` | `/etc/dae/config.dae` | dae 入口配置，geo 数据也放在它所在的目录 |
   | `service_name` | `dae` | init 脚本名 |
   | `enable_dae_install` | `1` | 由面板下载、安装、切换与回滚 dae |
   | `enable_geo_update` | `1` | 由面板一键更新 geo 数据 |
   | `trusted_proxies` | `127.0.0.0/8,::1/128` | 可信代理 CIDR，逗号分隔 |
   | `session_ttl` | `12h` | 会话有效期 |
   | `secure_cookie` | `0` | Cookie 是否仅 HTTPS |

   三条必须写进正文的说明：
   - `data_dir` **不要改到 `/var` 或 `/tmp` 下**——OpenWrt 上 `/var` 是 `/tmp` 的软链，即内存文件系统，重启即空，数据库、管理员账户和 dae 本地版本库会全部丢失。
   - **没有自升级与版本检查的开关**，因为 procd 后端下面板压根不注册这两项能力。原因要写清：
     它们都指向上游 `tuoro/kdae-panel`，那里的发布二进制不含 procd 后端；升级一次，面板就会以
     root 把自己换成一个只会调 `systemctl` 的程序，重启即不可用。面板设置页的「允许一键升级」
     开关在本部署里恒为置灰，「面板更新」卡片不会给出可用版本——这是预期行为，不是故障。
     **升级面板请安装新的 ipk。**
7. **升级与卸载**：升级装新 ipk 即可，`/etc/config/kdae-panel` 是 conffile 不会被覆盖；`opkg remove kdae-panel` 不会删 `/etc/kdae-panel` 与 `/etc/dae`，要连数据清掉需手工 `rm -rf`。
8. **日志功能的实际能力**（必须写，否则用户会以为面板日志坏了）：面板的日志页读的是
   OpenWrt 的系统日志环形缓冲区，**不是磁盘上的日志文件**。默认 `log_size` 只有 64 KiB，
   dae 在 `info` 级别下每条连接都记一行，缓冲区可能只装得下几分钟；重启路由器后全部清空。
   与 systemd 部署的 journald（可持久化、可按单元精确过滤、可翻很久以前）相比这是实质降级。
   缓解办法：
   ```sh
   uci set system.@system[0].log_size='256'   # 单位 KiB
   uci commit system
   /etc/init.d/log restart
   ```
   或把 dae 的 `log_level` 调到 `warn` 减少噪声（配置页的 global 段里改）。
   日志页的搜索框是在已取回的那几百条里做客户端过滤，缓冲区里没有的内容搜不出来。

9. **排障**：
   ```sh
   logread -e kdae-panel
   logread -e dae
   ubus call service list '{"name":"dae"}'
   /etc/init.d/dae enabled; echo $?
   curl http://127.0.0.1:2023/api/v1/health   # backend 字段应为 procd
   mount | grep bpf                            # dae 需要 bpffs
   ```
10. **真机验证清单**（安装后照做一遍）：装 ipk → LuCI 出现菜单 → 启动面板 → 打开一次性链接创建管理员 → 版本管理页首次安装 dae → 配置页写规则 → 启动 dae → 日志页有内容 → geo 更新一次 → 重启路由后面板与 dae 状态正确、管理员仍能登录。

- [ ] **Step 2: 改 `README.md`**

在「一键部署」之后插入一节：

```markdown
## OpenWrt / ImmortalWrt

immortalwrt 24.10.4（x86/64）上以 ipk 部署，附带 LuCI 入口与配置页。dae 的可执行文件、
配置与 geo 全部由面板管理，不经 opkg——这样 `opkg upgrade` 不会把你自己的分支构建盖回
官方版本。详见 [docs/openwrt.md](docs/openwrt.md)。
```

并在「功能」列表里把 "systemd 服务状态、启动、停止和重启" 改为 "systemd 与 OpenWrt procd 两套服务后端，自动探测；服务状态、启动、停止和重启"。

- [ ] **Step 3: 改 `docs/deployment.md`**

在「前置条件」里把 "Linux 与 systemd" 改为 "Linux，systemd 或 OpenWrt procd（自动探测，可用 `KDAE_PANEL_SERVICE_BACKEND` 强制）"，并在配置项表格里补三行：

```
| `KDAE_PANEL_SERVICE_BACKEND` | `auto` | 服务后端：`auto`（存在 `/sbin/procd` 即 procd）、`systemd`、`procd` |
| `KDAE_PANEL_LOCK_SELF_UPDATE_PREFERENCE` | `false` | 把自升级开关固定为部署方给出的值，面板内不可修改 |
```

在「面板一键自升级」小节末尾补一段说明锁定语义：部署方打开锁定后，界面上的开关只读，`/var/lib/kdae-panel/self-update.json` 既不读也不写。

- [ ] **Step 4: 改 `docs/architecture.md`**

新增一小节「服务后端」：说明 `internal/host` 暴露 `Manager` 接口、两套实现的分工（systemd 用 `systemctl show` + `journalctl --output json`；procd 用 `ubus call service list` + init 脚本 + `logread`），后端由 `/sbin/procd` 是否存在自动探测、可显式覆盖，选中的结果记进启动日志并由 `/api/v1/health` 的 `backend` 字段暴露。同时说明 `daeinstall.unitProvisioner` 的存在理由：systemd 下服务定义由面板写、卸载时删；procd 下它属于 `kdae-panel` 软件包，面板只校验。

- [ ] **Step 5: 改 `SECURITY.md`**

这一步不能省。`SECURITY.md` 现在的「安全边界」列着"systemd 服务采用能力白名单和文件系统保护"——**OpenWrt 部署完全没有这一层**，procd 不提供 `ProtectSystem` / `CapabilityBoundingSet` / `NoNewPrivileges` 的等价物。文档继续这么写，等于对 OpenWrt 用户宣称一份并不存在的防护。

在「安全边界」的项目符号列表里，把

```markdown
- systemd 服务采用能力白名单和文件系统保护；
```

改为

```markdown
- systemd 部署的服务单元采用能力白名单和文件系统保护；**OpenWrt/procd 部署没有等价机制**，
  面板以完整 root 权限运行，详见下方「OpenWrt 部署的安全差异」；
```

在「已知边界」之前新增一节：

```markdown
## OpenWrt 部署的安全差异

`luci-app-kdae-panel` 以 ipk 部署在 OpenWrt/ImmortalWrt 上，procd 没有 systemd 那套沙箱原语。
差异必须说清，而不是让人以为 systemd 单元里的防护到处都在：

- **没有文件系统保护**：不存在 `ProtectSystem=strict` / `ReadWritePaths` 的等价物。面板能写整个
  文件系统，而不是被限制在 `/etc/dae` 与数据目录。上游文档里"把某目录加入 ReadWritePaths"
  一类的排障步骤在这里不适用，对应的错误提示也已去掉。
- **没有能力白名单**：面板持有完整 root 权限，而不是 `CAP_KILL` + `CAP_NET_ADMIN` 两项。
- **没有 `ProtectHome`**：`/root/.local/share/dae` 对面板可见，geo 缺失提示因此会直说"未找到"。
- **默认开启的能力更多**：`enable_dae_install` 与 `enable_geo_update` 都默认为 `1`。前者与 systemd
  发行单元一致；后者上游默认关闭，理由是"给部署新增一条常态化的联网取字节 → 以 root 写系统目录
  的路径"。这里默认打开是因为 dae 的 geo 数据在本部署中也由面板管理，而 `enable_dae_install=1`
  已经引入了同一条路径且权限更大——再关掉 geo 只是让用户手工放文件，并不减少攻击面。
  **但如果你把 `enable_dae_install` 关掉，就该一并把 `enable_geo_update` 关掉**，否则会在一个
  没有沙箱的环境里单独留下那条路径。
- **数据目录是 `/etc/kdae-panel` 而非 `/var/lib/kdae-panel`**：OpenWrt 的 `/var` 是内存文件系统。
  「部署要求」里"不要让非管理员写入"的目录，在这里指 `/etc/kdae-panel` 与 `/etc/config/kdae-panel`。

结论没有变：面板是高权限系统管理服务，只应在可信内网使用，不要把端口暴露到公网。
但在 OpenWrt 上，"面板缺陷升级为任意代码执行"的门槛比 systemd 部署更低。
```

同时把「部署要求」里的

```markdown
- 不要让非管理员写入 `/etc/kdae-panel`、`/var/lib/kdae-panel`、dae 配置或面板二进制；
```

改为

```markdown
- 不要让非管理员写入 `/etc/kdae-panel`、`/var/lib/kdae-panel`（OpenWrt 上是 `/etc/kdae-panel`）、
  dae 配置或面板二进制；
```

- [ ] **Step 6: 全量验证**

Run: `go test ./... && go vet ./... && npm run typecheck --prefix web && npm test --prefix web`
Expected: 全部 PASS

- [ ] **Step 7: Commit**

```bash
git add docs README.md SECURITY.md
git commit -m "docs: 补上 OpenWrt 部署、服务后端与安全差异说明"
```
