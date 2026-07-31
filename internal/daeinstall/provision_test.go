package daeinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

const testUnit = `[Unit]
Description=dae Service

[Service]
Type=notify
ExecStartPre=/usr/bin/dae validate -c /etc/dae/config.dae
ExecStart=/usr/bin/dae run --disable-timestamp -c /etc/dae/config.dae
ExecReload=/usr/bin/dae reload $MAINPID

[Install]
WantedBy=multi-user.target
`

func testBundle() upstream.Bundle {
	return upstream.Bundle{
		Binary:      elf("v1"),
		Unit:        []byte(testUnit),
		EmptyConfig: []byte(SeedConfig),
		GeoIP:       []byte("geoip-data"),
		GeoSite:     []byte("geosite-data"),
	}
}

// newFreshInstaller 构造一台"还没装过 dae"的机器：
// 服务不存在，可执行文件与配置也都不存在。
func newFreshInstaller(t *testing.T) (*Installer, *fakeService, string) {
	t.Helper()
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, service)
	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Remove(installer.configPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	service.execStart = "" // 尚未作为 systemd 服务安装
	// 单元目录默认是 /etc/systemd/system，测试必须改到临时目录
	installer.unitDir = testDir(t)
	return installer, service, binaryPath
}

// newFreshProcdInstaller 与 newFreshInstaller 相同，但走 procd 后端：
// 还没装过 dae 的机器上，服务不存在，可执行文件与配置也都不存在。
// procd 没有单元目录这个概念，不需要像 newFreshInstaller 那样另外指定。
func newFreshProcdInstaller(t *testing.T) (*Installer, *fakeService, string) {
	t.Helper()
	service := &fakeService{}
	installer, binaryPath := newTestInstallerWithBackend(t, &fakeFetcher{}, service, host.BackendProcd)
	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Remove(installer.configPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	service.execStart = ""
	return installer, service, binaryPath
}

func TestProvisionReportsReadyOnFreshMachine(t *testing.T) {
	installer, _, _ := newFreshInstaller(t)

	provision := installer.Provision(context.Background())
	if !provision.Possible {
		t.Fatalf("空机器上应当可以首次安装: %+v", provision)
	}
	if provision.Installed {
		t.Fatal("没有服务时不应报告已安装")
	}
	// 必须明确告知不会自动启动
	joined := strings.Join(provision.Notes, " ")
	if !strings.Contains(joined, "不会自动启动") {
		t.Fatalf("应说明装完不自动启动: %v", provision.Notes)
	}
}

func TestProvisionRefusesWhenServiceExists(t *testing.T) {
	installer, service, binaryPath := newFreshInstaller(t)
	// 已有 dae 服务，且它启动的文件确实在
	service.execStart = binaryPath
	if err := os.WriteFile(binaryPath, elf("v1"), 0o755); err != nil {
		t.Fatal(err)
	}

	provision := installer.Provision(context.Background())
	if provision.Possible || !provision.Installed {
		t.Fatalf("已有服务时应引导去做版本切换: %+v", provision)
	}
}

// 单元在、可执行文件不在时，升级路径会说"目标不存在"。首次安装若也以
// "已有服务"为由拒绝，面板就再没有任何办法修好这台机器。
func TestProvisionAllowsRepairWhenUnitExistsButBinaryMissing(t *testing.T) {
	installer, service, binaryPath := newFreshInstaller(t)
	service.execStart = binaryPath // 单元指向的正是面板要写的位置，但文件不存在

	provision := installer.Provision(context.Background())
	if !provision.Possible {
		t.Fatalf("单元在而二进制丢失时应当允许补齐: %+v", provision)
	}
	if !strings.Contains(strings.Join(provision.Notes, " "), "补齐") {
		t.Fatalf("应说明这次安装是在补齐丢失的文件: %v", provision.Notes)
	}
}

// 单元指向别处而那个文件也不在时，补齐会装到错误的位置，必须拒绝并说明怎么改。
func TestProvisionRefusesRepairWhenUnitPointsElsewhere(t *testing.T) {
	installer, service, _ := newFreshInstaller(t)
	service.execStart = filepath.Join(testDir(t), "elsewhere", "dae")

	provision := installer.Provision(context.Background())
	if provision.Possible {
		t.Fatalf("单元指向别处时不应报告可以安装: %+v", provision)
	}
	if !strings.Contains(strings.Join(provision.Blockers, " "), "KDAE_PANEL_DAE_BINARY") {
		t.Fatalf("应指明该改哪个配置项: %v", provision.Blockers)
	}
}

// 状态查不出来时绝不能当成"这台机器上没有 dae"——那会把一次 systemctl 抽风
// 变成一次无备份的覆盖安装。
func TestProvisionBlocksWhenServiceStatusUnreadable(t *testing.T) {
	installer, service, _ := newFreshInstaller(t)
	service.statusErr = errors.New("systemctl 不可用")

	provision := installer.Provision(context.Background())
	if provision.Possible {
		t.Fatalf("状态查不出来时不应报告可以安装: %+v", provision)
	}
	if !strings.Contains(strings.Join(provision.Blockers, " "), "不能确认") {
		t.Fatalf("应说明拒绝的理由: %v", provision.Blockers)
	}
}

func TestProvisionReportsUnwritableDirectories(t *testing.T) {
	installer, _, _ := newFreshInstaller(t)
	// 祖先是普通文件而非目录：这条路径永远建不出来，
	// 对应现实中把路径配错、或挂载点缺失的情形
	blocker := filepath.Join(testDir(t), "a-file")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	installer.unitDir = filepath.Join(blocker, "systemd")

	provision := installer.Provision(context.Background())
	if provision.Possible {
		t.Fatal("目录不可写时不应报告可以安装")
	}
	if len(provision.Blockers) == 0 || !strings.Contains(strings.Join(provision.Blockers, " "), "ReadWritePaths") {
		t.Fatalf("应指明需要加入 ReadWritePaths: %v", provision.Blockers)
	}
}

// 对照组：确认收紧 procd 分支文案没有把 systemd 分支一并清空——
// 这条 blocker 仍要指向真实存在的单元与机制，而不是两边都变得含糊其辞。
func TestProvisionUnwritableBlockerKeepsSystemdGuidanceOnSystemd(t *testing.T) {
	installer, _, _ := newFreshInstaller(t)
	blocker := filepath.Join(testDir(t), "a-file")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	installer.binaryPath = filepath.Join(blocker, "dae")

	provision := installer.Provision(context.Background())
	if provision.Possible {
		t.Fatal("目录不可写时不应报告可以安装")
	}
	joined := strings.Join(provision.Blockers, " ")
	if !strings.Contains(joined, "ReadWritePaths") || !strings.Contains(joined, "kdae-panel.service") {
		t.Fatalf("systemd 下应保留原有指引（单元名 + ReadWritePaths）: %v", provision.Blockers)
	}
}

// procd 部署既没有 systemd 单元也没有 ReadWritePaths 这套沙箱机制，照着一个
// 不存在的东西去排查只会让用户白费一轮时间。用真实的 0o000 权限位模拟目录
// 不可写——对照 geodata 包 TestMissingWarningHedgesWhenHomeHidden 的写法，
// 而不是像上面的对照组那样借用"祖先是文件"的技巧，两者都要能触发同一条检查。
func TestProvisionUnwritableBlockerAvoidsSystemdVocabularyOnProcd(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root 能写任何目录，无法模拟权限拒绝")
	}
	installer, _, _ := newFreshProcdInstaller(t)
	unwritable := t.TempDir()
	if err := os.Chmod(unwritable, 0o000); err != nil {
		t.Fatalf("设置目录权限: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(unwritable, 0o755) })
	// procdUnits.WritableDirs() 是空的：procd 部署没有单元目录需要预检。
	// 唯一还会被检查的是二进制与配置目录，因此把二进制目录换成不可写的那个。
	installer.binaryPath = filepath.Join(unwritable, "dae")

	provision := installer.Provision(context.Background())
	if provision.Possible {
		t.Fatal("目录不可写时不应报告可以安装")
	}
	joined := strings.Join(provision.Blockers, " ")
	for _, forbidden := range []string{"systemd", "systemctl", "ReadWritePaths", ".service", "服务单元"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("procd 下的 blocker 不该出现 %q: %v", forbidden, provision.Blockers)
		}
	}
}

// procdManager.Status 约定永不返回错误（见 host/procd.go），但那只是写在注释
// 里的承诺，没有测试守着。这里用会报错的 fake service 模拟那条承诺被打破的
// 情形，确认此时的 blocker 引用的是这台机器上真实存在的 init 脚本，
// 而不是硬编码的 dae.service——procd 部署里根本没有这个文件。
func TestProvisionStatusUnreadableBlockerReferencesRealPathOnProcd(t *testing.T) {
	service := &fakeService{statusErr: errors.New("ubus 不可用")}
	installer, _ := newTestInstallerWithBackend(t, &fakeFetcher{}, service, host.BackendProcd)

	provision := installer.Provision(context.Background())
	joined := strings.Join(provision.Blockers, " ")
	if !strings.Contains(joined, filepath.FromSlash("/etc/init.d")) {
		t.Fatalf("procd 下应引用 init 脚本路径: %v", provision.Blockers)
	}
	if strings.Contains(joined, "dae.service") {
		t.Fatalf("procd 下不该出现 dae.service: %v", provision.Blockers)
	}
}

// 探测不该在文件系统上留下痕迹：Provision 会被界面轮询反复调用，
// 而其中一个探测目标是 /etc/systemd/system——systemd 对它有 inotify 监视。
func TestProvisionLeavesNoTraces(t *testing.T) {
	installer, _, _ := newFreshInstaller(t)
	before := listDir(t, installer.unitDir)

	for range 3 {
		installer.Provision(context.Background())
	}
	if after := listDir(t, installer.unitDir); !slices.Equal(before, after) {
		t.Fatalf("探测后目录内容变化: %v -> %v", before, after)
	}
}

// 目标目录尚不存在时，探测其最近的已存在祖先即可——安装时会由我们创建它。
func TestProvisionAcceptsMissingDirectoryWithWritableParent(t *testing.T) {
	installer, _, _ := newFreshInstaller(t)
	installer.unitDir = filepath.Join(installer.unitDir, "not-created-yet")

	provision := installer.Provision(context.Background())
	if !provision.Possible {
		t.Fatalf("上级目录可写时应当可以安装: %+v", provision)
	}
	if _, err := os.Stat(installer.unitDir); !os.IsNotExist(err) {
		t.Fatal("探测不应创建目标目录")
	}
}

func listDir(t *testing.T, directory string) []string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	return names
}

