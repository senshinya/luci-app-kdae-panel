# kdae-panel

`kdae-panel` 是面向 [dae](https://github.com/daeuniverse/dae) 及其兼容分支的零侵入式 Web 管理面板。

面板不引用 dae 的内部 Go 包，也不读取其内部 eBPF Map。它只依赖 dae 的公开命令、`.dae` 配置文件、systemd 和 journald，因此 dae 内部重构、协议实现变化和普通配置字段新增通常不需要同步修改面板。

## 界面预览

| 运行概览 | 代理编排 |
| :--: | :--: |
| ![运行概览](docs/screenshots/dashboard.png) | ![代理编排](docs/screenshots/orchestration.png) |

| dae 版本管理 | 登录 |
| :--: | :--: |
| ![dae 版本管理](docs/screenshots/versions.png) | ![登录](docs/screenshots/login.png) |

*截图来自本地演示环境，节点、订阅与延迟均为示例数据。*

## 功能

- 通过 `dae export outline` 动态发现当前版本的配置结构；
- systemd 与 OpenWrt procd 两套服务后端，自动探测；服务状态、启动、停止和重启；
- dae 无损重载、暂停和 sysdump 诊断；
- `global`、节点、订阅、分组与路由的可视化编排：全局设置覆盖 dae 当前公开的字段，实际支持项和默认值由本机二进制的 `export outline` 动态确认，不兼容字段会明确标记；支持分享链接批量导入、订阅与分组过滤条件编辑、逐条路由编辑，以及 GFW/中国列表/全局/MAC 常用路由模板；复杂规则可直接在当前页面编辑对应节原文，注释与未涉及的配置节保持不变；
- 订阅离线缓存开关（dae 的 `-file` 持久化）、立即刷新与按间隔自动刷新；
- 在官方 dae 发布与 kdae 分支 CI 构建之间安装、切换、回滚或卸载，安装前校验并在失败时自动恢复；下载过的二进制会保存在本地版本库，后续切换无需联网，并可逐个清理；机器上没有 dae 时可完成首次安装，卸载时可分别选择保留或删除配置与 geo 数据（默认保留，版本管理默认开启）；
- 一键更新 geo 数据，可在 Loyalsoldier 与 v2fly 两套规则集之间切换：校验 sha256、就地替换 dae 实际读取的那一份、只 reload 不重启，失败自动还原（独立开关，默认关闭）；支持每天到每 30 天的定时自动更新，来源沿用上次、绝不静默切换规则集；
- 面板自身的新版本提醒：读取本仓库最新发布并长时缓存，设置页支持立即检查，可用 `KDAE_PANEL_DISABLE_UPDATE_CHECK` 整体关闭；
- 面板一键自升级：默认开启，可在设置页直接开关；校验 sha256、用新二进制自证可运行后再替换并重启自身，保留上一版供人工还原；
- 面板主机侧的节点 TCP 直连延迟探测；
- 原始配置编辑、独立校验、并发冲突检测和事务保存；
- 保存前备份、原子替换及重载失败后的磁盘回滚；
- 配置历史浏览与指定版本恢复；
- journald 结构化日志浏览、搜索和级别筛选；
- SQLite 管理员账户、Argon2id 密码摘要和服务端会话；
- SameSite/HttpOnly Cookie、CSRF 校验、同源检查和登录限速；
- Vue 3 响应式管理界面，前端资源嵌入单个 Go 二进制；
- Linux `amd64`、`arm64` 和 `riscv64` 发布构建。

## 一键部署

在有 systemd 的 Linux（amd64 / arm64 / riscv64）上，以 root 执行：

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/tuoro/kdae-panel/main/scripts/get.sh)"
```

脚本会下载最新发布包、比对 `SHA256SUMS`、安装并启动服务。固定安装某个版本：

```bash
KDAE_PANEL_VERSION=v0.1.0 bash -c "$(curl -fsSL https://raw.githubusercontent.com/tuoro/kdae-panel/main/scripts/get.sh)"
```

这条命令等于信任本仓库与 GitHub：校验和与发布包同处一个 Release，防传输损坏，防不住发布者本身。不接受这个前提、或网络无法直连 GitHub 时，请手动到 [Releases](https://github.com/tuoro/kdae-panel/releases) 下载对应架构的 `kdae-panel_linux_<arch>.tar.gz` 与 `SHA256SUMS`，用 `sha256sum -c --ignore-missing SHA256SUMS` 核对（清单含全部架构，勿直接整份 `-c`）后运行包内的 `install.sh`——或直接用下面的源码方式。发布包另附构建来源证明，验证方式见 [docs/deployment.md](docs/deployment.md)。

若这台机器上还没有 dae，装好面板后可直接在版本管理页完成 dae 的首次安装。安装完成后的访问方式见下方「首次访问」。

## OpenWrt / ImmortalWrt

immortalwrt 24.10.4（x86/64）上以 ipk 部署，附带 LuCI 入口与配置页。dae 的可执行文件、
配置与 geo 全部由面板管理，不经 opkg——这样 `opkg upgrade` 不会把你自己的分支构建盖回
官方版本。详见 [docs/openwrt.md](docs/openwrt.md)。

## 从源码安装

依赖 Go 1.25.12+、Node.js 22+，运行环境需要 systemd：

```bash
git clone https://github.com/tuoro/kdae-panel.git
cd kdae-panel
npm ci --prefix web
make build
sudo ./scripts/install.sh
```

## 首次访问

新安装默认监听 `0.0.0.0:2023`，因此本机和同一局域网内的设备都能访问：

```text
http://<面板机器的内网 IP>:2023
```

首次安装完成后，脚本会枚举本机内网 IPv4，并在终端的「首次访问地址」下直接打印完整的一次性初始化链接；多网卡机器可能出现多条，选择当前设备能访问的一条即可。页面会自动完成授权，注册表单只需填写用户名和密码。创建管理员后初始化接口会永久关闭，一次性链接的临时文件也会立即删除。

已有安装不会在升级时覆盖 `/etc/kdae-panel/kdae-panel.env`；若原来仍是 `127.0.0.1:2023`，请将 `KDAE_PANEL_LISTEN` 改为 `0.0.0.0:2023` 并重启面板。局域网直连使用明文 HTTP，只适合可信内网；跨不可信网络访问请使用 HTTPS 反向代理或 SSH 隧道。

## 卸载

以下命令须在 root shell 中执行：

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/tuoro/kdae-panel/main/scripts/uninstall.sh)"
```

