package upstream

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

type staticGitHubToken string

func (token staticGitHubToken) GitHubToken() string { return string(token) }

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func testHTTPClient(transport http.RoundTripper, now func() time.Time) *httpClient {
	return &httpClient{
		client:            &http.Client{Transport: transport},
		githubTokenSource: emptyGitHubTokenSource{},
		now:               now,
		jsonCache:         make(map[string]jsonCacheEntry),
		inflight:          make(map[string]*jsonCall),
	}
}

func jsonResponse(status int, body string, headers http.Header) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestCachedJSONReusesResultAndBacksOffAfterRateLimit(t *testing.T) {
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	var requests atomic.Int32
	limited := false
	client := testHTTPClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		if limited {
			return jsonResponse(http.StatusForbidden, "", http.Header{
				"X-RateLimit-Remaining": []string{"0"},
				"X-RateLimit-Reset":     []string{"1785459600"},
			}), nil
		}
		return jsonResponse(http.StatusOK, `{"value":7}`, nil), nil
	}), func() time.Time { return now })

	var first, second struct {
		Value int `json:"value"`
	}
	const endpoint = "https://api.github.com/repos/daeuniverse/dae/releases"
	if err := client.getJSON(context.Background(), endpoint, &first); err != nil {
		t.Fatal(err)
	}
	if err := client.getJSON(context.Background(), endpoint, &second); err != nil {
		t.Fatal(err)
	}
	if first.Value != 7 || second.Value != 7 || requests.Load() != 1 {
		t.Fatalf("缓存结果 = %d/%d，请求数 = %d", first.Value, second.Value, requests.Load())
	}

	// 缓存到期后的第一次请求遇到限流，返回旧结果并重新进入退避窗口；
	// 紧接着的页面刷新不应再打 GitHub。
	now = now.Add(jsonCacheTTL + time.Second)
	limited = true
	for range 2 {
		var stale struct {
			Value int `json:"value"`
		}
		if err := client.getJSON(context.Background(), endpoint, &stale); err != nil {
			t.Fatal(err)
		}
		if stale.Value != 7 {
			t.Fatalf("旧结果 = %d，期望 7", stale.Value)
		}
	}
	if requests.Load() != 2 {
		t.Fatalf("限流后的请求数 = %d，期望 2", requests.Load())
	}
}

func TestCachedJSONCoalescesConcurrentRequests(t *testing.T) {
	gate := make(chan struct{})
	var requests atomic.Int32
	client := testHTTPClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		<-gate
		return jsonResponse(http.StatusOK, `{"ok":true}`, nil), nil
	}), time.Now)

	const workers = 12
	errorsSeen := make(chan error, workers)
	for range workers {
		go func() {
			var payload struct {
				OK bool `json:"ok"`
			}
			if err := client.getJSON(context.Background(), "https://api.github.com/meta", &payload); err != nil {
				errorsSeen <- err
				return
			}
			if !payload.OK {
				errorsSeen <- errors.New("响应内容错误")
				return
			}
			errorsSeen <- nil
		}()
	}
	for requests.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	close(gate)
	for range workers {
		if err := <-errorsSeen; err != nil {
			t.Fatal(err)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("并发请求数 = %d，期望 1", requests.Load())
	}
}

func TestGitHubTokenIsOnlySentToAPI(t *testing.T) {
	client := &httpClient{githubTokenSource: staticGitHubToken("secret-token")}
	for _, item := range []struct {
		target string
		want   bool
	}{
		{"https://api.github.com/repos/daeuniverse/dae/releases", true},
		{"https://github.com/daeuniverse/dae/releases/download/v1/dae.zip", false},
		{"https://nightly.link/olicesx/dae/actions/runs/1/dae.zip", false},
	} {
		request, err := http.NewRequest(http.MethodGet, item.target, nil)
		if err != nil {
			t.Fatal(err)
		}
		client.authorizeGitHubAPI(request)
		got := request.Header.Get("Authorization") != ""
		if got != item.want {
			t.Fatalf("%s Authorization 存在 = %t，期望 %t", item.target, got, item.want)
		}
		if item.want && request.Header.Get("Authorization") != "Bearer secret-token" {
			t.Fatalf("Authorization = %q", request.Header.Get("Authorization"))
		}
	}
}

