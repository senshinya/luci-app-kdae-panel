# HTTP API

所有接口以 `/api/v1` 为前缀，响应使用 UTF-8 JSON。除健康检查、认证状态、首次初始化和登录外，接口都需要有效会话。

## 认证

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/auth/status` | 初始化和登录状态 |
| `POST` | `/auth/bootstrap` | 将一次性初始化链接中的 token 兑换为短时 HttpOnly Cookie |
| `POST` | `/auth/setup` | 使用短时初始化授权创建首个管理员，仅可成功一次 |
| `POST` | `/auth/login` | 登录并设置 HttpOnly Cookie |
| `POST` | `/auth/logout` | 注销当前会话 |
| `POST` | `/auth/password` | 修改密码并注销旧会话 |

登录、初始化和状态响应会返回 `csrfToken`。所有已登录的非只读请求必须增加：

```http
X-CSRF-Token: <csrfToken>
```

浏览器会话 Cookie 名为 `kdae_panel_session`，属性为 `HttpOnly`、`SameSite=Strict`，可配置 `Secure`。

未初始化时，`/auth/status` 会返回 `bootstrapRequired: true`。前端从安装脚本所示 URL 的 `#bootstrap=...` 片段读取 token，调用 `/auth/bootstrap` 兑换一个有效期 10 分钟、`HttpOnly`、`SameSite=Strict` 的初始化 Cookie，并立即从地址栏清除片段。`/auth/setup` 的 JSON 只包含用户名和密码，不再传输 bootstrap token。显式配置 `KDAE_PANEL_BOOTSTRAP_TOKEN` 时，初始化链接会基于该固定值生成；管理员创建成功后，发行单元的临时链接文件会被删除。

## dae 能力

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/health` | 面板健康状态、版本与服务后端（`backend` 为 `systemd` 或 `procd`） |
| `GET` | `/dae/capabilities` | dae 可用性、版本和命令能力 |
| `GET` | `/dae/outline` | 当前 dae 导出的动态配置结构 |

`backend` 说明面板正在用哪一套接口管理 dae：存在 `/sbin/procd` 时自动选 `procd`，否则 `systemd`；
可用 `KDAE_PANEL_SERVICE_BACKEND` 强制。后端选错的症状是服务控制全部失败，而那个现场离原因很远，
因此把结论直接暴露在健康检查里。

## 配置

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/config` | 入口配置文本、SHA-256 和文件元数据 |
| `POST` | `/config/validate` | 只校验候选内容 |
| `PUT` | `/config` | 保存候选内容，可选择立即重载 |
| `GET` | `/config/backups` | 列出自动备份和手动配置存档 |
| `POST` | `/config/backups` | 将当前入口配置保存为带名称、备注的手动存档 |
| `PUT` | `/config/backups/{id}` | 修改存档名称和备注 |
| `DELETE` | `/config/backups/{id}` | 删除存档内容及其元数据 |
| `POST` | `/config/backups/{id}/restore` | 恢复指定备份或存档 |

创建和编辑存档请求体：

```json
{
  "name": "稳定线路",
  "note": "家庭网络使用"
}
```

`name` 必填，去除首尾空白后最多 80 个字符；`note` 可选，最多 500 个字符。没有名称和备注的旧备份在前端显示为“自动备份”，也可以通过编辑接口补充。备份内容仍是独立的 `.dae` 文件，名称和备注保存在同编号的 `.meta.json` 文件中；删除存档会同时删除两者。

保存示例：

```json
{
  "content": "global { ... }\nrouting { fallback: direct }\n",
  "expectedHash": "提交编辑前读取到的 SHA-256",
  "apply": true
}
```

入口配置已经存在时，`expectedHash` 必填且不匹配时返回 HTTP `409`，防止覆盖外部修改；新建入口配置时必须为空。`apply` 默认为 `true`。

配置保存、备份恢复和服务控制操作会共享串行门；已有操作执行时返回 `409 operation_in_progress`，避免多个控制动作交叉执行。

所有备份（包括手动存档）共用最多 50 份、总大小 256 MiB 的保留上限。达到上限时按文件创建时间清理最旧的备份，手动存档的元数据会随对应内容一起清理。

常见错误码：

