# 安装部署与升级

## 前置条件

- Linux，systemd 或 OpenWrt procd（自动探测，可用 `KDAE_PANEL_SERVICE_BACKEND` 强制）；本页的
  一键部署脚本与源码安装步骤只覆盖 systemd 路径，OpenWrt/ImmortalWrt 上的 ipk 部署见
  [docs/openwrt.md](openwrt.md)；
- 已安装并能够通过 `systemctl status dae` 正常运行的 dae——若这台机器上还没有 dae，可直接使用默认开启的版本管理完成首次安装；
- `/etc/dae/config.dae` 是实际入口配置（首次安装时由面板写入不劫持流量的种子配置）；
- 构建阶段需要 Go 1.25.12+ 和 Node.js 22+；
- 运行阶段不需要 Node.js。

## 一键部署

以下命令须在 root shell 中执行（OpenWrt 默认登录即 root；普通发行版可先运行 `sudo -i`）：

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/tuoro/kdae-panel/main/scripts/get.sh)"
```

脚本按 `uname -m` 选择发布资产（amd64 / arm64 / riscv64），从 GitHub Release 的 latest 直链下载、比对 `SHA256SUMS` 后运行包内的 `install.sh`，效果与源码安装完全一致。设置 `KDAE_PANEL_VERSION=v0.1.0` 可固定版本。

三点如实说明：

- **信任边界**：`curl | bash` 等于信任本仓库与 GitHub。校验和与发布包由同一个发布者签出、放在同一个 Release，防的是传输损坏与不完整下载，防不住发布者本身。发布包另附 GitHub OIDC 来源证明，可用 `gh attestation verify kdae-panel_linux_<arch>.tar.gz --repo tuoro/kdae-panel` 进一步确认归档确实由本仓库的发布流程构建——它不依赖与包同源的清单，防得住"资产被事后替换"，仍防不住发布者提交的代码本身。
- **网络前提**：`raw.githubusercontent.com` 与 `github.com` 都必须可达。无法直连时，请在能访问的机器上手动下载 `kdae-panel_linux_<arch>.tar.gz` 与 `SHA256SUMS` 两个文件，核对通过后拷到目标机器解压，运行包内 `install.sh`。核对命令：

  ```bash
  # SHA256SUMS 列有全部三个架构；只下载了一个包时必须加 --ignore-missing，
  # 否则会因另两个文件不存在而报错。预期恰好输出一行 "…tar.gz: OK"。
  sha256sum -c --ignore-missing SHA256SUMS
  ```

- **可重复执行**：脚本可用于升级——`install.sh` 会覆盖二进制与服务单元并重启面板，但不覆盖已有的 `/etc/kdae-panel/kdae-panel.env`。

安装完成后的访问方式见下方「首次访问」。

## 从源码安装

```bash
git clone https://github.com/tuoro/kdae-panel.git
cd kdae-panel
npm ci --prefix web
make build
sudo ./scripts/install.sh
```

安装内容：

```text
/usr/bin/kdae-panel
/etc/kdae-panel/kdae-panel.env
/etc/systemd/system/kdae-panel.service
/var/lib/kdae-panel/panel.db
/var/lib/kdae-panel/backups/
```

安装脚本不会覆盖现有 `/etc/kdae-panel/kdae-panel.env`，也不会修改 dae 配置。

## 首次访问

新安装默认监听 `0.0.0.0:2023`，同时接受本机和局域网连接：

```text
http://<面板机器的内网 IP>:2023
```

首次安装完成后，脚本会枚举本机 RFC1918 内网 IPv4，并在终端的「首次访问地址」下直接打印完整的一次性初始化链接。多网卡机器可能出现多条，选择当前设备能访问的那一条即可；只有完全检测不到内网地址时才回退到本机链接。发行单元用权限为 `0600` 的 `/run/kdae-panel/setup-url` 把本次链接交给安装脚本，不依赖 journald 文本格式；页面完成管理员创建后，该临时文件立即删除，初始化接口也永久关闭。

`0.0.0.0` 只是监听通配地址，不能直接放进浏览器，所以不会作为链接输出。通过 HTTPS 反向代理访问时，保留任一链接的 `/setup#bootstrap=...` 部分，并将协议和主机替换为实际面板地址；URL 片段不会发送给反向代理或写入访问日志。局域网直连是明文 HTTP，只适合可信内网；跨不可信网络请使用 HTTPS 或 SSH 隧道。

