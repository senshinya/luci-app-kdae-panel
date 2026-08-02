package geodata

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
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
	activeState string
	mainPID     int
}

func (s *fakeService) Status(context.Context) (host.Status, error) {
	if s.err != nil {
		return host.Status{}, s.err
	}
	return host.Status{
		Environment: s.environment,
		ActiveState: s.activeState,
		MainPID:     s.mainPID,
	}, nil
}

type fakeReloader struct {
	calls int
	pids  []int
	// failFirst 让第一次 reload 失败，模拟 dae 不接受新 geo 数据。
	failFirst bool
}

func (r *fakeReloader) Reload(context.Context) error {
	r.calls++
	if r.failFirst && r.calls == 1 {
		return errors.New("code twitter not found in /etc/dae/geosite.dat")
	}
	return nil
}

func (r *fakeReloader) ReloadPID(_ context.Context, pid int) error {
	r.pids = append(r.pids, pid)
	return r.Reload(context.Background())
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
		Service:    &fakeService{activeState: "active", mainPID: 4321},
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
	if !slices.Equal(reloader.pids, []int{4321}) {
		t.Fatalf("应使用 systemd MainPID reload，实际 PID = %v", reloader.pids)
	}
	if status.Managed == nil || status.Managed.Tag != "202607252248" {
		t.Fatalf("应记录更新到哪一版: %+v", status.Managed)
	}
}

// dae 会分别沿搜索路径查找两个文件；它们可能各自在不同目录生效。
// 更新必须就地替换每一份，不能再把两者强行搬到第一个文件所在的目录。
func TestUpdateWritesFilesToSeparateEffectiveDirectories(t *testing.T) {
	manager, _, _, directory := newTestManager(t)
	assetDir := filepath.Join(directory, "asset")
	systemDir := filepath.Join(directory, "system")
	seedGeo(t, assetDir, upstream.GeoIPName, "old-geoip")
	seedGeo(t, systemDir, upstream.GeoSiteName, "old-geosite")
	systemDirs = []string{systemDir}
	manager.service = &fakeService{
		environment: map[string]string{LocationAssetEnv: assetDir},
		activeState: "active",
		mainPID:     4321,
	}

	data, err := manager.Download(context.Background(), upstream.GeoSourceLoyalsoldier)
	if err != nil {
		t.Fatal(err)
	}
	status, err := manager.Apply(context.Background(), data)
	if err != nil {
		t.Fatalf("分目录更新应成功: %v", err)
	}
	wants := map[string]string{
		filepath.Join(assetDir, upstream.GeoIPName):    "new-geoip",
		filepath.Join(systemDir, upstream.GeoSiteName): "new-geosite",
	}
	for path, want := range wants {
		content, readErr := os.ReadFile(path)
		if readErr != nil || string(content) != want {
			t.Fatalf("%s 未原位更新: content=%q err=%v", path, content, readErr)
		}
	}
	if status.TargetDir != "" {
		t.Fatalf("分目录更新不应伪造公共目录，实际 %q", status.TargetDir)
	}
	if _, err := os.Stat(filepath.Join(directory, upstream.GeoIPName)); !os.IsNotExist(err) {
		t.Fatal("不应在配置目录生成额外的 geoip.dat")
	}
	if _, err := os.Stat(filepath.Join(directory, upstream.GeoSiteName)); !os.IsNotExist(err) {
		t.Fatal("不应在配置目录生成额外的 geosite.dat")
	}
}

func TestSeparateDirectoryUpdateRollsBothFilesBack(t *testing.T) {
	manager, _, reloader, directory := newTestManager(t)
	first := filepath.Join(directory, "first")
	second := filepath.Join(directory, "second")
	seedGeo(t, first, upstream.GeoIPName, "old-geoip")
	seedGeo(t, second, upstream.GeoSiteName, "old-geosite")
	systemDirs = []string{second}
	manager.service = &fakeService{
		environment: map[string]string{LocationAssetEnv: first},
		activeState: "active",
		mainPID:     4321,
	}
	reloader.failFirst = true

	data, err := manager.Download(context.Background(), upstream.GeoSourceLoyalsoldier)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(context.Background(), data); err == nil {
		t.Fatal("reload 失败时分目录事务也应失败")
	}
	for path, want := range map[string]string{
		filepath.Join(first, upstream.GeoIPName):    "old-geoip",
		filepath.Join(second, upstream.GeoSiteName): "old-geosite",
	} {
		content, readErr := os.ReadFile(path)
		if readErr != nil || string(content) != want {
			t.Fatalf("%s 未共同回滚: content=%q err=%v", path, content, readErr)
		}
	}
}

