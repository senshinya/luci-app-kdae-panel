package upstream

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	apiTimeout      = 20 * time.Second
	downloadTimeout = 5 * time.Minute
	// GitHub 的 JSON 响应远小于此,超过即视为异常。
	maxAPIBytes = 8 << 20
	// GitHub 匿名额度只有每个出口 IP 每小时 60 次。版本列表变化不频繁，
	// 短时缓存可以避免多人反复打开页面时把额度耗在相同响应上。
	jsonCacheTTL        = 10 * time.Minute
	maxJSONCacheEntries = 32
	// dae 各架构的发布包目前在 20MB 上下,留足余量同时挡住无限响应体。
	MaxAssetBytes = 128 << 20
	userAgent     = "kdae-panel"
)

// httpClient 收敛所有出站请求的超时与重定向策略。
type httpClient struct {
	client            *http.Client
	githubTokenSource GitHubTokenSource
	validateTarget    func(context.Context, string) error
	now               func() time.Time

	cacheMu   sync.Mutex
	jsonCache map[string]jsonCacheEntry
	inflight  map[string]*jsonCall
}

// GitHubTokenSource 让凭据可以由设置页即时更新，而不需要重启面板。
// 实现只需返回当前值；httpClient 不接触持久化细节。
type GitHubTokenSource interface {
	GitHubToken() string
}

type emptyGitHubTokenSource struct{}

func (emptyGitHubTokenSource) GitHubToken() string { return "" }

type jsonCacheEntry struct {
	body      []byte
	fetchedAt time.Time
}

type jsonCall struct {
	done chan struct{}
	body []byte
	err  error
}

// allowedHosts 只约束由面板主动发起的第一跳。
// 重定向终点故意不做域名白名单:GitHub 的资产落点在
// objects.githubusercontent.com / release-assets.githubusercontent.com /
// *.blob.core.windows.net 之间迁移过,写死必然过时,而过时的后果是有人被迫
// 去关掉校验。终点的可信度改由带外取得的 sha256 承担,这里只强制那些不会
// 过时的不变量:必须 https、跳数有限、不得转发凭据、不得落到内网地址。
var allowedHosts = map[string]bool{
	"api.github.com": true,
	"github.com":     true,
	"nightly.link":   true,
}

func newHTTPClient() *httpClient {
	return newHTTPClientWithTokenSource(emptyGitHubTokenSource{})
}

func newHTTPClientWithTokenSource(source GitHubTokenSource) *httpClient {
	return newHTTPClientWithValidators(source, checkFirstHopContext, checkHTTPSRedirectTarget)
}

// newCustomHTTPClient 供管理员配置的自定义来源使用。它不携带 GitHub Token，
// 且首跳和每次重定向都必须解析到公网 HTTPS 地址。
func newCustomHTTPClient() *httpClient {
	return newHTTPClientWithValidators(emptyGitHubTokenSource{}, checkPublicHTTPSTarget, checkPublicHTTPSTarget)
}

func newHTTPClientWithValidators(
	source GitHubTokenSource,
	validateTarget func(context.Context, string) error,
	validateRedirect func(context.Context, string) error,
) *httpClient {
	if source == nil {
		source = emptyGitHubTokenSource{}
	}
	if validateTarget == nil {
		validateTarget = checkFirstHopContext
	}
	if validateRedirect == nil {
		validateRedirect = checkHTTPSRedirectTarget
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	// 代理地址常常就是本机(在 GitHub 不可直连的网络里，代理往往正是 dae 自己)，
	// 因此内网地址检查必须放过管理员显式配置的代理，只约束直连的最终目标。
	proxies := &proxyAddresses{}
	transport := &http.Transport{
		Proxy: func(request *http.Request) (*url.URL, error) {
			proxyURL, err := http.ProxyFromEnvironment(request)
			if err == nil && proxyURL != nil {
				proxies.remember(proxyURL)
			}
			return proxyURL, err
		},
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: time.Second,
		IdleConnTimeout:       30 * time.Second,
		DialContext:           guardedDial(dialer, proxies),
	}
	return &httpClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   downloadTimeout,
			CheckRedirect: func(request *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("重定向次数过多")
				}
				if err := validateRedirect(request.Context(), request.URL.String()); err != nil {
					return fmt.Errorf("拒绝重定向：%w", err)
				}
				// Go 只在跨站时剥离部分请求头，这里显式清干净，
				// 免得将来给请求加上凭据后被重定向带去第三方。
				request.Header.Del("Authorization")
				request.Header.Del("Cookie")
				return nil
			},
		},
		githubTokenSource: source,
		validateTarget:    validateTarget,
		now:               time.Now,
		jsonCache:         make(map[string]jsonCacheEntry),
		inflight:          make(map[string]*jsonCall),
	}
}

