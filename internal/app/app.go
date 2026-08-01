package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tuoro/kdae-panel/internal/atomicfile"
	"github.com/tuoro/kdae-panel/internal/auth"
	"github.com/tuoro/kdae-panel/internal/configstore"
	"github.com/tuoro/kdae-panel/internal/dae"
	"github.com/tuoro/kdae-panel/internal/daeinstall"
	"github.com/tuoro/kdae-panel/internal/geodata"
	"github.com/tuoro/kdae-panel/internal/githubauth"
	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/netprobe"
	"github.com/tuoro/kdae-panel/internal/panelupdate"
	"github.com/tuoro/kdae-panel/internal/schedule"
	"github.com/tuoro/kdae-panel/internal/upstream"
	"github.com/tuoro/kdae-panel/internal/webui"
)

type App struct {
	handler    http.Handler
	closers    []io.Closer
	operations *sync.Mutex
}

type DaeService interface {
	Inspect(ctx context.Context) dae.Report
	Outline(ctx context.Context) (dae.Outline, error)
	Reload(ctx context.Context) error
	Suspend(ctx context.Context, abort bool) error
	Sysdump(ctx context.Context) (dae.Sysdump, error)
}

type ConfigurationService interface {
	Read(ctx context.Context) (configstore.Document, error)
	Validate(ctx context.Context, content string) error
	Save(ctx context.Context, content, expectedHash string, apply bool) (configstore.SaveResult, error)
	ListBackups(ctx context.Context) ([]configstore.Backup, error)
	CreateBackup(ctx context.Context, name, note string) (configstore.Backup, error)
	UpdateBackup(ctx context.Context, backupID, name, note string) (configstore.Backup, error)
	DeleteBackup(ctx context.Context, backupID string) error
	Restore(ctx context.Context, backupID, expectedHash string, apply bool) (configstore.SaveResult, error)
}

type Dependencies struct {
	Dae            DaeService
	Configuration  ConfigurationService
	Host           HostService
	Authentication AuthenticationService
	Probe          ProbeService
	Schedule       ScheduleService
	GeoSchedule    ScheduleService
	Install        InstallService
	Geo            GeoService
	PanelRelease   PanelReleaseChecker
	PanelUpdate    PanelUpdateService
	GitHub         GitHubCredentialService
}

type AuthenticationService interface {
	Initialized(ctx context.Context) (bool, error)
	Setup(ctx context.Context, username, password string) (auth.Session, error)
	Login(ctx context.Context, username, password string) (auth.Session, error)
	GetSession(ctx context.Context, token string) (auth.Session, error)
	Logout(ctx context.Context, token string) error
	ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) (auth.Session, error)
}

type HostService interface {
	Status(ctx context.Context) (host.Status, error)
	Action(ctx context.Context, action host.Action) error
	Logs(ctx context.Context, limit int) ([]host.LogEntry, error)
	Interfaces(ctx context.Context) ([]host.NetworkInterface, error)
}