func TestPublicAddressRejectsInternalTargets(t *testing.T) {
	// 重定向终点不做域名白名单，因此这一层是防 SSRF 的实际关口。
	// ::ffff: 前缀的映射地址必须与其 IPv4 形态判定一致，否则等于留了后门。
	internal := []string{
		"127.0.0.1", "::1", "::ffff:127.0.0.1",
		"10.0.0.1", "172.16.0.1", "192.168.1.1", "::ffff:192.168.1.1",
		"169.254.169.254", "::ffff:169.254.169.254", // 云元数据服务
		"fe80::1", "fc00::1", "fd00::1",
		"224.0.0.1", "ff02::1",
		"0.0.0.0", "::",
		// netip 的内建判断覆盖不到，但同样不该被连上
		"100.64.0.1", "100.100.100.100", // 运营商级 NAT / Tailscale
		"198.18.0.1", "198.19.255.254", // 常被用作 fake-ip
		"240.0.0.1", "255.255.255.255", "0.1.2.3",
		"2001:db8::1",
		// 把 IPv4 嵌进 v6 的几种写法：主机跑着 NAT64/6to4 时，
		// 内核会把它们还原成内网 IPv4 再发出去
		"64:ff9b::a9fe:a9fe", "64:ff9b::7f00:1", // NAT64 众所周知前缀 → 169.254.169.254 / 127.0.0.1
		"64:ff9b:1::a9fe:a9fe",                // NAT64 本地专用前缀
		"2002:a9fe:a9fe::1", "2002:7f00:1::1", // 6to4
		"192.88.99.1",          // 6to4 中继任播
		"::7f00:1",             // 已废弃的 IPv4-compatible
		"2001::1", "2001:2::1", // Teredo / 基准测试段
	}
	for _, value := range internal {
		ip, err := netip.ParseAddr(value)
		if err != nil {
			t.Fatalf("%s 解析失败: %v", value, err)
		}
		if publicAddress(ip) {
			t.Fatalf("%s 不应被视为公网地址", value)
		}
	}

	// 2001:4860:4860::8888 钉住 2001::/23 没有把真实分配出去的 2001: 段一并封死
	public := []string{
		"140.82.121.4", "1.1.1.1", "2606:4700::1111", "::ffff:8.8.8.8",
		"2001:4860:4860::8888", "2606:50c0:8000::153", "2a04:4e42::644",
	}
	for _, value := range public {
		ip, err := netip.ParseAddr(value)
		if err != nil {
			t.Fatal(err)
		}
		if !publicAddress(ip) {
			t.Fatalf("%s 应被视为公网地址", value)
		}
	}
}

// 这才是真正的 SSRF 关口：必须对"解析到内网的域名"生效，而不只是字面 IP。
func TestGuardedDialRejectsHostnamesResolvingInternally(t *testing.T) {
	dial := guardedDial(&net.Dialer{Timeout: 5 * time.Second}, &proxyAddresses{})

	// localhost 会解析到 127.0.0.1，域名形式同样必须被拦下
	for _, address := range []string{"localhost:80", "127.0.0.1:80", "[::1]:80"} {
		conn, err := dial(context.Background(), "tcp", address)
		if err == nil {
			_ = conn.Close()
			t.Fatalf("%s 应被拒绝", address)
		}
		if !strings.Contains(err.Error(), "非公网地址") {
			t.Fatalf("%s 的拒绝原因 = %v", address, err)
		}
	}
}

func TestGuardedDialAllowsConfiguredProxy(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		if conn, err := listener.Accept(); err == nil {
			_ = conn.Close()
		}
	}()

	// 在 GitHub 不可直连的网络里，代理往往就是本机上的 dae，必须放行
	address := listener.Addr().String()
	proxies := &proxyAddresses{}
	proxies.remember(&url.URL{Scheme: "http", Host: address})
	conn, err := guardedDial(&net.Dialer{Timeout: 5 * time.Second}, proxies)(context.Background(), "tcp", address)
	if err != nil {
		t.Fatalf("已配置的代理应被放行: %v", err)
	}
	_ = conn.Close()
}

