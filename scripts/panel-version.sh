#!/bin/sh
# 算出面板与软件包的版本号。三种格式共用同一次 tag 推导与同一份解析。
#
#   scripts/panel-version.sh                    # 编进二进制的版本（semver）
#   scripts/panel-version.sh --format=ipk       # opkg 的包版本
#   scripts/panel-version.sh --format=apk       # apk 的包版本
#   scripts/panel-version.sh v1.2.0             # 发布：tag 原样（参数为空等同于不给）
#   scripts/panel-version.sh --format=ipk v1.2.0
#
# Makefile 与 .github/workflows/openwrt.yml 共用这一份实现。两处各写各的时，快照
# 版本号的形状漂移过——而症状只在用户的路由器上看得见。
#
# ── 三个比较域，一份来源 ────────────────────────────────────────────────
#
# semver 归面板自己用：拿它和最新发布 tag 比对（internal/app/panelupdate_handlers.go
# 的 parseSemver 与 versionBehind），决定要不要提示有新版本。
#
#   - 认不出（分支名 "main"、PR 的 "123/merge"）→ 静默放弃比对，设置页显示"本部署
#     不做新版本检查"，而检查能力其实好好地注册着。
#   - 排在 tag 之前（git describe 原样的 v1.0.0-8-g<hash>——连字符后的部分都会被当成
#     预发布段）→ 快照机器上横幅恒亮"面板有新版本 v1.0.0"，而那一版比机器上正在跑的
#     代码还旧。所以领先量写成 build metadata（+git<n>.<hash>）：semver 规定它不参与
#     优先级比较，快照因此与所基于的 tag 平级，等真出了新发布仍会如实提示。
#
# ipk 与 apk 归包管理器用，规则完全是另一套，且两边互不相同。它们要满足的硬约束
# 只有一条：**后构建的包一定装得到先构建的包上面**。opkg 和 apk 都拒绝降级安装，
# 而 LuCI 的软件包页面没有任何入口能传 --force-downgrade，一旦排序反了，用户在
# 网页上就再也装不上新包。因此三条链路都必须单调：
#
#   1.1.0  <  1.1.0+git1.<hash>  <  1.1.0+git7.<hash>  <  1.1.1        （ipk）
#   1.1.0  <  1.1.0_git1~<hash>  <  1.1.0_git7~<hash>  <  1.1.1        （apk）
#
# 快照基于**最新的 tag**而不是写死的 0.0.1：写死意味着装着正式发布的机器再装快照
# 会被判成降级，而那正是没有入口可救的那种情况。
#
# ipk 侧的依据是 dpkg 的比较规则：串尾排在任何字符（'~' 除外）之前，所以
# 1.1.0+git7 大于 1.1.0；"+git" 之后是数字段，按数值比较，领先量单调则包版本单调。
# 预发布 tag 用 '~'（dpkg 里唯一排在串尾之前的记号），1.1.0~rc1 因此低于 1.1.0。
#
# apk 侧的依据是 apk-tools src/version.c：后缀表里 git 声明在 NONE 之后
# （alpha/beta/pre/rc 在前，cvs/svn/git/hg/p 在后），而收尾比较只把
# suffix < SUFFIX_NONE 的判成更小，所以 1.1.0_git7 大于 1.1.0。领先量落在
# _suf{#} 的数字段上按数值比较。预发布用 _rc1 这种前置后缀，同样低于 1.1.0。
# apk 的版本文法不接受 '+'，短哈希只能挂在 '~' 引出的 commit 字段里，
# 所以两种格式必须分叉，不能共用一个串。
set -eu

format=semver
case "${1-}" in
--format=*)
	format="${1#--format=}"
	shift
	;;
esac
case "${format}" in
semver | ipk | apk) ;;
*)
	echo "panel-version: 未知格式 ${format}，只支持 semver、ipk、apk" >&2
	exit 1
	;;
esac

release="${1-}"

# base 是所基于的 tag（去掉前导 v），ahead 是领先它的提交数，hash 是短哈希。
# 发布构建 ahead 恒为 0 且没有 hash——那一版就是 tag 本身。
ahead=0
hash=""
dirty=""

if [ -n "${release}" ]; then
	base="${release#v}"
else
	# KDAE_PANEL_DESCRIBE 只为测试留：不必真造一个带 tag 的仓库就能钉住转换规则。
	describe="${KDAE_PANEL_DESCRIBE-}"
	if [ -z "${describe}" ]; then
		# 取"可达 tag 里版本号最高的那个"，而不是 git describe 的"距离最近的那个"。
		#
		# 这个仓库会合并上游 tuoro/kdae-panel，上游带着它自己那套版本号相互重叠的
		# tag（v1.0.2…v1.0.6）。合并之后它们同样可达，且往往比本仓库自己的 tag 更近。
		# 用最近的那个，一次上游合并就能让包版本从 1.1.0+gitN 掉回 1.0.6+gitM——
		# 后构建的包反而更小，装不上去，而这正是必须守住的那条约束。
		#
		# 只在正式 tag（vX.Y.Z）里挑：git 的 version sort 默认把 v1.1.0-rc1 排在
		# v1.1.0 之后，拿预发布当基准会让快照落到 1.1.0~rc1+gitN，反而低于已发布的
		# 1.1.0。仓库里只有预发布 tag 时才退而用它们。
		tag="$(git tag --merged HEAD --sort=-v:refname 2>/dev/null |
			grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n 1 || true)"
		if [ -z "${tag}" ]; then
			tag="$(git tag --merged HEAD --sort=-v:refname 2>/dev/null |
				grep -E '^v[0-9]+\.[0-9]+\.[0-9]+-' | head -n 1 || true)"
		fi
		if [ -z "${tag}" ]; then
			echo "panel-version: 找不到可达的 tag（CI 里需要 checkout 带 fetch-depth: 0）" >&2
			exit 1
		fi
		count="$(git rev-list --count "${tag}..HEAD")"
		short="$(git rev-parse --short HEAD)"
		describe="${tag}-${count}-g${short}"
		git diff --quiet HEAD 2>/dev/null || describe="${describe}-dirty"
	fi

	# vX.Y.Z[-pre][-<n>-g<hash>][-dirty] 拆成四段。
	remainder="${describe}"
	case "${remainder}" in
	*-dirty)
		dirty=1
		remainder="${remainder%-dirty}"
		;;
	esac
	case "${remainder}" in
	*-[0-9]*-g[0-9a-f]*)
		hash="${remainder##*-g}"
		rest="${remainder%-g*}"
		ahead="${rest##*-}"
		base="${rest%-*}"
		;;
	*)
		base="${remainder}"
		;;
	esac
	base="${base#v}"
