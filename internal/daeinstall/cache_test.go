package daeinstall

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/upstream"
)

func TestAcquireReusesVerifiedLocalVersion(t *testing.T) {
	fetcher := &fakeFetcher{binary: elf("v2"), assetPlatform: "x86_64"}
	installer, _ := newTestInstaller(t, fetcher, &fakeService{})

	first, cached, err := installer.Acquire(
		context.Background(), upstream.SourceOfficial, "v2.0.0", "v2.0.0", false)
	if err != nil || cached {
		t.Fatalf("首次获取 = cached %t, err %v", cached, err)
	}
	fetcher.binary = elf("unexpected-network-copy")
	second, cached, err := installer.Acquire(
		context.Background(), upstream.SourceOfficial, "v2.0.0", "v2.0.0", false)
	if err != nil || !cached {
		t.Fatalf("再次获取 = cached %t, err %v", cached, err)
	}
	if string(second.Binary) != string(first.Binary) || fetcher.fetches != 1 {
		t.Fatalf("没有复用首次下载: binary %q, fetches %d", second.Binary, fetcher.fetches)
	}
	if first.Platform != "x86_64" || second.Platform != "x86_64" {
		t.Fatalf("下载与缓存命中都应保留实际资产变体: first=%q second=%q", first.Platform, second.Platform)
	}
	if fetcher.binaryFetches != 1 || fetcher.bundleFetches != 0 {
		t.Fatalf("已有 dae 时应只取二进制: binary=%d bundle=%d", fetcher.binaryFetches, fetcher.bundleFetches)
	}
}

func TestAcquireOldCacheKeepsUnknownAssetPlatform(t *testing.T) {
	installer, _ := newTestInstaller(t, &fakeFetcher{binary: elf("network")}, &fakeService{})
	platform, err := upstream.DetectPlatform()
	if err != nil {
		t.Fatal(err)
	}
	// 省略 assetPlatform 会生成与旧版本相同的元数据；读取时不能把缓存键使用的
	// 主机首选平台冒充成当初实际下载的资产变体。
	if err := installer.cache.store(
		upstream.SourceOfficial, "v1.0.0", "v1.0.0", platform.Name, "", elf("old")); err != nil {
		t.Fatal(err)
	}
	bundle, cached, err := installer.Acquire(
		context.Background(), upstream.SourceOfficial, "v1.0.0", "v1.0.0", false)
	if err != nil || !cached {
		t.Fatalf("读取旧缓存 = cached %t, err %v", cached, err)
	}
	if bundle.Platform != "" {
		t.Fatalf("旧缓存的实际资产变体应保持未知，得到 %q", bundle.Platform)
	}
}

func TestAcquireFullBundleBypassesBinaryCache(t *testing.T) {
	fetcher := &fakeFetcher{binary: elf("cached-binary")}
	installer, _ := newTestInstaller(t, fetcher, &fakeService{})
	if _, _, err := installer.Acquire(
		context.Background(), upstream.SourceOfficial, "v2.0.0", "v2.0.0", false); err != nil {
		t.Fatal(err)
	}

	// 首次安装还需要 service、种子配置和 geo，不能把仅含二进制的缓存当成完整包。
	fetcher.binary = elf("fresh-bundle")
	bundle, cached, err := installer.Acquire(
		context.Background(), upstream.SourceOfficial, "v2.0.0", "v2.0.0", true)
	if err != nil {
		t.Fatal(err)
	}
	if cached || fetcher.fetches != 2 || string(bundle.Binary) != string(elf("fresh-bundle")) {
		t.Fatalf("完整包获取错误: cached %t, fetches %d, binary %q", cached, fetcher.fetches, bundle.Binary)
	}
	if fetcher.binaryFetches != 1 || fetcher.bundleFetches != 1 {
		t.Fatalf("首次安装应重新获取完整包: binary=%d bundle=%d", fetcher.binaryFetches, fetcher.bundleFetches)
	}
}

func TestAcquireDiscardsCorruptCacheAndDownloadsAgain(t *testing.T) {
	fetcher := &fakeFetcher{binary: elf("v2")}
	installer, _ := newTestInstaller(t, fetcher, &fakeService{})
	if _, _, err := installer.Acquire(
		context.Background(), upstream.SourceKdae, "30187784287", "d63a0c1", false); err != nil {
		t.Fatal(err)
	}
	platform, err := upstream.DetectPlatform()
	if err != nil {
		t.Fatal(err)
	}
	path := installer.cache.path(upstream.SourceKdae, "30187784287", platform.Name)
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(-1, 2); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	last := []byte{0}
	if _, err := file.Read(last); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte{last[0] ^ 0xff}, mustFileSize(t, path)-1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	fetcher.binary = elf("v3")
	bundle, cached, err := installer.Acquire(
		context.Background(), upstream.SourceKdae, "30187784287", "d63a0c1", false)
	if err != nil || cached {
		t.Fatalf("损坏后获取 = cached %t, err %v", cached, err)
	}
	if string(bundle.Binary) != string(elf("v3")) || fetcher.fetches != 2 {
		t.Fatalf("应重新下载: binary %q, fetches %d", bundle.Binary, fetcher.fetches)
	}
}