// 上游若变更单元里的默认路径，改写会静默失效并写出指向别处的单元。
// 那样 dae 起不来，而错误现场离真正的原因很远，因此必须当场拒绝。
func TestFirstInstallRefusesUnitItCannotRetarget(t *testing.T) {
	installer, _, _ := newFreshInstaller(t)
	bundle := testBundle()
	bundle.Unit = []byte("[Service]\nExecStart=/opt/somewhere/dae run -c /opt/somewhere/config.dae\n")

	_, err := installer.FirstInstall(context.Background(), bundle,
		upstream.SourceOfficial, "v2.0.0", "v2.0.0")
	if err == nil || !strings.Contains(err.Error(), "无法把它改写为") {
		t.Fatalf("无法改写的单元应被拒绝，得到 %v", err)
	}
	if _, err := os.Stat(filepath.Join(installer.unitDir, "dae.service")); !os.IsNotExist(err) {
		t.Fatal("被拒绝时不应写下任何单元")
	}
}

func TestFirstInstallRefusesUnitWithoutExecStart(t *testing.T) {
	installer, _, _ := newFreshInstaller(t)
	bundle := testBundle()
	bundle.Unit = []byte("[Unit]\nDescription=没有 ExecStart\n")

	_, err := installer.FirstInstall(context.Background(), bundle,
		upstream.SourceOfficial, "v2.0.0", "v2.0.0")
	if err == nil || !strings.Contains(err.Error(), "没有 ExecStart") {
		t.Fatalf("缺少 ExecStart 的单元应被拒绝，得到 %v", err)
	}
}

