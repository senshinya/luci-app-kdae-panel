# OpenWrt / ImmortalWrt 部署

## 适用范围

本页描述 OpenWrt / ImmortalWrt 上的软件包部署，由两个软件包组成：

- `kdae-panel`：面板本体、procd 启动脚本（面板自身与 dae 各一份）、UCI 配置模板；
- `luci-app-kdae-panel`：LuCI 入口，依赖 `kdae-panel`，架构为 `all`/noarch。

Release 附带的产物覆盖两条版本线、三种架构：

| OpenWrt | 包格式 | 包管理器 | 架构 |
|---|---|---|---|
| 24.10 | `.ipk` | opkg | `x86_64`、`aarch64_generic`、`i386_pentium4` |
| 25.12 | `.apk` | apk | `x86_64`、`aarch64_generic`、`i386_pentium4` |

产物名带 `-openwrt-24.10` / `-openwrt-25.12` 后缀，对应上表的版本线。**包用官方 OpenWrt SDK
构建，ImmortalWrt 装的是同一批文件**：同版本同架构下两者的包架构标识、内核模块依赖名完全一致，
依赖在设备上按本机软件源解析，因此不另发一套 ImmortalWrt 专用包。

架构选哪个，在设备上直接问包管理器：24.10 用 `opkg print-architecture`，25.12 用
`apk --print-arch`。绝大多数 x86 软路由是 `x86_64`，ARM64 设备（armsr、rockchip、树莓派 4 等）
是 `aarch64_generic`。装错架构的包会被包管理器直接拒绝，不会装出一个跑不起来的面板。

本仓库只发这一条部署路径：包装到 `/usr/bin`、`/etc/init.d`、`/etc/config`，服务由 procd 管理。
面板的代码本身也支持 systemd 后端（自动探测），但那条路径没有发布产物，要用得自己从源码构建。

## 与官方 dae 包的关系

`kdae-panel` 的 control 文件声明 `CONFLICTS:=dae`：装了本包就装不上 OpenWrt/ImmortalWrt 官方源里的
`dae` 包，opkg 也会拒绝在两者共存的情况下继续。这是有意为之——本包把 dae 的可执行文件、配置与 geo
数据全部交给面板的版本管理页管理，不经包管理器安装或升级；如果同时装着官方 `dae` 包，一次
`opkg upgrade`（25.12 上是 `apk upgrade`）就可能把面板管理的分支构建盖回官方版本，而面板的账本
对此一无所知。

**25.12 上没有这条声明**：OpenWrt 的 apk 打包路径（`include/package-pack.mk` 里的 `apk mkpkg`）
根本不传冲突字段，`CONFLICTS:=dae` 在 apk 包里会被丢掉。互斥退化成文件冲突——两个包都装
`/etc/init.d/dae`，apk 拒绝覆盖已属于别的包的文件，装的时候仍然会失败，只是报错说的是文件
冲突而不是包冲突。处理方式一样：先 `apk del dae`。

dae 运行所需的内核模块不受这条冲突声明影响，仍由包管理器正常安装：它们是 `kdae-panel` 的依赖
（`kmod-sched-core`、`kmod-sched-bpf`、`kmod-veth`、`kmod-nft-bridge`），
装包时自动拉取，随官方源正常接受安全更新。`ca-bundle` 同样是依赖，供面板发起的 HTTPS 请求（下载
dae 版本、geo 数据、面板自身的新版本查询）验证证书。

`kmod-xdp-sockets-diag` **刻意不在依赖里**。ImmortalWrt 发这个包、它的 `dae` 包也依赖它，但官方
OpenWrt 的 24.10 与 25.12 都不发——本包由官方 SDK 构建、也要能装在官方系统上，依赖一个那边根本
不存在的包只会让安装直接失败。它是 AF_XDP 套接字的诊断模块，dae 的转发路径不用它；在 ImmortalWrt
上确实需要时自己装一次即可（`opkg install kmod-xdp-sockets-diag`）。

## 安装

24.10（opkg / ipk），把 `x86_64` 换成你的架构：

```sh
opkg update
opkg install ./kdae-panel_*_x86_64-openwrt-24.10.ipk \
             ./luci-app-kdae-panel_*_all-openwrt-24.10.ipk
```

25.12（apk）：

```sh
apk update
apk add --allow-untrusted ./kdae-panel-*-x86_64-openwrt-25.12.apk \
                          ./luci-app-kdae-panel-*-openwrt-25.12.apk
```