| HTTP | code | 含义 |
|---|---|---|
| `400` | `configuration_backup_invalid` | 存档名称或备注不符合长度要求 |
| `409` | `configuration_conflict` | 磁盘内容已经变化 |
| `422` | `configuration_invalid` | dae 拒绝候选配置 |
| `502` | `configuration_apply_failed` | 保存后重载失败，响应包含回滚状态 |

## dae 版本管理

默认开启。显式设置 `KDAE_PANEL_ENABLE_DAE_INSTALL=false` 时，以下接口一律返回 `503 dae_install_disabled`。

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/dae/install` | 当前安装状态与正在进行的任务 |
| `GET` | `/dae/versions?source=official\|kdae` | 列出上游与本地版本 |
| `POST` | `/dae/install` | 开始安装指定版本 |
| `DELETE` | `/dae/cache` | 删除指定版本的本地缓存 |
| `POST` | `/dae/rollback` | 回滚到上一版本 |
| `POST` | `/dae/uninstall` | 卸载面板管理的 dae，可选清理配置与 geo 数据 |

安装请求体：

```json
{ "source": "kdae", "ref": "30187784287", "label": "d63a0c1" }
```

`source` 只接受 `official` 与 `kdae` 两个枚举值，仓库地址在代码中写死，不接受外部指定。`ref` 对官方来源是发布 tag，对 kdae 是构建编号。`GET /dae/versions` 另接受 `limit` 参数（1–100，默认 30），超出范围返回 `400 invalid_limit`。

版本响应在上游字段之外附带 `cached`、`cachedAt`、`cachedBytes`；只存在于本机、不在当前上游清单中的版本还会带 `cachedOnly`。已过期的 kdae 构建只要本地缓存完整仍然可切换；上游暂时不可访问时，只要存在缓存也会返回本地版本。缓存按来源、版本与本机 CPU 平台隔离，真正安装前会重新计算二进制 SHA-256，而不是只信任缓存索引。

GitHub JSON 元数据另有 10 分钟进程内缓存；同 URL 的并发请求只访问上游一次，刷新失败时继续使用最近成功结果。凭据管理端点如下，任何响应都只返回 `configured` 与 `source`，不会返回 Token：

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/settings/github` | 查询是否配置 GitHub Token 及来源（`panel` / `environment`） |
| `PUT` | `/settings/github` | 保存 `{"token":"..."}`，写入 `0600` 独立文件并立即生效 |
| `DELETE` | `/settings/github` | 清除面板保存的 Token；环境变量管理时返回 `409` |

删除缓存请求体：

```json
{ "source": "official", "ref": "v2.0.0" }
```

删除只影响 `/var/lib/kdae-panel/dae-versions/` 下的对应缓存，不修改当前运行的 `/usr/bin/dae`，也不删除安装事务的上一版回滚点。版本不存在返回 `404 cached_version_not_found`。

机器上还没有 dae 时，`GET /dae/install` 的响应会附带 `provision` 字段，说明首次安装是否可行、将要写入哪些路径、以及缺少哪些可写目录。此时提交安装会走首次安装：除可执行文件外还写入 geo 数据与种子配置。systemd 后端下还会写入 `dae.service` 单元并执行 `daemon-reload`；procd 后端下服务定义由 ipk 软件包自带的 `/etc/init.d/<name>` 提供，面板只校验它存在，不写入也不重新加载。安装完成后**不会启动服务**。

任务进行中（`downloading`/`applying`）的响应**不含** `provision`：该字段要靠实际试写目标目录才能算出来，而界面每两秒轮询一次——systemd 后端下，其中一个探测目标正是 systemd 在 inotify 监视的单元目录，反复试写并不是没有代价。客户端应沿用上一次拿到的值，而不是当作"首次安装已不可行"。

安装、回滚与卸载都立即返回 `202` 与任务快照，由客户端轮询 `GET /dae/install` 获取进度。安装任务依次经过 `downloading`、`applying`；命中本地版本时仍从 `downloading` 开始，但任务会带 `cached: true` 并很快进入替换阶段。回滚与卸载直接进入 `applying`，终态均为 `done` 或 `failed`。同一时刻只允许一个版本管理任务，重复提交返回 `409 install_in_progress`。

卸载请求体可选，零值是安全默认：

```json
{ "purgeConfig": false, "purgeGeo": false }
```