// 可执行文件是按条目名从 zip 里挑的，必须确认它真是 ELF。
func TestFirstInstallRejectsNonELFBinary(t *testing.T) {
	installer, _, binaryPath := newFreshInstaller(t)
	bundle := testBundle()
	bundle.Binary = []byte("#!/bin/sh\necho not a binary\n")

	_, err := installer.FirstInstall(context.Background(), bundle,
		upstream.SourceOfficial, "v2.0.0", "v2.0.0")
	if err == nil || !strings.Contains(err.Error(), "不是 ELF") {
		t.Fatalf("非 ELF 内容应被拒绝，得到 %v", err)
	}
	if _, err := os.Stat(binaryPath); !os.IsNotExist(err) {
		t.Fatal("被拒绝时不应写下可执行文件")
	}
}

func TestUnitExecStartIgnoresExecStartPre(t *testing.T) {
	// ExecStartPre 也以 ExecStart 开头，前缀匹配必须区分开
	unit := "[Service]\nExecStartPre=/usr/bin/dae validate -c /etc/dae/config.dae\n" +
		"ExecStart=/usr/bin/dae run -c /etc/dae/config.dae\n"
	if got := unitExecStart(unit); got != "/usr/bin/dae run -c /etc/dae/config.dae" {
		t.Fatalf("ExecStart = %q", got)
	}
}