`--allow-untrusted` 是必需的：Release 附带的包没有用设备信任的密钥签名，apk 默认拒绝安装
未签名的本地文件。

文件名里的版本号随发布走（Release 附的包用该 Release 的 tag，去掉前导 `v`），因此这里用通配符而不是
写死某一版。安装脚本会调用 `/etc/init.d/kdae-panel enable` 与 `start`：面板装完立即可用，重启路由器
后也会自动拉起，不需要再手工执行 `start`。

如果这台机器上已经装过官方 `dae` 包（例如之前手动装的），`kdae-panel` 会装不上（24.10 上是包冲突，
25.12 上是 `/etc/init.d/dae` 的文件冲突），需要先卸载：

```sh
opkg remove dae dae-geoip dae-geosite     # 24.10
apk del dae dae-geoip dae-geosite         # 25.12
```

`/etc/dae` 下的配置与 geo 数据不会被 `opkg remove` 删除，装好 `kdae-panel` 后面板能直接读到它们
（首次安装时若 `/etc/dae/config.dae` 已存在，面板不会覆盖）。

## 首次访问

LuCI → 服务 → kdae 面板：页面上半部分「服务状态」区块就是初始化入口，还没有创建管理员时会直接
显示一次性初始化链接（点击即可跳转完成注册）。这条链接来自面板启动时写下的交接文件，也可以在
shell 里直接读取：

```sh
cat /var/run/kdae-panel/setup-url
logread -e kdae-panel
```

两种方式取到的是同一组链接（多网卡机器可能不止一条，选择当前设备能访问的一条即可）。管理员创建
成功后，交接文件立即删除，初始化接口永久关闭，此后再次访问这两个入口都不会再给出可用链接。

## 安装 dae

面板的版本管理页（登录后在导航中找到）可以在官方 dae 发布与 kdae 分支的 CI 构建之间选择安装。
机器上还没有 dae 时走的是"首次安装"：面板会写入可执行文件、geo 数据和一份种子配置——它带一套
可直接编辑的 AliDNS/Google DNS 默认 DNS，但不声明任何网卡，因而不会接管透明代理流量。

**安装完成后不会自动启动 dae。** dae 是透明代理，在一台你正通过面板本身访问的路由器上，配置不当
地启动可能直接切断你和它的连接。请先在面板的配置管理页写好实际规则，再到服务控制页手动启动。

`/etc/init.d/dae` 这份启动脚本本身随 `kdae-panel` 软件包一起安装，不由面板生成，面板只负责替换
它启动的可执行文件。

**dae 默认不开机自启。** 软件包的 `postinst` 只 `enable` 面板自己：dae 装完之后是否该随系统起来，
取决于你的配置写好了没有，包管理器无从判断。确认配置无误、手动启动过一次之后，在 LuCI 的
「服务 → kdae 面板」状态块里点「设为自启」，或者等价地执行：

```sh
/etc/init.d/dae enable
```

没有这一步，路由器重启后 dae 不会回来，代理会静默失效——面板自己不受影响，它是 `enable` 过的。

## UCI 配置项表

节名固定为 `kdae-panel.main`。修改后需要执行：

```sh
/etc/init.d/kdae-panel restart
```

（通过 LuCI 表单保存时会自动触发同样的重启，不需要手动执行上面的命令。）

| 选项 | 默认值 | 说明 |
|---|---|---|
| `enabled` | `1` | 开机自启；关闭后面板不会随系统启动，已在跑的实例不受影响 |
| `listen_addr` | `0.0.0.0` | 监听地址，默认接受本机与局域网连接 |
| `listen_port` | `2026` | 监听端口；上游的 systemd 部署默认 2023，这里避开 daed 占用的同一个端口 |
| `dae_binary` | `/usr/bin/dae` | dae 可执行文件路径；面板与 `/etc/init.d/dae` 读的是同一个值 |
| `dae_config` | `/etc/dae/config.dae` | dae 入口配置；geo 数据也放在它所在的目录 |
| `service_name` | `dae` | dae 的 init 脚本名（LuCI 表单未暴露这一项） |
| `enable_dae_install` | `1` | 允许面板下载、安装、切换与回滚 dae；关闭后版本管理页不可用 |
| `enable_geo_update` | `1` | 允许面板一键更新 geo 数据 |
| `trusted_proxies` | `127.0.0.0/8,::1/128` | 可信代理 CIDR，逗号分隔 |
| `session_ttl` | `12h` | 会话绝对有效期 |
| `secure_cookie` | `0` | Cookie 是否仅 HTTPS 发送 |

