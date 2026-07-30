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
	configuration, err := configstore.NewManager(cfg.DaeConfigPath, cfg.BackupDir, daeClient)
	if err != nil {
		return nil, fmt.Errorf("初始化配置管理器: %w", err)
	}
	hostManager, err := host.New(host.Options{
		Backend:     host.BackendAuto,
		ServiceName: cfg.ServiceName,
		DaeBinary:   cfg.DaeBinary,
		Systemctl:   cfg.Systemctl,
		Journalctl:  cfg.Journalctl,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化主机服务管理器: %w", err)
	}
	authStore, err := auth.Open(cfg.DatabasePath, cfg.SessionTTL)
	if err != nil {
		return nil, fmt.Errorf("初始化认证服务: %w", err)
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
		Dae:            daeClient,
		Configuration:  configuration,
		Host:           hostManager,
		Authentication: authStore,
		Probe:          netprobe.New(),
	}
	if cfg.EnableDaeInstall {
		installer, err := daeinstall.New(daeinstall.Options{
			BinaryPath:  cfg.DaeBinary,
			ConfigPath:  cfg.DaeConfigPath,
			StatePath:   cfg.InstallStatePath,
			ServiceName: cfg.ServiceName,
			Fetcher:     upstream.NewDefaultRegistry(),
			Service:     hostManager,
			Logger:      logger,
		})
		if err != nil {
			_ = authStore.Close()
			return nil, fmt.Errorf("初始化 dae 版本管理: %w", err)
		}
		dependencies.Install = installer
	}
	if cfg.EnableGeoUpdate {
		manager, err := geodata.New(geodata.Options{
			ConfigPath: cfg.DaeConfigPath,
			StatePath:  cfg.GeoStatePath,
			Fetcher:    upstream.NewGeoRegistry(),
			Service:    hostManager,
			Reloader:   daeClient,
			Logger:     logger,
		})
		if err != nil {
			_ = authStore.Close()
			return nil, fmt.Errorf("初始化 geo 数据更新: %w", err)
		}
		dependencies.Geo = manager
	}
	updater, err := panelupdate.New(panelupdate.Options{
		Version:    cfg.Version,
		BackupPath: cfg.PanelBackupPath,
		Enabled:    cfg.EnableSelfUpdate,
		Fetcher:    upstream.NewPanelFetcher(),
		Service:    hostManager,
		Logger:     logger,
	})
	if err != nil {
		_ = authStore.Close()
		return nil, fmt.Errorf("初始化面板自升级: %w", err)
	}
	dependencies.PanelUpdate = updater
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
				return dependencies.Dae.Reload(ctx)
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
	if dependencies.Geo != nil {
		geo = newGeoUpdater(dependencies.Geo, operations, logger)
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
		})
	})
	panelRelease := dependencies.PanelRelease
	if panelRelease == nil && !cfg.DisableUpdateCheck {
		panelRelease = func(ctx context.Context) (string, error) {
			return upstream.LatestPanelRelease(ctx, upstream.PanelRepoOwner, upstream.PanelRepoName)
		}
	}
	registerPanelUpdateRoutes(router, cfg.Version, panelRelease, dependencies.PanelUpdate, operations, logger)
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
	registerUpstreamRoutes(router, dependencies.Install, operations, logger)
	registerGeoRoutes(router, geo)
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
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
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
	if err != nil || (host != "" && host != "0.0.0.0") {
		return []string{fallback}
	}

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
		if ip = ip.To4(); ip == nil || !ip.IsPrivate() {
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
