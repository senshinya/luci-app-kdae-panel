package webui

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
)

func get(t *testing.T, target string) *http.Response {
	t.Helper()
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder.Result()
}

// Vite 有时会生成 _common-*.js。普通 go:embed 会静默排除以下划线或点
// 开头的文件，因此逐个核对磁盘产物，避免发布包直到浏览器运行时才白屏。
func TestAllDistFilesAreEmbedded(t *testing.T) {
	onDisk := os.DirFS(".")
	err := fs.WalkDir(onDisk, "dist", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if _, err := fs.Stat(assets, name); err != nil {
			t.Errorf("构建产物 %s 没有被嵌入: %v", name, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// 目录也让 fs.Stat 成功，早先因此被交给 FileServer，而 assets/ 下没有
// index.html，于是吐出一份目录清单——一个没人打算提供的端点。
func TestDirectoryRequestsFallBackToIndex(t *testing.T) {
	for _, target := range []string{"/assets/", "/assets", "/assets/../assets/"} {
		response := get(t, target)
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s 状态码 = %d", target, response.StatusCode)
		}
		if strings.Contains(string(body), "<pre>") || strings.Contains(string(body), "</a>\n") {
			t.Fatalf("%s 返回了目录清单:\n%s", target, string(body)[:min(400, len(body))])
		}
		if !strings.Contains(string(body), "<div id=\"app\">") {
			t.Fatalf("%s 应回退到 SPA 入口页，实际:\n%s", target, string(body)[:min(200, len(body))])
		}
	}
}

// 前端路由的路径在磁盘上不存在，必须回退到入口页而不是 404。
func TestUnknownRoutesFallBackToIndex(t *testing.T) {
	for _, target := range []string{"/dashboard", "/versions", "/config"} {
		response := get(t, target)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s 状态码 = %d", target, response.StatusCode)
		}
	}
}

// firstAsset 从嵌入产物里找一个 .js 资源做样本，没有就跳过测试。
func firstAsset(t *testing.T) string {
	t.Helper()
	dist, err := fs.Sub(assets, "dist")
	if err != nil {
		t.Fatal(err)
	}
	var sample string
	err = fs.WalkDir(dist, "assets", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && path.Ext(name) == ".js" && sample == "" {
			sample = name
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if sample == "" {
		t.Skip("嵌入产物里没有 .js 资源")
	}
	return sample
}

// 回退不能把真实资源也吃掉：构建产物必须原样送出。
func TestRealAssetsAreServed(t *testing.T) {
	sample := firstAsset(t)

	response := get(t, "/"+sample)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("%s 状态码 = %d", sample, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	// 送出的必须是这个资源本身，而不是被兜底换成了入口页
	if strings.Contains(string(body), "<div id=\"app\">") {
		t.Fatalf("%s 被错误地回退成了入口页", sample)
	}
	if len(body) == 0 {
		t.Fatalf("%s 内容为空", sample)
	}
}

func TestFaviconIsEmbeddedAndServed(t *testing.T) {
	response := get(t, "/favicon.svg")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("favicon 状态码 = %d", response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.Contains(contentType, "image/svg+xml") {
		t.Fatalf("favicon Content-Type = %q", contentType)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "<svg") || strings.Contains(string(body), "<div id=\"app\">") {
		t.Fatalf("favicon 没有返回 SVG 资源: %q", string(body))
	}
}

// embed 的文件没有修改时间，响应里发不出 Last-Modified/ETag，浏览器会
// 启发式缓存入口页，面板升级后用户停留在旧界面。资产靠文件名里的内容
// 哈希放心长缓存，入口页（含 SPA 兜底与直接请求）必须每次取新。
func TestCacheHeaders(t *testing.T) {
	for _, target := range []string{"/", "/index.html", "/dashboard", "/assets/"} {
		response := get(t, target)
		if got := response.Header.Get("Cache-Control"); got != "no-cache" {
			t.Fatalf("%s Cache-Control = %q，期望 no-cache", target, got)
		}
	}

	response := get(t, "/"+firstAsset(t))
	want := "public, max-age=31536000, immutable"
	if got := response.Header.Get("Cache-Control"); got != want {
		t.Fatalf("资产 Cache-Control = %q，期望 %q", got, want)
	}
}
