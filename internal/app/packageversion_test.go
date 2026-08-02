package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 这一整个文件守的是同一条约束：**后构建的包一定装得到先构建的包上面**。
//
// opkg 与 apk 都拒绝降级安装，而 LuCI 的软件包页面没有任何入口能传
// --force-downgrade——排序一旦反了，用户在网页上就再也装不上新包，只能进 shell。
// 症状离原因很远（一个"Not downgrading package …"，或者更糟，一句
// "is up to date" 加退出码 0），所以规则在这里就钉死。
//
// 两套比较规则各自照抄自权威实现：ipk 侧是 dpkg 的 verrevcmp（deb-version(7)），
// apk 侧是 apk-tools 的 src/version.c。照抄而不是"意会"，是因为两边都有反直觉的
// 地方：dpkg 里字母排在所有非字母之前、'~' 排在串尾之前；apk 里 _git 是**后置**
// 后缀（排在裸版本之后），而 _rc 是前置后缀（排在裸版本之前）。

func versionScript(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("脚本需要 POSIX sh")
	}
	return filepath.Join("..", "..", "scripts", "panel-version.sh")
}

// packageVersion 跑脚本取某种格式的版本号。describe 为空时脚本自己去问 git。
func packageVersion(t *testing.T, format, describe, release string) string {
	t.Helper()
	command := exec.Command("sh", versionScript(t), "--format="+format, release)
	command.Env = append(os.Environ(), "KDAE_PANEL_DESCRIBE="+describe)
	out, err := command.Output()
	if err != nil {
		t.Fatalf("panel-version.sh --format=%s describe=%q release=%q: %v", format, describe, release, err)
	}
	return strings.TrimSpace(string(out))
}

// ── dpkg：deb-version(7) 的 verrevcmp ────────────────────────────────────

// dpkgOrder 是 dpkg 给每个字符定的序：字母按本身的值，'~' 排在最前（比串尾还前），
// 其余非字母整体抬高 256 排到字母之后，串尾与数字都是 0。
func dpkgOrder(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return 0
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return int(c)
	case c == '~':
		return -1
	case c == 0:
		return 0
	default:
		return int(c) + 256
	}
}

func dpkgCompare(a, b string) int {
	at := func(s string, i int) byte {
		if i < len(s) {
			return s[i]
		}
		return 0
	}
	isDigit := func(c byte) bool { return c >= '0' && c <= '9' }

	i, j := 0, 0
	for i < len(a) || j < len(b) {
		firstDiff := 0
		for (at(a, i) != 0 && !isDigit(at(a, i))) || (at(b, j) != 0 && !isDigit(at(b, j))) {
			ac, bc := dpkgOrder(at(a, i)), dpkgOrder(at(b, j))
			if ac != bc {
				return ac - bc
			}
			i++
			j++
		}
		for at(a, i) == '0' {
			i++
		}
		for at(b, j) == '0' {
			j++
		}
		for isDigit(at(a, i)) && isDigit(at(b, j)) {
			if firstDiff == 0 {
				firstDiff = int(at(a, i)) - int(at(b, j))
			}
			i++
			j++
		}
		if isDigit(at(a, i)) {
			return 1
		}
		if isDigit(at(b, j)) {
			return -1
		}
		if firstDiff != 0 {
			return firstDiff
		}
	}
	return 0
}

// ── apk：apk-tools src/version.c ─────────────────────────────────────────

// apk 的 token 枚举顺序本身参与比较（收尾时 token 值大的那个反而更小），
// 所以常量必须与上游同序。
const (
	apkTokenInitialDigit = iota
	apkTokenDigit
	apkTokenLetter
	apkTokenSuffix
	apkTokenSuffixNo
	apkTokenCommitHash
	apkTokenRevisionNo
	apkTokenEnd
	apkTokenInvalid
)

// 后缀表的声明顺序即优先级。none 之前的（alpha/beta/pre/rc）是预发布，排在裸版本
// 之前；之后的（cvs/svn/git/hg/p）排在裸版本之后。git 在 none 之后——这正是快照包
// 能装到正式发布上面的依据。
var apkSuffixes = []string{"alpha", "beta", "pre", "rc", "", "cvs", "svn", "git", "hg", "p"}

const apkSuffixNone = 4