func TestFirstInstallLandsEveryArtifact(t *testing.T) {
	installer, service, binaryPath := newFreshInstaller(t)
	unitDir := installer.unitDir

	status, err := installer.FirstInstall(context.Background(), testBundle(),
		upstream.SourceOfficial, "v2.0.0", "v2.0.0")
	if err != nil {
		t.Fatalf("首次安装失败: %v", err)
	}

	if content, err := os.ReadFile(binaryPath); err != nil || string(content) != string(elf("v1")) {
		t.Fatalf("可执行文件 = %q, err = %v", content, err)
	}
	configDir := filepath.Dir(installer.configPath)
	for name, want := range map[string]string{
		"geoip.dat":   "geoip-data",
		"geosite.dat": "geosite-data",
	} {
		// geo 数据必须落在配置目录：那是 dae 搜索顺序里的最高优先级，
		// 也是面板在 ProtectSystem=strict 下唯一写得进去的地方
		content, err := os.ReadFile(filepath.Join(configDir, name))
		if err != nil || string(content) != want {
			t.Fatalf("%s = %q, err = %v", name, content, err)
		}
	}
	config, err := os.ReadFile(installer.configPath)
	if err != nil || strings.TrimSpace(string(config)) != SeedConfig {
		t.Fatalf("种子配置 = %q, err = %v", config, err)
	}

	unit, err := os.ReadFile(filepath.Join(unitDir, "dae.service"))
	if err != nil {
		t.Fatal(err)
	}
	// 单元里的路径必须改写成面板实际使用的路径
	if !strings.Contains(string(unit), binaryPath) || !strings.Contains(string(unit), installer.configPath) {
		t.Fatalf("单元未改写为实际路径:\n%s", unit)
	}
	if strings.Contains(string(unit), "/usr/bin/dae") {
		t.Fatalf("单元里仍残留默认路径:\n%s", unit)
	}

	// 必须 daemon-reload，否则 systemd 看不到新单元；且不应启动服务
	if len(service.actions) != 1 || service.actions[0] != host.ActionDaemonReload {
		t.Fatalf("应当只执行 daemon-reload，实际 %v", service.actions)
	}
	_ = status
}

func TestFirstInstallKeepsExistingConfig(t *testing.T) {
	installer, _, _ := newFreshInstaller(t)
	existing := "global { log_level: debug }\n"
	if err := os.WriteFile(installer.configPath, []byte(existing), 0o640); err != nil {
		t.Fatal(err)
	}

	if _, err := installer.FirstInstall(context.Background(), testBundle(),
		upstream.SourceOfficial, "v2.0.0", "v2.0.0"); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(installer.configPath); string(content) != existing {
		t.Fatalf("既有配置不应被覆盖，现在是 %q", content)
	}
}

