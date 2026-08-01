package daeinstall

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/dae"
	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

type fakeFetcher struct {
	binary        []byte
	assetPlatform string
	versions      []upstream.Version
	listErr       error
	resolveErr    error
	fetchErr      error
	fetches       int
	binaryFetches int
	bundleFetches int
}

func (f *fakeFetcher) List(context.Context, upstream.Source, int) ([]upstream.Version, error) {
	return f.versions, f.listErr
}

func (f *fakeFetcher) Resolve(_ context.Context, _ upstream.Source, _ string, platform upstream.Platform) (upstream.Asset, error) {
	selected := f.assetPlatform
	if selected == "" {
		selected = platform.Name
	}
	return upstream.Asset{Platform: selected}, f.resolveErr
}

func (f *fakeFetcher) FetchBinary(context.Context, upstream.Asset) ([]byte, error) {
	f.fetches++
	f.binaryFetches++
	return f.binary, f.fetchErr
}

func (f *fakeFetcher) FetchBundle(context.Context, upstream.Asset) (upstream.Bundle, error) {
	f.fetches++
	f.bundleFetches++
	return upstream.Bundle{Binary: f.binary}, f.fetchErr
}

// fakeProbe 按二进制内容决定行为，从而模拟"新版本跑不起来/不认配置"。
type fakeProbe struct {
	content string
}

func (p fakeProbe) Inspect(context.Context) dae.Report {
	if strings.Contains(p.content, "broken") {
		return dae.Report{Problem: "无法执行"}
	}
	return dae.Report{Available: true, Version: "dae " + p.content}
}

func (p fakeProbe) Validate(context.Context, string) error {
	if strings.Contains(p.content, "rejects-config") {
		return errors.New("配置里有新版本不认识的字段")
	}
	return nil
}

type fakeService struct {
	execStart     string
	unitPath      string
	unitFileState string
	actions       []host.Action
	// actionErrors 可让某个动作的前若干次调用失败；切片会按调用顺序消费。
	actionErrors map[host.Action][]error
	// failRestart 指定第几次 restart 之后服务起不来（从 1 起算，0 表示始终能起来）。
	//
	// 刻意不按"第几次状态查询"计数：那个次数取决于事务内部调了多少次 Status，
	// 改一行无关代码就会让断言悄悄失去意义；而且它还要靠观察窗口的墙钟时间
	// 才能"数"到预期的那一次，本身就会偶发假失败。
	failRestart int
	restarts    int
	// restartsGrow 为真时，每次状态查询都让 systemd 的重启计数加一，
	// 模拟"两次采样之间服务已经崩过一轮又被拉起来"——ActiveState 全程 active。
	restartsGrow bool
	restartsAt   uint64
	// pidChurn 为真时每次状态查询都换一个 pid，模拟"两次采样之间服务已经
	// 崩过一轮又被拉起来"，而 ActiveState 全程 active。procd 不暴露重启
	// 计数器，这是那边唯一能发现崩溃循环的信号。
	pidChurn bool
	// churnAfter 让 pid 从第几次状态查询之后才开始每次都变，用来模拟
	// "先稳住、稳定期结束之后才开始崩"的服务。pidChurn 是从头就崩，
	// 两者覆盖的是稳定期与观察期两段不同的判定。
	churnAfter  int
	samples     int
	pidAt       int
	nextPID     int
	activeState string
	statusErr   error
	actionErr   error
	// logs 供 explainRestartFailure 截取，用来给重启失败附上真实原因。
	logs []host.LogEntry
	// staleSamples / inactiveSamples / newPID 模拟 procd 的异步重启：
	// Action(restart) 立刻返回，之后的前 staleSamples 次状态查询看到的仍是
	// 正在退出的旧实例（active + 旧 pid），再之后的 inactiveSamples 次两个
	// 实例都不在（inactive），最后才稳定在新 pid。没有这套模拟，fakeService
	// 就是个瞬时纯函数，procd 上每次安装都会失败回滚这件事在测试里根本看不见。
	staleSamples    int
	inactiveSamples int
	newPID          int
	pendingStale    int
	pendingInactive int
}

// initialPID 是首个实例的主进程号，重启会换成别的号。
const initialPID = 4321

