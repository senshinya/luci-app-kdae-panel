include $(TOPDIR)/rules.mk

PKG_NAME:=kdae-panel

# 版本号必须能被外部覆盖。opkg 只按包名与版本判断该不该动手：装同名同版本的
# ipk 会打印 "installed in root is up to date"、退出码 0、什么都不做。写死
# 1.0.0-1 的话，每次 Release 附的都是元数据一致而内容不同的包，用户以为升级
# 成功、实际还跑着旧二进制——而 procd 部署刻意没有面板自升级，ipk 是唯一通道。
KDAE_PANEL_VERSION?=
PKG_VERSION:=$(if $(KDAE_PANEL_VERSION),$(KDAE_PANEL_VERSION),0.0.0)
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
  # 架构闸门列的是 target 的 Target-Arch 符号（target-metadata.pl 里 select $(target->arch)）：
  # x86/64 是 x86_64，armsr/armv8 是 aarch64，x86/generic 是 i386。包内容本身与架构无关
  # （装的是 CI 交叉编译好的静态二进制），闸门只是挡住没出过对应二进制的架构。
  # 闸门不匹配时 make defconfig 会静默丢掉 CONFIG_PACKAGE_kdae-panel，因此 CI 在
  # defconfig 之后立刻断言配置项还在——否则报错会推迟到 compile，离原因很远。
  DEPENDS:=@(x86_64||aarch64||i386) +kmod-sched-core +kmod-sched-bpf +kmod-veth \
	+kmod-nft-bridge +kmod-xdp-sockets-diag +ca-bundle
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
# 必须显式启动。升级同一个包时 opkg 先跑旧包的 prerm（stop + disable）再跑新包的
# postinst，只 enable 的净效果是每次升级后面板都停着——而 procd 部署刻意没有面板
# 自升级，ipk 是唯一的升级通道，停在这里的面板没有别的办法自己回来。
# 重复执行无害：init 脚本内部有 enabled 开关守卫，关掉时 start 不会拉起实例。
/etc/init.d/kdae-panel start
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
