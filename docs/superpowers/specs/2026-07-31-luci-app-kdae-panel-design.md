# luci-app-kdae-panel 设计

把 kdae-panel 改造为可在 immortalwrt 24.10.4（x86/64）上以 ipk 部署的 LuCI 应用。

## 目标与非目标

**目标**

- 面板在 procd 系统上完整可用：服务状态、启停、日志、配置编排、dae 版本管理、geo 更新。
- dae 完全由面板管理（下载、安装、切换、回滚、卸载），不经 opkg，使 kdae 分支 CI 构建不会被 `opkg upgrade` 覆盖。
- 提供 LuCI 菜单入口与配置页，配置走 UCI。
- CI 产出可直接 `opkg install` 的 x86_64 ipk。

**非目标**

- 不用 LuCI JS 重写面板界面。Vue 前端与 Go HTTP 层原样保留。
- 不支持 x86_64 以外的架构（本次只针对用户机器；包 Makefile 本身与架构无关，扩展只需在 CI 矩阵加 SDK）。
- 不做 iframe 内嵌，面板仍以新标签打开。
- systemd 分支的行为不做任何改变。

## 约束

- immortalwrt 24.10.4 用 opkg / ipk（apk 是 25.x 起），LuCI 是客户端 JS 框架（`htdocs/luci-static/resources/view/*.js` + `menu.d` + `acl.d`），不是 Lua CBI。
- immortalwrt 24.10 feed 里的 golang 到不了 go.mod 要求的 1.25.0，因此 Go 二进制在 CI 用官方 Go 交叉编译，SDK 只负责打包。
- 依赖是纯 Go（`modernc.org/sqlite`，无 cgo），`CGO_ENABLED=0` 静态二进制在 musl 上可直接运行。
- dae 需要的内核模块只能由 opkg 安装，面板代替不了。

## 一、包设计

一个仓库产出两个 ipk。

### `kdae-panel`

| 文件 | 说明 |
|---|---|
| `/usr/bin/kdae-panel` | 预编译静态二进制，前端已嵌入 |
| `/etc/init.d/kdae-panel` | 面板的 procd 脚本 |
| `/etc/init.d/dae` | dae 的 procd 脚本（**由本包提供，不是面板运行时生成**） |
| `/etc/config/kdae-panel` | UCI 默认配置 |
| `/etc/dae/`（空目录） | dae 配置与 geo 的落地目录 |

- `DEPENDS:=+kmod-sched-core +kmod-sched-bpf +kmod-veth +kmod-nft-bridge +kmod-xdp-sockets-diag +ca-bundle`
  （`ca-bundle` 必需：面板要通过 HTTPS 访问 GitHub 拉 dae 版本清单与 geo 数据。）
- `CONFLICTS:=dae` — 装了本包就装不上官方 `dae` 包，`opkg upgrade` 不会把分支构建盖回官方版本。
- postinst：`/etc/init.d/kdae-panel enable`，不自动 start（首次要先看 UCI 配置）。
- prerm：`/etc/init.d/kdae-panel stop; /etc/init.d/kdae-panel disable`。
- 只有 `/etc/config/kdae-panel` 列进 `conffiles`。两个 init 脚本**刻意不是** conffile：面板会解析 `/etc/init.d/dae` 来确定 dae 的实际启动路径，脚本与面板必须同版本演进；设成 conffile 会让老脚本永久留在机器上，而面板的解析逻辑已经往前走了。

### `luci-app-kdae-panel`

- `DEPENDS:=+kdae-panel +luci-base`
- `htdocs/luci-static/resources/view/kdae-panel/panel.js`
- `root/usr/share/luci/menu.d/luci-app-kdae-panel.json`
- `root/usr/share/rpcd/acl.d/luci-app-kdae-panel.json`
- 界面文案直接写中文源字符串，不提供 `.po`，因此打包无需 `po2lmo`。

## 二、Go 侧改造

### 2.1 `internal/host` 抽出后端接口

```
internal/host/host.go       Manager 接口 + Options + 后端探测 + 构造入口
internal/host/systemd.go    现 manager.go 改名，实现逐字不变（类型改名 systemdManager）
internal/host/procd.go      新增
internal/host/interfaces.go 接收者从 *Manager 改为 interfaceLister，两个后端各内嵌一份
```

