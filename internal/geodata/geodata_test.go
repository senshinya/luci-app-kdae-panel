package geodata

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

type fakeFetcher struct {
	release upstream.GeoRelease
	files   map[string][]byte
	err     error
	// requested 记录实际被请求的来源，用于断言选择确实透传到了上游。
	requested upstream.GeoSource
}

func (f *fakeFetcher) Sources() []upstream.GeoSourceInfo {
	return []upstream.GeoSourceInfo{
		{Source: upstream.GeoSourceLoyalsoldier, Label: "Loyalsoldier"},
		{Source: upstream.GeoSourceV2fly, Label: "v2fly"},
	}
}

func (f *fakeFetcher) Latest(_ context.Context, source upstream.GeoSource) (upstream.GeoRelease, error) {
	f.requested = source
	if f.err != nil {
		return upstream.GeoRelease{}, f.err
	}
	release := f.release
	release.Source = source
	return release, nil
}

func (f *fakeFetcher) Fetch(_ context.Context, release upstream.GeoRelease) (upstream.GeoData, error) {
	if f.err != nil {
		return upstream.GeoData{}, f.err
	}
	return upstream.GeoData{Release: release, Files: f.files}, nil
}

type fakeService struct {
	environment map[string]string
	err         error
}

func (s *fakeService) Status(context.Context) (host.Status, error) {
	if s.err != nil {
		return host.Status{}, s.err
	}
	return host.Status{Environment: s.environment}, nil
}

type fakeReloader struct {
	calls int
	// failFirst 让第一次 reload 失败，模拟 dae 不接受新 geo 数据。
	failFirst bool
}

func (r *fakeReloader) Reload(context.Context) error {
	r.calls++
	if r.failFirst && r.calls == 1 {
		return errors.New("dae 拒绝了新的 geo 数据")
	}
	return nil
}

