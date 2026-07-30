# OpenWrt / ImmortalWrt 部署

## 适用范围

本页描述 immortalwrt 24.10.4（x86/64）上的 ipk 部署，由两个软件包组成：

- `kdae-panel`：面板本体、procd 启动脚本（面板自身与 dae 各一份）、UCI 配置模板；
- `luci-app-kdae-panel`：LuCI 入口，依赖 `kdae-panel`。

与 [docs/deployment.md](deployment.md) 描述的 systemd 一键部署是两条独立的路径：systemd 版靠
`scripts/get.sh`/`scripts/install.sh` 装到 `/usr/bin`、`/etc/systemd/system`；OpenWrt 版靠 opkg
装到 `/usr/bin`、`/etc/init.d`、`/etc/config`，服务由 procd 管理，不涉及 systemd 也不涉及一键部署脚本。

## 与官方 dae 包的关系

`kdae-panel` 的 control 文件声明 `CONFLICTS:=dae`：装了本包就装不上 OpenWrt/ImmortalWrt 官方源里的
`dae` 包，opkg 也会拒绝在两者共存的情况下继续。这是有意为之——本包把 dae 的可执行文件、配置与 geo
数据全部交给面板的版本管理页管理，不经 opkg 安装或升级；如果同时装着官方 `dae` 包，一次
`opkg upgrade` 就可能把面板管理的分支构建盖回官方版本，而面板的账本对此一无所知。

dae 运行所需的内核模块不受这条冲突声明影响，仍由 opkg 正常安装：它们是 `kdae-panel` 的依赖
（`kmod-sched-core`、`kmod-sched-bpf`、`kmod-veth`、`kmod-nft-bridge`、`kmod-xdp-sockets-diag`），
装包时自动拉取，随官方源正常接受安全更新。`ca-bundle` 同样是依赖，供面板发起的 HTTPS 请求（下载
dae 版本、geo 数据、面板自身的新版本查询）验证证书。

## 安装

```sh
opkg update
opkg install ./kdae-panel_1.0.0-1_x86_64.ipk ./luci-app-kdae-panel_1.0.0-1_all.ipk
/etc/init.d/kdae-panel start
```

安装脚本已经调用 `/etc/init.d/kdae-panel enable`，重启路由器后面板会自动拉起；执行上面的 `start`
只是让本次安装立即生效，不必等下次开机。

如果这台机器上已经装过官方 `dae` 包（例如之前用 `opkg install dae` 手动装的），`CONFLICTS` 会让
`kdae-panel` 装不上，需要先卸载：

```sh
opkg remove dae dae-geoip dae-geosite
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
机器上还没有 dae 时走的是"首次安装"：面板会写入可执行文件、geo 数据和一份不劫持任何流量的种子
配置（`global {} routing {}`）。

**安装完成后不会自动启动 dae。** dae 是透明代理，在一台你正通过面板本身访问的路由器上，配置不当
地启动可能直接切断你和它的连接。请先在面板的配置管理页写好实际规则，再到服务控制页手动启动。

`/etc/init.d/dae` 这份启动脚本本身随 `kdae-panel` 软件包一起安装，不由面板生成，面板只负责替换
它启动的可执行文件。开机自启同样需要显式声明：

```sh
/etc/init.d/dae enable
```

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
| `listen_port` | `2023` | 监听端口 |
| `data_dir` | `/etc/kdae-panel` | 数据库、配置备份、状态文件与 dae 本地版本库的位置 |
| `dae_binary` | `/usr/bin/dae` | dae 可执行文件路径；面板与 `/etc/init.d/dae` 读的是同一个值 |
| `dae_config` | `/etc/dae/config.dae` | dae 入口配置；geo 数据也放在它所在的目录 |
| `service_name` | `dae` | dae 的 init 脚本名（LuCI 表单未暴露这一项） |
| `enable_dae_install` | `1` | 允许面板下载、安装、切换与回滚 dae；关闭后版本管理页不可用 |
| `enable_geo_update` | `1` | 允许面板一键更新 geo 数据 |
| `trusted_proxies` | `127.0.0.0/8,::1/128` | 可信代理 CIDR，逗号分隔 |
| `session_ttl` | `12h` | 会话绝对有效期 |
| `secure_cookie` | `0` | Cookie 是否仅 HTTPS 发送 |

以上 12 项与 `/etc/config/kdae-panel` 的默认模板逐项一致；LuCI 的设置表单暴露其中 11 项，
`service_name` 有意不放进界面（改错这一项会让面板和 init 脚本互相认不出对方）。

三条必须知道的说明：

- **`data_dir` 不要改到 `/var` 或 `/tmp` 下。** OpenWrt 上 `/var` 是 `/tmp` 的软链，两者都是内存
  文件系统，重启即空。数据库、管理员账户和 dae 本地版本库一旦落在这里，重启路由器后会全部丢失。
  默认值 `/etc/kdae-panel` 在 overlay 上，是持久的。
- **没有自升级开关，也没有版本检查开关。** 计划阶段设想过给面板加一个一键升级自身的能力，但
  该能力（`internal/panelupdate`）取件坐标写死指向上游仓库 `tuoro/kdae-panel`，那里发布的二进制
  不含本文档描述的 procd 后端。一旦在这个部署上打开它并触发一次升级，面板会以 root 权限把自己
  替换成一个只认识 `systemctl` 的程序，重启后彻底无法工作——这不是需要提示用户注意的风险，而是
  必然发生的故障，因此 procd 后端下面板启动时压根不构造这项能力，也不做版本检查。设置页的
  「允许一键升级」开关在本部署里恒为置灰，「面板更新」卡片不会给出可用版本。**这是预期行为，
  不是故障。升级面板请安装新的 ipk。**
- 上述两项在 systemd 部署（[docs/deployment.md](deployment.md)）里以
  `KDAE_PANEL_ENABLE_SELF_UPDATE` 和 `KDAE_PANEL_DISABLE_UPDATE_CHECK` 两个环境变量存在；
  OpenWrt 的 `/etc/config/kdae-panel` 里没有对应选项，不要照搬那份文档去找它们。

## 升级与卸载

**升级面板**：安装新版本的 `kdae-panel_*.ipk` 即可覆盖旧的可执行文件与启动脚本；
`/etc/config/kdae-panel` 声明为 conffile，`opkg install` 不会覆盖你已经改过的配置。

**卸载**：

```sh
opkg remove kdae-panel luci-app-kdae-panel
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

## 排障

```sh
logread -e kdae-panel
logread -e dae
ubus call service list '{"name":"dae"}'
/etc/init.d/dae enabled; echo $?
curl http://127.0.0.1:2023/api/v1/health   # backend 字段应为 procd
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

装 ipk 后建议完整走一遍以下步骤，确认服务后端探测、procd 状态查询、崩溃循环检测和日志功能都按
预期工作：

1. 安装两个 ipk；
2. LuCI 菜单「服务 → kdae 面板」应当出现；
3. 启动面板（`/etc/init.d/kdae-panel start` 或在 LuCI 状态块里点「启动」）；
4. 打开一次性链接创建管理员；
5. 在版本管理页完成 dae 的首次安装；
6. 在配置页写好实际规则；
7. 在服务控制页启动 dae；
8. 日志页应当能看到 dae 的输出；
9. 在 geo 数据页触发一次更新；
10. 重启路由器：面板与 dae 应自动拉起且状态显示正确，日志页会是空的（缓冲区已清空，符合预期），
    用此前创建的账户仍能登录（管理员账户存在 `/etc/kdae-panel` 下的 SQLite 数据库里，不受重启
    影响）。