func TestUpdateWhileServiceStoppedWaitsForNextStart(t *testing.T) {
	manager, _, reloader, directory := newTestManager(t)
	manager.service = &fakeService{activeState: "failed"}
	seedGeo(t, directory, upstream.GeoIPName, "old-geoip")
	seedGeo(t, directory, upstream.GeoSiteName, "old-geosite")

	data, err := manager.Download(context.Background(), upstream.GeoSourceLoyalsoldier)
	if err != nil {
		t.Fatal(err)
	}
	status, err := manager.Apply(context.Background(), data)
	if err != nil {
		t.Fatalf("dae 未运行时仍应完成文件更新: %v", err)
	}
	if reloader.calls != 0 {
		t.Fatalf("dae 未运行时不应调用 reload，实际 %d 次", reloader.calls)
	}
	if status.ServiceState != ServiceStateInactive {
		t.Fatalf("服务状态 = %q，期望 %q", status.ServiceState, ServiceStateInactive)
	}
	for name, want := range map[string]string{
		upstream.GeoIPName: "new-geoip", upstream.GeoSiteName: "new-geosite",
	} {
		content, readErr := os.ReadFile(filepath.Join(directory, name))
		if readErr != nil || string(content) != want {
			t.Fatalf("%s 未保留新数据: content=%q err=%v", name, content, readErr)
		}
	}
}

// 服务状态查询失败时拿不到 DAE_LOCATION_ASSET。若仍按空环境变量继续更新，
// 面板会把文件写进配置目录，而运行中的 dae 继续读取真正的 asset 目录；API
// 报成功但更新永远不生效。查询恢复后，同一个 Manager 必须能够正常重试。
func TestUpdateWaitsForTrustedServiceSearchPathAndCanRetry(t *testing.T) {
	manager, _, reloader, directory := newTestManager(t)
	assetDir := filepath.Join(directory, "dae-assets")
	for _, name := range Names {
		seedGeo(t, assetDir, name, "old-"+name)
	}
	service := &fakeService{err: errors.New("ubus temporarily unavailable")}
	manager.service = service
	data := upstream.GeoData{Files: map[string][]byte{
		upstream.GeoIPName:   []byte("new-geoip"),
		upstream.GeoSiteName: []byte("new-geosite"),
	}}

	if _, err := manager.Apply(context.Background(), data); err == nil {
		t.Fatal("服务搜索路径不可信时必须拒绝更新")
	} else if !strings.Contains(err.Error(), "ubus temporarily unavailable") {
		t.Fatalf("错误应保留服务查询失败原因: %v", err)
	}
	for _, name := range Names {
		if _, err := os.Stat(filepath.Join(directory, name)); !os.IsNotExist(err) {
			t.Fatalf("查询失败时不应向猜测出的配置目录写入 %s: %v", name, err)
		}
	}
	if reloader.calls != 0 {
		t.Fatalf("查询失败时不应 reload，实际 %d 次", reloader.calls)
	}

	service.err = nil
	service.environment = map[string]string{LocationAssetEnv: assetDir}
	service.activeState = "active"
	service.mainPID = 4321
	if _, err := manager.Apply(context.Background(), data); err != nil {
		t.Fatalf("服务查询恢复后应能用同一个 Manager 重试: %v", err)
	}
	for name, want := range map[string]string{
		upstream.GeoIPName: "new-geoip", upstream.GeoSiteName: "new-geosite",
	} {
		content, err := os.ReadFile(filepath.Join(assetDir, name))
		if err != nil || string(content) != want {
			t.Fatalf("%s 应写入实际 asset 目录: content=%q err=%v", name, content, err)
		}
	}
}