以上 11 项与 `/etc/config/kdae-panel` 的默认模板逐项一致；LuCI 的设置表单暴露其中 10 项，
`service_name` 有意不放进界面（改错这一项会让面板和 init 脚本互相认不出对方）。

四条必须知道的说明：

- **数据目录固定为 `/etc/kdae-panel`。** OpenWrt 上 `/var` 是 `/tmp` 的软链，两者都是内存文件系统，
  因此 init 脚本不再读取可配置路径，而是把数据库、管理员账户、备份、GitHub Token、自定义 geo
  来源与 dae 本地版本库逐项放进这个 overlay 目录。`/etc/kdae-panel` 本身若是符号链接，服务会
  拒绝启动，避免以 root 跟随到意外位置。
  `/etc/config/kdae-panel` 是 conffile，升级不覆盖，所以此前改过 `data_dir` 的机器仍留着旧值。
  init 发现它与 `/etc/kdae-panel` 不一致时**拒绝启动**并写一条 syslog——默默忽略等于从一个空目录
  重新开始：数据库和管理员账户还在老地方，界面上却什么都没有，且没有任何报错指向原因。
  按提示把数据搬过去，再 `uci delete kdae-panel.main.data_dir && uci commit kdae-panel` 即可。
- **没有自升级开关，但版本检查照做。** 计划阶段设想过给面板加一个一键升级自身的能力，但
  该能力（`internal/panelupdate`）取件坐标写死指向上游仓库 `tuoro/kdae-panel`，那里发布的二进制
  不含本文档描述的 procd 后端。一旦在这个部署上打开它并触发一次升级，面板会以 root 权限把自己
  替换成一个只认识 `systemctl` 的程序，重启后彻底无法工作——这不是需要提示用户注意的风险，而是
  必然发生的故障，因此 procd 后端下面板启动时压根不构造这项能力，设置页也就没有「允许一键升级」
  这个开关（取而代之的是一句说明和上面那条 `opkg install` 命令）。**这是预期行为，不是故障。**
  但**检查仍然做**：能不能自己动手升级，与该不该知道有新版本，是两件事。procd 下面板问的是
  `senshinya/luci-app-kdae-panel` 的最新 release——那才是这台机器装得上的那条版本线，沿用上游
  `tuoro/kdae-panel` 的 tag 报出来的"新版本"既装不上也不该装。
- 版本检查在 systemd 后端下由 `KDAE_PANEL_DISABLE_UPDATE_CHECK` 控制，自升级由
  `KDAE_PANEL_ENABLE_SELF_UPDATE` 控制；OpenWrt 的 `/etc/config/kdae-panel` 里两个选项都没有，
  不要照着别处的说明来找。
- **「面板更新」卡片如果写着「本部署不做新版本检查」**，那是面板认不出自己的版本号、因而放弃了
  比对，不是这个部署没有检查能力。比对只在版本形如 `vX.Y.Z` 时进行
  （`internal/app/panelupdate_handlers.go` 的 `parseSemver`），认不出就宁可不提示，也不拿一个没法
  比较的字符串去和 tag 硬比。正式发布的软件包不会出现这种情况；自己从源码编译时要让
  `main.version` 带上版本号（仓库根目录的 `make build` 用 `scripts/panel-version.sh` 自动填），
  否则二进制里的版本是 `dev`。
- **快照包（非 release 构建）不会提示"有新版本"指向它已经包含的那个发布。** 快照的版本号形如
  `v1.0.0+git8.5df15b7`，加号后面是"在 v1.0.0 之后又领先 8 个提交"；按 semver 规范这段
  build metadata 不参与比较，所以它与 v1.0.0 平级，而不是排在它前面。等真出了 v1.1.0，
  同一个快照会照常提示。写成 `v1.0.0-8-g5df15b7` 或 `v0.0.1-git144.5df15b7` 就成了预发布段，
  横幅会一直劝用户装回比在跑的代码更旧的包——`scripts/panel-version.sh` 存在的理由就是这个。

## 升级与卸载

**升级面板**：

```sh
# 24.10
opkg install ./kdae-panel_*_x86_64-openwrt-24.10.ipk \
             ./luci-app-kdae-panel_*_all-openwrt-24.10.ipk
# 25.12
apk add --allow-untrusted ./kdae-panel-*-x86_64-openwrt-25.12.apk \
                          ./luci-app-kdae-panel-*-openwrt-25.12.apk
```

