package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/tuoro/kdae-panel/internal/app"
	"github.com/tuoro/kdae-panel/internal/host"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("程序退出", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := app.DefaultConfig()
	sessionTTLDefault, err := envDuration("KDAE_PANEL_SESSION_TTL", cfg.SessionTTL)
	if err != nil {
		return err
	}
	secureCookieDefault, err := envBool("KDAE_PANEL_SECURE_COOKIE", cfg.SecureCookie)
	if err != nil {
		return err
	}
	enableDaeInstallDefault, err := envBool("KDAE_PANEL_ENABLE_DAE_INSTALL", cfg.EnableDaeInstall)
	if err != nil {
		return err
	}
	enableGeoUpdateDefault, err := envBool("KDAE_PANEL_ENABLE_GEO_UPDATE", cfg.EnableGeoUpdate)
	if err != nil {
		return err
	}
	disableUpdateCheckDefault, err := envBool("KDAE_PANEL_DISABLE_UPDATE_CHECK", cfg.DisableUpdateCheck)
	if err != nil {
		return err
	}
	enableSelfUpdateDefault, err := envBool("KDAE_PANEL_ENABLE_SELF_UPDATE", cfg.EnableSelfUpdate)
	if err != nil {
		return err
	}
	listen := flag.String("listen", envOr("KDAE_PANEL_LISTEN", cfg.ListenAddress), "HTTP 监听地址")
	bootstrapToken := flag.String("bootstrap-token", envOr("KDAE_PANEL_BOOTSTRAP_TOKEN", cfg.BootstrapToken), "首次初始化 bootstrap token")
	setupURLFile := flag.String("setup-url-file", envOr("KDAE_PANEL_SETUP_URL_FILE", cfg.SetupURLFile), "首次初始化链接临时文件")
	trustedProxies := flag.String("trusted-proxies", envOr("KDAE_PANEL_TRUSTED_PROXIES", cfg.TrustedProxies), "可信反向代理 CIDR，逗号分隔")
	daeBinary := flag.String("dae-binary", envOr("KDAE_PANEL_DAE_BINARY", cfg.DaeBinary), "dae 可执行文件路径")
	daeConfig := flag.String("dae-config", envOr("KDAE_PANEL_DAE_CONFIG", cfg.DaeConfigPath), "dae 入口配置文件路径")
	backupDir := flag.String("backup-dir", envOr("KDAE_PANEL_BACKUP_DIR", cfg.BackupDir), "配置备份目录")
	// 这个 flag 在 procd 部署上同样生效（openwrt/kdae-panel 的 init 脚本会传它），
	// 帮助文案不能只讲 systemd 那一半，否则 procd 用户会以为它跟自己无关。
	serviceName := flag.String("service-name", envOr("KDAE_PANEL_SERVICE_NAME", cfg.ServiceName), "dae 服务名（systemd 下是单元名，procd 下是 /etc/init.d/ 下的脚本名）")
	systemctl := flag.String("systemctl", envOr("KDAE_PANEL_SYSTEMCTL", cfg.Systemctl), "systemctl 可执行文件路径（仅 systemd 后端使用）")
	journalctl := flag.String("journalctl", envOr("KDAE_PANEL_JOURNALCTL", cfg.Journalctl), "journalctl 可执行文件路径（仅 systemd 后端使用）")
	serviceBackend := flag.String("service-backend", envOr("KDAE_PANEL_SERVICE_BACKEND", string(cfg.ServiceBackend)), "服务后端：auto、systemd 或 procd")
	databasePath := flag.String("database", envOr("KDAE_PANEL_DATABASE", cfg.DatabasePath), "面板 SQLite 数据库路径")
	schedulePath := flag.String("schedule-file", envOr("KDAE_PANEL_SCHEDULE_FILE", cfg.SchedulePath), "订阅自动刷新设置文件路径")
	installStatePath := flag.String("install-state-file", envOr("KDAE_PANEL_INSTALL_STATE_FILE", cfg.InstallStatePath), "dae 版本安装状态文件路径")
	githubTokenPath := flag.String("github-token-file", envOr("KDAE_PANEL_GITHUB_TOKEN_FILE", cfg.GitHubTokenPath), "GitHub API Token 持久化文件路径")
	geoStatePath := flag.String("geo-state-file", envOr("KDAE_PANEL_GEO_STATE_FILE", cfg.GeoStatePath), "geo 数据更新状态文件路径")
	geoSchedulePath := flag.String("geo-schedule-file", envOr("KDAE_PANEL_GEO_SCHEDULE_FILE", cfg.GeoSchedulePath), "geo 数据自动更新设置文件路径")
	geoSourcesPath := flag.String("geo-sources-file", envOr("KDAE_PANEL_GEO_SOURCES_FILE", cfg.GeoSourcesPath), "自定义 geo 数据来源文件路径")
	enableDaeInstall := flag.Bool("enable-dae-install", enableDaeInstallDefault, "允许通过面板安装与切换 dae 版本")
	enableGeoUpdate := flag.Bool("enable-geo-update", enableGeoUpdateDefault, "兼容旧版本；Geo 数据管理现已始终启用")
	disableUpdateCheck := flag.Bool("disable-update-check", disableUpdateCheckDefault, "关闭面板自身的新版本检查")
	enableSelfUpdate := flag.Bool("enable-self-update", enableSelfUpdateDefault, "允许面板一键升级自身")
	panelBackupPath := flag.String("panel-backup-file", envOr("KDAE_PANEL_BACKUP_FILE", cfg.PanelBackupPath), "自升级时保留的上一版面板二进制路径")
	sessionTTL := flag.Duration("session-ttl", sessionTTLDefault, "登录会话有效期")
	secureCookie := flag.Bool("secure-cookie", secureCookieDefault, "仅通过 HTTPS 发送登录 Cookie")
	showVersion := flag.Bool("version", false, "显示版本")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return nil
	}
	cfg.ListenAddress = *listen
	cfg.BootstrapToken = *bootstrapToken
	cfg.SetupURLFile = *setupURLFile
	cfg.TrustedProxies = *trustedProxies
	cfg.DaeBinary = *daeBinary
	cfg.DaeConfigPath = *daeConfig
	cfg.BackupDir = *backupDir
	cfg.ServiceName = *serviceName
	cfg.Systemctl = *systemctl
	cfg.Journalctl = *journalctl
	cfg.ServiceBackend = host.Backend(*serviceBackend)
	cfg.DatabasePath = *databasePath
	cfg.SchedulePath = *schedulePath
	cfg.InstallStatePath = *installStatePath
	cfg.GitHubTokenPath = *githubTokenPath
	cfg.GeoStatePath = *geoStatePath
	cfg.GeoSchedulePath = *geoSchedulePath
	cfg.GeoSourcesPath = *geoSourcesPath
	cfg.EnableDaeInstall = *enableDaeInstall
	cfg.EnableGeoUpdate = *enableGeoUpdate
	cfg.DisableUpdateCheck = *disableUpdateCheck
	cfg.EnableSelfUpdate = *enableSelfUpdate
	cfg.PanelBackupPath = *panelBackupPath
	cfg.SessionTTL = *sessionTTL
	cfg.SecureCookie = *secureCookie
	cfg.Version = version

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	application, err := app.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("初始化应用: %w", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			logger.Error("关闭应用资源失败", "error", err)
		}
	}()

	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           application.Handler(),
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      180 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("面板服务已启动", "listen", cfg.ListenAddress, "version", cfg.Version)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP 服务: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("关闭 HTTP 服务: %w", err)
	}
	logger.Info("面板服务已停止")
	return nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("环境变量 %s=%q 不是有效布尔值: %w", key, value, err)
	}
	return parsed, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("环境变量 %s=%q 不是有效时长: %w", key, value, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("环境变量 %s 必须大于 0", key)
	}
	return parsed, nil
}