升级不会覆盖已有的 `/etc/kdae-panel/kdae-panel.env`。旧安装若要采用新默认值，需要手动把 `KDAE_PANEL_LISTEN` 改为 `0.0.0.0:2023` 后重启。

## 配置项

编辑 `/etc/kdae-panel/kdae-panel.env` 后执行：

```bash
sudo systemctl restart kdae-panel
```

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `KDAE_PANEL_LISTEN` | `0.0.0.0:2023` | HTTP 监听地址；默认接受本机与所有 IPv4 网卡的连接 |
| `KDAE_PANEL_BOOTSTRAP_TOKEN` | 空 | 一次性初始化链接的根凭证；留空时启动自动生成，发行单元会写入临时交接文件并同时记录 `setup_url` 日志 |
| `KDAE_PANEL_TRUSTED_PROXIES` | `127.0.0.0/8,::1/128` | 可以转发客户端地址和协议的代理 CIDR，逗号分隔 |
| `KDAE_PANEL_DAE_BINARY` | `/usr/bin/dae` | dae 二进制路径 |
| `KDAE_PANEL_DAE_CONFIG` | `/etc/dae/config.dae` | dae 入口配置 |
| `KDAE_PANEL_SERVICE_NAME` | `dae` | systemd 单元名（procd 后端下是 init 脚本名） |
| `KDAE_PANEL_SERVICE_BACKEND` | `auto` | 服务后端：`auto`（存在 `/sbin/procd` 即判为 procd，否则 systemd）、`systemd`、`procd`；也可用 `--service-backend` 命令行参数指定，选中结果会记入启动日志并由 `GET /api/v1/health` 的 `backend` 字段暴露 |
| `KDAE_PANEL_SYSTEMCTL` | `/usr/bin/systemctl` | systemctl 路径（仅 systemd 后端使用） |
| `KDAE_PANEL_JOURNALCTL` | `/usr/bin/journalctl` | journalctl 路径（仅 systemd 后端使用） |
| `KDAE_PANEL_DATABASE` | `/var/lib/kdae-panel/panel.db` | 认证数据库 |
| `KDAE_PANEL_BACKUP_DIR` | `/var/lib/kdae-panel/backups` | 自动备份与手动配置存档目录；存档名称和备注位于对应的 `.meta.json` 文件 |
| `KDAE_PANEL_SCHEDULE_FILE` | `/var/lib/kdae-panel/schedule.json` | 订阅自动刷新的设置与上次执行时间 |
| `KDAE_PANEL_INSTALL_STATE_FILE` | `/var/lib/kdae-panel/dae-install.json` | dae 版本安装记录，同目录还存放回滚点与 `dae-versions/` 本地版本库 |
| `KDAE_PANEL_GITHUB_TOKEN_FILE` | `/var/lib/kdae-panel/github-token` | 设置页保存 GitHub API Token 的独立文件，权限 `0600` |
| `KDAE_PANEL_GITHUB_TOKEN` | 空 | 可选 GitHub API Token；非空时优先于设置页文件且不能从 UI 修改，只需公开仓库只读权限 |
| `KDAE_PANEL_ENABLE_DAE_INSTALL` | `true` | 允许通过面板首次安装、升级与切换 dae 版本 |
| `KDAE_PANEL_GEO_STATE_FILE` | `/var/lib/kdae-panel/geo-update.json` | geo 数据更新记录 |
| `KDAE_PANEL_GEO_SCHEDULE_FILE` | `/var/lib/kdae-panel/geo-schedule.json` | geo 自动更新的设置与上次执行时间 |
| `KDAE_PANEL_GEO_SOURCES_FILE` | `/var/lib/kdae-panel/geo-sources.json` | 自定义 geo 来源，权限 `0600` |
| `KDAE_PANEL_ENABLE_GEO_UPDATE` | `true` | 旧版启动参数兼容项；Geo 管理现已始终可用 |
| `KDAE_PANEL_DISABLE_UPDATE_CHECK` | `false` | 关闭面板自身的新版本检查（检查只读取本仓库 releases/latest 的 tag，结果缓存 6 小时） |
| `KDAE_PANEL_ENABLE_SELF_UPDATE` | `true` | 面板一键升级的初始值；设置页保存过选择后以 UI 偏好为准 |
| `KDAE_PANEL_BACKUP_FILE` | `/var/lib/kdae-panel/kdae-panel.previous` | 自升级保留的上一版面板二进制 |
| `KDAE_PANEL_SESSION_TTL` | `12h` | 会话绝对有效期 |
| `KDAE_PANEL_SECURE_COOKIE` | `false` | Cookie 是否仅允许 HTTPS |