`purgeConfig` 与 `purgeGeo` 相互独立，只有显式设为 `true` 才删除对应数据。geo 清理覆盖 dae 搜索路径里所有面板可见的副本，受 `ProtectHome=true` 隐藏的 `/root/.local/share/dae` 不在其中。卸载只接受面板有安装账本且二进制摘要未漂移的 dae；systemd 后端下还要求服务单元位于面板管理的标准路径，procd 后端没有单元可校验——服务定义属于 ipk 软件包，不属于某一次 dae 安装。它会停止并禁用 dae，移除可执行文件与版本回滚记录；systemd 后端下一并移除服务单元并执行 `daemon-reload`，procd 后端下服务定义由软件包保留、不删除也无需重新加载。文件移除、可选的数据清理与服务定义变更属于同一事务，失败时会恢复文件、开机启动状态和原运行状态。

读取缓存、下载与校验不占用全局控制门，只有替换与重启阶段才进入串行区，避免几十兆的 I/O 把配置保存一并堵住。普通升级和切换只缓存可执行文件；首次安装还需要种子配置与 geo 数据（systemd 后端下还需要用来渲染服务单元的模板），因此即使该版本已有二进制缓存，也会重新取得并校验完整发布包。

校验和缺失或格式不符时拒绝安装，没有跳过校验的开关。kdae 的构建产物保留 90 天，过期版本在列表中标记为不可安装；面板只接受本仓库自己的构建，解析时会重新核对 `head_repository`、事件类型、分支与工作流文件路径四项。

## geo 数据更新

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/dae/geo` | geo 数据现状、可选来源与正在进行的任务 |
| `POST` | `/dae/geo` | 更新到指定来源的最新版 |
| `GET` | `/dae/geo/sources` | 列出管理员保存的自定义来源 |
| `POST` | `/dae/geo/sources` | 添加自定义来源 |
| `PUT` | `/dae/geo/sources/{id}` | 修改自定义来源 |
| `DELETE` | `/dae/geo/sources/{id}` | 删除未在使用的自定义来源 |

Geo 数据管理默认开启，由 `KDAE_PANEL_ENABLE_GEO_UPDATE`（OpenWrt 上是 UCI 的 `enable_geo_update`）控制，与 dae 版本管理互不影响；关闭时以下端点一律返回 `503 geo_update_disabled`。

更新请求体（可省略，此时沿用 `status.defaultSource`）：

```json
{ "source": "loyalsoldier" }
```

`source` 接受内置的 `loyalsoldier`、`v2fly`，或由来源管理接口生成的 `custom:<id>`；未知或已经删除的来源返回 `400 invalid_geo_source`。不同来源的规则集可能不同，切换会改变 `geosite:` 规则匹配的域名集合。

自定义来源请求体包含 `label`、`geoipUrl`、`geoipSha256Url`、`geositeUrl`、`geositeSha256Url`。四条地址都必须是公网 HTTPS；保存时拒绝 userinfo、内网字面地址与 URL 片段，下载首跳和每次重定向会重新解析 DNS，并在实际连接前再次拒绝非公网地址。自定义请求使用独立客户端，不携带 GitHub Token。每个数据文件上限 64 MiB，校验文件上限 64 KiB，没有跳过 SHA-256 的开关。来源保存在权限 `0600` 的 `KDAE_PANEL_GEO_SOURCES_FILE`；当前更新记录正在引用的来源不能直接删除，需先用另一个来源成功更新。

`GET` 返回 `status.sources`（每个来源的标识、展示名、全部信任根仓库与说明）、`status.defaultSource`（界面该预选哪个——用过就是上次那个）、`status.targetDir`（本次会写入哪个目录）、`status.searchPath`（dae 的完整查找顺序）、每个文件的实际路径与大小，以及 `files[].shadowed`——被优先级更高的副本遮蔽、因而不会生效的同名文件。

`POST` 立即返回 `202` 与任务快照，进度靠轮询 `GET /dae/geo`，阶段与安装任务一致（`downloading` → `applying` → `done`/`failed`）。同一时刻只允许一个 geo 任务，重复提交返回 `409 geo_update_in_progress`；它与安装任务各有各的任务槽，但落盘阶段共用全局控制门。

dae 正在运行时，更新会把服务后端记录的 PID（systemd 的 `MainPID`、procd 的实例 PID）显式传给 `dae reload`，不依赖 `/var/run/dae.pid`，也不重启服务。dae 未运行时只更新文件并成功结束，下一次启动会直接读取新数据；此时无法借助 reload 检查配置引用的 Geo 分类是否存在，若新数据仍缺少分类，下一次启动仍会失败。若运行中的 dae 不接受新数据，面板会自动还原旧文件并再 reload 一次，任务标记为 `failed`。

dae 的 `validate` 不检查 `geoip:` / `geosite:` 分类是否真实存在。Geo 更新重载、启动、重启或版本切换因分类缺失失败时，面板会从 dae 命令输出或本次操作后的服务日志（systemd 后端读 journald，procd 后端读 `logread`）明确指出缺失分类并引导到 Geo 数据页；版本切换仍按原事务回滚二进制。

## 定时任务（订阅自动刷新 / geo 自动更新）

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/schedule/reload` | 读取订阅自动刷新的设置与执行状态 |
| `PUT` | `/schedule/reload` | 更新订阅自动刷新设置 |
| `GET` | `/schedule/geo` | 读取 geo 自动更新的设置与执行状态 |
| `PUT` | `/schedule/geo` | 更新 geo 自动更新设置 |

