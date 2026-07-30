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
	SessionTTL       time.Duration
	SecureCookie     bool
	// EnableDaeInstall 打开通过面板安装与切换 dae 版本的能力。
	// 默认开启，发行单元同时开放默认二进制与 systemd 单元目录；不需要版本管理的
	// 部署仍可显式关闭开关并收紧 ReadWritePaths。
	EnableDaeInstall bool
	// GeoStatePath 记录面板上次把 geo 数据更新到了哪一版。
	GeoStatePath string
	// GeoSchedulePath 持久化 geo 数据自动更新的设置与上次执行时间。
	GeoSchedulePath string
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
	// EnableGeoUpdate 打开一键更新 geo 数据的能力，与 EnableDaeInstall 相互独立。
	//
	// 分成两个开关是有意为之：更新 geo 只写 dae 的数据目录（通常就是已经可写的
	// 配置目录），既不碰可执行文件也不碰 systemd 单元，不具备"面板缺陷升级为
	// 任意代码执行"的性质。把它并进 EnableDaeInstall，等于逼着只想刷新 geo 的
	// 人把 dae 二进制目录也交出去——那反而更不安全。
	//
	// 仍然默认关闭：它给部署新增了一条常态化的"联网取字节→以 root 写系统目录"
	// 路径，而这条路径在默认部署里本来并不存在。
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
		GeoStatePath:     "/var/lib/kdae-panel/geo-update.json",
		GeoSchedulePath:  "/var/lib/kdae-panel/geo-schedule.json",
		PanelBackupPath:  "/var/lib/kdae-panel/kdae-panel.previous",
		SessionTTL:       12 * time.Hour,
		EnableDaeInstall: true,
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
	if c.PanelBackupPath == "" {
		c.PanelBackupPath = defaults.PanelBackupPath
	}
	if c.InstallStatePath == "" {
		c.InstallStatePath = defaults.InstallStatePath
	}
	if c.SessionTTL <= 0 {
		c.SessionTTL = defaults.SessionTTL
	}
	return c
}