type apkToken struct {
	kind   int
	value  string
	number int
	suffix int
}

// apkTokenize 复刻 token_next：digit{.digit}...{letter}{_suf{#}}...{~hash}{-r#}
func apkTokenize(version string) []apkToken {
	tokens := []apkToken{}
	kind := apkTokenInitialDigit
	rest := version

	digits := func() (string, int) {
		n := 0
		for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
			n++
		}
		value := rest[:n]
		rest = rest[n:]
		number := 0
		for _, c := range []byte(value) {
			number = number*10 + int(c-'0')
		}
		return value, number
	}

	value, number := digits()
	if value == "" {
		return []apkToken{{kind: apkTokenInvalid}}
	}
	tokens = append(tokens, apkToken{kind: kind, value: value, number: number})

	for len(rest) > 0 {
		switch c := rest[0]; {
		case c >= 'a' && c <= 'z':
			if kind > apkTokenDigit {
				return append(tokens, apkToken{kind: apkTokenInvalid})
			}
			kind = apkTokenLetter
			tokens = append(tokens, apkToken{kind: kind, value: rest[:1]})
			rest = rest[1:]
		case c == '.', c >= '0' && c <= '9':
			if c == '.' {
				if kind > apkTokenDigit {
					return append(tokens, apkToken{kind: apkTokenInvalid})
				}
				rest = rest[1:]
			}
			switch kind {
			case apkTokenInitialDigit, apkTokenDigit:
				kind = apkTokenDigit
			case apkTokenSuffix:
				kind = apkTokenSuffixNo
			default:
				return append(tokens, apkToken{kind: apkTokenInvalid})
			}
			value, number := digits()
			if value == "" {
				return append(tokens, apkToken{kind: apkTokenInvalid})
			}
			tokens = append(tokens, apkToken{kind: kind, value: value, number: number})
		case c == '_':
			if kind > apkTokenSuffixNo {
				return append(tokens, apkToken{kind: apkTokenInvalid})
			}
			rest = rest[1:]
			n := 0
			for n < len(rest) && rest[n] >= 'a' && rest[n] <= 'z' {
				n++
			}
			name := rest[:n]
			rest = rest[n:]
			suffix := -1
			for index, candidate := range apkSuffixes {
				if candidate != "" && candidate == name {
					suffix = index
				}
			}
			if suffix < 0 {
				return append(tokens, apkToken{kind: apkTokenInvalid})
			}
			kind = apkTokenSuffix
			tokens = append(tokens, apkToken{kind: kind, value: name, suffix: suffix})
		case c == '~':
			if kind >= apkTokenCommitHash {
				return append(tokens, apkToken{kind: apkTokenInvalid})
			}
			rest = rest[1:]
			n := 0
			for n < len(rest) && (rest[n] >= '0' && rest[n] <= '9' || rest[n] >= 'a' && rest[n] <= 'f') {
				n++
			}
			if n == 0 {
				return append(tokens, apkToken{kind: apkTokenInvalid})
			}
			kind = apkTokenCommitHash
			tokens = append(tokens, apkToken{kind: kind, value: rest[:n]})
			rest = rest[n:]
		case c == '-':
			if kind >= apkTokenRevisionNo || !strings.HasPrefix(rest, "-r") {
				return append(tokens, apkToken{kind: apkTokenInvalid})
			}
			rest = rest[2:]
			kind = apkTokenRevisionNo
			value, number := digits()
			if value == "" {
				return append(tokens, apkToken{kind: apkTokenInvalid})
			}
			tokens = append(tokens, apkToken{kind: kind, value: value, number: number})
		default:
			return append(tokens, apkToken{kind: apkTokenInvalid})
		}
	}
	return append(tokens, apkToken{kind: apkTokenEnd})
}