两组端点的请求与响应完全同构。geo 自动更新随 `KDAE_PANEL_ENABLE_GEO_UPDATE` 一起出现，
到点后重新下载校验并只 reload 不重启；来源沿用面板记录的上一个，绝不自动切换规则集。

```json
{
  "enabled": true,
  "intervalMinutes": 1440
}
```

响应在此基础上追加 `lastRunAt`、`lastError` 和 `nextRunAt`。

dae 只在重载时重新拉取 `subscription` 链接，因此"订阅定时刷新"的实现就是按间隔执行一次 `dae reload`。每轮开始前尝试获取全局控制锁，锁被占用时跳过当轮并把原因记入 `lastError`，不会与用户发起的操作交叉。

间隔取值范围为 5 分钟到 30 天。设置与上次执行时间一起持久化（订阅刷新在 `KDAE_PANEL_SCHEDULE_FILE`，geo 在 `KDAE_PANEL_GEO_SCHEDULE_FILE`），下一轮按"上次执行 + 间隔"排期，因此面板重启或提交无变化的设置都不会把倒计时重新拉满；停机期间错过的轮次会在启动一分钟后补做。

重载应用的是磁盘上的当前配置，所以之前用 `apply: false` 保存但未应用的改动会随这次刷新一并生效。

订阅内容本身的缓存由 dae 负责：把链接的 scheme 写成带 `-file` 后缀的形式（如 `https-file://`），dae 会将拉取成功的内容保存到 `config_dir/persist.d/<tag>.sub`，并在后续拉取失败时回退使用。面板只负责在配置里维护这一行，不自行下载或缓存订阅内容。

## 面板自身更新

procd 部署没有这项能力：`internal/app/app.go` 探测到 procd 后端时既不构造自升级管理器，
也不发起面板自身的新版本检查——不是默认关闭，而是根本不存在。理由：上游发布的面板二进制不含
procd 后端，升级一次就会把自己换成一个只会调用 systemctl 的程序，重启即不可用；面板的版本线
也和上游 tag 不是一回事，检查只会给出误导性的提示。procd 部署走 `opkg upgrade` 升级。

procd 下，以下四个接口的实际行为：`GET /panel/update`、`POST /panel/update/check` 只返回
`check`（`current`、`checkedAt`；不发起联网检查，`latest` 缺失，`updateAvailable` 恒为
`false`），不含 `status`、不含 `job`；`PUT /panel/update/preference`、`POST /panel/update`
恒返回 `503 panel_self_update_unavailable`（"当前部署不支持面板自升级"）。