func TestFirstInstallRefusesToOverwriteExistingUnit(t *testing.T) {
	installer, _, _ := newFreshInstaller(t)
	unitDir := installer.unitDir
	existing := "[Unit]\nDescription=用户自己写的\n"
	if err := os.WriteFile(filepath.Join(unitDir, "dae.service"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := installer.FirstInstall(context.Background(), testBundle(),
		upstream.SourceOfficial, "v2.0.0", "v2.0.0")
	if err == nil || !strings.Contains(err.Error(), "不覆盖既有服务单元") {
		t.Fatalf("不应覆盖用户已有的单元，得到 %v", err)
	}
	if content, _ := os.ReadFile(filepath.Join(unitDir, "dae.service")); string(content) != existing {
		t.Fatalf("既有单元被改动了: %q", content)
	}
}

func TestFirstInstallFallsBackToBuiltinSeedConfig(t *testing.T) {
	installer, _, _ := newFreshInstaller(t)
	bundle := testBundle()
	bundle.EmptyConfig = nil // kdae 的构建不带 empty.dae

	if _, err := installer.FirstInstall(context.Background(), bundle,
		upstream.SourceKdae, "30187784287", "d63a0c1"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(installer.configPath)
	if err != nil || strings.TrimSpace(string(content)) != SeedConfig {
		t.Fatalf("应回退到内置种子配置，实际 %q, err = %v", content, err)
	}
}

// daemon-reload 失败后重试不能被自己上一轮写下的单元卡死。
// 此时 systemd 还不认识那个单元，所以 Provision 仍认为没装；若 writeUnit
// 一律拒绝覆盖，用户就永远走不完首次安装。
func TestFirstInstallRetryAfterDaemonReloadFailure(t *testing.T) {
	installer, service, binaryPath := newFreshInstaller(t)
	service.actionErr = errors.New("systemctl daemon-reload 失败")

	if _, err := installer.FirstInstall(context.Background(), testBundle(),
		upstream.SourceOfficial, "v2.0.0", "v2.0.0"); err == nil {
		t.Fatal("daemon-reload 失败时首次安装应报错")
	}
	// 单元已经落盘，但 systemd 还不知道它
	unitPath := filepath.Join(installer.unitDir, "dae.service")
	if _, err := os.Stat(unitPath); err != nil {
		t.Fatalf("单元应已写入: %v", err)
	}

	service.actionErr = nil
	if _, err := installer.FirstInstall(context.Background(), testBundle(),
		upstream.SourceOfficial, "v2.0.0", "v2.0.0"); err != nil {
		t.Fatalf("重试应当成功，得到: %v", err)
	}
	if content, _ := os.ReadFile(binaryPath); string(content) != string(elf("v1")) {
		t.Fatalf("重试后二进制内容 = %q", content)
	}
}

// 种子配置不能声明网卡，否则首次启动就会劫持流量、可能切断管理员自己的连接。
func TestSeedConfigHijacksNothing(t *testing.T) {
	for _, forbidden := range []string{"wan_interface", "lan_interface"} {
		if strings.Contains(SeedConfig, forbidden) {
			t.Fatalf("种子配置不应包含 %s：它会让 dae 一启动就劫持流量", forbidden)
		}
	}
}

func TestRetargetUnit(t *testing.T) {
	rendered := retargetUnit(testUnit, "/opt/dae/bin/dae", "/opt/dae/config.dae")
	if strings.Contains(rendered, "/usr/bin/dae") || strings.Contains(rendered, "/etc/dae/config.dae") {
		t.Fatalf("默认路径未被替换:\n%s", rendered)
	}
	for _, want := range []string{
		"ExecStartPre=/opt/dae/bin/dae validate -c /opt/dae/config.dae",
		"ExecStart=/opt/dae/bin/dae run --disable-timestamp -c /opt/dae/config.dae",
		"ExecReload=/opt/dae/bin/dae reload $MAINPID",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("缺少 %q:\n%s", want, rendered)
		}
	}
}