安装新版本会覆盖旧的可执行文件与启动脚本；`/etc/config/kdae-panel` 声明为 conffile，
两种包管理器都不会覆盖你已经改过的配置。

因此默认端口从 2023 改成 2026 之后，**老安装升级上来仍然监听 2023**——conffile 保留的就是这个
行为。想跟上新默认值，手动改一次：

```sh
uci set kdae-panel.main.listen_port=2026
uci commit kdae-panel
/etc/init.d/kdae-panel restart
```

升级过程中面板会短暂停止：opkg 先执行旧包的 `prerm`（`stop` + `disable`），换完文件后再执行新包的
`postinst`（`enable` + `start`），因此命令返回时面板已经带着新版本重新跑起来了。apk 走的是自己的
`post-upgrade` 钩子，内容由同一份 `postinst` 生成，净效果相同。dae 本身不受影响，它是独立的 procd
服务，升级面板不会中断代理。

**包管理器只按包名和版本判断该不该动手**（下面以 opkg 为例，apk 同理），两种情况下它会拒绝装：

- **版本相同**（例如从 CI 产物页下载了同一次构建）：打印
  `Package kdae-panel (…) installed in root is up to date.`，退出码 0、什么都不做——这时面板
  **没有**被换掉。强制覆盖加 `--force-reinstall`。
- **版本被判定为更旧**：打印 `Not downgrading package … on root from A to B.`，同样什么都不做。
  强制覆盖加 `--force-downgrade`。

正式 Release 的版本号单调递增，不会遇到第二种。CI 产物（非 Release 构建）用的是
`0.0.1+<commit 数>.<短哈希>`，commit 数保证了后一次 push 的包一定算升级。

25.12 的 apk 产物版本是 `0.0.1_git<commit 数>~<短哈希>`：apk 的版本语法不接受 `+`，`_git` 是
它认识的后缀，commit 数落在后缀的数字段上，按数值比较，单调性与 ipk 侧一致。两种格式的 `0.0.1`
基版本都远低于任何正式发布。

版本号里这两处细节都是踩出来的：

- **后缀先放 commit 数，不能只放短哈希。** opkg 用的是 dpkg 那套比较规则，**字母排在数字
  之前**：`fe515cb` 与 `f961b58` 比到第二位是 `e` vs `9`，字母优先，于是先构建的那个反而
  "更大"，装后一个会被判成降级。commit 数单调，数字段又按数值比较，才轮得到"后一次 push
  一定算升级"。
- **基版本是 `0.0.1` 而不是 `0.0.0`。** 数字打头的字符串首个非数字块是空的，空排在任何字母
  之前，所以 `0.0.0+124.abcdefg` 反倒小于历史上那批 `0.0.0+fe515cb`。抬一位就盖过去了，
  同时 `0.0.1` 仍然远低于任何正式发布（比到第二段是 `0` < `8`）。

后一条尤其要紧，因为 **LuCI 的软件包页面没有传 `--force-downgrade` 的入口**——它构造 opkg
参数时只认 `--autoremove` 和 `--force-overwrite`，上传安装走的也是同一条路。真在页面上撞到
降级拒绝，只能先把 `luci-app-kdae-panel` 移除再上传：依赖方向是它依赖 `kdae-panel` 而不是
反过来，移除它不会动面板服务，dae 代理也不断。

**面板不会自己升级自己**，但会告诉你有没有新版本：面板设置页的「面板更新」卡片会查
`senshinya/luci-app-kdae-panel` 的最新 release 并与当前版本比对。它刻意只检查不动手——上游
`tuoro/kdae-panel` 发布的二进制不含 procd 后端，让面板自我替换一次就会把它换成一个只会调用
`systemctl` 的程序，重启即不可用。因此那张卡片上没有「一键升级」开关，只有版本号和上面那条
安装命令。

**卸载**：

```sh
opkg remove kdae-panel luci-app-kdae-panel   # 24.10
apk del kdae-panel luci-app-kdae-panel       # 25.12
```

不会删除 `/etc/kdae-panel`（面板数据）与 `/etc/dae`（dae 配置和 geo 数据）。要连数据一起清掉，
需要在卸载后手工执行：

```sh
rm -rf /etc/kdae-panel /etc/dae
```

