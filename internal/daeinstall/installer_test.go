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
	binary     []byte
	versions   []upstream.Version
	listErr    error
	resolveErr error
	fetchErr   error
	fetches    int
}

func (f *fakeFetcher) List(context.Context, upstream.Source, int) ([]upstream.Version, error) {
	return f.versions, f.listErr
}

func (f *fakeFetcher) Resolve(context.Context, upstream.Source, string, upstream.Platform) (upstream.Asset, error) {
	return upstream.Asset{}, f.resolveErr
}

func (f *fakeFetcher) FetchBundle(context.Context, upstream.Asset) (upstream.Bundle, error) {
	f.fetches++
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
	pidChurn    bool
	pidAt       int
	activeState string
	statusErr   error
	actionErr   error
}

func (s *fakeService) Action(_ context.Context, action host.Action) error {
	s.actions = append(s.actions, action)
	if action == host.ActionRestart {
		s.restarts++
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
	if s.pidChurn {
		s.pidAt++
	} else if s.pidAt == 0 {
		s.pidAt = 4321
	}
	return host.Status{
		ActiveState:   state,
		SubState:      "running",
		ExecStartPath: s.execStart,
		UnitPath:      s.unitPath,
		UnitFileState: s.unitFileState,
		Restarts:      s.restartsAt,
		MainPID:       s.pidAt,
	}, nil
}

func newTestInstaller(t *testing.T, fetcher *fakeFetcher, service *fakeService) (*Installer, string) {
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
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		// 这批测试验证的是 systemd 路径的行为；留空会走 host.Backend.Resolve
		// 的自动探测，让测试结果取决于运行机器上有没有 /sbin/procd。显式钉住
		// 后，测试断言（如 unitDir 下 dae.service 落盘）在任何机器上都成立。
		ServiceBackend: host.BackendSystemd,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 测试不必真的观察十秒，但仍多次采样以覆盖"起来后又崩"的判定
	installer.health = 3 * time.Millisecond
	installer.interval = time.Millisecond
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

func TestInstallUpgrade(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")

	status, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0")
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
	if !status.RollbackAvailable {
		t.Fatal("升级后应可回滚")
	}
	if len(service.actions) != 1 || service.actions[0] != host.ActionRestart {
		t.Fatalf("应当重启服务（eBPF 需重新挂载），实际 %v", service.actions)
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

	status, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0")
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

	_, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0")
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
	service.execStart = "" // dae 尚未作为 systemd 服务安装

	_, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0")
	if err == nil || !strings.Contains(err.Error(), "尚未作为 systemd 服务安装") {
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

	_, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0")
	if err == nil || !strings.Contains(err.Error(), "首次安装") {
		t.Fatalf("目标不存在时应指引用官方安装器，得到 %v", err)
	}
}

func TestInstallRejectsBinaryThatCannotRun(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")

	_, err := installer.Install(context.Background(), elf("broken"), upstream.SourceOfficial, "v9.9.9", "v9.9.9")
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

	_, err := installer.Install(context.Background(), elf("rejects-config"), upstream.SourceOfficial, "v3.0.0", "v3.0.0")
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
	service := &fakeService{failRestart: 1}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")

	_, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0")
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

	if _, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2"); err != nil {
		t.Fatal(err)
	}
	// 此刻回滚点是 v1。再装一个起不来的 v3，回滚点应当还是 v1。
	service.failRestart = 2
	if _, err := installer.Install(context.Background(), elf("v3"), upstream.SourceOfficial, "v3.0.0", "v3"); err == nil {
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

	_, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2")
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
func TestInstallRejectsPIDChurnDuringHealthWindow(t *testing.T) {
	service := &fakeService{activeState: "active", pidChurn: true}
	installer, _ := newTestInstaller(t, &fakeFetcher{}, service)

	err := installer.waitHealthy(context.Background())
	if err == nil {
		t.Fatal("观察窗口内 pid 变化应当判定为不稳定")
	}
	if !strings.Contains(err.Error(), "进程号") {
		t.Fatalf("错误信息 = %q，应当说明是进程号变化", err.Error())
	}
}

func TestRollbackRestoresPreviousVersion(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")

	if _, err := installer.Install(context.Background(), elf("v2"), upstream.SourceKdae, "30187784287", "d63a0c1"); err != nil {
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
	if _, err := installer.Install(context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0"); err != nil {
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
	if _, err := installer.Install(context.Background(), nil, upstream.SourceOfficial, "v1.0.0", "v1.0.0"); err == nil {
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