// guardedDial 拒绝连往内网地址。
//
// 判断必须放在 Dialer.Control 里:DialContext 收到的是 URL 里的主机名,
// 对域名做 netip.ParseAddr 只会失败,等于完全不设防;Control 则在每次实际
// connect 之前拿到解析后的具体地址,因此 DNS 重绑定同样逃不掉。
func guardedDial(dialer *net.Dialer, proxies *proxyAddresses) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if proxies.contains(address) {
			return dialer.DialContext(ctx, network, address)
		}
		guarded := *dialer
		guarded.Control = func(_, resolved string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(resolved)
			if err != nil {
				return fmt.Errorf("无法解析连接地址 %q: %w", resolved, err)
			}
			ip, err := netip.ParseAddr(host)
			if err != nil {
				return fmt.Errorf("无法解析连接地址 %q", resolved)
			}
			if !publicAddress(ip) {
				return fmt.Errorf("拒绝连接到非公网地址 %s", ip)
			}
			return nil
		}
		return guarded.DialContext(ctx, network, address)
	}
}

// proxyAddresses 记录环境变量里配置的代理地址。
// 这些地址由管理员显式设置，能改动它们的人本就控制着面板环境，
// 因此放行它们不会削弱针对重定向的防护。
type proxyAddresses struct {
	mu    sync.RWMutex
	known map[string]bool
}

func (p *proxyAddresses) remember(proxyURL *url.URL) {
	address := proxyURL.Host
	if proxyURL.Port() == "" {
		port := "80"
		if proxyURL.Scheme == "https" {
			port = "443"
		}
		address = net.JoinHostPort(proxyURL.Hostname(), port)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.known == nil {
		p.known = make(map[string]bool)
	}
	p.known[address] = true
}

func (p *proxyAddresses) contains(address string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.known[address]
}

// reservedRanges 是 netip 的内建判断没覆盖、但同样不该出现在上游下载链路上的网段。
// 100.64/10 是运营商级 NAT，也是 Tailscale 的地址段；198.18/15 常被用作 fake-ip；
// 240/4 与 0/8 是保留段。
//
// IPv6 侧还得挡住几类"把 IPv4 地址嵌进 v6 地址"的写法。主机上跑着 NAT64/CLAT 或
// 6to4 隧道时（纯 IPv6 云环境的常态，而面板恰恰常装在这种机器上），内核会把
// 64:ff9b::a9fe:a9fe、2002:a9fe:a9fe::1 还原成 169.254.169.254 再发出去，
// 而 netip 只把它们当普通全球单播，publicAddress 会直接放行。
// 这一层不能省：client.go 刻意不对重定向终点做域名白名单，全靠"不得落到内网地址"
// 这条不变量兜底，v6 侧漏一个口子等于整条防线在这类主机上失效。
var reservedRanges = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"), // 6to4 中继任播
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("::/96"),          // 已废弃的 IPv4-compatible，如 ::7f00:1
	netip.MustParsePrefix("64:ff9b::/96"),   // NAT64 众所周知前缀，RFC 6052
	netip.MustParsePrefix("64:ff9b:1::/48"), // NAT64 本地专用前缀，RFC 8215
	netip.MustParsePrefix("100::/64"),
	// 2001::/23 是 IETF 协议分配段，一条覆盖 Teredo(2001::/32)、
	// 基准测试段(2001:2::/48，即 198.18/15 的 v6 对等段)等；
	// 2001:db8::/32 在它之外，仍需单列。
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"), // 6to4
}

// publicAddress 排除回环、私网、链路本地与组播等不该出现在上游下载链路上的地址。
// 云元数据服务(169.254.169.254)正落在链路本地段内。
func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return false
	}
	for _, prefix := range reservedRanges {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func (c *httpClient) getJSON(ctx context.Context, url string, destination any) error {
	body, err := c.cachedJSON(ctx, url)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("解析上游响应: %w", err)
	}
	return nil
}