### dae 版本管理

新安装默认开启，发行单元已经允许写入默认二进制目录 `/usr/bin` 和服务单元目录 `/etc/systemd/system`，因此可以直接完成首次安装、升级与版本切换。dae 若实际位于其他目录，先用 `systemctl show dae --property=ExecStart` 确认路径，再通过 `systemctl edit kdae-panel` 把该目录加入 `ReadWritePaths`。

版本列表、官方 Release 元数据和 kdae Actions 产物摘要依赖 GitHub API。面板会将 JSON 元数据缓存 10 分钟、合并相同的并发请求，并在上游短暂限流时沿用最近一次成功结果；从列表直接安装时还会复用已经核验过的 Release/run 信息。匿名调用仍受同一出口 IP 每小时 60 次限制，共享公网 IP 或频繁管理多台机器时，建议在「面板设置 → GitHub API」填写只读 Token，认证额度通常为每用户每小时 5000 次。Token 只保存于服务器，不会回传前端；也可以通过 `KDAE_PANEL_GITHUB_TOKEN` 交给部署系统管理。

不需要版本管理的部署可以把 `KDAE_PANEL_ENABLE_DAE_INSTALL` 改为 `false`，并用 systemd drop-in 收紧上述写目录。允许写 root 的可执行文件和服务单元意味着面板缺陷可能升级为任意代码执行，这是默认便利性所接受的权限代价。

首次安装会写入可执行文件、geo 数据、服务单元，以及一份带默认 DNS 但不声明网卡的种子配置（仅在配置不存在时）。**它不会自动启动 dae**——请先在配置管理页写好规则再手动启动，否则透明代理可能切断你当前的连接。已存在的服务单元与配置一律不覆盖；旧配置缺少 `dns` 节时，配置页面只生成待保存草稿，不会在打开页面时静默改磁盘。

版本页也可以卸载 dae。确认框分别提供“同时删除主配置文件”和“同时删除面板可见的全部 geo 数据副本”两个选项，默认都不勾选；因此常规卸载只删除面板管理的 dae 可执行文件、标准路径下的 systemd 单元和版本回滚记录，配置、订阅与 geo 数据默认保留。选择删除的数据会进入同一个可回滚事务。受面板沙箱隐藏的 `/root/.local/share/dae` 无法代为删除，界面会明确说明。为避免误删包管理器或用户手工维护的程序，没有面板安装记录、二进制摘要已经漂移，或服务单元不在标准路径时，自动卸载会被拒绝。

版本切换下载的二进制缓存在安装状态文件同目录的 `dae-versions/`。缓存不会随 dae 卸载而删除，便于稍后重新安装；可在版本表格逐个清理。卸载 kdae-panel 时，默认仍保留该目录，`KDAE_PANEL_PURGE=true` 的清除模式会随 `/var/lib/kdae-panel` 一并移除默认位置的数据。若安装状态文件被改到默认数据目录之外，缓存也会跟随到那个目录，清除面板前应先在版本页删除或自行处理。