`Manager` 从结构体变为接口，方法集与 `app.HostService` 一致：

```go
type Manager interface {
    Status(ctx context.Context) (Status, error)
    Action(ctx context.Context, action Action) error
    RestartSelf(ctx context.Context) error
    Logs(ctx context.Context, limit int) ([]LogEntry, error)
    Interfaces(ctx context.Context) ([]NetworkInterface, error)
}
```

`Interfaces` 的实现是纯 `net.Interfaces()`，与后端无关。`Manager` 变成接口后它不能再挂在具体类型上，因此抽成零字段的 `interfaceLister` 结构体，`systemdManager` 与 `procdManager` 各内嵌它一次，方法集自动带上，逻辑一行不改。

后端选择由 `Backend` 显式表达，不藏在探测函数里：

```go
type Backend string
const (
    BackendAuto    Backend = "auto"    // 存在 /sbin/procd 即 procd，否则 systemd
    BackendSystemd Backend = "systemd"
    BackendProcd   Backend = "procd"
)
```

构造入口从三个位置参数换成 Options，否则 procd 后端要塞进 systemctl/journalctl 两个用不上的参数：

```go
type Options struct {
    Backend     Backend
    ServiceName string
    DaeBinary   string // ExecStartPath 回退链的最后一级
    Systemctl   string // 仅 systemd 后端
    Journalctl  string // 仅 systemd 后端
    Runner      command.Runner
    Timeout     time.Duration
}
func New(opts Options) (Manager, error)
```

`app.New` 里那一处 `host.NewManager(...)` 改为 `host.New(host.Options{...})`；现有测试用的 `NewManagerWithRunner` 由 `Options.Runner` / `Options.Timeout` 承接。

`app.Config` 增加 `ServiceBackend Backend`，可由 `--service-backend` / `KDAE_PANEL_SERVICE_BACKEND` 覆盖，默认 `auto`。选中的后端记进启动日志，并通过 `/api/v1/health` 暴露，让"面板以为自己在 systemd 上"这类错配一眼可见。

### 2.2 procd 后端

**Status** 以 ubus 为主、`pidof` 为辅。参考实现只用 `pidof` 有个硬伤：dae 停止时拿不到 `ExecStartPath`，而 `daeinstall.Provision` 正是靠这个字段判断机器上有没有 dae——会把每台停着 dae 的机器误判成"未安装"，从而允许一次无备份的覆盖安装。

```
ubus call service list '{"name":"dae"}'
→ {"dae":{"instances":{"instance1":{"running":true,"pid":1234,
     "command":["/usr/bin/dae","run","-c","/etc/dae/config.dae"]}}}}
```

字段映射：

| Status 字段 | procd 来源 |
|---|---|
| `Name` | 配置的服务名 |
| `ActiveState` / `SubState` | ubus `running` → `active`/`running`，否则 `inactive`/`dead` |
| `MainPID` | ubus `pid`，回退 `pidof <name>` 首个 |
| `ExecStartPath` | ubus `command[0]`；ubus 无实例时回退到面板配置的 `dae_binary` |
| `UnitFileState` | `/etc/init.d/dae enabled` 退出码 → `enabled` / `disabled`；脚本不存在 → 空 |
| `UnitPath` | `/etc/init.d/<serviceName>` |
| `MemoryBytes` | `/proc/<pid>/status` 的 `VmRSS` |
| `CPUUsageNanoseconds` | `/proc/<pid>/stat` 的 utime+stime × 10ms |
| `Restarts` | 恒为 0；字段带 `omitempty`，不进 JSON，前端自然不展示 |
| `Environment` | `/proc/<pid>/environ`（NUL 分隔）；进程没跑时为空 |
| `Description` | 固定 `"procd service <name>"` |

`ExecStartPath` 的回退是刻意的：`daeinstall` 用它做替换目标，返回空会让首次安装误判，返回错误路径会让替换打到别处。回退到面板配置的 `dae_binary` 是安全的——`/etc/init.d/dae` 与面板的 `--dae-binary` 读的是**同一份 UCI**，两者不可能分叉，因此不需要解析 init 脚本，也就不存在"脚本被改过而面板不知道"这一类分歧。

