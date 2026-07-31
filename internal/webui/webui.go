package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// all: 必须保留：Vite 可能生成 _common-*.js，而普通 embed 模式会静默排除
// 以下划线或点开头的文件，导致浏览器拿到 SPA 回退页而无法启动。
//
//go:embed all:dist
var assets embed.FS

func Handler() http.Handler {
	dist, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// embed 进来的文件没有修改时间，FileServer 发不出 Last-Modified/ETag，
		// 浏览器无从协商就会启发式缓存入口页——面板升级后用户会停留在旧界面。
		// 因此默认 no-cache，每次取新（入口页不足 1KB，代价可忽略）；
		// assets/ 下的文件名都带内容哈希，内容一变引用必变，放心长缓存。
		writer.Header().Set("Cache-Control", "no-cache")

		requestedPath := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if requestedPath != "." && requestedPath != "" {
			// 只有真正的文件才交给 FileServer。目录也让 Stat 成功，而 assets/ 下
			// 没有 index.html，FileServer 会吐出一份目录清单——那是个没人打算
			// 提供的端点。目录一律走 SPA 兜底，交给前端路由。
			if info, err := fs.Stat(dist, requestedPath); err == nil && !info.IsDir() {
				if strings.HasPrefix(requestedPath, "assets/") {
					writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(writer, request)
				return
			}
		}

		request.URL.Path = "/"
		files.ServeHTTP(writer, request)
	})
}