func TestCheckFirstHopRestrictsSchemeAndHost(t *testing.T) {
	allowed := []string{
		"https://api.github.com/repos/x/y/releases",
		"https://github.com/x/y/releases/download/v1/a.zip",
		"https://nightly.link/x/y/actions/runs/1/a.zip",
	}
	for _, target := range allowed {
		if err := checkFirstHop(target); err != nil {
			t.Fatalf("%s 应被允许: %v", target, err)
		}
	}

	rejected := map[string]string{
		"http://api.github.com/x":       "HTTPS",
		"https://evil.example.com/x":    "允许列表",
		"https://127.0.0.1:2023/x":      "允许列表",
		"https://api.github.com.evil/x": "允许列表",
		"://broken":                     "无效",
	}
	for target, expect := range rejected {
		err := checkFirstHop(target)
		if err == nil {
			t.Fatalf("%s 应被拒绝", target)
		}
		if !strings.Contains(err.Error(), expect) {
			t.Fatalf("%s 的错误 = %v，期望包含 %q", target, err, expect)
		}
	}
}

func TestRedirectValidationKeepsGitHubCDNOpenButRejectsPrivateTargets(t *testing.T) {
	if err := checkHTTPSRedirectTarget(context.Background(),
		"https://release-assets.githubusercontent.com/github-production-release-asset/file.zip?sig=secret"); err != nil {
		t.Fatalf("内置下载必须允许 GitHub 变动的 CDN 终点：%v", err)
	}
	for _, target := range []string{
		"http://release-assets.githubusercontent.com/file.zip",
		"https://127.0.0.1/file.zip",
		"https://user:secret@example.com/file.zip",
	} {
		if err := checkHTTPSRedirectTarget(context.Background(), target); err == nil {
			t.Fatalf("不安全重定向 %q 应被拒绝", target)
		}
	}
}

func TestCustomTargetRejectsHostnameResolvingInternally(t *testing.T) {
	err := checkPublicHTTPSTarget(context.Background(), "https://localhost/geoip.dat")
	if err == nil || !strings.Contains(err.Error(), "非公网地址") {
		t.Fatalf("localhost 应在请求前被拒绝：%v", err)
	}
}

func TestCustomTargetParseErrorDoesNotExposeQuerySecret(t *testing.T) {
	_, err := parsePublicHTTPSURL("https://example.com/%zz?token=should-not-leak")
	if err == nil {
		t.Fatal("无效 URL 应被拒绝")
	}
	if strings.Contains(err.Error(), "should-not-leak") {
		t.Fatalf("解析错误泄露了查询凭据：%v", err)
	}
}

func TestKdaeProviderRejectsUntrustedRuns(t *testing.T) {
	provider := NewKdaeProvider(nil, "olicesx", "dae", "kdae", "build.yml")
	trusted := workflowRun{
		Conclusion: "success", HeadBranch: "kdae", Event: "push",
		Path: ".github/workflows/build.yml",
	}
	trusted.HeadRepository.FullName = "olicesx/dae"
	if !provider.trustworthy(trusted) {
		t.Fatal("本仓库自己的 push 构建应被接受")
	}

	// 每一项单独破坏都必须导致拒绝
	cases := map[string]func(*workflowRun){
		"来自 fork 的 PR": func(r *workflowRun) { r.HeadRepository.FullName = "attacker/dae"; r.Event = "pull_request" },
		"仅事件为 PR":      func(r *workflowRun) { r.Event = "pull_request" },
		"仓库字段缺失":       func(r *workflowRun) { r.HeadRepository.FullName = "" },
		"分支不符":         func(r *workflowRun) { r.HeadBranch = "main" },
		"构建未成功":        func(r *workflowRun) { r.Conclusion = "failure" },
		"结论字段缺失":       func(r *workflowRun) { r.Conclusion = "" },
		"事件字段缺失":       func(r *workflowRun) { r.Event = "" },
		"来自同仓库其它工作流":   func(r *workflowRun) { r.Path = ".github/workflows/lint.yml" },
		"工作流字段缺失":      func(r *workflowRun) { r.Path = "" },
		"仓库名大小写不同但同仓库": func(r *workflowRun) { r.HeadRepository.FullName = "OLICESX/DAE" },
	}
	for name, mutate := range cases {
		run := trusted
		mutate(&run)
		got := provider.trustworthy(run)
		want := name == "仓库名大小写不同但同仓库" // GitHub 仓库名大小写不敏感
		if got != want {
			t.Fatalf("%s: trustworthy = %v，期望 %v", name, got, want)
		}
	}
}