func (s *fakeService) Action(_ context.Context, action host.Action) error {
	s.actions = append(s.actions, action)
	if action == host.ActionRestart {
		s.restarts++
		// 重启一发出就进入中间态；在此之前的状态查询（预检、目标解析）
		// 不该被这套脚本消费掉。
		s.pendingStale, s.pendingInactive = s.staleSamples, s.inactiveSamples
		if s.pidAt == 0 {
			s.pidAt = initialPID
		}
		// 重启必然拉起一个新进程，systemd 与 procd 都如此。旧实现让 pid 跨重启
		// 保持不变，等于替安装事务掩盖了"面板可能把旧实例当成新实例"这件事。
		s.nextPID = s.newPID
		if s.nextPID == 0 {
			s.nextPID = s.pidAt + 1000
		}
	}
	if failures := s.actionErrors[action]; len(failures) > 0 {
		s.actionErrors[action] = failures[1:]
		if failures[0] != nil {
			return failures[0]
		}
	}
	return s.actionErr
}

func (s *fakeService) Status(context.Context) (host.Status, error) {
	if s.statusErr != nil {
		return host.Status{}, s.statusErr
	}
	state := s.activeState
	if state == "" {
		state = "active"
	}
	if s.failRestart > 0 && s.restarts == s.failRestart {
		state = "failed"
	}
	if s.restartsGrow {
		s.restartsAt++
	}
	s.samples++
	if s.pidAt == 0 {
		s.pidAt = initialPID
	}
	subState := "running"
	switch {
	case s.pendingStale > 0:
		// 旧实例还在退出：procd 已经发了 SIGTERM，但 dae 要卸载 eBPF 挂载点，
		// 这段时间里查到的仍是旧实例，pid 也还是旧的。
		s.pendingStale--
	case s.pendingInactive > 0:
		// 旧实例已退，新实例还没被拉起来——procd 不会在旧进程退干净前 service add。
		s.pendingInactive--
		state, subState = "inactive", "dead"
	default:
		if s.nextPID != 0 {
			s.pidAt, s.nextPID = s.nextPID, 0
		}
		if s.pidChurn || (s.churnAfter > 0 && s.samples > s.churnAfter) {
			s.pidAt++
		}
	}
	pid := s.pidAt
	if state != "active" {
		pid = 0
		if subState == "running" {
			subState = "dead"
		}
	}
	return host.Status{
		ActiveState:   state,
		SubState:      subState,
		ExecStartPath: s.execStart,
		UnitPath:      s.unitPath,
		UnitFileState: s.unitFileState,
		Restarts:      s.restartsAt,
		MainPID:       pid,
	}, nil
}

func (s *fakeService) Logs(context.Context, int) ([]host.LogEntry, error) {
	return append([]host.LogEntry(nil), s.logs...), nil
}

// newTestInstaller 构造一个 systemd 后端的 Installer。这批测试验证的是
// systemd 路径的行为；留空会走 host.Backend.Resolve 的自动探测，让测试结果
// 取决于运行机器上有没有 /sbin/procd。显式钉住后，测试断言（如 unitDir 下
// dae.service 落盘）在任何机器上都成立。
func newTestInstaller(t *testing.T, fetcher *fakeFetcher, service *fakeService) (*Installer, string) {
	t.Helper()
	return newTestInstallerWithBackend(t, fetcher, service, host.BackendSystemd)
}

