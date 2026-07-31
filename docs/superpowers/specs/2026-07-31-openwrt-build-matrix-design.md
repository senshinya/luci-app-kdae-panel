# OpenWrt 软件包构建矩阵设计

把 `openwrt.yml` 从「immortalwrt 24.10.4 单一 x86_64 ipk」扩成官方 OpenWrt SDK 的
2 版本 × 3 架构矩阵，架构覆盖对齐 [QiuSimons/luci-app-daed](https://github.com/QiuSimons/luci-app-daed/releases)。

## 目标与非目标

**目标**

- 产出 OpenWrt 24.10（ipk）与 25.12（apk）两条版本线的软件包。
- 架构覆盖 `x86_64`、`aarch64_generic`、`i386_pentium4`。
- 包 Makefile 解除只允许 x86_64 的架构闸门。
- 校验步骤对两种包格式都成立，不是只对 ipk 断言、apk 裸奔。

**非目标**

- 不再用 immortalwrt SDK。同版本同 pkgarch 的包在 immortalwrt 上一样能装（pkgarch
  一致、kmod 依赖名一致，依赖在设备上按本机软件源解析），维护两套 SDK 只会让 CI 时长
  和 Release 页资产数翻倍。
- 不扩到 armv7 等 32 位 ARM：面板与 dae 的 32 位 ARM 组合没有验证过，参考项目也没发。
- 不动 systemd 路径的 `scripts/build-release.sh`，它按 GOARCH 出 tar.gz，与 ipk/apk 无关。

## 一、构建矩阵

| openwrt | SDK target | pkgarch | GOARCH | 格式 | gcc |
|---|---|---|---|---|---|
| 24.10.8 | x86/64 | `x86_64` | amd64 | ipk | 13.3.0 |
| 24.10.8 | armsr/armv8 | `aarch64_generic` | arm64 | ipk | 13.3.0 |
| 24.10.8 | x86/generic | `i386_pentium4` | 386 | ipk | 13.3.0 |
| 25.12.5 | x86/64 | `x86_64` | amd64 | apk | 14.3.0 |
| 25.12.5 | armsr/armv8 | `aarch64_generic` | arm64 | apk | 14.3.0 |
| 25.12.5 | x86/generic | `i386_pentium4` | 386 | apk | 14.3.0 |

SDK URL 由这几列拼出：

```
https://downloads.openwrt.org/releases/<openwrt>/targets/<target>/openwrt-sdk-<openwrt>-<target 用 - 连>_gcc-<gcc>_musl.Linux-x86_64.tar.zst
```

`fail-fast: false`：一个架构编不出来不该让其余五个白跑。

## 二、换发行版暴露出来的两处依赖问题

改用官方 SDK 后，六个 job 全部在 `make defconfig` 之后丢掉 `CONFIG_PACKAGE_kdae-panel`。
二分定位（逐条依赖单独试一遍 defconfig）的结论：

- **`+kmod-xdp-sockets-diag` 必须去掉。** 官方 OpenWrt 的 24.10.8 与 25.12.5 在 x86/64、
  armsr/armv8、x86/generic 的 kmods 目录里都没有这个包，ImmortalWrt 才有（它的 `dae` 包也
  依赖它），所以旧流水线一直没暴露。这不是 SDK 挑剔——依赖一个官方源里不存在的包，用户在
  官方 OpenWrt 上 `opkg install` 同样会失败。它是 AF_XDP 套接字的诊断模块，dae 的转发路径
  不用它。另外四个 kmod 与架构闸门本身都没问题（各自单独试都能选上）。
- **包要挂成 feed，不能拷进 `package/`。** 拷进去的包不经过 feeds，依赖也就没人替它装：
  官方 SDK 的 `package/` 里没有 `ca-bundle`，`+ca-bundle` 指向不存在的包。改成
  `src-link` 一个本地 feed + `feeds install -a -p kdae`，`scripts/feeds` 的 `install_src`
  会顺着 `DEPENDS` 递归把依赖从 base/luci feed 装进 `package/feeds/`。这也是
  QiuSimons/luci-app-daed 用的那个 action 的做法。

两处的共同症状都是 defconfig 静默丢包，日志里只有一条不起眼的 `has a dependency on ...,
which does not exist`。CI 在 defconfig 之后立刻断言配置项还在，就是为了让这类故障停在这里。

## 三、包 Makefile 的架构闸门

`kdae-panel` 现在是 `DEPENDS:=@x86_64 ...`，改成 `@(x86_64||aarch64||i386)`。

这三个是 target 的 `Target-Arch` 符号（`scripts/target-metadata.pl` 里 `select $target->{arch}`），
armsr/armv8 是 `aarch64`，x86/generic 是 `i386`。闸门不匹配时 `make defconfig` 会**静默**把
`CONFIG_PACKAGE_kdae-panel=m` 丢掉，后面的 `make package/kdae-panel/compile` 才报一个和真实
原因隔了十万八千里的错。因此 CI 在 defconfig 之后立刻断言两个包的配置项还在。

包内容与架构无关（一个已经交叉编译好的静态二进制被 `$(CP)` 进去），所以不需要为每个架构
准备不同的打包逻辑，只需要每个架构一份对应 GOARCH 的二进制。

## 四、版本号按包格式分叉

apk 的版本语法是 `digit{.digit}...{letter}{_suf{#}}...{~hash}{-r#}`（apk-tools `src/version.c`），
`+` 不在其中，现有的 `0.0.1+<commit数>.<短哈希>` 只能给 ipk 用。

| 触发 | ipk 版本 | apk 版本 |
|---|---|---|
| release | `<tag 去 v>` | `<tag 去 v>` |
| 其它 | `0.0.1+<commit数>.<短哈希>` | `0.0.1_git<commit数>~<短哈希>` |

apk 侧的排序性质与 ipk 侧一致：`_git` 是后置后缀，`0.0.1_git124` 排在 `0.0.1` 之后但仍远低于
任何 `0.8.x` 正式版；`<commit数>` 是 `_suf{#}` 的数字段，按数值比较，因此后一次 push 一定算升级。

Release 事件下两种格式拿到的是同一个纯 tag 版本号，不分叉。

## 五、产物命名

apk 的文件名里**没有架构**（`package-pack.mk` 里是 `<name>-<version>.apk`），三个架构的
`kdae-panel-0.8.8-r1.apk` 会在 Release 页撞名。所有产物统一补后缀，与参考项目一致：

- `kdae-panel_<ver>-r1_<pkgarch>-openwrt-24.10.ipk`
- `kdae-panel-<ver>-r1-<pkgarch>-openwrt-25.12.apk`
- `luci-app-kdae-panel_<ver>-r1_all-openwrt-24.10.ipk`
- `luci-app-kdae-panel-<ver>-r1-openwrt-25.12.apk`

LuCI 包是 `all`/noarch，三个架构的 job 出的是同一份内容，只从每条版本线的 `x86_64` job 上传，
否则 Release 页会有三份同名资产互相覆盖。

## 六、校验

ipk 沿用现在的断言（解 tar 看 control 与文件表），新增一条 `Architecture` 必须等于本 job 的 pkgarch。

apk 是 ADB 二进制格式，`tar` 打不开，改用 SDK 自带的 `staging_dir/host/bin/apk`：

- `apk adbdump --format json <pkg>` 断版本、`arch`、依赖；
- `apk --allow-untrusted extract --destination <dir> <pkg>` 后断文件路径。

**apk 包不带 `Conflicts`**：OpenWrt 的 apk 打包路径（`package-pack.mk` 的 `apk mkpkg`）根本不传
这个字段，`CONFLICTS:=dae` 在 25.12 上会被静默丢掉，所以那条断言只对 ipk 生效。25.12 上与官方
`dae` 包的互斥退化成文件冲突——两个包都装 `/etc/init.d/dae`，apk 会拒绝覆盖。这一点写进
`docs/openwrt.md`，不用 `!dae` 塞进 `DEPENDS` 去 hack（会干扰 OpenWrt 的依赖扫描）。

## 七、Release 附加

`download-artifact` 改用 `pattern: kdae-panel-pkg-*` + `merge-multiple: true`，把六个 job 的产物
一起附到 Release。写权限仍只给发布路径上的那个 job。

**触发方式必须改成显式调用。** `openwrt.yml` 原来只挂 `release: published`，而本仓库的 Release 是
`release.yml` 用 `GITHUB_TOKEN` 创建的——这种 release 不会触发新的工作流，那个 job 从来没有、
也永远不会运行。装机验证（`release-smoke.yml`）当初踩的是同一个坑，解法也照搬：`openwrt.yml`
增加 `workflow_call`（输入 `version`），`release.yml` 在 Release 建好后作为 `packages` job 调用它。
`release: published` 触发器保留，供网页上手工建 release 的场景。

预发布 tag 的版本后缀两种格式都要改写：opkg 按 dpkg 规则比较，`1.0.0-rc1` 排在 `1.0.0` **之后**，
装过 RC 的机器再装正式版会被判成降级，dpkg 里"排在前面"的记号是 `~`；apk 的文法里 `-` 只能引出
`-r<数字>` 修订号，`1.0.0-rc1` 是非法版本，`mkpkg` 会拒绝，它认的是 `_rc1`。因此 `v1.0.0-rc1`
分别落成 `1.0.0~rc1`（ipk）与 `1.0.0_rc1`（apk），两者都排在正式版之前。后缀要写成 apk 认识的
形式且不含点：`rc1`、`beta2`，不要 `rc.1`。

## 八、文档

`README.md` 的「OpenWrt / ImmortalWrt」段与 `docs/openwrt.md`：支持矩阵、`opkg install` 与
`apk add --allow-untrusted` 两套安装命令、immortalwrt 用同一批包的说明、25.12 上冲突声明的差异。

## 已知风险

- **`i386_pentium4` 的 dae 可用性未验证**：dae 官方有 `x86_32` 资产，面板的
  `internal/upstream/platform.go` 已映射 `386→x86_32`，但 32 位 x86 上 dae 的 eBPF 实际能否工作
  没有在真机验证过。参考项目发这个架构，跟随它。
- **apk 校验命令首次上 CI 才能验证**：`apk adbdump` / `apk extract` 的行为按 apk-tools 文档
  与 `package-pack.mk` 推定，本地无法预演（SDK 是 Linux x86_64 二进制）。
