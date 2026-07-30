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