`Environment` 只在进程运行时才有。这不是缺陷：进程没跑时 geo 的搜索顺序退化为"配置文件所在目录优先"，而 init 脚本设的 `DAE_LOCATION_ASSET` 正是配置文件所在目录，两者本就重合。

`/proc` 的读取路径抽成包级变量 `procRoot`（默认 `/proc`），测试可指向临时目录。

**Action**：`/etc/init.d/<name> start|stop|restart|enable|disable`；`daemon-reload` 直接返回 nil（procd 每次执行 init 脚本都会重读，没有等价动作）。

**RestartSelf**：

```go
exec: sh -c "setsid /etc/init.d/kdae-panel restart >/dev/null 2>&1 &"
```

`setsid` 不能省。面板执行的 init 脚本是自己的子进程，procd 停掉面板实例时会一并杀掉它，重启命令会先于重启本身死掉。`setsid` 让它脱离面板的会话与进程组。

**Logs**：`logread -e <serviceName>`。init 脚本设了 `procd_set_param stdout 1` / `stderr 1`，dae 的日志经 procd 进 syslog。解析容忍两种 logread 前缀格式：

```
Fri Jul 31 01:00:00 2026 daemon.info dae[1234]: level=info msg="..."
2026-07-31 01:00:00 host dae.info dae[1234]: message
```

消息体若形如 dae 的 `level=… msg="…"`，进一步解析出 `level` 与 `msg`，映射到 `Priority`；解析不出就整行作为 message、级别 info。不实现"读 `/var/log/dae/dae.log`"——init 脚本由本包提供，日志去向是我们自己定的，多一条猜测路径只会在出问题时增加排查面。

### 2.3 `daeinstall` 适配

把"服务单元"这件事从 `Installer` 里抽成接口，systemd 分支的实现是现有代码原样搬家：

```go
// unitProvisioner 描述一个后端如何提供、校验与移除 dae 的服务定义。
type unitProvisioner interface {
    // Path 返回服务定义文件的位置，仅用于回报给界面。
    Path() string
    // Plan 在写任何文件之前判定服务定义能否就位。
    // inPlace 为真表示已存在且可用，不需要写。
    Plan(bundle upstream.Bundle) (content string, inPlace bool, err error)
    // Commit 落地服务定义；inPlace 时不做任何事。
    Commit(ctx context.Context, content string, inPlace bool) error
    // WritableDirs 是首次安装需要写入、因而必须预检的目录。
    WritableDirs() []string
    // RemovablePaths 是卸载时应当一并删除的服务定义文件；procd 下为空。
    RemovablePaths() []string
    // VerifyRemovable 在卸载前确认服务定义确实归面板管理；procd 下无事可做。
    VerifyRemovable(status host.Status, binaryPath string) error
    // Detect 判定机器上是否已有 dae 服务。
    Detect(ctx context.Context, status host.Status) unitDetection
}

// unitDetection 是"这台机器上已经有 dae 了吗"的判定结果。
type unitDetection struct {
    Installed bool
    // Blocker 非空表示不能首次安装，直接作为拒绝理由。
    Blocker string
    // Notes 是不阻断但用户应当知道的情况。
    Notes []string
}
```

- `systemdUnits`：现 `provision.go` 的 `planUnit` / `render` 原样搬迁（`retargetUnit` / `execStartBinary` / `unitExecStart` 留在 provision.go 供两处共用），`Commit` 写单元后 `daemon-reload`，`RemovablePaths` 返回单元路径，`VerifyRemovable` 承接现 `uninstallTarget` 末段的单元校验。
- `procdUnits`：init 脚本归 ipk，因此
  - `Plan` 退化为校验 `/etc/init.d/dae` 存在且是普通文件；`inPlace` 恒为 true。不再解析脚本内容——脚本与面板同读一份 UCI，不可能对 dae 的位置产生分歧。
  - `Commit` 空操作。
  - `RemovablePaths` 与 `VerifyRemovable` 空操作（脚本属于 ipk，卸载 dae 不该删它，也就没有"删对了没有"可校验）。systemd 现有的单元校验（路径必须是标准位置、必须是普通文件、ExecStart 必须与要删的二进制一致、拒绝 `enabled-runtime`）原样搬进 `systemdUnits.VerifyRemovable`，一条不减。
  - `Installed` 判定：`os.Stat(binaryPath)` 成功即已装。不看 ExecStartPath 是因为 dae 停止时它来自回退链，回退到"面板配置的路径"会恒为真。

