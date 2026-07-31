package upstream

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func validCustomGeoSource() CustomGeoSource {
	return CustomGeoSource{
		Label:            "自建规则",
		GeoIPURL:         "https://assets.example.com/geoip.dat",
		GeoIPSHA256URL:   "https://checks.example.com/geoip.dat.sha256sum",
		GeoSiteURL:       "https://assets.example.com/geosite.dat",
		GeoSiteSHA256URL: "https://checks.example.com/geosite.dat.sha256sum",
	}
}

func TestCustomGeoStorePersistsCRUD(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "geo-sources.json")
	store, err := openCustomGeoStore(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if initial := store.list(); initial == nil || len(initial) != 0 {
		t.Fatalf("空来源列表必须是非 nil 空数组：%#v", initial)
	}
	created, err := store.create(validCustomGeoSource())
	if err != nil {
		t.Fatal(err)
	}
	if created.Source != GeoSource("custom:"+created.ID) {
		t.Fatalf("来源 ID 未规范化：%+v", created)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != customGeoStoreMode {
			t.Fatalf("来源文件权限 = %o，期望 %o", info.Mode().Perm(), customGeoStoreMode)
		}
	}

	reopened, err := openCustomGeoStore(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.list(); len(got) != 1 || got[0].ID != created.ID {
		t.Fatalf("重启后来源未恢复：%+v", got)
	}
	updated := validCustomGeoSource()
	updated.Label = "更新后的来源"
	if _, err := reopened.update(created.ID, updated); err != nil {
		t.Fatal(err)
	}
	if err := reopened.delete(created.ID); err != nil {
		t.Fatal(err)
	}
	if got := reopened.list(); len(got) != 0 {
		t.Fatalf("删除后仍有来源：%+v", got)
	}
	if got := reopened.list(); got == nil {
		t.Fatal("删除最后一个来源后必须返回空数组，不能返回 null")
	}
}

func TestCustomGeoSourceRejectsUnsafeURLs(t *testing.T) {
	cases := []string{
		"http://example.com/geoip.dat",
		"https://user:secret@example.com/geoip.dat",
		"https://127.0.0.1/geoip.dat",
		"https://192.168.1.2/geoip.dat",
		"https://[::1]/geoip.dat",
	}
	for _, target := range cases {
		source := validCustomGeoSource()
		source.GeoIPURL = target
		if err := normalizeCustomGeoSource(&source, "0123456789abcdef"); err == nil {
			t.Fatalf("不安全地址 %q 应被拒绝", target)
		}
	}
}

func TestCustomGeoSourceTrimsURLWhitespace(t *testing.T) {
	source := validCustomGeoSource()
	source.GeoIPURL = "  https://assets.example.com/geoip.dat  "
	if err := normalizeCustomGeoSource(&source, "0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	if source.GeoIPURL != "https://assets.example.com/geoip.dat" {
		t.Fatalf("链接未规范化：%q", source.GeoIPURL)
	}
}

func TestCustomGeoProviderDownloadsAndVerifiesBothFiles(t *testing.T) {
	contents := map[string][]byte{
		"/geoip.dat":   []byte("geoip-content"),
		"/geosite.dat": []byte("geosite-content"),
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "" {
			t.Error("自定义来源不应收到 Authorization")
		}
		if content, ok := contents[request.URL.Path]; ok {
			_, _ = writer.Write(content)
			return
		}
		name := strings.TrimSuffix(request.URL.Path, ".sha256sum")
		content, ok := contents[name]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		digest := sha256.Sum256(content)
		_, _ = fmt.Fprintf(writer, "%x  %s\n", digest, strings.TrimPrefix(name, "/"))
	}))
	defer server.Close()

	client := &httpClient{
		client:            server.Client(),
		githubTokenSource: emptyGitHubTokenSource{},
		validateTarget:    func(context.Context, string) error { return nil },
	}
	source := CustomGeoSource{
		ID:               "0123456789abcdef",
		Source:           "custom:0123456789abcdef",
		Label:            "测试来源",
		GeoIPURL:         server.URL + "/geoip.dat",
		GeoIPSHA256URL:   server.URL + "/geoip.dat.sha256sum",
		GeoSiteURL:       server.URL + "/geosite.dat",
		GeoSiteSHA256URL: server.URL + "/geosite.dat.sha256sum",
	}
	provider := &customGeoProvider{client: client, source: source}
	release, err := provider.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data, err := provider.Fetch(context.Background(), release)
	if err != nil {
		t.Fatal(err)
	}
	for route, want := range contents {
		name := strings.TrimPrefix(route, "/")
		if got := data.Files[name]; string(got) != string(want) {
			t.Fatalf("%s = %q，期望 %q", name, got, want)
		}
	}
	if !strings.Contains(data.Release.Tag, "geoip ") || !strings.Contains(data.Release.Tag, "geosite ") {
		t.Fatalf("版本标识没有摘要：%q", data.Release.Tag)
	}
}