只想升级已有的 dae 时不需要这一步。若你更习惯官方工具，[dae-installer](https://github.com/daeuniverse/dae-installer) 依然可用，两者互不冲突。

### 面板一键自升级

新安装默认开启。有新版本时，界面顶部会显示「立即升级」按钮：面板下载发布包、比对 `SHA256SUMS`、用新二进制自证能在本机运行，然后替换自己并请求 systemd 重启。设置页的「允许一键升级」开关可以随时关闭或重新开启，不需要 SSH；普通的新版本提示不会随之关闭。

界面选择原子写入 `/var/lib/kdae-panel/self-update.json`，重启后保持，并优先于 `KDAE_PANEL_ENABLE_SELF_UPDATE` 给出的初始值。这样从旧版本升级、环境文件仍写着 `false` 的实例，也可以直接在新版本横幅中选择「启用并升级」，以后不再需要改环境文件。卸载面板时该偏好默认随其他数据保留，`KDAE_PANEL_PURGE=true` 才会清除。

**这个开关的代价要说透**：它让面板能改写自己的可执行文件，因此面板本身的任何可利用缺陷都能被写成持久化的任意代码——严重程度不低于默认开启的 dae 版本管理。不接受这一权限时可在设置页关闭；新版本提醒仍然可用，也可以重跑一键部署命令完成整包升级。

**没有自动回滚。** 被替换、被重启的是当前进程自己，一旦 systemd 把它停掉就无从执行补救。风险因此前移：替换之前先运行新二进制的 `-version` 让它自证能在这台机器上跑起来，版本对不上或跑不起来就中止，原文件一个字节都不动。替换时把上一版复制到 `KDAE_PANEL_BACKUP_FILE`，万一新版本起不来：

```bash
rollback=$(sudo mktemp /usr/bin/.kdae-panel-rollback.XXXXXX)
trap 'sudo rm -f "$rollback"' EXIT
sudo install -m0755 /var/lib/kdae-panel/kdae-panel.previous "$rollback"
sudo mv -f "$rollback" /usr/bin/kdae-panel
trap - EXIT
sudo systemctl restart kdae-panel
```

升级期间面板会短暂无法访问（通常几秒）。**dae 与代理流量完全不受影响**——面板只是管理界面，它的重启不碰 dae 的任何东西。

一键自升级只替换面板二进制（前端已嵌入其中），不会扩大权限去改写 `/usr/share`、
systemd 单元或 env 模板；这些配套文件仍属于最近一次完整安装的版本。Release notes 若注明单元或
脚本有变更，请重跑一键部署完成整包升级。卸载时优先使用上面的联网命令获取最新脚本。

### Geo 数据管理

侧栏的「Geo 数据」是独立入口，不需要开启 dae 版本管理，也不再要求修改环境文件。旧部署残留的 `KDAE_PANEL_ENABLE_GEO_UPDATE=false` 只作为兼容参数接受，不会隐藏页面。

通常不需要额外放宽 `ReadWritePaths`：面板更新的是 dae **当前实际读取**的那份 geo，而它多半就在配置目录（已经可写）。若你的 geo 在 `/usr/local/share/dae`（例如用 `dae-installer` 装的），界面会明确提示该目录不可写以及要追加哪一条。

界面内置两个来源：

| 来源 | 仓库 | 适合谁 |
|---|---|---|
| Loyalsoldier 规则集 | `Loyalsoldier/v2ray-rules-dat` | 想要更细分类（`geosite:gfw`、`geosite:greatfire` 等）、每天更新 |
| v2fly 官方 | `v2fly/geoip` + `v2fly/domain-list-community` | 想与 dae 发布包保持同一套数据，切过去不会改变现有规则的含义 |

两点务必知悉：

- **切换来源会改变路由行为。** 两套规则集里同名分类所含的域名不同，切换后 `geosite:` 开头的路由规则匹配的范围会变，而 dae 不会因此报错。界面只在切换时警告，沿用同一来源不会反复打扰。
- **dae 运行时会触发 `dae reload <MainPID>`。** PID 直接取自 systemd，不依赖 `/var/run/dae.pid`。新连接不受影响，但进行中的长连接（大文件下载、SSH、串流）最多约 10 秒后可能被断开。若 dae 不接受新数据，面板会自动还原旧文件并重新加载。dae 未运行时只更新文件，下一次启动会读取新数据；由于此时无法通过 reload 检查配置引用的 Geo 分类，若新数据仍缺少分类，下一次启动仍会失败。

「来源管理」可以添加多组自定义来源，分别填写 `geoip.dat`、`geosite.dat` 与各自的 SHA-256 校验文件直链。只接受公网 HTTPS；每次重定向都重新检查解析地址，自定义下载不携带 GitHub Token，也不能关闭校验。链接可能带查询参数，因此配置单独保存在权限 `0600` 的 `KDAE_PANEL_GEO_SOURCES_FILE`，不会进入配置历史或普通日志。

若路由规则引用当前数据里不存在的分类，dae 会在启动时报类似 `country code ... not found in .../geoip.dat`，但 `dae validate` 仍然成功。面板会从 Geo 更新的 reload 输出，或启动、重启和版本切换后的近期日志中直接指出缺失的 `geoip:` / `geosite:` 分类；此时应在 Geo 数据页更新或切换到包含该分类的来源，或者修改路由规则。切换二进制本身不能修复数据分类缺失。

## HTTPS

不建议直接将面板的 HTTP 端口暴露到公网。保持监听回环地址，并使用反向代理提供 TLS。

Nginx 示例：

```nginx
server {
    listen 443 ssl http2;
    server_name panel.example.com;

    ssl_certificate     /etc/letsencrypt/live/panel.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/panel.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:2023;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

面板只接受可信代理 CIDR 转发的地址和协议。可信代理报告 HTTPS 时，Cookie 会自动增加 `Secure` 并发送 HSTS；仍建议显式设置：

```bash
KDAE_PANEL_SECURE_COOKIE=true
```

面板的同源检查同时比较浏览器 `Origin` 的协议和主机，因此反向代理必须传递原始 Host 和正确的 `X-Forwarded-Proto`。不要信任公网来源的转发头。

## 权限模型

当前 systemd 单元以 root 运行，因为面板需要同时完成以下操作：

- 原子写入 `/etc/dae`；
- 向 dae 进程发送重载或暂停信号；
- 通过 systemd 启停服务；
- 读取系统日志和 sysdump。

单元通过 `ProtectSystem`、`ProtectHome`、`NoNewPrivileges`、能力白名单、地址族限制和只读系统路径降低暴露面。默认只保留 `CAP_KILL`、`CAP_NET_ADMIN`，可写路径仅为 `/etc/dae` 和 `/var/lib/kdae-panel`；`/run` 只读即可连接 systemd socket 并读取 dae 状态文件。`ProtectProc=invisible` 会隐藏其他进程，但保留 `/proc/sys/net`，供 dae sysdump 采集 sysctl。不要移除登录认证后对外开放，也不要让其他用户写入环境文件、数据库或面板二进制。

## 升级面板

```bash
git pull --ff-only
npm ci --prefix web
make build
sudo ./scripts/install.sh
```

数据库使用向前兼容的幂等迁移；安装脚本保留现有账户和环境配置。

## 升级 dae

建议先下载新二进制到临时位置：

```bash
/tmp/dae-new --version
sudo /tmp/dae-new validate -c /etc/dae/config.dae
```

校验通过后再替换 dae 并重启：

```bash
sudo install -m0755 /tmp/dae-new /usr/bin/dae
sudo systemctl restart dae
```

刷新面板后，它会重新执行 `--help` 和 `export outline`，自动读取新版本能力。生产环境仍应保留旧二进制，以便遇到上游破坏性变化时回滚。

## 卸载

一键卸载（信任边界是一键部署的子集：只有 `raw.githubusercontent.com` 上 main 分支的脚本本身，不涉及 Release 资产，因此也没有校验和环节）：

以下命令同样须在 root shell 中执行：

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/tuoro/kdae-panel/main/scripts/uninstall.sh)"
```

安装时（无论一键部署还是源码安装）也会在本地落一份等效脚本，离线可用（早期安装的机器没有这份副本，重跑一次安装即可补上）：

```bash
sudo bash /usr/share/kdae-panel/uninstall.sh
```

源码检出还在的话，`sudo ./scripts/uninstall.sh` 同样等效。

默认保留 `/etc/kdae-panel` 与 `/var/lib/kdae-panel`（配置、账户数据库、配置备份、dae 回滚副本），但主服务单元及其 `kdae-panel.service.d` override 会一并移除，避免重装后意外恢复高权限。确认数据不再需要后可用清除模式重跑。本地副本会随普通卸载一起移除，因此重跑要用一键命令（或源码检出）：

```bash
sudo KDAE_PANEL_PURGE=true bash -c "$(curl -fsSL https://raw.githubusercontent.com/tuoro/kdae-panel/main/scripts/uninstall.sh)"
```

清除模式会按 env 文件里配置的实际路径删除数据（数据库、订阅刷新、geo 更新与安装状态文件即使被挪到默认目录之外也会被找到），随后删除上述两个目录本身。唯一的例外是备份目录：它需要以 root 递归删除，取值又来自 env 配置，因此只有位于默认数据目录 `/var/lib/kdae-panel` 之内时才自动删，挪到别处的会打印路径请你确认后手工处理——env 里一个手滑的取值不该变成 root 下的 `rm -rf`。

**任何模式都不触碰 dae**：它的服务、二进制、`/etc/dae` 配置与 geo 数据原样保留；`/etc/dae` 下的 geo 文件（无论是面板首次安装写入还是自行放置的）也会留下（dae 还在用），脚本会把它们的位置与大小如实列出。清除模式删掉 env 文件后，重装会使用发布包内的默认配置——自定义过 env 的话请先自行备份一份。

## 排障

```bash
systemctl status kdae-panel
journalctl -u kdae-panel -n 200 --no-pager
curl http://127.0.0.1:2023/api/v1/health
/usr/bin/dae export outline
/usr/bin/dae validate -c /etc/dae/config.dae
```

若服务操作返回权限错误，先执行：

```bash
systemd-analyze security kdae-panel.service
systemctl cat kdae-panel.service
```

某些发行版或自定义 dae 可能需要调整 systemd 单元的能力白名单；修改前应明确缺失的具体系统调用或能力，避免直接移除所有沙箱设置。