提到 systemd 的用户可见文案按后端分支，procd 下不能出现 systemd 词汇：

- `provision.go` 的 "请在 kdae-panel.service 的 ReadWritePaths 中加入该目录" → "面板无法写入 %s：%v"。
- `provision.go` 的 "%s 已存在但 systemd 里没有对应的服务" → "…但 %s 里没有对应的服务定义"。
- `installer.go:258` 的 "dae 尚未作为 systemd 服务安装" → "机器上找不到 dae 的服务定义"。
- `panelupdate.go` 的 "自升级需要在 kdae-panel.service 的 ReadWritePaths 中加入该目录" → 去掉 systemd 部分。

`FirstInstall` 的顺序不变（geo → 种子配置 → 二进制 → 服务定义），只是 procd 下最后一步是空操作。仍然不自动启动 dae。

`uninstall.go` 同样按后端分流：procd 下卸载 = `stop` + `disable` + 删二进制 + 按选项删配置/geo；init 脚本保留。

### 2.4 `geodata`

- init 脚本设 `DAE_LOCATION_ASSET=/etc/dae`，geo 落在面板已可写的配置目录，无需放宽任何东西。
- `SandboxHiddenDir`（`/root/.local/share/dae`）在 procd 下不存在沙箱遮挡问题。`MissingWarning` 不需要从上层透传后端标志：它自己探测 `/root` 是否因权限而读不到（`ProtectHome=true` 正是这个症状），读不到就留余地，读得到就直说"未找到"。这比透传布尔更准——systemd 部署若没开 ProtectHome，同样该直说。签名不变。
- `systemDirs` 不变（`/usr/local/share/dae`、`/usr/share/dae` 在 OpenWrt 上同样是 dae 的搜索路径）。

### 2.5 `panelupdate`

默认关闭（UCI `enable_self_update=0`），因为包由 opkg 管理，自升级替换 `/usr/bin/kdae-panel` 后 opkg 的文件账本会与实际不符。能力本身保留可用，`RestartSelf` 走 2.2 的 procd 实现。文档说明推荐路径是装新 ipk。

**必须一并修掉的双真相源。** `panelupdate.New()` 里的 `loadPreference()` 会用数据目录下的 `self-update.json` 覆盖命令行传入的初始值。上游这么设计是对的：systemd 部署下用户只有 env 文件可改，让界面的选择赢，省得为一个开关去 SSH。但到了 LuCI 部署，同一个布尔有了 UCI 与偏好文件两个真相源，而用户看得见的那个（LuCI）反而不生效——这是移植引进的缺陷，不是上游的。

修法是让部署方能显式锁定这一项：

- `panelupdate.Options` 增加 `PreferenceLocked bool`；`app.Config` 增加 `LockSelfUpdatePreference bool`，由 `--lock-self-update-preference` / `KDAE_PANEL_LOCK_SELF_UPDATE_PREFERENCE` 控制，`/etc/init.d/kdae-panel` 恒传该标志。
- 锁定时 `New()` 跳过 `loadPreference()`，`SetEnabled` 直接返回错误而不写盘，偏好文件既不读也不写。
- `panelupdate.Status` 增加 `Locked bool` 与 `LockedReason string`；`PUT /api/v1/panel/update/preference` 在锁定时返回 `409 self_update_preference_locked`。
- `web/src/views/SettingsView.vue` 的自升级开关在 `locked` 时置灰，提示改为「该项由 LuCI → 服务 → kdae 面板 → 设置 管理」。`web/src/types/api.ts` 同步字段。
- systemd 部署不传这个标志，锁定为假，行为逐字不变。

这样"你在哪儿看到开关，改它就生效"才真正成立：能改的地方一处，改不了的地方明确说改不了，而不是改完被悄悄覆盖。