fi

# 预发布段（1.2.0-rc1 的 rc1）在三个域里的记号各不相同，先摘出来单独处理。
prerelease=""
case "${base}" in
*-*)
	prerelease="${base#*-}"
	base="${base%%-*}"
	;;
esac

case "${format}" in
semver)
	version="v${base}"
	[ -n "${prerelease}" ] && version="${version}-${prerelease}"
	# 领先量与短哈希写成 build metadata：semver 规定它不参与优先级比较，快照
	# 因此与所基于的 tag 平级，而不是排到它前面。
	if [ "${ahead}" -gt 0 ] && [ -n "${hash}" ]; then
		version="${version}+git${ahead}.${hash}"
		[ -n "${dirty}" ] && version="${version}.dirty"
	elif [ -n "${dirty}" ]; then
		version="${version}+dirty"
	fi
	;;
ipk | apk)
	# 脏工作区不打包：包版本里没有位置如实表达"这不是某个提交的内容"，
	# 而一个版本号对不上内容的包发出去之后无从追查。CI 永远是干净的。
	if [ -n "${dirty}" ]; then
		echo "panel-version: 工作区有未提交的改动，拒绝据此生成 ${format} 包版本" >&2
		exit 1
	fi
	version="${base}"
	if [ -n "${prerelease}" ]; then
		# 预发布 tag 的后缀要写成 apk 认识的形式且不含点：rc1、beta2，不要 rc.1。
		# apk 的文法里 '-' 只能引出 -r<数字> 修订号，1.0.0-rc1 是非法版本，mkpkg
		# 会直接拒绝；'.' 也不在 _suf 的字母段里。
		case "${prerelease}" in
		*[!0-9A-Za-z]*)
			echo "panel-version: 预发布段 ${prerelease} 含有 apk 不接受的字符，请写成 rc1 这种形式" >&2
			exit 1
			;;
		esac
		if [ "${format}" = "apk" ]; then
			version="${version}_${prerelease}"
		else
			# dpkg 里 '~' 是唯一排在串尾之前的记号，1.0.0~rc1 因此低于 1.0.0；
			# 直接用 '-rc1' 会排到 1.0.0 之后，装过 RC 的机器再装正式版被判成降级。
			version="${version}~${prerelease}"
		fi
	fi
	# 非发布构建恒定带上领先段，ahead 为 0 也不省。省掉的话，正好落在 tag 上的
	# 那次手动构建会产出与正式发布同名同版本的包，opkg 打印 "up to date"、
	# 退出码 0、什么都不做，用户以为装上了却还跑着旧包。
	if [ -n "${hash}" ]; then
		if [ "${format}" = "apk" ]; then
			version="${version}_git${ahead}~${hash}"
		else
			version="${version}+git${ahead}.${hash}"
		fi
	fi
	;;
esac

# 版本号写错的症状发生在用户的路由器上（一个不声不响关掉的检查，或一条在网页上
# 消不掉的"拒绝降级"），所以在这里就失败掉。
case "${format}" in
semver)
	printf '%s' "${version}" |
		grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?([+][0-9A-Za-z.-]+)?$' || {
		echo "panel-version: ${version} 不是 semver，面板会据此关掉新版本检查" >&2
		exit 1
	}
	# 上面那条拦不住"转换没生效"：git describe 的原样输出 vX.Y.Z-<n>-g<hash> 本身就是
	# 合法 semver，只是那个领先段会被当成预发布，排到 tag 之前——正是要修的那个 bug。
	# 上面的解析只认小写十六进制哈希，认不出时宁可在这里失败，也不要漏一个更旧的版本号出去。
	if printf '%s' "${version}" | grep -Eq -- '-[0-9]+-g[0-9A-Za-z]+'; then
		echo "panel-version: ${version} 还带着 git describe 的领先段，它会排在所基于的 tag 之前" >&2
		exit 1
	fi
	;;
ipk)
	printf '%s' "${version}" |
		grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(~[0-9A-Za-z]+)?([+]git[0-9]+\.[0-9a-f]+)?$' || {
		echo "panel-version: ${version} 不是本流水线约定的 ipk 版本形状" >&2
		exit 1
	}
	;;
apk)
	printf '%s' "${version}" |
		grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(_[0-9A-Za-z]+)?(_git[0-9]+~[0-9a-f]+)?$' || {
		echo "panel-version: ${version} 不是本流水线约定的 apk 版本形状" >&2
		exit 1
	}
	;;
esac
printf '%s\n' "${version}"