默认移除程序、服务单元及其 systemd override，配置、账户数据库与配置备份全部保留；清理 override 是为了避免重装后意外恢复 `/usr/bin` 等高权限写路径。要连数据一并清除，在命令前加 `KDAE_PANEL_PURGE=true`。两种模式都不触碰 dae——它的服务、二进制、配置与 geo 数据原样保留。安装时会在本地落一份等效脚本，离线也可卸载：`sudo bash /usr/share/kdae-panel/uninstall.sh`；一键自升级只替换二进制，这份离线脚本仍属于最近一次完整安装的版本，联网时优先使用上面的最新脚本。

## 开发

```bash
npm install --prefix web
npm run build --prefix web
go run ./cmd/kdae-panel \
  --database ./data/panel.db \
  --backup-dir ./data/backups \
  --schedule-file ./data/schedule.json \
  --dae-config ./data/config.dae
```

前后端分离开发：

```bash
# 终端一
go run ./cmd/kdae-panel --database ./data/panel.db --schedule-file ./data/schedule.json

# 终端二，Vite 会代理 /api 到 127.0.0.1:2023
npm run dev --prefix web
```

验证全部代码：

```bash
npm run typecheck --prefix web
npm test --prefix web
npm run build --prefix web
go test ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
```

改动前端依赖后，务必先 `rm -rf web/node_modules web/*.tsbuildinfo` 再 `npm ci --prefix web` 复验：`vue-tsc -b` 是增量构建，而 `npm install` 不会删掉已不在 package.json 里的包，两者叠加会让本地 typecheck 用着旧状态通过，到 CI 的干净环境才失败。

`@types/katex` 看起来无人引用，实际是 naive-ui 类型定义的依赖（`config-provider/src/katex.d.ts` 与 `equation/src/Equation.d.ts` 直接 `import 'katex'`）。源码里搜不到它，删掉即 typecheck 失败。

## 上游兼容

面板启动后直接执行当前安装的 dae：

```text
dae --version
dae --help
dae export outline
dae validate -c <候选配置>
dae reload
dae suspend
dae sysdump
```

配置字段和默认值来自 `export outline`，配置正确性以 `validate` 的结果为准。面板不会复制 dae 的完整配置模型，也不会将未知配置字段静默删除。

完全零适配无法覆盖上游主动删除公开命令或改变命令语义的情况。CI 会持续验证面板自身契约；建议对生产环境的 dae 版本进行固定，并在升级前使用新二进制验证现有配置。

仓库的 `kdae 上游兼容` 工作流每周检出并构建 `olicesx/dae:kdae`，使用真实二进制验证能力发现、outline、配置校验和 sysdump 文件契约。上游发生破坏性变化时会直接产生失败记录。

## 文档

- [架构与兼容策略](docs/architecture.md)
- [安装部署与升级](docs/deployment.md)
- [OpenWrt / ImmortalWrt 部署](docs/openwrt.md)
- [HTTP API](docs/api.md)
- [安全策略](SECURITY.md)

## 许可证

GNU Affero General Public License v3.0，详见 [LICENSE](LICENSE)。