面板自身的新版本检查（`disable_update_check`）在本包里默认**关闭检查**（`disable_update_check=1`）。它读的是上游 `tuoro/kdae-panel` 的 releases/latest，与本 ipk 的版本线不是一回事，提示只会误导。想跟踪上游时可在 LuCI 页面打开。

Go module 路径保持 `github.com/tuoro/kdae-panel` 不改：重命名要动全部 import，而唯一的收益是让上面两个功能指向本仓库——这两个功能在本部署里都默认关闭。

### 2.6 首次访问链接

init 脚本传 `--setup-url-file=/var/run/kdae-panel/setup-url`，并在 `start_service` 前 `mkdir -p -m 0700 /var/run/kdae-panel`。链接同时进 syslog。LuCI 页面读这个文件直接渲染成可点链接。

## 三、init 脚本

### `/etc/init.d/kdae-panel`

`USE_PROCD=1`，`START=99` / `STOP=10`。从 UCI `kdae-panel.main` 读取全部配置，逐项以命令行参数传给二进制（面板的 flag 已覆盖所有配置项，不需要 env 文件）。

```
start_service()
  config_load kdae-panel
  mkdir -p -m 0700 /var/run/kdae-panel
  procd_open_instance
  procd_set_param command /usr/bin/kdae-panel \
      --listen <listen_addr>:<listen_port> \
      --dae-binary <dae_binary> --dae-config <dae_config> \
      --service-name <service_name> \
      --enable-dae-install=<0|1> --enable-geo-update=<0|1> \
      --enable-self-update=<0|1> --lock-self-update-preference \
      --disable-update-check=<0|1> \
      --secure-cookie=<0|1> --trusted-proxies <…> --session-ttl <…> \
      --database <data_dir>/panel.db --backup-dir <data_dir>/backups \
      --schedule-file <data_dir>/schedule.json \
      --install-state-file <data_dir>/dae-install.json \
      --geo-state-file <data_dir>/geo-update.json \
      --geo-schedule-file <data_dir>/geo-schedule.json \
      --panel-backup-file <data_dir>/kdae-panel.previous \
      --setup-url-file /var/run/kdae-panel/setup-url
  procd_set_param respawn
  procd_set_param stdout 1
  procd_set_param stderr 1
  procd_close_instance

service_triggers()
  procd_add_reload_trigger "kdae-panel"
```

**数据目录默认 `/etc/kdae-panel`，不是 systemd 部署用的 `/var/lib/kdae-panel`。** OpenWrt 上 `/var` 是 `/tmp` 的软链，即内存文件系统——把数据库、管理员账户、配置备份和 dae 本地版本库放在那里，一次重启全部消失。`/etc` 在 overlay 上，是这台机器上唯一确定持久的可写位置。目录由 `start_service` 按 UCI 的 `data_dir` 创建，权限 0700（配置里可能有订阅地址与节点凭据）。

反过来，一次性初始化链接**就该**是易失的：它本来每次重启都要重新生成，因此留在 `/var/run/kdae-panel/setup-url`。

### `/etc/init.d/dae`

`USE_PROCD=1`，`START=90` / `STOP=15`。**默认不 enable**——dae 是透明代理，配置没写好就开机自启会切断管理连接，enable 由用户在面板或 LuCI 页面显式执行。

```
start_service()
  [ -x /usr/bin/dae ] || { echo "dae 尚未安装，请先在面板的版本管理页安装" >&2; return 1; }
  mount | grep -q "type bpf" || mount -t bpf bpf /sys/fs/bpf 2>/dev/null
  procd_open_instance
  procd_set_param command /usr/bin/dae run -c /etc/dae/config.dae
  procd_set_param env DAE_LOCATION_ASSET=/etc/dae
  procd_set_param respawn
  procd_set_param stdout 1
  procd_set_param stderr 1
  procd_set_param limits nofile="1048576 1048576"
  procd_close_instance

reload_service()
  /usr/bin/dae reload   # 无损重载，不重启进程
```

bpffs 挂载是必需的：dae 要 pin eBPF map，`/sys/fs/bpf` 未挂载时启动失败。