以下响应结构与升级机制的说明仅适用于 systemd 后端：

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/panel/update` | 新版本检查结果与自升级状态 |
| `POST` | `/panel/update/check` | 立即绕过缓存检查面板新版本 |
| `PUT` | `/panel/update/preference` | 在 UI 中持久化一键升级开关 |
| `POST` | `/panel/update` | 触发一键自升级 |

`GET` 响应里的 `check` 含 `current`、`latest`、`updateAvailable`、`checkedAt`，检查失败时带 `error`；
结果按 TTL 缓存（成功 6 小时、失败 15 分钟），dev 构建不发起检查，
`KDAE_PANEL_DISABLE_UPDATE_CHECK=true` 时不再联网、恒不提示。
手动检查接口返回同样的 `check`、`status` 与 `job` 结构，会绕过成功缓存；同一面板在 1 分钟冷却期内重复调用直接返回上次结果。

正式部署的响应始终带 `status`（`enabled`、是否可升级、二进制路径、上一版副本位置）
与 `job`（任务进度）。`PUT /panel/update/preference` 接受 `{"enabled":true|false}`，
原子保存到面板数据目录并返回新状态；关闭时 `POST` 返回 `409 panel_self_update_disabled`。

`POST` 可选 `{"version":"v0.2.0"}`，省略则取最新正式发布；立即返回 `202` 并在后台执行：
下载 → 比对 `SHA256SUMS` → 新二进制 `-version` 自证 → 备份上一版 → 原子替换 →
`systemctl restart --no-block` 重启自身。下载、校验、自证或备份失败时原文件不动；
原子改名后的目录同步或重启请求失败则可能已经留下新二进制，任务会明确报错并要求人工确认或重启。

**没有自动回滚**：被替换、被重启的是当前进程自己，systemd 停掉它之后无从补救。
上一版副本保留在 `KDAE_PANEL_BACKUP_FILE`，还原步骤见 [deployment.md](deployment.md)。

## 网络探测

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/net/latency` | 从面板主机对目标做 TCP 握手延迟探测 |

请求与响应示例：

```json
{
  "targets": [
    { "host": "hk.example.com", "port": 443 }
  ]
}
```

```json
{
  "results": [
    { "host": "hk.example.com", "port": 443, "reachable": true, "latencyMs": 42.7 }
  ]
}
```

单次最多 64 个目标，单目标超时 4 秒，同一时刻最多 16 个并发拨号（上限属于面板进程，多个并发请求共享）。

`latencyMs` 是从发起拨号到 TCP 连接建立的耗时。目标为域名时它包含名称解析时间，因此冷缓存下的首次结果会偏高。该值反映面板主机到节点服务器的可达性，不等同于 dae 内部按 `tcp_check_url`/`udp_check_dns` 进行的健康检查，也不是 dae 选路时使用的延迟。

还有一层需要注意：dae 配置 `wan_interface` 时会劫持本机进程发出的流量，只有 dae 自身的连接凭 `so_mark_from_dae` 豁免。面板与 dae 同机运行，因此探测连接同样会进入 dae 的转发平面并按 routing 规则选路，测到的可能是经代理转发的路径而非物理直连。

单个目标不合法只影响它自己那条结果（`reachable: false` 并带 `error`），不会让整批探测失败；只有请求为空或超过 64 个目标才返回 `400`。目标列表会记入面板日志以供审计。

目标地址来自管理员自己的 dae 配置，可能合法指向内网或回环地址，因此服务端不按地址段过滤；该端点与其他写接口一样要求有效会话与 CSRF 令牌。

## 服务与日志

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/host/interfaces` | 本机网络接口及其 IP/CIDR 地址，供 global 接口选择器使用 |
| `GET` | `/service` | 服务状态与资源数据（systemd 后端读 `systemctl show`，procd 后端读 ubus 与 `/proc`） |
| `POST` | `/service/actions/start` | 启动 dae |
| `POST` | `/service/actions/stop` | 停止 dae |
| `POST` | `/service/actions/restart` | 重启 dae |
| `POST` | `/service/actions/reload` | 执行 `dae reload` |
| `POST` | `/service/actions/suspend` | 执行 `dae suspend` |
| `GET` | `/logs?limit=200` | 最近 1–500 条服务日志（systemd 后端读 journald，procd 后端读 `logread`） |
| `GET` | `/diagnostics/sysdump` | 执行 dae sysdump，并以 `application/gzip` 下载生成的归档 |

所有动作名和参数都由服务端白名单决定。URL、请求体和查询参数都不能注入额外命令参数。

## 错误格式

```json
{
  "error": {
    "code": "configuration_invalid",
    "message": "dae 配置校验失败：..."
  }
}
```

认证失败返回 `401`，CSRF 或来源检查失败返回 `403`，登录限速返回 `429` 并带 `Retry-After`。
