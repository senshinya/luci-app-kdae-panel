#!/bin/sh
# 算出编进面板二进制的版本号（go build -ldflags "-X main.version=…"）。
#
#   scripts/panel-version.sh          # 从 git 推
#   scripts/panel-version.sh v1.2.0   # 发布：tag 原样（参数为空等同于不给）
#
# Makefile 与 .github/workflows/openwrt.yml 共用这一份实现。两处各写各的时，快照
# 版本号的形状漂移过——而症状只在用户的路由器上看得见。
#
# 面板拿这个版本号和最新发布 tag 比对（internal/app/panelupdate_handlers.go 的
# parseSemver 与 versionBehind），因此它必须是 semver，而且不能低于所基于的 tag：
#
#   - 认不出（分支名 "main"、PR 的 "123/merge"）→ 静默放弃比对，设置页显示"本部署
#     不做新版本检查"，而检查能力其实好好地注册着。
#   - 排在 tag 之前（写死的 v0.0.1-git<n>.<hash>，或 git describe 原样的
#     v1.0.0-8-g<hash>——连字符后的部分都会被当成预发布段）→ 快照机器上横幅恒亮
#     "面板有新版本 v1.0.0"，而那一版比机器上正在跑的代码还旧。
#
# 所以领先 tag 的提交数与短哈希写成 build metadata（+git<n>.<hash>）：semver 规定它
# 不参与优先级比较，快照因此与所基于的 tag 平级，等真出了新发布仍会如实提示。
#
# 软件包版本（KDAE_PANEL_VERSION）是另一回事，不要拿这里的产出去填：opkg 按 dpkg
# 规则比、apk 有自己的版本文法，两边都不认 semver，详见流水线里那一大段注释。
set -eu

release="${1-}"
if [ -n "${release}" ]; then
	version="v${release#v}"
else
	# KDAE_PANEL_DESCRIBE 只为测试留：不必真造一个带 tag 的仓库就能钉住转换规则。
	describe="${KDAE_PANEL_DESCRIBE-}"
	if [ -z "${describe}" ]; then
		describe="$(git describe --tags --dirty 2>/dev/null)" || {
			echo "panel-version: 取不到可达的 tag（CI 里需要 checkout 带 fetch-depth: 0）" >&2
			exit 1
		}
	fi
	# v1.0.0-8-g5df15b7[-dirty] → v1.0.0+git8.5df15b7[.dirty]
	# v1.0.0[-dirty]            → v1.0.0[+dirty]
	version="$(printf '%s' "${describe}" | sed -E \
		-e 's/-([0-9]+)-g([0-9a-f]+)/+git\1.\2/' \
		-e 's/[+]([^+]*)-dirty$/+\1.dirty/' \
		-e 's/-dirty$/+dirty/')"
fi

# 版本号写错的症状发生在用户的路由器上（一个不声不响关掉的检查，或一条消不掉的
# 升级横幅），所以在这里就失败掉。
printf '%s' "${version}" |
	grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?([+][0-9A-Za-z.-]+)?$' || {
	echo "panel-version: ${version} 不是 semver，面板会据此关掉新版本检查" >&2
	exit 1
}
# 上面那条拦不住"转换没生效"：git describe 的原样输出 vX.Y.Z-<n>-g<hash> 本身就是
# 合法 semver，只是那个领先段会被当成预发布，排到 tag 之前——正是要修的那个 bug。
# 上面的 sed 只认小写十六进制哈希，认不出时宁可在这里失败，也不要漏一个更旧的版本号出去。
if printf '%s' "${version}" | grep -Eq -- '-[0-9]+-g[0-9A-Za-z]+'; then
	echo "panel-version: ${version} 还带着 git describe 的领先段，它会排在所基于的 tag 之前" >&2
	exit 1
fi
printf '%s\n' "${version}"