// cachedJSON 缓存 GitHub JSON 元数据，并合并相同 URL 的并发请求。
//
// 列表、Release 与 Actions 元数据都远比二进制稳定。上游暂时限流或不可达时，
// 最近一次成功结果仍比把版本管理整页打断更有用；安装阶段还有摘要校验与来源复核，
// 不会因为这层短时缓存而放宽信任边界。
func (c *httpClient) cachedJSON(ctx context.Context, target string) ([]byte, error) {
	now := c.now()
	c.cacheMu.Lock()
	if cached, ok := c.jsonCache[target]; ok && now.Sub(cached.fetchedAt) < jsonCacheTTL {
		body := append([]byte(nil), cached.body...)
		c.cacheMu.Unlock()
		return body, nil
	}
	if call, ok := c.inflight[target]; ok {
		c.cacheMu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-call.done:
			return append([]byte(nil), call.body...), call.err
		}
	}
	call := &jsonCall{done: make(chan struct{})}
	c.inflight[target] = call
	stale, hasStale := c.jsonCache[target]
	c.cacheMu.Unlock()

	requestCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	body, err := c.get(requestCtx, target, maxAPIBytes, "application/vnd.github+json", false)
	cancel()
	if err == nil && !json.Valid(body) {
		err = errors.New("解析上游响应: JSON 格式无效")
	}
	if err != nil && hasStale {
		// 失败后同样延长缓存窗口，避免每个页面请求都立即重试一个已限流的接口。
		body, err = append([]byte(nil), stale.body...), nil
	}

	c.cacheMu.Lock()
	if err == nil {
		c.storeJSON(target, body, now)
	}
	call.body, call.err = append([]byte(nil), body...), err
	delete(c.inflight, target)
	close(call.done)
	c.cacheMu.Unlock()
	return append([]byte(nil), body...), err
}

// storeJSON 保持缓存有界。容量很小时线性寻找最旧项比再维护一套 LRU 更精简。
func (c *httpClient) storeJSON(target string, body []byte, fetchedAt time.Time) {
	if _, exists := c.jsonCache[target]; !exists && len(c.jsonCache) >= maxJSONCacheEntries {
		var oldestTarget string
		var oldest time.Time
		for key, entry := range c.jsonCache {
			if oldestTarget == "" || entry.fetchedAt.Before(oldest) {
				oldestTarget, oldest = key, entry.fetchedAt
			}
		}
		delete(c.jsonCache, oldestTarget)
	}
	c.jsonCache[target] = jsonCacheEntry{body: append([]byte(nil), body...), fetchedAt: fetchedAt}
}

func (c *httpClient) getText(ctx context.Context, url string) (string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	body, err := c.get(requestCtx, url, maxAPIBytes, "", false)
	return string(body), err
}

// download 取回资产内容,内容长度硬上限为 limit。
func (c *httpClient) download(ctx context.Context, url string, limit int64) ([]byte, error) {
	requestCtx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	return c.get(requestCtx, url, limit, "", true)
}