两个脚本都从 `config_load kdae-panel` 读 `dae_binary` 与 `dae_config`，与面板的命令行参数同源。这就是"面板不解析 init 脚本"的前提：UCI 是唯一真相源，脚本与面板不可能对 dae 的位置产生分歧。

## 四、LuCI 页面

菜单：`admin/services/kdae-panel`，标题「kdae 面板」，`order 60`。

单个 `view.js`：上半是状态块，下半是设置表单。状态块**不做成 CBI 表单项**——`form.DummyValue` 是为"展示一个 UCI 值"设计的，往里塞按钮和链接要整个覆写它的 `render`，LuCI 一改内部约定就会碎。直接建 DOM，再与 `m.render()` 的结果拼起来。

**状态块**

- `kdae-panel` 与 `dae` 两个服务各一行：运行状态徽标（取自 `luci.getInitList` / `service list`）、启动/停止/重启按钮、开机自启开关。
- 「打开面板」按钮：`window.open('http://' + location.hostname + ':' + port + '/')`。端口取自 UCI。
- 若 `/var/run/kdae-panel/setup-url` 可读且非空，显示醒目的一次性初始化链接（渲染为 `<a>`），并说明"创建管理员后此链接自动失效"。

**设置表单**

UCI `kdae-panel.main`（`config kdae-panel 'main'`）：

| 选项 | 默认 | 说明 |
|---|---|---|
| `enabled` | `1` | 开机自启（映射到 init enable/disable） |
| `listen_addr` | `0.0.0.0` | 监听地址 |
| `listen_port` | `2023` | 监听端口 |
| `data_dir` | `/etc/kdae-panel` | 数据库、备份、状态文件与 dae 本地版本库；不可放到 `/var` 或 `/tmp`（内存文件系统） |
| `dae_binary` | `/usr/bin/dae` | dae 可执行文件；面板与 `/etc/init.d/dae` 读同一个值 |
| `dae_config` | `/etc/dae/config.dae` | dae 入口配置 |
| `service_name` | `dae` | init 脚本名 |
| `enable_dae_install` | `1` | 面板管理 dae 版本 |
| `enable_geo_update` | `1` | 面板管理 geo 数据 |
| `enable_self_update` | `0` | 面板自升级（默认关，走 opkg）。此处即唯一真相源，面板设置页的同名开关被锁定为只读 |
| `disable_update_check` | `1` | 关闭新版本检查（检查的是上游仓库，与本 ipk 版本线无关） |
| `trusted_proxies` | `127.0.0.0/8,::1/128` | 可信代理 CIDR |
| `session_ttl` | `12h` | 会话有效期 |
| `secure_cookie` | `0` | Cookie 仅 HTTPS |

保存 → `uci commit` → `/etc/init.d/kdae-panel restart`。

ACL（`acl.d/luci-app-kdae-panel.json`）：读写 uci `kdae-panel`；`file` 读 `/var/run/kdae-panel/setup-url`；`ubus` 的 `service list`；init 控制 `kdae-panel` 与 `dae`。

面板保持 `frame-ancestors 'none'` 与 `X-Frame-Options: DENY`，不做 iframe。

## 五、构建与 CI

`.github/workflows/openwrt.yml`，触发：push main / PR / release published。

1. `actions/setup-go@v5` (1.25) + `actions/setup-node@v4` (22)
2. `make web-install && make web-build`
3. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=<ref>" -o kdae-panel`
4. 下载并解压 `https://downloads.immortalwrt.org/releases/24.10.4/targets/x86/64/immortalwrt-sdk-24.10.4-x86-64_gcc-13.3.0_musl.Linux-x86_64.tar.zst`
5. `./scripts/feeds update base luci packages && ./scripts/feeds install luci-base`
   （只装 `luci-base`：包 Makefile 需要 `feeds/luci/luci.mk`，不需要整套 LuCI 应用。）
6. 把仓库的 `openwrt/` 目录软链到 SDK 的 `package/kdae`
7. 往 `.config` 追加 `CONFIG_PACKAGE_kdae-panel=m` 与 `CONFIG_PACKAGE_luci-app-kdae-panel=m`，`make defconfig`，
   然后 `make package/kdae-panel/compile package/luci-app-kdae-panel/compile -j$(nproc) V=s`