func apkBlobSort(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func apkTokenCompare(a, b apkToken) int {
	sign := func(x int) int {
		if x < 0 {
			return -1
		}
		if x > 0 {
			return 1
		}
		return 0
	}
	switch a.kind {
	case apkTokenDigit:
		// 任一侧带前导零就退回字符串比较（Gentoo 那套规矩）。
		if strings.HasPrefix(a.value, "0") || strings.HasPrefix(b.value, "0") {
			return apkBlobSort(a.value, b.value)
		}
		return sign(a.number - b.number)
	case apkTokenInitialDigit, apkTokenSuffixNo, apkTokenRevisionNo:
		return sign(a.number - b.number)
	case apkTokenLetter:
		return apkBlobSort(a.value, b.value)
	case apkTokenSuffix:
		return sign(a.suffix - b.suffix)
	default:
		return apkBlobSort(a.value, b.value)
	}
}

func apkCompare(a, b string) int {
	ta, tb := apkTokenize(a), apkTokenize(b)
	index := 0
	for index < len(ta) && index < len(tb) &&
		ta[index].kind == tb[index].kind && ta[index].kind < apkTokenEnd {
		if r := apkTokenCompare(ta[index], tb[index]); r != 0 {
			return r
		}
		index++
	}
	left, right := ta[min(index, len(ta)-1)], tb[min(index, len(tb)-1)]
	if left.kind == right.kind {
		return 0
	}
	// 收尾：还没结束的那一方更大，除非它接的是表示预发布的后缀。
	if left.kind == apkTokenSuffix && left.suffix < apkSuffixNone {
		return -1
	}
	if right.kind == apkTokenSuffix && right.suffix < apkSuffixNone {
		return 1
	}
	if left.kind > right.kind {
		return -1
	}
	if right.kind > left.kind {
		return 1
	}
	return 0
}

// ── 先证明两套比较规则本身是对的 ─────────────────────────────────────────

// 比较器抄错了，下面所有排序断言都会变成自说自话。这里用两边文档与源码里明写
// 的那几条反直觉规则做自检。
func TestPackageComparatorsMatchTheirUpstreamRules(t *testing.T) {
	for _, item := range []struct{ a, b string }{
		{"1.0.0~rc1", "1.0.0"},   // dpkg：'~' 排在串尾之前
		{"1.0.0", "1.0.0+git1"},  // dpkg：串尾排在任何普通字符之前
		{"1.0.9", "1.0.10"},      // dpkg：数字段按数值比，不是字典序
		{"1.0.0a", "1.0.0+git2"}, // dpkg：字母排在所有非字母之前，'a' 因此小于 '+'
	} {
		if dpkgCompare(item.a, item.b) >= 0 {
			t.Fatalf("dpkg: %q 应当小于 %q", item.a, item.b)
		}
		if dpkgCompare(item.b, item.a) <= 0 {
			t.Fatalf("dpkg: %q 应当大于 %q", item.b, item.a)
		}
	}
	if dpkgCompare("1.0.0", "1.0.0") != 0 {
		t.Fatal("dpkg: 同一个版本必须相等")
	}

	for _, item := range []struct{ a, b string }{
		{"1.0.0_rc1", "1.0.0"},      // apk：rc 在 none 之前，是预发布
		{"1.0.0", "1.0.0_git1~abc"}, // apk：git 在 none 之后，是后置后缀
		{"1.0.9", "1.0.10"},         // apk：数字段按数值比
		{"1.0.0_rc1", "1.0.0_rc2"},  // apk：后缀数字段按数值比
	} {
		if apkCompare(item.a, item.b) >= 0 {
			t.Fatalf("apk: %q 应当小于 %q", item.a, item.b)
		}
		if apkCompare(item.b, item.a) <= 0 {
			t.Fatalf("apk: %q 应当大于 %q", item.b, item.a)
		}
	}
	if apkCompare("1.0.0", "1.0.0") != 0 {
		t.Fatal("apk: 同一个版本必须相等")
	}
}

// ── 形状 ────────────────────────────────────────────────────────────────

func TestPackageVersionShapes(t *testing.T) {
	for _, item := range []struct{ format, describe, release, want string }{
		{format: "ipk", release: "v1.2.0", want: "1.2.0"},
		{format: "apk", release: "v1.2.0", want: "1.2.0"},
		// 预发布 tag 的记号两边不同，且都必须排在正式版之前。
		{format: "ipk", release: "v1.2.0-rc1", want: "1.2.0~rc1"},
		{format: "apk", release: "v1.2.0-rc1", want: "1.2.0_rc1"},
		// 快照基于最新的 tag，不再是写死的 0.0.1。
		{format: "ipk", describe: "v1.1.0-6-g4afbecb", want: "1.1.0+git6.4afbecb"},
		{format: "apk", describe: "v1.1.0-6-g4afbecb", want: "1.1.0_git6~4afbecb"},
		// 正好落在 tag 上的手动构建仍带领先段：省掉的话包名与正式发布完全一致，
		// opkg 会打印 "up to date"、退出码 0、什么都不装。
		{format: "ipk", describe: "v1.1.0-0-g4afbecb", want: "1.1.0+git0.4afbecb"},
		{format: "apk", describe: "v1.1.0-0-g4afbecb", want: "1.1.0_git0~4afbecb"},
		// 预发布 tag 之后的快照，两段后缀并存。
		{format: "ipk", describe: "v1.2.0-rc1-3-gabc1234", want: "1.2.0~rc1+git3.abc1234"},
		{format: "apk", describe: "v1.2.0-rc1-3-gabc1234", want: "1.2.0_rc1_git3~abc1234"},
	} {
		got := packageVersion(t, item.format, item.describe, item.release)
		if got != item.want {
			t.Fatalf("format=%s describe=%q release=%q 得到 %q，期望 %q",
				item.format, item.describe, item.release, got, item.want)
		}
	}
}

// 脏工作区不打包：包版本里没有位置如实表达"这不是某个提交的内容"，而一个版本号
// 对不上内容的包发出去之后无从追查。semver 侧相反——本地 make build 要能用。
func TestPackageVersionRefusesDirtyTree(t *testing.T) {
	for _, format := range []string{"ipk", "apk"} {
		command := exec.Command("sh", versionScript(t), "--format="+format, "")
		command.Env = append(os.Environ(), "KDAE_PANEL_DESCRIBE=v1.1.0-6-g4afbecb-dirty")
		if out, err := command.Output(); err == nil {
			t.Fatalf("format=%s 脏工作区应当让构建失败，实际输出 %q", format, strings.TrimSpace(string(out)))
		}
	}
	if got := packageVersion(t, "ipk", "v1.1.0-6-g4afbecb", ""); got == "" {
		t.Fatal("干净的工作区应当照常出版本号")
	}
}

// ── 那条硬约束 ──────────────────────────────────────────────────────────

// 一次完整的版本线：发布 → 若干次快照 → 下一个发布 → 再快照。每一步都必须严格
// 大于前一步，否则那一步在用户的路由器上就是装不上去的。
func TestPackageVersionsInstallOverEachOther(t *testing.T) {
	timeline := []struct{ describe, release string }{
		{release: "v1.1.0"},                 // 正式发布 1.1.0
		{describe: "v1.1.0-1-gaaaaaaa"},     // 之后的第一次 push
		{describe: "v1.1.0-2-gbbbbbbb"},     // 第二次
		{describe: "v1.1.0-10-gccccccc"},    // 第十次：两位数不能被当成字典序
		{release: "v1.2.0-rc1"},             // 预发布
		{describe: "v1.2.0-rc1-1-gddddddd"}, // 预发布之后的快照
		{release: "v1.2.0"},                 // 正式发布 1.2.0
		{describe: "v1.2.0-1-geeeeeee"},     // 新一轮快照，领先量重新从 1 开始
	}

	for _, format := range []string{"ipk", "apk"} {
		compare := dpkgCompare
		if format == "apk" {
			compare = apkCompare
		}
		previous := ""
		previousLabel := ""
		for _, step := range timeline {
			current := packageVersion(t, format, step.describe, step.release)
			label := step.describe + step.release
			if previous != "" && compare(previous, current) >= 0 {
				t.Fatalf("%s: %s（%s）没有大于 %s（%s）——这一版在路由器上装不上去，"+
					"而 LuCI 的软件包页面没有 --force-downgrade 的入口",
					format, current, label, previous, previousLabel)
			}
			previous, previousLabel = current, label
		}
	}
}

// 包管理器看到的不是裸版本，而是拼上 PKG_RELEASE 之后的整串：control 里是
// "Version: <版本>-1"，apk 的元数据里是 "<版本>-r1"。apk 的版本文法很严——
// 版本非法时 mkpkg 直接拒绝打包，所以这里连同修订号一起过一遍词法。
func TestFullPackageVersionsRemainValidWithRevision(t *testing.T) {
	for _, describe := range []string{
		"v1.1.0-6-g4afbecb", "v1.1.0-0-g4afbecb", "v1.2.0-rc1-3-gabc1234",
	} {
		full := packageVersion(t, "apk", describe, "") + "-r1"
		tokens := apkTokenize(full)
		for _, token := range tokens {
			if token.kind == apkTokenInvalid {
				t.Fatalf("apk 版本 %q 词法非法，mkpkg 会拒绝打包", full)
			}
		}
		if last := tokens[len(tokens)-1]; last.kind != apkTokenEnd {
			t.Fatalf("apk 版本 %q 没有正常收尾: %+v", full, last)
		}
	}
	// 带上修订号之后相对顺序不变——PKG_RELEASE 恒为 1，不该参与决定谁更新。
	older := packageVersion(t, "apk", "v1.1.0-6-g4afbecb", "") + "-r1"
	newer := packageVersion(t, "apk", "v1.1.0-7-gbbbbbbb", "") + "-r1"
	if apkCompare(older, newer) >= 0 {
		t.Fatalf("apk: %q 应当小于 %q", older, newer)
	}
	olderIpk := packageVersion(t, "ipk", "v1.1.0-6-g4afbecb", "") + "-1"
	newerIpk := packageVersion(t, "ipk", "v1.1.0-7-gbbbbbbb", "") + "-1"
	if dpkgCompare(olderIpk, newerIpk) >= 0 {
		t.Fatalf("ipk: %q 应当小于 %q", olderIpk, newerIpk)
	}
}

// 旧方案写死 0.0.1 打头，装着这种快照的机器必须能直接装上新方案的包。
// 这条不成立的话，所有现存快照用户都要进 shell 才能升级。
func TestNewPackageVersionsSupersedeTheOldZeroBase(t *testing.T) {
	for _, format := range []string{"ipk", "apk"} {
		compare := dpkgCompare
		legacy := "0.0.1+177.4afbecb"
		if format == "apk" {
			compare = apkCompare
			legacy = "0.0.1_git177~4afbecb"
		}
		current := packageVersion(t, format, "v1.1.0-6-g4afbecb", "")
		if compare(legacy, current) >= 0 {
			t.Fatalf("%s: 新版本 %s 没有大于旧方案的 %s", format, current, legacy)
		}
	}
}

// ── tag 的挑法 ──────────────────────────────────────────────────────────

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// 版本号要从"可达 tag 里最高的那个"推，而不是 git describe 的"距离最近的那个"。
//
// 这个仓库会合并上游 tuoro/kdae-panel，上游带着自己那套版本号相互重叠的 tag。
// 合并之后它们同样可达，而且往往比本仓库自己的 tag 更近——用最近的那个，一次上游
// 合并就能让包版本从 1.1.0+gitN 掉回 1.0.6+gitM，后构建的包反而装不上去。
func TestVersionPrefersHighestReachableTagNotNearest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("脚本需要 POSIX sh")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "scripts", "panel-version.sh"))
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	git(t, directory, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(directory, "f"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, directory, "add", "f")
	git(t, directory, "commit", "-qm", "A")
	git(t, directory, "tag", "v1.1.0") // 本仓库的发布
	if err := os.WriteFile(filepath.Join(directory, "f"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, directory, "commit", "-qam", "B")
	git(t, directory, "checkout", "-q", "-b", "up", "HEAD~1")
	if err := os.WriteFile(filepath.Join(directory, "g"), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, directory, "add", "g")
	git(t, directory, "commit", "-qm", "C")
	git(t, directory, "tag", "v1.0.6") // 上游的发布：版本号更低，但离 HEAD 更近
	git(t, directory, "checkout", "-q", "main")
	git(t, directory, "merge", "-q", "--no-ff", "up", "-m", "Merge upstream")

	// 先确认这个仓库确实构成了那个陷阱，否则测试通过得毫无意义。
	if nearest := git(t, directory, "describe", "--tags"); !strings.HasPrefix(nearest, "v1.0.6-") {
		t.Fatalf("这个仓库没能复现最近的 tag 反而更低的场景，git describe = %q", nearest)
	}

	command := exec.Command("sh", script, "--format=ipk")
	command.Dir = directory
	out, err := command.Output()
	if err != nil {
		t.Fatalf("panel-version.sh: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if !strings.HasPrefix(got, "1.1.0+git") {
		t.Fatalf("应当基于版本号最高的可达 tag v1.1.0，实际 %q", got)
	}
}
