package app

import (
	"time"

	"github.com/tuoro/kdae-panel/internal/host"
)

type Config struct {
	ListenAddress  string
	Version        string
	BootstrapToken string
	// SetupURLFile 是 systemd 安装流程读取的一次性链接交接文件。
	// 留空表示只写服务日志，便于非 systemd 环境直接运行。
	SetupURLFile   string
	TrustedProxies string
	DaeBinary      string
	DaeConfigPath  string
	BackupDir      string
	ServiceName    string
	Systemctl      string
	Journalctl     string
	// ServiceBackend 选择用 systemd 还是 procd 管理 dae。
	// 留空即 auto：存在 /sbin/procd 就用 procd，否则 systemd。
	ServiceBackend host.Backend
	DatabasePath   string
	SchedulePath   string
	// InstallStatePath 记录面板装了哪个 dae 版本，并存放回滚用的上一版二进制。
	InstallStatePath string
	// GitHubTokenPath 保存管理员在设置页填写的 GitHub API Token。
	// Token 本身不进入数据库、配置历史或 API 响应。
	GitHubTokenPath string
	SessionTTL      time.Duration
	SecureCookie    bool
	// EnableDaeInstall 打开通过面板安装与切换 dae 版本的能力。
	// 默认开启，发行单元同时开放默认二进制与 systemd 单元目录；不需要版本管理的
	// 部署仍可显式关闭开关并收紧 ReadWritePaths。
	EnableDaeInstall bool
	// GeoStatePath 记录面板上次把 geo 数据更新到了哪一版。
	GeoStatePath string
	// GeoSchedulePath 持久化 geo 数据自动更新的设置与上次执行时间。
	GeoSchedulePath string
	// GeoSourcesPath 保存管理员添加的自定义 geo 数据直链。
	GeoSourcesPath string
	// PanelBackupPath 存放自升级时被替换掉的上一版面板二进制。
	PanelBackupPath string
	// EnableSelfUpdate 是尚未在界面保存偏好时采用的初始值。
	// 默认开启；管理员可在设置页随时关闭，选择会持久化在面板数据目录。
	EnableSelfUpdate bool
	// DisableUpdateCheck 关闭面板自身的新版本检查。
	//
	// 检查默认开启：它只读取本仓库 releases/latest 的 tag，结果长时间缓存。
	// 但"完全不开安装与 geo 功能的部署从不外联"曾是可以宣称的性质，
	// 这个开关把决定权留给在意这件事的人。
	DisableUpdateCheck bool
	// EnableGeoUpdate 控制 Geo 数据管理是否启用（真实开关，非兼容字段）。
	// 关闭时 dependencies.Geo 为 nil，geo 相关接口一律返回 503
	// geo_update_disabled。OpenWrt 上由 UCI 的 enable_geo_update 透传
	// （见 openwrt/kdae-panel/files/kdae-panel.init）；下载仍受公网
	// HTTPS、体积与 SHA-256 三重约束。
	EnableGeoUpdate bool
}

func DefaultConfig() Config {
	return Config{
		ListenAddress:  "0.0.0.0:2023",
		Version:        "dev",
		TrustedProxies: "127.0.0.0/8,::1/128",
		DaeBinary:      "dae",
		DaeConfigPath:  "/etc/dae/config.dae",
		BackupDir:      "/var/lib/kdae-panel/backups",
		ServiceName:    "dae",
		Systemctl:      "systemctl",
		Journalctl:     "journalctl",
		ServiceBackend: host.BackendAuto,
		DatabasePath:   "/var/lib/kdae-panel/panel.db",
		SchedulePath:   "/var/lib/kdae-panel/schedule.json",
		// 上一版二进制也放在这个前缀下，因此目录必须在 ReadWritePaths 内。
		InstallStatePath: "/var/lib/kdae-panel/dae-install.json",
		GitHubTokenPath:  "/var/lib/kdae-panel/github-token",
		GeoStatePath:     "/var/lib/kdae-panel/geo-update.json",
		GeoSchedulePath:  "/var/lib/kdae-panel/geo-schedule.json",
		GeoSourcesPath:   "/var/lib/kdae-panel/geo-sources.json",
		PanelBackupPath:  "/var/lib/kdae-panel/kdae-panel.previous",
		SessionTTL:       12 * time.Hour,
		EnableDaeInstall: true,
		EnableGeoUpdate:  true,
		EnableSelfUpdate: true,
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	if c.ListenAddress == "" {
		c.ListenAddress = defaults.ListenAddress
	}
	if c.Version == "" {
		c.Version = defaults.Version
	}
	if c.TrustedProxies == "" {
		c.TrustedProxies = defaults.TrustedProxies
	}
	if c.DaeBinary == "" {
		c.DaeBinary = defaults.DaeBinary
	}
	if c.DaeConfigPath == "" {
		c.DaeConfigPath = defaults.DaeConfigPath
	}
	if c.BackupDir == "" {
		c.BackupDir = defaults.BackupDir
	}
	if c.ServiceName == "" {
		c.ServiceName = defaults.ServiceName
	}
	if c.Systemctl == "" {
		c.Systemctl = defaults.Systemctl
	}
	if c.Journalctl == "" {
		c.Journalctl = defaults.Journalctl
	}
	if c.ServiceBackend == "" {
		c.ServiceBackend = defaults.ServiceBackend
	}
	if c.DatabasePath == "" {
		c.DatabasePath = defaults.DatabasePath
	}
	if c.SchedulePath == "" {
		c.SchedulePath = defaults.SchedulePath
	}
	if c.GeoStatePath == "" {
		c.GeoStatePath = defaults.GeoStatePath
	}
	if c.GeoSchedulePath == "" {
		c.GeoSchedulePath = defaults.GeoSchedulePath
	}
	if c.GeoSourcesPath == "" {
		c.GeoSourcesPath = defaults.GeoSourcesPath
	}
	if c.PanelBackupPath == "" {
		c.PanelBackupPath = defaults.PanelBackupPath
	}
	if c.InstallStatePath == "" {
		c.InstallStatePath = defaults.InstallStatePath
	}
	if c.GitHubTokenPath == "" {
		c.GitHubTokenPath = defaults.GitHubTokenPath
	}
	if c.SessionTTL <= 0 {
		c.SessionTTL = defaults.SessionTTL
	}
	return c
}