8. 产物在 `bin/packages/x86_64/base/kdae-panel_*.ipk` 与 `bin/packages/x86_64/luci/luci-app-kdae-panel_*.ipk`
   （luci.mk 把 LuCI 包放进 `luci` 子目录），一并上传 artifact；release 事件时附到 Release

预编译二进制通过 `PKG_BUILD_DIR` 外的绝对路径传入包 Makefile（`KDAE_PANEL_BIN` 变量），`Build/Compile` 为空，`Package/kdae-panel/install` 直接 `$(INSTALL_BIN)` 该文件。包 Makefile 在缺少该变量时报明确错误，而不是装出一个空文件。

## 六、测试

**新增 Go 单测**（`internal/host/procd_test.go`）

- ubus JSON 解析：运行中 / 已停止 / 服务不存在 / ubus 不可用四种输入。
- `ExecStartPath` 回退：ubus 有实例取 `command[0]`，ubus 无实例回退到配置的 `dae_binary`（服务停止时也不能为空）。
- `/etc/init.d/dae enabled` 退出码 → `UnitFileState` 映射；脚本不存在 → `LoadState` 为 `not-found`。
- `logread` 两种前缀格式 + dae `level=…/msg=…` 消息体解析。
- `/proc` 读取（`procRoot` 指向临时目录）：VmRSS、utime/stime、environ。
- `RestartSelf` 命令行含 `setsid`。

**新增 Go 单测**（`internal/daeinstall/procd_units_test.go`）

- `procdUnits.Plan`：脚本不存在 / 脚本指向别的二进制 / 脚本一致，三种结果。
- `procdUnits.Installed`：二进制在 / 不在。
- `FirstInstall` 在 procd 下不写服务定义、不调 daemon-reload。

**新增 Go 单测**（`internal/panelupdate/panelupdate_test.go` 增补）

- `PreferenceLocked=true` 时：`New()` 不读已存在的偏好文件（初始值原样保留）；`SetEnabled` 返回错误且不创建偏好文件；`Status().Locked` 为真。
- `PreferenceLocked=false` 时行为与现在逐字一致（回归）。
- handler 层：锁定时 `PUT /api/v1/panel/update/preference` 返回 409 `self_update_preference_locked`。

**回归**：现有全部 systemd 测试必须原样通过（后端抽象与偏好锁定都不得改变 systemd 行为）。`go test ./...`、`go vet ./...`、`npm run typecheck`、`npm test`。

**打包验证**（CI）：ipk 生成成功；`tar -xOf … ./control` 断言 `Depends` 含全部 kmod 与 `ca-bundle`、`Conflicts: dae`；`data.tar.gz` 内含 `/usr/bin/kdae-panel`、两个 init 脚本、`/etc/config/kdae-panel`。

**真机验证清单**（用户执行，写入 `docs/openwrt.md`）：安装 ipk → LuCI 出现菜单 → 启动面板 → 打开一次性链接创建管理员 → 版本管理页首次安装 dae → 写配置 → 启动 dae → 日志页有内容 → geo 更新 → 重启路由后状态正确。

## 七、文档

- 新增 `docs/openwrt.md`：前提、安装、UCI 配置项表、与官方 `dae` 包冲突的说明、升级与卸载、排障（`logread -e kdae-panel`、`ubus call service list`、`/etc/init.d/dae enabled`）。
- `README.md` 增补 immortalwrt 部署段落，链到 `docs/openwrt.md`。
- `docs/deployment.md` 与 `docs/architecture.md` 补一句后端抽象的存在与选择规则。

## 八、实施顺序

1. `internal/host` 抽接口 + systemd 搬家（纯重构，测试必须全绿）
2. procd 后端 + 单测
3. 后端选择接进 `app.Config` / flag / health
4. `daeinstall` 抽 `unitProvisioner` + procd 实现 + 单测
5. `geodata` 的后端相关分支
6. `panelupdate` 偏好锁定（Go + Vue + api.md）
7. init 脚本与 UCI 默认配置
7. 两个包的 Makefile
8. LuCI 页面（menu.d / acl.d / view.js）
9. CI workflow
10. 文档