// get 取回 target 的响应体，长度上限为 limit。
// identity 为真时禁用传输层压缩，用于必须逐字节比对校验和的资产下载。
func (c *httpClient) get(ctx context.Context, target string, limit int64, accept string, identity bool) ([]byte, error) {
	validateTarget := c.validateTarget
	if validateTarget == nil {
		validateTarget = checkFirstHopContext
	}
	if err := validateTarget(ctx, target); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("构造上游请求: %w", err)
	}
	request.Header.Set("User-Agent", userAgent)
	if identity {
		// 下载资产时让落盘的字节就是链路上的字节，便于校验和比对。
		// 这条只对资产成立：加在 JSON 接口上纯属自伤——版本列表是高度可压缩的
		// 文本，禁用压缩会让它多传十几倍字节，还得挤进 20 秒的接口超时里。
		request.Header.Set("Accept-Encoding", "identity")
	}
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	c.authorizeGitHubAPI(request)

	response, err := c.client.Do(request)
	if err != nil {
		// 不能直接 %w 包裹 *url.Error：它带着完整 URL，
		// 而签名下载地址的查询串里含有临时凭据，会随错误进日志和 API 响应。
		return nil, fmt.Errorf("请求上游 %s 失败: %s", redact(target), redactError(err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, describeHTTPError(response, target)
	}
	// 先信任 Content-Length 做早期拒绝，真正的上限仍由 LimitReader 保证。
	if response.ContentLength > limit {
		return nil, fmt.Errorf("上游响应 %d 字节，超过 %d 上限", response.ContentLength, limit)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("读取上游响应: %w", err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("上游响应超过 %d 字节上限", limit)
	}
	return body, nil
}

// authorizeGitHubAPI 只把令牌发给 API 首跳。发布资产和 nightly.link 不需要令牌，
// CheckRedirect 还会再次清除 Authorization，避免凭据跟随重定向外泄。
func (c *httpClient) authorizeGitHubAPI(request *http.Request) {
	if !strings.EqualFold(request.URL.Hostname(), "api.github.com") {
		return
	}
	token := strings.TrimSpace(c.githubTokenSource.GitHubToken())
	if token == "" {
		return
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func describeHTTPError(response *http.Response, target string) error {
	switch response.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("上游不存在该资源: %s", redact(target))
	case http.StatusForbidden, http.StatusTooManyRequests:
		// 匿名调用 GitHub API 是每 IP 每小时 60 次，触顶时这里给出可读提示。
		if response.Header.Get("X-RateLimit-Remaining") == "0" {
			reset := response.Header.Get("X-RateLimit-Reset")
			return fmt.Errorf("GitHub 接口调用频率已达上限，请稍后重试或配置 KDAE_PANEL_GITHUB_TOKEN（重置时间戳 %s）", reset)
		}
		return fmt.Errorf("上游拒绝访问（HTTP %d）", response.StatusCode)
	default:
		return fmt.Errorf("上游返回 HTTP %d", response.StatusCode)
	}
}

// checkFirstHop 确认面板主动发起的请求指向已知主机。
// 这些地址全部由本包自己拼装，校验只是防止将来有人把外部字符串接进来。
func checkFirstHopContext(_ context.Context, target string) error {
	return checkFirstHop(target)
}

func checkFirstHop(target string) error {
	parsed, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("上游地址无效: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("上游地址必须使用 HTTPS")
	}
	if !allowedHosts[parsed.Hostname()] {
		return fmt.Errorf("上游主机 %s 不在允许列表内", parsed.Hostname())
	}
	return nil
}

func checkHTTPSRedirectTarget(_ context.Context, target string) error {
	_, err := parsePublicHTTPSURL(target)
	return err
}

// checkPublicHTTPSTarget 校验自定义来源。保存配置时只做语法检查；真正请求前和
// 每次重定向都会在这里重新解析 DNS，并拒绝任一非公网结果。随后 guardedDial 还会
// 在 connect 前检查实际地址，堵住检查与连接之间的 DNS 重绑定窗口。
func checkPublicHTTPSTarget(ctx context.Context, target string) error {
	parsed, err := parsePublicHTTPSURL(target)
	if err != nil {
		return err
	}
	host := parsed.Hostname()
	if address, err := netip.ParseAddr(host); err == nil {
		if !publicAddress(address) {
			return fmt.Errorf("拒绝连接到非公网地址 %s", address)
		}
		return nil
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("解析自定义来源主机 %s: %w", host, err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("自定义来源主机 %s 没有可用地址", host)
	}
	for _, address := range addresses {
		if !publicAddress(address) {
			return fmt.Errorf("自定义来源主机 %s 解析到非公网地址 %s", host, address)
		}
	}
	return nil
}

func parsePublicHTTPSURL(target string) (*url.URL, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		// url.ParseError 会带回完整原文，而自定义链接的查询串可能含临时凭据。
		// 对外只说明格式问题，不能把原始链接送进 API 错误或 journald。
		return nil, errors.New("下载地址格式无效")
	}
	if parsed.Scheme != "https" {
		return nil, errors.New("下载地址必须使用 HTTPS")
	}
	if parsed.Hostname() == "" {
		return nil, errors.New("下载地址缺少主机名")
	}
	if parsed.User != nil {
		return nil, errors.New("下载地址不能包含用户名或密码")
	}
	if parsed.Fragment != "" {
		return nil, errors.New("下载地址不能包含片段")
	}
	if address, err := netip.ParseAddr(parsed.Hostname()); err == nil && !publicAddress(address) {
		return nil, fmt.Errorf("下载地址不能指向非公网地址 %s", address)
	}
	return parsed, nil
}

// redact 去掉签名下载地址里的查询串，避免把临时凭据写进日志或错误消息。
func redact(target string) string {
	if index := strings.IndexByte(target, '?'); index >= 0 {
		return target[:index]
	}
	return target
}

// redactError 取出底层错误原因，丢掉 *url.Error 携带的完整地址。
func redactError(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err.Error()
	}
	return err.Error()
}