// newTestInstallerWithBackend 与 newTestInstaller 相同，但允许调用方指定服务
// 后端。procd 分支的文案（如去 systemd 化相关的 blocker）必须真的构造一个
// procd 后端的 Installer 才测得到——共用只认 systemd 的构造函数，procd 分支
// 就永远没有测试覆盖。
func newTestInstallerWithBackend(t *testing.T, fetcher *fakeFetcher, service *fakeService, backend host.Backend) (*Installer, string) {
	t.Helper()
	directory := testDir(t)
	binaryPath := filepath.Join(directory, "bin", "dae")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	service.execStart = binaryPath

	installer, err := New(Options{
		BinaryPath: binaryPath,
		ConfigPath: filepath.Join(directory, "config.dae"),
		StatePath:  filepath.Join(directory, "state", "dae-install.json"),
		Fetcher:    fetcher,
		Service:    service,
		NewProbe: func(path string) Probe {
			content, err := os.ReadFile(path)
			if err != nil {
				return fakeProbe{content: "broken"}
			}
			return fakeProbe{content: string(content)}
		},
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		ServiceBackend: backend,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 测试不必真的观察十秒，但仍多次采样以覆盖"起来后又崩"的判定
	installer.health = 3 * time.Millisecond
	installer.interval = time.Millisecond
	// 稳定期同样按毫秒计：真值 15 秒会让"服务永远起不来"的用例每条卡十几秒。
	// 仍留出十几个采样间隔，异步重启的模拟才有机会走完中间态。
	installer.settle = 50 * time.Millisecond
	if err := os.WriteFile(filepath.Join(directory, "config.dae"), []byte("global {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return installer, binaryPath
}

// elf 构造带 ELF 魔数的假二进制。安装事务会拒绝替换非 ELF 目标
// （ExecStart 可能指向名叫 dae 的启动脚本），因此测试数据必须真实。
func elf(content string) []byte {
	return append([]byte("\x7fELF"), content...)
}

// seed 预置一个"已安装"的 dae，因为面板只做升级与切换，不做首次安装。
func seed(t *testing.T, binaryPath, content string) {
	t.Helper()
	if err := os.WriteFile(binaryPath, elf(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

// ExecStart 是 systemd 的概念。procd 上服务由 /etc/init.d/dae 拉起，命令行来自
// UCI 的 dae_binary；照着 ExecStart 去找，在这台机器上找不到任何东西。
func TestAssertExecutableFixHintFollowsBackend(t *testing.T) {
	for _, testCase := range []struct {
		backend  host.Backend
		want     string
		unwanted string
	}{
		{host.BackendSystemd, "ExecStart", "dae_binary"},
		{host.BackendProcd, "dae_binary", "ExecStart"},
	} {
		t.Run(string(testCase.backend), func(t *testing.T) {
			installer, binaryPath := newTestInstallerWithBackend(
				t, &fakeFetcher{}, &fakeService{}, testCase.backend)
			// 名叫 dae 的启动包装脚本：文件名过关，文件头不是 ELF。
			if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\nexec /opt/dae \"$@\"\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			err := installer.assertExecutable(binaryPath)
			if err == nil {
				t.Fatal("包装脚本应当被拒绝")
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("提示 %q 应包含 %q", err, testCase.want)
			}
			if strings.Contains(err.Error(), testCase.unwanted) {
				t.Fatalf("提示 %q 不该出现 %q", err, testCase.unwanted)
			}
		})
	}
}

func TestInstallUpgrade(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")

	status, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0", "x86_64_v2_sse")
	if err != nil {
		t.Fatalf("升级失败: %v", err)
	}
	content, err := os.ReadFile(binaryPath)
	if err != nil || string(content) != string(elf("v2")) {
		t.Fatalf("二进制内容 = %q, err = %v", content, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(binaryPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Fatalf("二进制应可执行，权限 = %v", info.Mode().Perm())
		}
	}
	if !status.Present || status.Managed == nil || status.Managed.Ref != "v2.0.0" {
		t.Fatalf("状态异常: %+v", status)
	}
	if status.Managed.Platform != "x86_64_v2_sse" {
		t.Fatalf("安装账本的实际资产变体 = %q", status.Managed.Platform)
	}
	if !status.RollbackAvailable {
		t.Fatal("升级后应可回滚")
	}
	if len(service.actions) != 1 || service.actions[0] != host.ActionRestart {
		t.Fatalf("应当重启服务（eBPF 需重新挂载），实际 %v", service.actions)
	}
}

func TestPreflightValidatesWithoutReplacingBinary(t *testing.T) {
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, service)
	seed(t, binaryPath, "v1")

	result, err := installer.Preflight(context.Background(), elf("v2"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Compatible || !result.ConfigPresent || result.Version != "dae \x7fELFv2" {
		t.Fatalf("预检结果异常: %+v", result)
	}
	if content, _ := os.ReadFile(binaryPath); string(content) != string(elf("v1")) {
		t.Fatalf("预检不应替换当前二进制: %q", content)
	}
	if len(service.actions) != 0 {
		t.Fatalf("预检不应控制服务: %v", service.actions)
	}
}

func TestPreflightReportsConfigurationIncompatibility(t *testing.T) {
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, &fakeService{})
	seed(t, binaryPath, "v1")

	result, err := installer.Preflight(context.Background(), elf("rejects-config"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Compatible || !strings.Contains(result.ValidationError, "不认识的字段") {
		t.Fatalf("应明确报告配置不兼容: %+v", result)
	}
}

func TestInstallPreservesInactiveServiceState(t *testing.T) {
	service := &fakeService{activeState: "inactive"}
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, service)
	seed(t, binaryPath, "v1")

	status, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0", "")
	if err != nil {
		t.Fatalf("切换未运行的 dae 失败: %v", err)
	}
	if len(service.actions) != 0 {
		t.Fatalf("切换前服务未运行，不应触发 systemd 动作，实际 %v", service.actions)
	}
	if status.ServiceActive {
		t.Fatal("切换后服务应保持未运行")
	}
	if content, _ := os.ReadFile(binaryPath); string(content) != string(elf("v2")) {
		t.Fatalf("二进制内容 = %q，应已切换到 v2", content)
	}
}

func TestInstallTargetsServiceExecStartNotConfiguredPath(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, configured := newTestInstaller(t, fetcher, service)
	seed(t, configured, "v1")

	// 服务实际启动的是另一个目录下的 dae：必须替换它，
	// 否则会出现"装成功但仍跑旧版本"的静默假成功
	actualDir := filepath.Join(filepath.Dir(filepath.Dir(configured)), "usr-local-bin")
	if err := os.MkdirAll(actualDir, 0o755); err != nil {
		t.Fatal(err)
	}
	actual := filepath.Join(actualDir, "dae")
	if err := os.WriteFile(actual, elf("running"), 0o755); err != nil {
		t.Fatal(err)
	}
	service.execStart = actual

	status, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0", "")
	if err != nil {
		t.Fatalf("安装失败: %v", err)
	}
	if content, _ := os.ReadFile(actual); string(content) != string(elf("v2")) {
		t.Fatalf("应替换服务实际启动的文件，其内容 = %q", content)
	}
	if content, _ := os.ReadFile(configured); string(content) != string(elf("v1")) {
		t.Fatalf("配置项指向的文件不应被动，其内容 = %q", content)
	}
	if status.BinaryPath != actual {
		t.Fatalf("报告的路径 = %q，应为服务实际启动的 %q", status.BinaryPath, actual)
	}
	// 两者不一致必须提示，否则用户不知道自己改的是哪个
	if len(status.Warnings) == 0 {
		t.Fatal("路径不一致时应给出警告")
	}
}

func TestInstallRefusesUnrelatedExecStartTarget(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")

	// 单元配置有误、ExecStart 指向别的程序时，覆盖它就是在破坏无关软件
	unrelated := filepath.Join(filepath.Dir(binaryPath), "some-other-daemon")
	if err := os.WriteFile(unrelated, []byte("not dae"), 0o755); err != nil {
		t.Fatal(err)
	}
	service.execStart = unrelated

	_, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0", "")
	if err == nil || !strings.Contains(err.Error(), "为避免覆盖无关程序") {
		t.Fatalf("非 dae 目标应被拒绝，得到 %v", err)
	}
	if content, _ := os.ReadFile(unrelated); string(content) != "not dae" {
		t.Fatalf("无关文件不应被改动，内容 = %q", content)
	}
}

func TestInstallRefusesWhenServiceHasNoExecStart(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, _ := newTestInstaller(t, fetcher, service)
	service.execStart = "" // 机器上找不到 dae 的服务定义

	_, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0", "")
	if err == nil || !strings.Contains(err.Error(), "找不到 dae 的服务定义") {
		t.Fatalf("没有服务时应拒绝安装，得到 %v", err)
	}
	if status := installer.Status(context.Background()); status.Ready {
		t.Fatalf("没有服务时不应报告就绪: %+v", status)
	}
}

func TestInstallRefusesFirstTimeInstall(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	// 目标不存在：面板不做首次安装，也不去猜该往哪装
	_ = os.Remove(binaryPath)

	_, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0", "")
	if err == nil || !strings.Contains(err.Error(), "首次安装") {
		t.Fatalf("目标不存在时应指引用官方安装器，得到 %v", err)
	}
}

func TestInstallRejectsBinaryThatCannotRun(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")

	_, err := installer.Install(context.Background(), elf("broken"), upstream.SourceOfficial, "v9.9.9", "v9.9.9", "")
	if err == nil || !strings.Contains(err.Error(), "无法运行") {
		t.Fatalf("跑不起来的版本应在替换前被拒绝，得到 %v", err)
	}
	if content, _ := os.ReadFile(binaryPath); string(content) != string(elf("v1")) {
		t.Fatalf("失败后磁盘内容 = %q，应保持不变", content)
	}
	if len(service.actions) != 0 {
		t.Fatalf("预检失败不应触发重启，实际 %v", service.actions)
	}
}

func TestInstallRejectsBinaryThatRejectsConfig(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")

	_, err := installer.Install(context.Background(), elf("rejects-config"), upstream.SourceOfficial, "v3.0.0", "v3.0.0", "")
	if err == nil || !strings.Contains(err.Error(), "拒绝当前配置") {
		t.Fatalf("不认配置的版本应被拒绝，得到 %v", err)
	}
	if content, _ := os.ReadFile(binaryPath); string(content) != string(elf("v1")) {
		t.Fatalf("失败后磁盘内容 = %q，应保持不变", content)
	}
}

func TestInstallRollsBackWhenServiceFailsToStart(t *testing.T) {
	fetcher := &fakeFetcher{}
	// 装上去的那次重启起不来；随后的回滚重启能起来
	service := &fakeService{
		failRestart: 1,
		logs:        []host.LogEntry{{Message: "country code twitter not found in /etc/dae/geoip.dat"}},
	}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")

	_, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0", "")
	if err == nil {
		t.Fatal("服务起不来时安装应失败")
	}
	var applyErr *ApplyError
	if !errors.As(err, &applyErr) {
		t.Fatalf("应返回 ApplyError，得到 %T: %v", err, err)
	}
	// 这两个标志是用户唯一能据以判断"现在到底什么状态"的东西，
	// 早先 ServiceRecovered 从未被赋值，成败两种结果都在说假话。
	if !applyErr.RolledBack || !applyErr.ServiceRecovered {
		t.Fatalf("旧版本已还原且重启成功，应如实报告: %+v（%v）", applyErr, applyErr)
	}
	if strings.Contains(applyErr.Error(), "服务仍未恢复") {
		t.Fatalf("服务已恢复，错误描述不应说仍未恢复: %v", applyErr)
	}
	if !strings.Contains(applyErr.Error(), "geoip:twitter") || !strings.Contains(applyErr.Error(), "Geo 数据") {
		t.Fatalf("版本切换失败应指出 Geo 分类根因：%v", applyErr)
	}
	if content, _ := os.ReadFile(binaryPath); string(content) != string(elf("v1")) {
		t.Fatalf("回滚后磁盘内容 = %q，应恢复为旧版本", content)
	}
	// 磁盘已退回旧版本，此前的回滚点必须原样保留，不能被本次失败的安装改写
	if content, err := os.ReadFile(installer.backupPath); err == nil {
		if string(content) == string(elf("v2")) {
			t.Fatal("失败的安装把回滚点改写成了新版本，用户再也回不去了")
		}
	}
}

// 安装失败并回滚后，原有的回滚点必须完好：backupPath 的含义始终是
// "磁盘上这一版的前一版"，被一次失败的安装提前覆盖就等于把它删了。
func TestFailedInstallKeepsExistingRollbackPoint(t *testing.T) {
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, service)
	seed(t, binaryPath, "v1")

	if _, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2", ""); err != nil {
		t.Fatal(err)
	}
	// 此刻回滚点是 v1。再装一个起不来的 v3，回滚点应当还是 v1。
	service.failRestart = 2
	if _, err := installer.Install(context.Background(), elf("v3"), upstream.SourceOfficial, "v3.0.0", "v3", ""); err == nil {
		t.Fatal("服务起不来时安装应失败")
	}
	backup, err := os.ReadFile(installer.backupPath)
	if err != nil {
		t.Fatalf("回滚点不该消失: %v", err)
	}
	if string(backup) != string(elf("v1")) {
		t.Fatalf("回滚点 = %q，应仍是 v1", backup)
	}
	if content, _ := os.ReadFile(binaryPath); string(content) != string(elf("v2")) {
		t.Fatalf("磁盘内容 = %q，应退回 v2", content)
	}
}

// 观察窗口两次采样之间跑完的崩溃-重启循环，只看 ActiveState 是发现不了的：
// 两次都是 active，中间其实已经挂掉并被 systemd 拉起来过。
func TestInstallDetectsCrashLoopWithinObservationWindow(t *testing.T) {
	service := &fakeService{restartsGrow: true}
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, service)
	// 零窗口刻意压住边界：第一次采样只能建立基线，即使窗口已经结束，
	// 也必须再采一次才能判断 NRestarts 是否增长。测试因此不依赖 CI 调度速度。
	installer.health = 0
	installer.interval = 0
	seed(t, binaryPath, "v1")

	_, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2", "")
	if err == nil {
		t.Fatal("观察窗口内服务反复重启，安装应失败")
	}
	if !strings.Contains(err.Error(), "又重启了") {
		t.Fatalf("错误应指出服务在观察窗口内重启过: %v", err)
	}
	if content, _ := os.ReadFile(binaryPath); string(content) != string(elf("v1")) {
		t.Fatalf("崩溃循环应触发回滚，磁盘内容 = %q", content)
	}
}

// procd 不暴露重启计数器，崩溃循环只能靠主进程号变化发现。
// 没有这一条，respawn 循环里反复崩溃的新版本会被判定为安装成功。
//
// pid 每次查询都换，稳定期的"连续两次相同"永远凑不齐——稳定期必须到点报失败，
// 而不是把这种服务当成"还在起来"一路放过去。
func TestInstallRejectsPIDChurnDuringHealthWindow(t *testing.T) {
	service := &fakeService{activeState: "active", pidChurn: true}
	installer, _ := newTestInstaller(t, &fakeFetcher{}, service)

	err := installer.waitHealthy(context.Background(), 0, time.Now().UTC())
	if err == nil {
		t.Fatal("观察窗口内 pid 变化应当判定为不稳定")
	}
	if !strings.Contains(err.Error(), "进程号") {
		t.Fatalf("错误信息 = %q，应当说明是进程号变化", err.Error())
	}
}

// procd 的 restart 展开成 stop + start 且立刻返回：面板会先看到正在退出的
// 旧实例，再看到两个实例都不在的空窗，最后才看到新实例。稳定期必须容忍
// 这段过程，否则每次版本切换都会回滚一个完全正常的版本，理由还是编造的
// （"主进程号从 X 变成 Y"——那其实是旧实例换成新实例）。
func TestInstallToleratesAsynchronousProcdRestart(t *testing.T) {
	service := &fakeService{staleSamples: 2, inactiveSamples: 3, newPID: 9876}
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, service)
	seed(t, binaryPath, "v1")

	status, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2", "")
	if err != nil {
		t.Fatalf("异步重启是 procd 的正常过程，不应判定为安装失败: %v", err)
	}
	if content, _ := os.ReadFile(binaryPath); string(content) != string(elf("v2")) {
		t.Fatalf("磁盘内容 = %q，新版本应当留在磁盘上而不是被回滚", content)
	}
	if status.Managed == nil || status.Managed.Ref != "v2.0.0" {
		t.Fatalf("账本应记下新版本: %+v", status.Managed)
	}
}

// 稳定期只是给"正在起来"留出时间，不是给"起不来"发免死金牌：
// 服务始终不 active 时必须到点失败，并且如实说出最后查到的状态。
func TestWaitHealthyFailsWhenServiceNeverSettles(t *testing.T) {
	service := &fakeService{activeState: "activating"}
	installer, _ := newTestInstaller(t, &fakeFetcher{}, service)

	err := installer.waitHealthy(context.Background(), 0, time.Now().UTC())
	if err == nil {
		t.Fatal("服务始终没起来时应当判定失败")
	}
	if !strings.Contains(err.Error(), "没有起来") || !strings.Contains(err.Error(), "activating") {
		t.Fatalf("错误信息 = %q，应当说明服务没起来并带上最后状态", err.Error())
	}
}

// 稳定期与观察期必须分开计时。合并计时的话，异步重启耗掉的时间会从观察窗口里
// 扣走，重启越慢观察越短——而重启慢恰恰最该多看两眼。这里让服务先走完异步重启
// 的中间态，稳定之后才开始崩：观察窗口若被稳定期吃掉，这次崩溃就会被漏掉。
func TestObservationWindowStartsAfterServiceSettles(t *testing.T) {
	// 前 4 次采样用于走完中间态并稳定下来（旧实例退出 → 空窗 → 新实例
	// 连续两次同号），第 5 次起 pid 每查一次变一次。
	service := &fakeService{staleSamples: 1, inactiveSamples: 1, newPID: 5000, churnAfter: 4}
	installer, _ := newTestInstaller(t, &fakeFetcher{}, service)
	_ = service.Action(context.Background(), host.ActionRestart)

	err := installer.waitHealthy(context.Background(), initialPID, time.Now().UTC())
	if err == nil {
		t.Fatal("服务稳定后又开始反复换 pid，观察期应当抓到")
	}
	if !strings.Contains(err.Error(), "观察窗口内挂掉") {
		t.Fatalf("错误信息 = %q，应当指出是观察窗口内的崩溃重启", err.Error())
	}
}

// 旧实例退出得慢时，重启后连续两次都能采到同一个旧 pid。只按"连续两次相同"
// 建立基线，会把旧实例当成新实例，等新实例真的起来再判成"观察窗口内挂过"——
// 这正是 procd 上回滚正常版本的第二种路径，必须靠"pid 要与重启前不同"挡住。
func TestSettleIgnoresLingeringPreviousInstance(t *testing.T) {
	service := &fakeService{staleSamples: 4, newPID: 7777}
	installer, _ := newTestInstaller(t, &fakeFetcher{}, service)
	_ = service.Action(context.Background(), host.ActionRestart)

	if err := installer.waitHealthy(context.Background(), initialPID, time.Now().UTC()); err != nil {
		t.Fatalf("旧实例慢慢退出是正常过程，不应判定为失败: %v", err)
	}
}

func TestRollbackRestoresPreviousVersion(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")

	if _, err := installer.Install(context.Background(), elf("v2"), upstream.SourceKdae, "30187784287", "d63a0c1", ""); err != nil {
		t.Fatal(err)
	}
	rolled, err := installer.Rollback(context.Background())
	if err != nil {
		t.Fatalf("回滚失败: %v", err)
	}
	if content, _ := os.ReadFile(binaryPath); string(content) != string(elf("v1")) {
		t.Fatalf("回滚后内容 = %q", content)
	}
	if !rolled.Present {
		t.Fatalf("回滚后状态异常: %+v", rolled)
	}
}

func TestRollbackRestoresPreviousAssetPlatform(t *testing.T) {
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, service)
	seed(t, binaryPath, "v1")
	if _, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial,
		"v2.0.0", "v2.0.0", "x86_64_v2_sse"); err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(context.Background(), elf("v3"), upstream.SourceKdae,
		"30187784287", "d63a0c1", "x86_64_v3_avx2"); err != nil {
		t.Fatal(err)
	}
	rolled, err := installer.Rollback(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rolled.Managed == nil || rolled.Managed.Platform != "x86_64_v2_sse" {
		t.Fatalf("回滚后实际资产变体没有随上一版账本恢复: %+v", rolled.Managed)
	}
}

func TestRollbackWithoutBackup(t *testing.T) {
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, &fakeService{})
	seed(t, binaryPath, "v1")
	if _, err := installer.Rollback(context.Background()); err == nil {
		t.Fatal("没有备份时回滚应报错")
	}
}

func TestStatusDetectsExternalReplacement(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")
	if _, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0", ""); err != nil {
		t.Fatal(err)
	}
	if status := installer.Status(context.Background()); status.Drifted {
		t.Fatal("刚装完不应报告被外部替换")
	}

	// 模拟有人绕过面板手动换了二进制
	if err := os.WriteFile(binaryPath, []byte("manually-replaced"), 0o755); err != nil {
		t.Fatal(err)
	}
	status := installer.Status(context.Background())
	if !status.Drifted {
		t.Fatal("外部替换后应被识别出来")
	}
	if !status.Present || !status.Ready {
		t.Fatalf("文件仍在，应报告已就绪: %+v", status)
	}
}

func TestInstallRejectsEmptyBinary(t *testing.T) {
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, &fakeService{})
	seed(t, binaryPath, "v1")
	if _, err := installer.Install(context.Background(), nil, upstream.SourceOfficial, "v1.0.0", "v1.0.0", ""); err == nil {
		t.Fatal("空内容应被拒绝")
	}
}

func TestDownloadReturnsVerifiedBinary(t *testing.T) {
	fetcher := &fakeFetcher{binary: elf("v2")}
	installer, _ := newTestInstaller(t, fetcher, &fakeService{})
	bundle, cached, err := installer.Acquire(
		context.Background(), upstream.SourceOfficial, "v2.0.0", "v2.0.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if cached {
		t.Fatal("第一次获取不应命中缓存")
	}
	if string(bundle.Binary) != string(elf("v2")) {
		t.Fatalf("下载内容 = %q", bundle.Binary)
	}
}
