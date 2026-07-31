include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-kdae-panel

# 与 kdae-panel 包同源同版本：两个包一起发布、一起安装，版本线分叉只会让
# "装的是哪一版"这个问题多一份要对的账。理由见 kdae-panel/Makefile。
KDAE_PANEL_VERSION?=
PKG_VERSION:=$(if $(KDAE_PANEL_VERSION),$(KDAE_PANEL_VERSION),0.0.0)
PKG_RELEASE:=1

PKG_LICENSE:=AGPL-3.0-only
PKG_MAINTAINER:=senshinya

LUCI_TITLE:=LuCI support for kdae-panel
LUCI_DESCRIPTION:=LuCI 入口与配置页：服务状态、一次性初始化链接、UCI 设置。
LUCI_DEPENDS:=+kdae-panel
LUCI_PKGARCH:=all

include $(TOPDIR)/feeds/luci/luci.mk

# call BuildPackage - OpenWrt buildroot signature