dae 本身（可执行文件、其运行状态）不受面板卸载影响；`/etc/init.d/dae` 脚本随 `kdae-panel` 包一起
移除，因此卸载面板之后 dae 也就没有了启动入口。

## 日志功能的实际能力

面板的日志页读的是 OpenWrt 的系统日志环形缓冲区（`logread`），**不是磁盘上的日志文件**。默认
`system.@system[0].log_size` 只有 64 KiB；dae 在 `info` 级别下每条连接都记一行，缓冲区可能只装得
下几分钟的量，更早的记录会被直接挤掉，**重启路由器后缓冲区清空，此前的全部日志一并消失**。

这与 systemd 部署使用的 journald 相比是实质降级：journald 默认可持久化到磁盘、可按单元精确
过滤、可以翻很久以前的记录；procd 的环形缓冲区三者都不具备。

缓解办法（不能根治，只能延长缓冲区能覆盖的时间窗口）：

```sh
uci set system.@system[0].log_size='256'   # 单位 KiB
uci commit system
/etc/init.d/log restart
```

或者把 dae 的 `log_level` 调到 `warn` 减少噪声（在面板配置页的 `global` 段里改）。

日志页的搜索框是在已经取回的那几百条记录里做客户端过滤，不在缓冲区之外查找——缓冲区里已经没有
的内容，无论怎么搜都搜不出来。

## 仪表盘指标的差异

内存、任务数、累计 CPU 三格在两种部署下含义一致（procd 取自 `/proc/<pid>/`，systemd 取自 cgroup；
dae 是单进程，两者的数值可比）。第四格不同：

| | systemd | procd |
|---|---|---|
| 第四格 | 重启次数（`NRestarts`）+ 退出状态 | 运行时长 + 是否开机自启 |

procd 不维护重启计数器，也不记录上次退出码，这两个数在 OpenWrt 上根本不存在。面板不会拿 0 去
充数——那会让"服务反复崩溃"看起来和"一次没崩过"一模一样，连带把版本切换的崩溃循环检测也架空。
崩溃循环检测因此改用另一个信号：主进程号在观察窗口内变化即判定为起来就崩，与 systemd 那条
`NRestarts` 递增的判据等效。

## 排障

```sh
logread -e kdae-panel
logread -e dae
ubus call service list '{"name":"dae"}'
/etc/init.d/dae enabled; echo $?
curl http://127.0.0.1:2026/api/v1/health   # backend 字段应为 procd
mount | grep bpf                            # dae 需要 bpffs
```

`/api/v1/health` 返回的 JSON 里 `backend` 字段应为 `"procd"`；如果显示 `"systemd"`，说明这台机器
上探测到了 `/sbin/procd` 之外的信号（或者被 `KDAE_PANEL_SERVICE_BACKEND`/`--service-backend`
显式覆盖过），启动日志里会记录选中的后端，可用 `logread -e kdae-panel | grep 已选定服务后端`
确认。

若「首次安装」或「版本管理」提示某个目录不可写，注意 OpenWrt 部署没有 systemd 的
`ReadWritePaths` 概念——procd 不做任何文件系统沙箱，面板本身以完整 root 权限运行，能写整个文件
系统。这类报错在 OpenWrt 上通常意味着实际的文件系统权限问题（例如目标目录所在分区以只读挂载），
而不是需要放宽某个服务单元的写白名单；请直接检查 `mount` 输出和目标路径的实际权限。

## 真机验证清单

装好软件包后建议完整走一遍以下步骤，确认服务后端探测、procd 状态查询、崩溃循环检测和日志功能都按
预期工作：

1. 安装两个软件包；
2. LuCI 菜单「服务 → kdae 面板」应当出现；
3. 启动面板（`/etc/init.d/kdae-panel start` 或在 LuCI 状态块里点「启动」）；
4. 打开一次性链接创建管理员；
5. 在版本管理页完成 dae 的首次安装；
6. 在配置页写好实际规则；
7. 在服务控制页启动 dae；
8. 日志页应当能看到 dae 的输出；
9. 在 geo 数据页触发一次更新；
10. 在 LuCI 状态块里给 dae 点「设为自启」——不做这一步，下一条一定失败；
11. 重启路由器：面板与 dae 应自动拉起且状态显示正确，日志页会是空的（缓冲区已清空，符合预期），
    用此前创建的账户仍能登录（管理员账户存在 `/etc/kdae-panel` 下的 SQLite 数据库里，不受重启
    影响）。