func TestVersionsMarksRemoteAndKeepsCachedOnlyEntries(t *testing.T) {
	now := time.Now().UTC()
	fetcher := &fakeFetcher{
		binary: elf("old"),
		versions: []upstream.Version{{
			Source: upstream.SourceOfficial, Ref: "v2.0.0", Label: "v2.0.0",
			PublishedAt: now, Installable: false,
		}},
	}
	installer, _ := newTestInstaller(t, fetcher, &fakeService{})
	if _, _, err := installer.Acquire(
		context.Background(), upstream.SourceOfficial, "v2.0.0", "v2.0.0", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := installer.Acquire(
		context.Background(), upstream.SourceOfficial, "v1.0.0", "v1.0.0", false); err != nil {
		t.Fatal(err)
	}

	versions, err := installer.Versions(context.Background(), upstream.SourceOfficial, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("版本数 = %d, 期望 2: %+v", len(versions), versions)
	}
	if !versions[0].Cached || !versions[0].Installable || versions[0].CachedOnly {
		t.Fatalf("过期上游版本应因本地缓存而可安装: %+v", versions[0])
	}
	if !versions[1].Cached || !versions[1].CachedOnly || versions[1].Ref != "v1.0.0" {
		t.Fatalf("上游列表外的本地版本应继续显示: %+v", versions[1])
	}
}

func TestVersionsRemainUsableWhenUpstreamIsOffline(t *testing.T) {
	fetcher := &fakeFetcher{binary: elf("v2")}
	installer, _ := newTestInstaller(t, fetcher, &fakeService{})
	if _, _, err := installer.Acquire(
		context.Background(), upstream.SourceOfficial, "v2.0.0", "v2.0.0", false); err != nil {
		t.Fatal(err)
	}
	fetcher.listErr = errors.New("offline")
	versions, err := installer.Versions(context.Background(), upstream.SourceOfficial, 30)
	if err != nil || len(versions) != 1 || !versions[0].CachedOnly {
		t.Fatalf("离线版本列表 = %+v, err %v", versions, err)
	}
}

func TestDeleteCachedDoesNotTouchInstalledBinary(t *testing.T) {
	fetcher := &fakeFetcher{binary: elf("v2")}
	installer, binaryPath := newTestInstaller(t, fetcher, &fakeService{})
	seed(t, binaryPath, "running")
	if _, _, err := installer.Acquire(
		context.Background(), upstream.SourceOfficial, "v2.0.0", "v2.0.0", false); err != nil {
		t.Fatal(err)
	}
	if err := installer.DeleteCached(upstream.SourceOfficial, "v2.0.0"); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(binaryPath); err != nil || string(got) != string(elf("running")) {
		t.Fatalf("删除缓存不应修改运行文件: %q, %v", got, err)
	}
	if err := installer.DeleteCached(upstream.SourceOfficial, "v2.0.0"); !errors.Is(err, ErrCachedVersionNotFound) {
		t.Fatalf("重复删除应报告不存在: %v", err)
	}
}

func TestSwitchCachesManagedVersionBeingReplaced(t *testing.T) {
	fetcher := &fakeFetcher{}
	service := &fakeService{}
	installer, binaryPath := newTestInstaller(t, fetcher, service)
	seed(t, binaryPath, "v1")
	if err := installer.writeState(&State{
		Source: upstream.SourceOfficial, Ref: "v1.0.0", Label: "v1.0.0",
		Platform: "x86_64_v2_sse", SHA256: digestBytes(elf("v1")),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(
		context.Background(), elf("v2"), upstream.SourceOfficial, "v2.0.0", "v2.0.0", ""); err != nil {
		t.Fatal(err)
	}
	platform, err := upstream.DetectPlatform()
	if err != nil {
		t.Fatal(err)
	}
	content, metadata, err := installer.cache.load(upstream.SourceOfficial, "v1.0.0", platform.Name)
	if err != nil || string(content) != string(elf("v1")) || metadata.Ref != "v1.0.0" {
		t.Fatalf("被替换版本未进入缓存: metadata %+v, content %q, err %v", metadata, content, err)
	}
	if metadata.AssetPlatform != "x86_64_v2_sse" {
		t.Fatalf("被替换版本的缓存丢失实际资产变体: %+v", metadata)
	}
}

func mustFileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}