func New(cfg Config, logger *slog.Logger) (*App, error) {
	cfg = cfg.withDefaults()
	daeClient := dae.NewClient(cfg.DaeBinary)
	hostManager, err := host.New(host.Options{
		Backend:     cfg.ServiceBackend,
		ServiceName: cfg.ServiceName,
		DaeBinary:   cfg.DaeBinary,
		Systemctl:   cfg.Systemctl,
		Journalctl:  cfg.Journalctl,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化主机服务管理器: %w", err)
	}
	resolvedBackend, err := cfg.ServiceBackend.Resolve()
	if err != nil {
		return nil, fmt.Errorf("解析服务后端: %w", err)
	}
	logger.Info("已选定服务后端", "backend", string(resolvedBackend))
	adoptRunningServiceBootState(hostManager, logger)
	daeService := newPIDReloadDaeService(daeClient, daeClient, daeClient, hostManager)
	configuration, err := configstore.NewManager(cfg.DaeConfigPath, cfg.BackupDir, daeService)
	if err != nil {
		return nil, fmt.Errorf("初始化配置管理器: %w", err)
	}
	authStore, err := auth.Open(cfg.DatabasePath, cfg.SessionTTL)
	if err != nil {
		return nil, fmt.Errorf("初始化认证服务: %w", err)
	}
	githubCredentials, err := githubauth.Open(cfg.GitHubTokenPath, os.Getenv("KDAE_PANEL_GITHUB_TOKEN"))
	if err != nil {
		_ = authStore.Close()
		return nil, fmt.Errorf("初始化 GitHub API 凭据: %w", err)
	}
	initialized, err := authStore.Initialized(context.Background())
	if err != nil {
		_ = authStore.Close()
		return nil, fmt.Errorf("检查管理员初始化状态: %w", err)
	}
	var setupURLs []string
	if !initialized {
		if cfg.BootstrapToken == "" {
			cfg.BootstrapToken, err = newBootstrapToken()
			if err != nil {
				_ = authStore.Close()
				return nil, err
			}
		}
		setupURLs = bootstrapSetupURLs(cfg.ListenAddress, cfg.BootstrapToken)
	}
	dependencies := Dependencies{
		Dae:            daeService,
		Configuration:  configuration,
		Host:           hostManager,
		Authentication: authStore,
		Probe:          netprobe.New(),
		GitHub:         githubCredentials,
	}
	if cfg.EnableDaeInstall {
		installer, err := daeinstall.New(daeinstall.Options{
			BinaryPath:     cfg.DaeBinary,
			ConfigPath:     cfg.DaeConfigPath,
			StatePath:      cfg.InstallStatePath,
			ServiceName:    cfg.ServiceName,
			Fetcher:        upstream.NewDefaultRegistryWithGitHubToken(githubCredentials),
			Service:        hostManager,
			Logger:         logger,
			ServiceBackend: cfg.ServiceBackend,
		})
		if err != nil {
			_ = authStore.Close()
			return nil, fmt.Errorf("初始化 dae 版本管理: %w", err)
		}
		dependencies.Install = installer
	}
	// 上游 v0.8.x 起把 geo 管理改成恒定可用，EnableGeoUpdate 退化成兼容字段。
	// 本分支保留这道闸：OpenWrt 上面板以完整 root 运行、没有任何沙箱，
	// enable_geo_update 是 UCI 里真实存在的开关，SECURITY.md 也按"关得掉"写。
	// 让它变成空开关，等于把我们自己的文档改成假的。
	// 上游的 registerGeoRoutes 本来就为 updater==nil 留了 503 geo_update_disabled
	// 分支，走这条路不需要偏离上游代码。
	if cfg.EnableGeoUpdate {
		geoRegistry, err := upstream.OpenGeoRegistryWithGitHubToken(githubCredentials, cfg.GeoSourcesPath)
		if err != nil {
			_ = authStore.Close()
			return nil, fmt.Errorf("初始化 geo 数据来源: %w", err)
		}
		manager, err := geodata.New(geodata.Options{
			ConfigPath:     cfg.DaeConfigPath,
			StatePath:      cfg.GeoStatePath,
			Fetcher:        geoRegistry,
			Service:        hostManager,
			Reloader:       daeClient,
			Logger:         logger,
			ServiceBackend: cfg.ServiceBackend,
		})
		if err != nil {
			_ = authStore.Close()
			return nil, fmt.Errorf("初始化 geo 数据更新: %w", err)
		}
		dependencies.Geo = manager
	}
	// procd 部署由 opkg 管理升级，这里根本不构造自升级能力。
	//
	// 不是"默认关"而是"不存在"：PanelFetcher 指向的是上游 tuoro/kdae-panel，
	// 那里的发布二进制不含 procd 后端。开启后升级一次，面板就会以 root 把自己
	// 替换成一个只会调 systemctl 的程序，重启即不可用。一个开了就砖的开关，
	// 靠默认值和文案是拦不住的。
	//
	// 检查保留，只是换一个仓库问：能不能自己升级，与该不该知道有新版本，
	// 是两件事。指向 PackageRepo（两个 ipk 的发布仓库）之后，"最新版本"说的
	// 正是用户 opkg 装得上的那一版；沿用上游坐标才是误导。
	if resolvedBackend == host.BackendProcd {
		if !cfg.DisableUpdateCheck {
			dependencies.PanelRelease = upstream.NewPackageReleaseChecker(githubCredentials)
		}
	} else {
		panelFetcher := upstream.NewPanelFetcherWithGitHubToken(githubCredentials)
		updater, err := panelupdate.New(panelupdate.Options{
			Version:    cfg.Version,
			BackupPath: cfg.PanelBackupPath,
			Enabled:    cfg.EnableSelfUpdate,
			Fetcher:    panelFetcher,
			Service:    hostManager,
			Logger:     logger,
		})
		if err != nil {
			_ = authStore.Close()
			return nil, fmt.Errorf("初始化面板自升级: %w", err)
		}
		dependencies.PanelUpdate = updater
		if !cfg.DisableUpdateCheck {
			dependencies.PanelRelease = panelFetcher.LatestVersion
		}
	}
	application, err := NewWithDependencies(cfg, logger, dependencies)
	if err != nil {
		_ = authStore.Close()
		return nil, err
	}
	application.closers = append(application.closers, authStore)
	if err := syncSetupURLFile(cfg.SetupURLFile, setupURLs); err != nil {
		_ = application.Close()
		return nil, fmt.Errorf("同步首次访问链接: %w", err)
	}
	for _, setupURL := range setupURLs {
		logger.Warn("首次初始化请打开一次性链接", "setup_url", setupURL)
	}
	return application, nil
}

// adoptRunningServiceBootState 迁移旧版面板留下的“正在运行但未启用”状态。
// 新版服务控制会在每次启停时同步开机状态；这里只补一次历史缺口，而且绝不根据
// inactive 状态自动 disable，避免面板与 dae 在开机时并行启动造成竞态。
func adoptRunningServiceBootState(service HostService, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	status, err := service.Status(ctx)
	if err != nil || status.ActiveState != "active" || status.UnitFileState != "disabled" {
		return
	}
	if err := service.Action(ctx, host.ActionEnable); err != nil {
		logger.Warn("无法把正在运行的 dae 迁移为随系统启动", "error", err)
		return
	}
	logger.Info("已把正在运行的 dae 迁移为随系统启动")
}

func NewWithDae(cfg Config, logger *slog.Logger, daeService DaeService) (*App, error) {
	return NewWithDependencies(cfg, logger, Dependencies{Dae: daeService})
}

func NewWithDependencies(cfg Config, logger *slog.Logger, dependencies Dependencies) (*App, error) {
	if dependencies.Dae == nil {
		return nil, errors.New("dae 服务不能为空")
	}
	proxyTrust, err := parseProxyTrust(cfg.TrustedProxies)
	if err != nil {
		return nil, err
	}
	// 后端在这里再解析一次：New 里解析好的那个传不进来，而本函数是测试直接
	// 调用的入口。Resolve 只做一次 os.Stat，解析失败时 New 早已报错退出，
	// 这里的兜底只为不 panic。健康检查与"功能未启用"的指引都要用它——
	// 那两处提示在两套部署里指向的是完全不同的开关。
	backend, err := cfg.ServiceBackend.Resolve()
	if err != nil {
		backend = cfg.ServiceBackend
	}
	router := http.NewServeMux()
	operations := &sync.Mutex{}
	application := &App{operations: operations}

	scheduleService := dependencies.Schedule
	if scheduleService == nil && cfg.SchedulePath != "" {
		runner, err := schedule.New(schedule.Options{
			Path:   cfg.SchedulePath,
			Name:   "订阅自动刷新",
			Logger: logger,
			Task: func(ctx context.Context) error {
				if !operations.TryLock() {
					return errors.New("另一个控制操作正在执行，本轮已跳过")
				}
				defer operations.Unlock()
				err := dependencies.Dae.Reload(ctx)
				if errors.Is(err, configstore.ErrReloadDeferred) {
					return nil
				}
				return err
			},
		})
		if err != nil {
			return nil, fmt.Errorf("初始化订阅自动刷新: %w", err)
		}
		application.closers = append(application.closers, runner)
		scheduleService = runner
	}

	// geo 更新器同时服务手动端点与定时任务，任务追踪器只有一份。
	var geo *geoUpdater
	var geoSources GeoSourceService
	if dependencies.Geo != nil {
		geo = newGeoUpdater(dependencies.Geo, operations, logger)
		geoSources, _ = dependencies.Geo.(GeoSourceService)
	}
	geoScheduleService := dependencies.GeoSchedule
	if geoScheduleService == nil && geo != nil && cfg.GeoSchedulePath != "" {
		runner, err := schedule.New(schedule.Options{
			Path:    cfg.GeoSchedulePath,
			Name:    "geo 数据自动更新",
			Logger:  logger,
			Timeout: geoUpdateTimeout,
			Task:    geo.runScheduled,
		})
		if err != nil {
			return nil, fmt.Errorf("初始化 geo 数据自动更新: %w", err)
		}
		application.closers = append(application.closers, runner)
		geoScheduleService = runner
	}
	router.HandleFunc("GET /api/v1/health", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]any{
			"status":  "ok",
			"version": cfg.Version,
			"backend": string(backend),
		})
	})
	// 检查哪个仓库由后端决定：procd 装的是 ipk，上游 tag 与它的版本线不是
	// 一回事。这个兜底分支是 NewWithDependencies 被直接调用时的唯一入口，
	// 不跟着分后端的话，New 里刚立的规矩在这里就破了。
	releaseOwner, releaseRepo := upstream.PanelRepoOwner, upstream.PanelRepoName
	if backend == host.BackendProcd {
		releaseOwner, releaseRepo = upstream.PackageRepoOwner, upstream.PackageRepoName
	}
	panelRelease := dependencies.PanelRelease
	if panelRelease == nil && !cfg.DisableUpdateCheck {
		panelRelease = func(ctx context.Context) (string, error) {
			return upstream.LatestPanelRelease(ctx, releaseOwner, releaseRepo)
		}
	}
	registerPanelUpdateRoutes(router, cfg.Version, panelRelease, dependencies.PanelUpdate,
		upstream.ReleasesURL(releaseOwner, releaseRepo), operations, logger)
	registerGitHubCredentialRoutes(router, dependencies.GitHub)
	router.HandleFunc("GET /api/v1/dae/capabilities", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, dependencies.Dae.Inspect(request.Context()))
	})
	router.HandleFunc("GET /api/v1/dae/outline", func(writer http.ResponseWriter, request *http.Request) {
		outline, err := dependencies.Dae.Outline(request.Context())
		if err != nil {
			writeAPIError(writer, http.StatusServiceUnavailable, "dae_outline_unavailable", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, outline)
	})
	registerConfigurationRoutes(router, dependencies.Configuration, operations)
	registerServiceRoutes(router, dependencies.Dae, dependencies.Host, operations)
	registerProbeRoutes(router, dependencies.Probe, logger)
	registerScheduleRoutes(router, "/api/v1/schedule/reload", scheduleService)
	registerScheduleRoutes(router, "/api/v1/schedule/geo", geoScheduleService)
	registerUpstreamRoutes(router, dependencies.Install, operations, logger, backend)
	registerGeoRoutes(router, geo, geoSources, backend)
	registerAuthenticationRoutes(router, dependencies.Authentication, cfg.SecureCookie, cfg.BootstrapToken, cfg.SetupURLFile, proxyTrust, logger)
	apiNotFound := func(writer http.ResponseWriter, _ *http.Request) {
		writeAPIError(writer, http.StatusNotFound, "api_not_found", "API 路径不存在")
	}
	router.HandleFunc("/api", apiNotFound)
	router.HandleFunc("/api/", apiNotFound)
	router.Handle("/", webui.Handler())

	var handler http.Handler = router
	if dependencies.Authentication != nil {
		handler = authenticationMiddleware(handler, dependencies.Authentication, cfg.SecureCookie, proxyTrust)
	}
	handler = securityHeaders(handler, proxyTrust)
	handler = recoverer(handler, logger)
	handler = requestLogger(logger, proxyTrust)(handler)
	application.handler = handler
	return application, nil
}