// 两套后端在状态查询失败时丢的东西不一样：systemd 丢的是单元里声明的
// DAE_LOCATION_ASSET（可以指向任意目录，写错就静默不生效），procd 丢的是实例 PID
// （搜索路径其实一个字都不差——dae.init 与面板读同一份 UCI）。写成一句通用话会把
// OpenWrt 用户支去查一个在他机器上不可能出错的东西。
func TestUnknownServiceStateProblemNamesTheRightLoss(t *testing.T) {
	manager, _, _, _ := newTestManager(t)
	manager.service = &fakeService{err: errors.New("systemctl exploded")}

	systemdProblem := manager.Status(context.Background()).Problem
	if !strings.Contains(systemdProblem, LocationAssetEnv) {
		t.Fatalf("systemd 下应点名读不到的 %s: %s", LocationAssetEnv, systemdProblem)
	}

	manager.backend = host.BackendProcd
	procdProblem := manager.Status(context.Background()).Problem
	if strings.Contains(procdProblem, LocationAssetEnv) {
		t.Fatalf("procd 下 %s 恒等于配置目录，提示不该点它: %s", LocationAssetEnv, procdProblem)
	}
	if !strings.Contains(procdProblem, "PID") {
		t.Fatalf("procd 下丢的是实例 PID，提示应点名它: %s", procdProblem)
	}
	for _, problem := range []string{systemdProblem, procdProblem} {
		if !strings.Contains(problem, "systemctl exploded") {
			t.Fatalf("两条提示都必须保留原始故障: %s", problem)
		}
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
	} else if !strings.Contains(err.Error(), "geosite:twitter") || !strings.Contains(err.Error(), "Geo 数据") {
		t.Fatalf("reload 失败应指出缺失分类和处理入口：%v", err)
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
	if !slices.Equal(reloader.pids, []int{4321, 4321}) {
		t.Fatalf("新旧数据都应使用 systemd MainPID reload，实际 PID = %v", reloader.pids)
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

	transaction := &geoTransaction{}
	defer transaction.cleanup()
	for _, name := range Names {
		if err := transaction.stage(name, filepath.Join(directory, name), []byte("new-"+name)); err != nil {
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

	transaction := &geoTransaction{}
	if err := transaction.stage(upstream.GeoIPName, filepath.Join(directory, upstream.GeoIPName), []byte("new-geoip")); err != nil {
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

// Status 已经把"存在回滚点"报成不可更新，事务里仍要再拦一道：真走到这里说明拦截
// 没生效，而 rename 覆盖回滚点就等于删掉仅存的那份旧数据。
func TestCommitRefusesToOverwriteRecoveryPoint(t *testing.T) {
	directory := testDirectory(t)
	final := seedGeo(t, directory, upstream.GeoIPName, "current-data")
	backup := seedGeo(t, directory, upstream.GeoIPName+rollbackSuffix, "last-known-good")
	transaction := &geoTransaction{}
	defer transaction.cleanup()
	if err := transaction.stage(upstream.GeoIPName, final, []byte("new-data")); err != nil {
		t.Fatal(err)
	}

	if err := transaction.commit(); err == nil {
		t.Fatal("既存 recovery 文件存在时 commit 必须拒绝覆盖")
	}
	for path, want := range map[string]string{final: "current-data", backup: "last-known-good"} {
		content, err := os.ReadFile(path)
		if err != nil || string(content) != want {
			t.Fatalf("%s 应保持 %q: content=%q err=%v", path, want, content, err)
		}
	}
}

// 回滚点意味着上一次事务没结算完，那份旧数据可能是仅存的一份。新的更新必须先被
// 拦下、由用户决定恢复还是清理，而不是照常写下去把它盖掉。
func TestUpdateRefusesWhileRollbackPointRemains(t *testing.T) {
	manager, _, _, directory := newTestManager(t)
	geoIP := seedGeo(t, directory, upstream.GeoIPName, "unverified-current")
	backup := seedGeo(t, directory, upstream.GeoIPName+rollbackSuffix, "last-known-good")
	geoSite := seedGeo(t, directory, upstream.GeoSiteName, "old-geosite")
	data := upstream.GeoData{Files: map[string][]byte{
		upstream.GeoIPName:   []byte("new-geoip"),
		upstream.GeoSiteName: []byte("new-geosite"),
	}}

	if _, err := manager.Apply(context.Background(), data); err == nil {
		t.Fatal("存在未处理的回滚点时应拒绝更新")
	}
	for path, want := range map[string]string{
		geoIP:   "unverified-current",
		geoSite: "old-geosite",
		backup:  "last-known-good",
	} {
		content, err := os.ReadFile(path)
		if err != nil || string(content) != want {
			t.Fatalf("%s 应保持 %q: content=%q err=%v", path, want, content, err)
		}
	}
}

// 进程若恰好死在"旧文件改名为回滚点、尚未换入新文件"之间，正式文件不存在。
// 残留扫描必须走完整条搜索路径：只看当前有效文件会退回配置目录，从此再也扫不到
// 低优先级目录里的那个回滚点，dae 一重启就因为读不到 geo 起不来。
func TestResidualScanFindsRollbackPointInLowPriorityDirectory(t *testing.T) {
	manager, _, reloader, directory := newTestManager(t)
	lowPriority := filepath.Join(directory, "system-share-dae")
	systemDirs = []string{lowPriority}
	backup := seedGeo(t, lowPriority, upstream.GeoIPName+rollbackSuffix, "last-known-good")

	status := manager.Status(context.Background())
	if status.Updatable {
		t.Fatalf("低优先级目录里的回滚点同样应拦下更新: %+v", status)
	}
	if len(status.Residuals) != 1 || status.Residuals[0].Path != backup || !status.Residuals[0].Restorable {
		t.Fatalf("应把它列为可恢复的回滚点: %+v", status.Residuals)
	}

	if _, err := manager.RestoreResidual(context.Background(), backup); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(lowPriority, upstream.GeoIPName))
	if err != nil || string(content) != "last-known-good" {
		t.Fatalf("旧数据应就地恢复: content=%q err=%v", content, err)
	}
	if _, err := os.Stat(filepath.Join(directory, upstream.GeoIPName)); !os.IsNotExist(err) {
		t.Fatalf("不应把它恢复到优先级更高的配置目录: %v", err)
	}
	if reloader.calls != 1 {
		t.Fatalf("恢复后应 reload 一次，实际 %d 次", reloader.calls)
	}
}

// systemd 的 ProtectHome 会故意把 dae 可能读取的 root asset 目录隐藏给面板。
// 扫不到那里是契约的一部分，但不能因此让可见目录里的残留也一并漏掉。
func TestResidualScanSurvivesUnreadableSandboxDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root 能读任何目录，无法模拟 ProtectHome")
	}
	base := testDirectory(t)
	hiddenParent := filepath.Join(base, "hidden-root")
	if err := os.Mkdir(hiddenParent, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(hiddenParent, 0o755) })
	previous := SandboxHiddenDir
	SandboxHiddenDir = filepath.Join(hiddenParent, ".local", "share", "dae")
	t.Cleanup(func() { SandboxHiddenDir = previous })

	visible := filepath.Join(base, "visible")
	backup := seedGeo(t, visible, upstream.GeoIPName+rollbackSuffix, "last-known-good")
	residuals := findResiduals([]string{SandboxHiddenDir, visible})
	if len(residuals) != 1 || residuals[0].Path != backup {
		t.Fatalf("不可读的沙箱目录不应吞掉可见目录里的残留: %+v", residuals)
	}
}

// 恢复被异常占用的正式路径挡住时，回滚点必须原样留着并给出准确路径，
// 让用户清掉障碍后还能重试——顺手删掉它等于把仅存的旧数据也扔了。
func TestResidualRestoreCanRetryAfterBlockerRemoved(t *testing.T) {
	manager, _, _, directory := newTestManager(t)
	final := filepath.Join(directory, upstream.GeoIPName)
	backup := seedGeo(t, directory, upstream.GeoIPName+rollbackSuffix, "last-known-good")
	if err := os.MkdirAll(filepath.Join(final, "blocker"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.RestoreResidual(context.Background(), backup); err == nil {
		t.Fatal("正式路径被目录占用时应拒绝恢复")
	}
	content, err := os.ReadFile(backup)
	if err != nil || string(content) != "last-known-good" {
		t.Fatalf("失败时回滚点必须保持不变: content=%q err=%v", content, err)
	}

	if err := os.RemoveAll(final); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.RestoreResidual(context.Background(), backup); err != nil {
		t.Fatalf("解除阻塞后应能恢复: %v", err)
	}
	if content, err := os.ReadFile(final); err != nil || string(content) != "last-known-good" {
		t.Fatalf("旧数据应回到正式位置: content=%q err=%v", content, err)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("恢复成功后不应留下回滚点: %v", err)
	}
}

func TestUpdateRemovesBackupAfterSuccess(t *testing.T) {
	manager, _, _, directory := newTestManager(t)
	seedGeo(t, directory, upstream.GeoIPName, "old-geoip")
	staleTemp := filepath.Join(directory, geoTempPrefix+"abandoned")
	if err := os.WriteFile(staleTemp, []byte("never-active"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * geoTempStaleAfter)
	if err := os.Chtimes(staleTemp, old, old); err != nil {
		t.Fatal(err)
	}

	data, err := manager.Download(context.Background(), upstream.GeoSourceLoyalsoldier)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	leftovers, globErr := filepath.Glob(filepath.Join(directory, "*"+rollbackSuffix))
	if globErr != nil || len(leftovers) != 0 {
		t.Fatalf("成功后不应留下回滚点: paths=%v err=%v", leftovers, globErr)
	}
	// 暂存文件也不该留下
	staged, _ := filepath.Glob(filepath.Join(directory, ".kdae-panel-*"))
	if len(staged) != 0 {
		t.Fatalf("不应留下暂存文件: %v", staged)
	}
}

func TestResidualCleanupPreservesOnlyRecoverableBackup(t *testing.T) {
	manager, _, reloader, directory := newTestManager(t)
	geoIP := seedGeo(t, directory, upstream.GeoIPName, "current-geoip")
	geoIPBackup := geoIP + rollbackSuffix
	if err := os.WriteFile(geoIPBackup, []byte("old-geoip"), 0o644); err != nil {
		t.Fatal(err)
	}
	geoSiteBackup := filepath.Join(directory, upstream.GeoSiteName) + rollbackSuffix
	if err := os.WriteFile(geoSiteBackup, []byte("old-geosite"), 0o644); err != nil {
		t.Fatal(err)
	}
	temporary := filepath.Join(directory, geoTempPrefix+"abandoned")
	if err := os.WriteFile(temporary, []byte("new-but-unused"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * geoTempStaleAfter)
	if err := os.Chtimes(temporary, old, old); err != nil {
		t.Fatal(err)
	}

	status := manager.Status(context.Background())
	if status.Updatable || len(status.Residuals) != 3 {
		t.Fatalf("回滚点应阻止继续更新并列出全部残留: %+v", status)
	}
	status, err := manager.CleanupResiduals(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(geoIPBackup); !os.IsNotExist(err) {
		t.Fatal("正式文件存在时，用户确认清理后应删除旧回滚点")
	}
	if _, err := os.Stat(temporary); !os.IsNotExist(err) {
		t.Fatal("Geo 专属暂存文件应被清理")
	}
	if _, err := os.Stat(geoSiteBackup); err != nil {
		t.Fatalf("正式文件缺失时必须保留唯一旧数据: %v", err)
	}
	if len(status.Residuals) != 1 || !status.Residuals[0].Restorable {
		t.Fatalf("清理后应只剩可恢复的回滚点: %+v", status.Residuals)
	}

	status, err = manager.RestoreResidual(context.Background(), geoSiteBackup)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(directory, upstream.GeoSiteName))
	if err != nil || string(content) != "old-geosite" {
		t.Fatalf("旧 geosite 应恢复到正式位置: content=%q err=%v", content, err)
	}
	if len(status.Residuals) != 0 || !status.Updatable {
		t.Fatalf("全部处理后应恢复可更新状态: %+v", status)
	}
	if reloader.calls != 1 {
		t.Fatalf("运行中的 dae 应在恢复后 reload，实际 %d 次", reloader.calls)
	}
}

func TestFreshGeoTemporaryFileIsNotTreatedAsResidual(t *testing.T) {
	manager, _, _, directory := newTestManager(t)
	path := filepath.Join(directory, geoTempPrefix+"still-in-use")
	if err := os.WriteFile(path, []byte("staging"), 0o600); err != nil {
		t.Fatal(err)
	}
	status := manager.Status(context.Background())
	if len(status.Residuals) != 0 {
		t.Fatalf("一小时内的暂存文件可能属于另一个面板进程，不应列为可清理残留: %+v", status.Residuals)
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

// 搜索顺序里同一个目录不得出现两次。
//
// procd 部署必然踩中：dae.init 设的 DAE_LOCATION_ASSET 就是 dirname $dae_config，
// 与配置目录是同一个。列两次的后果不是多查一遍那么无害——locate 会在那里命中同一个
// 文件两次，第二次记成"被遮蔽的副本"，界面于是给出一句自相矛盾的话：dae 只读
// /etc/dae/geoip.dat，而 /etc/dae/geoip.dat 里的副本不会生效、可以删掉。
func TestSearchPathDeduplicatesDirectories(t *testing.T) {
	for name, environment := range map[string]map[string]string{
		"与配置目录相同": {LocationAssetEnv: "/etc/dae"},
		"只差一个尾斜杠": {LocationAssetEnv: "/etc/dae/"},
	} {
		paths := SearchPath("/etc/dae/config.dae", environment)
		var hits int
		for _, path := range paths {
			if path == "/etc/dae" {
				hits++
			}
		}
		if hits != 1 {
			t.Fatalf("%s：/etc/dae 在搜索顺序里出现 %d 次: %v", name, hits, paths)
		}
	}
}

// 同一个文件经由两条路径被找到时不算副本。
//
// 告警的措辞是"可以删掉"，而这两条路径指向的是同一个 inode——照着删就把 dae 唯一
// 读得到的那份 geo 数据删了。目录是符号链接时同样成立。
func TestLocateDoesNotShadowFileWithItself(t *testing.T) {
	directory := testDirectory(t)
	real := filepath.Join(directory, "real")
	seedGeo(t, real, upstream.GeoIPName, "only-copy")

	link := filepath.Join(directory, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("本机不支持创建符号链接: %v", err)
	}
	for name, searchPath := range map[string][]string{
		"同一个目录列了两次": {real, real},
		"目录是符号链接":   {link, real},
	} {
		files := locate(searchPath, []string{upstream.GeoIPName})
		if len(files) != 1 || !files[0].Present {
			t.Fatalf("%s：应找到文件: %+v", name, files)
		}
		if len(files[0].Shadowed) != 0 {
			t.Fatalf("%s：生效的是 %s，却报称 %v 里的副本可以删掉——那是同一个文件",
				name, files[0].Path, files[0].Shadowed)
		}
	}
}

// 用户看到的是 Status 里的那句告警，这里把 procd 的实际配置原样摆一遍。
func TestStatusHasNoSelfShadowingWarning(t *testing.T) {
	directory := testDirectory(t)
	previousDirs := systemDirs
	systemDirs = nil
	t.Cleanup(func() { systemDirs = previousDirs })
	seedGeo(t, directory, upstream.GeoIPName, "effective")
	seedGeo(t, directory, upstream.GeoSiteName, "effective")

	manager, err := New(Options{
		ConfigPath: filepath.Join(directory, "config.dae"),
		StatePath:  filepath.Join(directory, "state", "geo-update.json"),
		Fetcher:    &fakeFetcher{},
		// dae.init 设的就是这个：DAE_LOCATION_ASSET = 配置文件所在目录。
		Service: &fakeService{
			activeState: "active",
			mainPID:     4321,
			environment: map[string]string{LocationAssetEnv: directory},
		},
		Reloader:       &fakeReloader{},
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		ServiceBackend: host.BackendProcd,
	})
	if err != nil {
		t.Fatal(err)
	}

	status := manager.Status(context.Background())
	for _, file := range status.Files {
		if len(file.Shadowed) != 0 {
			t.Fatalf("%s 生效于 %s，却报称 %v 里的副本可以删掉", file.Name, file.Path, file.Shadowed)
		}
	}
	for _, warning := range status.Warnings {
		if strings.Contains(warning, "可以删掉") {
			t.Fatalf("不该出现让用户删掉唯一生效文件的告警: %s", warning)
		}
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
func TestTargetPathsFollowEachEffectiveFile(t *testing.T) {
	directory := testDirectory(t)
	first := filepath.Join(directory, "first")
	second := filepath.Join(directory, "second")
	seedGeo(t, first, upstream.GeoIPName, "installed-by-dae-installer")
	seedGeo(t, second, upstream.GeoSiteName, "installed-by-another-tool")

	files := locate([]string{first, second}, Names)
	assignTargets(files, filepath.Join(directory, "etc-dae"))
	if files[0].TargetPath != filepath.Join(first, upstream.GeoIPName) ||
		files[1].TargetPath != filepath.Join(second, upstream.GeoSiteName) {
		t.Fatalf("两个文件应各自在生效位置更新: %+v", files)
	}
	if commonTargetDir(files) != "" {
		t.Fatalf("分目录时不应报告公共目标目录: %+v", files)
	}
}

func TestMissingTargetsFallBackToConfigDir(t *testing.T) {
	directory := testDirectory(t)
	configDir := filepath.Join(directory, "etc-dae")
	files := locate([]string{configDir, filepath.Join(directory, "system")}, Names)
	assignTargets(files, configDir)
	for _, file := range files {
		if file.TargetPath != filepath.Join(configDir, file.Name) {
			t.Fatalf("缺失文件应退回配置目录: %+v", file)
		}
	}
}

// 就地更新时"写下去的那份就是生效的那份"通常自动成立，但搜索路径上冒出一份优先级
// 更高的副本就不成立了——此时接口报成功而 dae 读的是别的数据，只能整个回滚。
func TestVerifyEffectiveFilesRequiresEachFileAtItsTarget(t *testing.T) {
	directory := testDirectory(t)
	high := filepath.Join(directory, "high")
	low := filepath.Join(directory, "low")
	seedGeo(t, high, upstream.GeoIPName, "high-geoip")
	seedGeo(t, low, upstream.GeoSiteName, "low-geosite")
	search := []string{high, low}
	targets := map[string]string{
		upstream.GeoIPName:   filepath.Join(high, upstream.GeoIPName),
		upstream.GeoSiteName: filepath.Join(low, upstream.GeoSiteName),
	}

	if err := verifyEffectiveFiles(search, targets); err != nil {
		t.Fatalf("两个文件都从各自的目标位置生效时应通过: %v", err)
	}
	// 优先级更高的目录里冒出一份 geosite，dae 从此读它而不是刚写的那份。
	seedGeo(t, high, upstream.GeoSiteName, "shadowing-geosite")
	if err := verifyEffectiveFiles(search, targets); err == nil ||
		!strings.Contains(err.Error(), upstream.GeoSiteName) {
		t.Fatalf("被更高优先级副本遮蔽时应拒绝结算，实际 %v", err)
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

func TestUpdateRejectsMissingGeoFileBeforeWriting(t *testing.T) {
	manager, _, reloader, directory := newTestManager(t)
	for _, name := range Names {
		seedGeo(t, directory, name, "old-"+name)
	}
	data := upstream.GeoData{Files: map[string][]byte{
		upstream.GeoIPName: []byte("new-geoip"),
	}}

	if _, err := manager.Apply(context.Background(), data); err == nil || !strings.Contains(err.Error(), upstream.GeoSiteName) {
		t.Fatalf("缺少 geosite 时应在写盘前拒绝，实际 %v", err)
	}
	if reloader.calls != 0 {
		t.Fatal("输入不完整时不应 reload")
	}
	for _, name := range Names {
		content, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || string(content) != "old-"+name {
			t.Fatalf("%s 不应被修改: content=%q err=%v", name, content, err)
		}
	}
}

func TestUpdateRejectsUnknownGeoFileBeforeWriting(t *testing.T) {
	manager, _, reloader, directory := newTestManager(t)
	data := upstream.GeoData{Files: map[string][]byte{
		upstream.GeoIPName:   []byte("new-geoip"),
		upstream.GeoSiteName: []byte("new-geosite"),
		"unexpected.dat":     []byte("unexpected"),
	}}

	if _, err := manager.Apply(context.Background(), data); err == nil || !strings.Contains(err.Error(), "unexpected.dat") {
		t.Fatalf("未知文件名应在写盘前拒绝，实际 %v", err)
	}
	if reloader.calls != 0 {
		t.Fatal("出现未知文件时不应 reload")
	}
	if _, err := os.Stat(filepath.Join(directory, "unexpected.dat")); !os.IsNotExist(err) {
		t.Fatalf("未知文件不应写入目标目录: %v", err)
	}
}