// testDirectory 在 Windows 上给刚关闭文件的过滤驱动一个短暂释放窗口。
//
// t.TempDir 的清理会立即把 RemoveAll 的一次性失败记为测试失败；Defender 等过滤
// 驱动偶尔还持有刚完成原子改名的目录项，随后目录已经是空的。这里仅重试清理，
// 超过窗口仍删不掉就照常让测试失败，避免掩盖真实的文件句柄泄漏。
func testDirectory(t *testing.T) string {
	t.Helper()
	directory, err := os.MkdirTemp("", "kdae-panel-geodata-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		deadline := time.Now().Add(2 * time.Second)
		for {
			err := os.RemoveAll(directory)
			if err == nil || os.IsNotExist(err) {
				return
			}
			if time.Now().After(deadline) {
				t.Errorf("清理临时目录 %s: %v", directory, err)
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	})
	return directory
}

func newTestManager(t *testing.T) (*Manager, *fakeFetcher, *fakeReloader, string) {
	t.Helper()
	directory := testDirectory(t)
	fetcher := &fakeFetcher{
		release: upstream.GeoRelease{Tag: "202607252248"},
		files: map[string][]byte{
			upstream.GeoIPName:   []byte("new-geoip"),
			upstream.GeoSiteName: []byte("new-geosite"),
		},
	}
	// 固定系统目录必须与本机隔离：/usr/local/share/dae 在开发者机器上可能
	// 真有 dae 的 geo 文件（Windows 还会解析到当前盘符），一旦被当成
	// "实际生效的那一份"，更新测试会写到沙盒之外。
	previousDirs := systemDirs
	systemDirs = nil
	t.Cleanup(func() { systemDirs = previousDirs })

	reloader := &fakeReloader{}
	manager, err := New(Options{
		ConfigPath: filepath.Join(directory, "config.dae"),
		StatePath:  filepath.Join(directory, "state", "geo-update.json"),
		Fetcher:    fetcher,
		Service:    &fakeService{},
		Reloader:   reloader,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		// 显式钉住后端：留空会走自动探测，"目录不可写"该建议什么就取决于
		// 跑测试的机器上有没有 /sbin/procd，断言随之失去意义。
		ServiceBackend: host.BackendSystemd,
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager, fetcher, reloader, directory
}

func seedGeo(t *testing.T, directory, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUpdateWritesBothFilesAndReloads(t *testing.T) {
	manager, _, reloader, directory := newTestManager(t)

	data, err := manager.Download(context.Background(), upstream.GeoSourceLoyalsoldier)
	if err != nil {
		t.Fatal(err)
	}
	status, err := manager.Apply(context.Background(), data)
	if err != nil {
		t.Fatalf("更新应成功: %v", err)
	}

	for name, want := range map[string]string{
		upstream.GeoIPName: "new-geoip", upstream.GeoSiteName: "new-geosite",
	} {
		content, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatalf("%s 应已写入: %v", name, err)
		}
		if string(content) != want {
			t.Fatalf("%s = %q，期望 %q", name, content, want)
		}
	}
	// 更新完必须让 dae 重新读，否则文件换了也不生效
	if reloader.calls != 1 {
		t.Fatalf("应恰好 reload 一次，实际 %d 次", reloader.calls)
	}
	if status.Managed == nil || status.Managed.Tag != "202607252248" {
		t.Fatalf("应记录更新到哪一版: %+v", status.Managed)
	}
}

// dae validate 察觉不到 geo 的问题，一份 dae 不接受的 geo 会让 reload 失败，
// 而 dae 不运行时流量就不再被透明代理接管——必须能退回原样。
func TestUpdateRestoresPreviousDataWhenReloadFails(t *testing.T) {
	manager, _, reloader, directory := newTestManager(t)
	reloader.failFirst = true
	seedGeo(t, directory, upstream.GeoIPName, "old-geoip")
	seedGeo(t, directory, upstream.GeoSiteName, "old-geosite")

	data, err := manager.Download(context.Background(), upstream.GeoSourceLoyalsoldier)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(context.Background(), data); err == nil {
		t.Fatal("reload 失败时更新应报错")
	}

	for name, want := range map[string]string{
		upstream.GeoIPName: "old-geoip", upstream.GeoSiteName: "old-geosite",
	} {
		content, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != want {
			t.Fatalf("%s = %q，应已还原为 %q", name, content, want)
		}
	}
	// 还原之后要再 reload 一次，让 dae 读回旧数据
	if reloader.calls != 2 {
		t.Fatalf("应在还原后再 reload 一次，实际共 %d 次", reloader.calls)
	}
	// 回滚点是临时的，不该留在磁盘上白占几十兆
	leftovers, _ := filepath.Glob(filepath.Join(directory, "*.kdae-panel-previous"))
	if len(leftovers) != 0 {
		t.Fatalf("不应留下回滚点: %v", leftovers)
	}
}

// commit 中途失败：备份已改名、新文件还没就位。此前 rollback 按 replaced 判断，
// 恰好跳过这个中间态，随后 cleanup 又把回滚点删了——旧数据彻底消失，
// 而 dae 下次重启会因为读不到 geo 直接起不来。
func TestCommitFailureKeepsOldDataInPlace(t *testing.T) {
	manager, _, reloader, directory := newTestManager(t)
	for _, name := range Names {
		seedGeo(t, directory, name, "old-"+name)
	}

	transaction := &geoTransaction{directory: directory}
	defer transaction.cleanup()
	for _, name := range Names {
		if err := transaction.stage(name, []byte("new-"+name)); err != nil {
			t.Fatal(err)
		}
	}
	// 抽掉第二个暂存文件，迫使它的 Replace 失败——此时第一个已经换完，
	// 第二个的旧文件刚被改名走。
	if err := os.Remove(transaction.staged[1].temp); err != nil {
		t.Fatal(err)
	}

	if err := transaction.commit(); err == nil {
		t.Fatal("暂存文件消失时 commit 应当失败")
	}
	for _, name := range Names {
		content, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatalf("%s 应当仍在原位: %v", name, err)
		}
		if string(content) != "old-"+name {
			t.Fatalf("%s = %q，应当仍是旧数据", name, content)
		}
	}
	transaction.cleanup()
	leftovers, _ := filepath.Glob(filepath.Join(directory, "*.kdae-panel-previous"))
	if len(leftovers) != 0 {
		t.Fatalf("成功还原后不该留下回滚点: %v", leftovers)
	}
	if reloader.calls != 0 {
		t.Fatal("commit 失败时不该 reload")
	}
	_ = manager
}

// 还原也失败时，回滚点是仅存的一份旧数据，绝不能被 cleanup 顺手删掉。
func TestFailedRollbackKeepsBackup(t *testing.T) {
	directory := testDirectory(t)
	seedGeo(t, directory, upstream.GeoIPName, "old-geoip")

	transaction := &geoTransaction{directory: directory}
	if err := transaction.stage(upstream.GeoIPName, []byte("new-geoip")); err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(directory, upstream.GeoIPName)
	backup := final + ".kdae-panel-previous"
	if err := os.Rename(final, backup); err != nil {
		t.Fatal(err)
	}
	transaction.staged[0].backup = backup
	// 把目标路径占成一个非空目录，让还原用的 rename 必定失败
	if err := os.MkdirAll(filepath.Join(final, "blocker"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := transaction.rollback(); err == nil {
		t.Fatal("还原不可能成功时 rollback 应当报错")
	}
	transaction.cleanup()
	if _, err := os.Stat(backup); err != nil {
		t.Fatalf("还原失败时回滚点必须留着，它是仅存的旧数据: %v", err)
	}
}

func TestUpdateRemovesBackupAfterSuccess(t *testing.T) {
	manager, _, _, directory := newTestManager(t)
	seedGeo(t, directory, upstream.GeoIPName, "old-geoip")

	data, err := manager.Download(context.Background(), upstream.GeoSourceLoyalsoldier)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	leftovers, _ := filepath.Glob(filepath.Join(directory, "*.kdae-panel-previous"))
	if len(leftovers) != 0 {
		t.Fatalf("成功后应删掉回滚点: %v", leftovers)
	}
	// 暂存文件也不该留下
	staged, _ := filepath.Glob(filepath.Join(directory, ".kdae-panel-*"))
	if len(staged) != 0 {
		t.Fatalf("不应留下暂存文件: %v", staged)
	}
}

// 选定的来源必须透传到上游，并如实记进账本——两个来源的规则集不是同一套，
// 记错了会让用户以为自己用的是另一套数据。
func TestUpdateRecordsChosenSource(t *testing.T) {
	manager, fetcher, _, _ := newTestManager(t)

	data, err := manager.Download(context.Background(), upstream.GeoSourceV2fly)
	if err != nil {
		t.Fatal(err)
	}
	if fetcher.requested != upstream.GeoSourceV2fly {
		t.Fatalf("上游收到的来源 = %q", fetcher.requested)
	}
	status, err := manager.Apply(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if status.Managed == nil || status.Managed.Source != upstream.GeoSourceV2fly {
		t.Fatalf("账本应记下用的是哪个来源: %+v", status.Managed)
	}
	// 用过哪个就沿用哪个：每次都重置回默认值等于诱导用户反复来回切，
	// 而来回切会改变 geosite: 规则的含义。
	if status.DefaultSource != upstream.GeoSourceV2fly {
		t.Fatalf("下次应预选上次用过的来源，实际 %q", status.DefaultSource)
	}
}

func TestStatusDefaultsToLoyalsoldierBeforeAnyUpdate(t *testing.T) {
	manager, _, _, _ := newTestManager(t)
	status := manager.Status(context.Background())
	if status.DefaultSource != upstream.GeoSourceLoyalsoldier {
		t.Fatalf("尚未更新过时应预选内置默认来源，实际 %q", status.DefaultSource)
	}
	if len(status.Sources) != 2 {
		t.Fatalf("应列出全部可选来源: %+v", status.Sources)
	}
}

// DAE_LOCATION_ASSET 的优先级高于一切。忽略它就会把 geo 写到 dae 根本不读的
// 地方，更新"成功"却毫无效果。
func TestSearchPathHonoursLocationAsset(t *testing.T) {
	paths := SearchPath("/etc/dae/config.dae", map[string]string{LocationAssetEnv: "/opt/geo"})
	if len(paths) == 0 || paths[0] != "/opt/geo" {
		t.Fatalf("DAE_LOCATION_ASSET 应排在最前: %v", paths)
	}
	if paths[1] != filepath.Dir("/etc/dae/config.dae") {
		t.Fatalf("配置目录应排在第二位: %v", paths)
	}
}

func TestSearchPathWithoutLocationAsset(t *testing.T) {
	paths := SearchPath("/etc/dae/config.dae", nil)
	if paths[0] != filepath.Dir("/etc/dae/config.dae") {
		t.Fatalf("没有环境变量时配置目录应排在最前: %v", paths)
	}
}

// dae 只读优先级最高的那一份，被遮蔽的副本必须说出来——否则用户会以为
// "我明明更新了却没生效"。
func TestStatusReportsShadowedCopies(t *testing.T) {
	manager, _, _, directory := newTestManager(t)
	system := filepath.Join(directory, "system")
	seedGeo(t, directory, upstream.GeoIPName, "effective")
	seedGeo(t, system, upstream.GeoIPName, "shadowed")

	files := locate([]string{directory, system}, []string{upstream.GeoIPName})
	if len(files) != 1 || !files[0].Present {
		t.Fatalf("应找到文件: %+v", files)
	}
	if files[0].Path != filepath.Join(directory, upstream.GeoIPName) {
		t.Fatalf("应以优先级最高的那一份为准: %s", files[0].Path)
	}
	if len(files[0].Shadowed) != 1 {
		t.Fatalf("应列出被遮蔽的副本: %+v", files[0].Shadowed)
	}
	_ = manager
}

// 就地更新实际生效的那一份，而不是无脑写死某个目录：dae-installer 把 geo 装在
// /usr/local/share/dae，改往配置目录写会生成一份优先级更高的副本，从此用户跑
// 上游更新脚本毫无效果且没有任何提示。
func TestTargetDirFollowsEffectiveFile(t *testing.T) {
	directory := testDirectory(t)
	system := filepath.Join(directory, "usr-local-share-dae")
	seedGeo(t, system, upstream.GeoIPName, "installed-by-dae-installer")

	files := locate([]string{filepath.Join(directory, "etc-dae"), system}, Names)
	target := targetDir(files, filepath.Join(directory, "etc-dae"))
	if target != system {
		t.Fatalf("应就地更新 %s，实际选了 %s", system, target)
	}
}

func TestTargetDirFallsBackToConfigDir(t *testing.T) {
	directory := testDirectory(t)
	configDir := filepath.Join(directory, "etc-dae")
	files := locate([]string{configDir, filepath.Join(directory, "system")}, Names)
	if target := targetDir(files, configDir); target != configDir {
		t.Fatalf("都不存在时应退回配置目录，实际 %s", target)
	}
}

func TestStatusReportsUnwritableTarget(t *testing.T) {
	manager, _, _, directory := newTestManager(t)
	// 祖先是普通文件而非目录：这条路径永远建不出来
	blocker := filepath.Join(directory, "a-file")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager.configPath = filepath.Join(blocker, "dae", "config.dae")

	status := manager.Status(context.Background())
	if status.Updatable {
		t.Fatal("目录不可写时不应报告可更新")
	}
	if !strings.Contains(status.Problem, "ReadWritePaths") {
		t.Fatalf("应指明需要加入 ReadWritePaths: %s", status.Problem)
	}

	// procd 上没有 kdae-panel.service，也没有 ReadWritePaths：面板由 procd
	// 直接拉起，写不进去就是目录本身的权限或挂载有问题。照着一个不存在的
	// 单元去排查，用户只会白费一轮时间。
	manager.backend = host.BackendProcd
	procdStatus := manager.Status(context.Background())
	if procdStatus.Updatable {
		t.Fatal("目录不可写时不应报告可更新")
	}
	for _, forbidden := range []string{"ReadWritePaths", "systemd", ".service"} {
		if strings.Contains(procdStatus.Problem, forbidden) {
			t.Fatalf("procd 下的提示 %q 不该出现 %q", procdStatus.Problem, forbidden)
		}
	}
	if !strings.Contains(procdStatus.Problem, "只读") {
		t.Fatalf("procd 下应指向目录本身的权限或挂载: %s", procdStatus.Problem)
	}
}

// 两个文件来自同一次发布，只换掉其中一个会让 dae 拿着两个版本的规则集跑，
// 而这种不一致既不报错也无从察觉。
// 面板能读到 /root 时，"未找到"就是未找到，不该再拿沙箱当挡箭牌。
// procd 部署没有 ProtectHome，用户看到含 ProtectHome 的措辞只会困惑。
func TestMissingWarningIsDirectWhenHomeVisible(t *testing.T) {
	visible := t.TempDir()
	previous := SandboxHiddenDir
	SandboxHiddenDir = filepath.Join(visible, "dae")
	t.Cleanup(func() { SandboxHiddenDir = previous })

	warning := MissingWarning([]string{t.TempDir()})
	if warning == "" {
		t.Fatal("geo 文件缺失时应当给出警告")
	}
	for _, forbidden := range []string{"ProtectHome", "systemd", "面板单元"} {
		if strings.Contains(warning, forbidden) {
			t.Fatalf("警告 %q 含沙箱措辞 %q", warning, forbidden)
		}
	}
	// 仅"不含 systemd 词汇"测不出分支选错——新旧两条兜底文案都不含这三个词。
	// 直接判别式：兜底文案才会把 SandboxHiddenDir 拼进去，直说文案不会；
	// 反过来断言直说文案的特征子串，双向确认确实走的是直说分支而非误判成兜底。
	if strings.Contains(warning, SandboxHiddenDir) {
		t.Fatalf("警告 %q 不该提到面板看不到的目录，那是兜底分支才会做的事", warning)
	}
	if !strings.Contains(warning, "未找到 geoip.dat / geosite.dat；路由规则用到") {
		t.Fatalf("警告 %q 应当是直说文案", warning)
	}
}

// 面板读不到 /root（ProtectHome=true）时必须留有余地：
// 文件可能就在那里而 dae 读得好好的，说死"未找到"会把正常系统报成故障。
func TestMissingWarningHedgesWhenHomeHidden(t *testing.T) {
	hidden := t.TempDir()
	unreadable := filepath.Join(hidden, "root")
	if err := os.Mkdir(unreadable, 0o000); err != nil {
		t.Fatalf("创建不可读目录: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o755) })
	if os.Geteuid() == 0 {
		t.Skip("root 能读任何目录，无法模拟沙箱遮挡")
	}
	previous := SandboxHiddenDir
	SandboxHiddenDir = filepath.Join(unreadable, ".local", "share", "dae")
	t.Cleanup(func() { SandboxHiddenDir = previous })

	warning := MissingWarning([]string{t.TempDir()})
	if !strings.Contains(warning, SandboxHiddenDir) {
		t.Fatalf("警告 %q 应当提到面板看不到的那个目录", warning)
	}
}

func TestApplyRejectsEmptyContent(t *testing.T) {
	manager, fetcher, reloader, directory := newTestManager(t)
	fetcher.files = map[string][]byte{
		upstream.GeoIPName:   []byte("new-geoip"),
		upstream.GeoSiteName: {},
	}
	seedGeo(t, directory, upstream.GeoIPName, "old-geoip")

	data, err := manager.Download(context.Background(), upstream.GeoSourceLoyalsoldier)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(context.Background(), data); err == nil {
		t.Fatal("内容为空时应拒绝写入")
	}
	if reloader.calls != 0 {
		t.Fatal("拒绝写入时不该 reload")
	}
	content, _ := os.ReadFile(filepath.Join(directory, upstream.GeoIPName))
	if string(content) != "old-geoip" {
		t.Fatalf("旧数据不该被动过: %q", content)
	}
}