func acquireOperation(writer http.ResponseWriter, operations *sync.Mutex) bool {
	if operations.TryLock() {
		return true
	}
	writeAPIError(writer, http.StatusConflict, "operation_in_progress", "另一个控制操作正在执行，请稍后重试")
	return false
}

func newBootstrapToken() (string, error) {
	content := make([]byte, 24)
	if _, err := rand.Read(content); err != nil {
		return "", fmt.Errorf("生成 bootstrap token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(content), nil
}

// syncSetupURLFile 只在等待首次初始化时留下链接；已初始化启动会清理旧文件。
func syncSetupURLFile(path string, setupURLs []string) error {
	if path == "" {
		return nil
	}
	if len(setupURLs) == 0 {
		return removeSetupURLFile(path)
	}
	content := []byte(strings.Join(setupURLs, "\n") + "\n")
	return atomicfile.Write(path, content, 0o600)
}

func removeSetupURLFile(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func bootstrapSetupURL(listenAddress, token string) string {
	fragment := "bootstrap=" + token
	rawFragment := "bootstrap=" + strings.ReplaceAll(url.QueryEscape(token), "+", "%20")
	host, port, err := net.SplitHostPort(listenAddress)
	if err != nil {
		return "/setup#" + rawFragment
	}
	switch host {
	case "", "0.0.0.0":
		host = "127.0.0.1"
	case "::":
		host = "::1"
	}
	return (&url.URL{
		Scheme:      "http",
		Host:        net.JoinHostPort(host, port),
		Path:        "/setup",
		Fragment:    fragment,
		RawFragment: rawFragment,
	}).String()
}

func bootstrapSetupURLs(listenAddress, token string) []string {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return []string{bootstrapSetupURL(listenAddress, token)}
	}
	return bootstrapSetupURLsForAddresses(listenAddress, token, addresses)
}

func bootstrapSetupURLsForAddresses(listenAddress, token string, addresses []net.Addr) []string {
	fallback := bootstrapSetupURL(listenAddress, token)
	host, port, err := net.SplitHostPort(listenAddress)
	if err != nil || !isWildcardListenHost(host) {
		return []string{fallback}
	}
	// 监听 0.0.0.0 的 socket 收不到 IPv6 连接，列出 v6 地址只会给出一条打不开的链接。
	allowIPv6 := host == "" || host == "::"

	seen := make(map[string]struct{})
	urls := make([]string, 0, len(addresses)+1)
	for _, address := range addresses {
		var ip net.IP
		switch value := address.(type) {
		case *net.IPNet:
			ip = value.IP
		case *net.IPAddr:
			ip = value.IP
		default:
			continue
		}
		if ipv4 := ip.To4(); ipv4 != nil {
			ip = ipv4
		} else if !allowIPv6 {
			continue
		}
		if !isLANAddress(ip) {
			continue
		}
		setupURL := bootstrapSetupURL(net.JoinHostPort(ip.String(), port), token)
		if _, exists := seen[setupURL]; exists {
			continue
		}
		seen[setupURL] = struct{}{}
		urls = append(urls, setupURL)
	}
	sort.Strings(urls)
	if len(urls) > 0 {
		return urls
	}
	return []string{fallback}
}

func isWildcardListenHost(host string) bool {
	return host == "" || host == "0.0.0.0" || host == "::"
}

// isLANAddress 判断一个接口地址值不值得写进一次性初始化链接。
//
// 链接的 fragment 里带着 bootstrap token，所以只列局域网地址。IPv4 收 RFC1918
// 与 CGNAT（100.64/10，ISP 下发给路由器的常见形态），公网 IPv4 一律排除：路由器
// 上的公网 v4 在 WAN 侧，把 token 渲染成一条指向公网的明文 HTTP 链接摆在管理员
// 面前，只会诱导他把它发到互联网上去。
//
// IPv6 相反，ULA 与全局单播都收：v6 局域网通常不做 NAT，br-lan 上那个 2001:
// 开头的地址就是局域网客户端要访问的地址，按"是不是公网"筛会把它整个筛掉。
//
// IsGlobalUnicast 已经排除了 loopback、unspecified、multicast、link-local 与
// IPv4 广播地址，不必再逐条重复。
func isLANAddress(ip net.IP) bool {
	if !ip.IsGlobalUnicast() {
		return false
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		// 100.64.0.0/10：第二个八位组的高两位恰好是 01。
		return ipv4.IsPrivate() || (ipv4[0] == 100 && ipv4[1]&0xc0 == 64)
	}
	return true
}

func writeAPIError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func (a *App) Close() error {
	var result error
	for index := len(a.closers) - 1; index >= 0; index-- {
		result = errors.Join(result, a.closers[index].Close())
	}
	return result
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status   int
	bytes    int
	username string
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Write(content []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	written, err := w.ResponseWriter.Write(content)
	w.bytes += written
	return written, err
}

func (w *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func requestLogger(logger *slog.Logger, proxyTrust proxyTrust) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			recorder := &loggingResponseWriter{ResponseWriter: writer}
			next.ServeHTTP(recorder, request)
			if recorder.status == 0 {
				recorder.status = http.StatusOK
			}
			logger.Info("HTTP 请求",
				"method", request.Method,
				"path", request.URL.Path,
				"status", recorder.status,
				"bytes", recorder.bytes,
				"client", proxyTrust.clientAddress(request),
				"user", recorder.username,
				"duration", time.Since(startedAt),
			)
		})
	}
}

func recoverer(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("HTTP 处理发生异常", "panic", recovered, "path", request.URL.Path)
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(writer, request)
	})
}
