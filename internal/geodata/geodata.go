// Package geodata 维护 dae 的 geo 数据文件（geoip.dat / geosite.dat）。
//
// 它和 dae 版本管理是两件事，刻意分开：更新 geo 只写一个数据目录，既不碰
// 可执行文件也不碰 systemd 单元，因此不具备"面板缺陷升级为任意代码执行"的
// 性质，不该逼着用户为了刷新 geo 而放宽二进制目录的写权限。
//
// dae 运行时只需 reload，不必 restart：reload 会重建控制平面并重新编译路由规则，
// 从而重读 geo 文件；dae 未运行时则只需落盘，下次启动会直接读取。这与替换二进制
// 必须 restart（要重挂 eBPF）不同。
package geodata

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// LocationAssetEnv 是 dae 用来指定 geo 数据目录的环境变量。
// 它的优先级高于所有默认搜索路径，不读它就无法确定 dae 究竟从哪里读。
const LocationAssetEnv = "DAE_LOCATION_ASSET"

const geoMode = 0o644

// Fetcher 是 geo 数据上游的消费者侧接口。
type Fetcher interface {
	Sources() []upstream.GeoSourceInfo
	Latest(ctx context.Context, source upstream.GeoSource) (upstream.GeoRelease, error)
	Fetch(ctx context.Context, release upstream.GeoRelease) (upstream.GeoData, error)
}

// SourceEditor 是 Fetcher 可选实现的自定义来源维护能力。
type SourceEditor interface {
	CustomSources() []upstream.CustomGeoSource
	CreateCustomSource(source upstream.CustomGeoSource) (upstream.CustomGeoSource, error)
	UpdateCustomSource(id string, source upstream.CustomGeoSource) (upstream.CustomGeoSource, error)
	DeleteCustomSource(id string) error
}

// ServiceController 读取 dae 服务状态，用于发现 DAE_LOCATION_ASSET。
type ServiceController interface {
	Status(ctx context.Context) (host.Status, error)
}

// Reloader 让 dae 重新读取 geo 数据。显式 PID 用于运行中且能取得服务状态的
// 后端（systemd、procd 皆可）；无参数形式仅在服务状态未知时使用。
type Reloader interface {
	Reload(ctx context.Context) error
	ReloadPID(ctx context.Context, pid int) error
}

type ServiceState string

const (
	ServiceStateActive   ServiceState = "active"
	ServiceStateInactive ServiceState = "inactive"
	ServiceStateUnknown  ServiceState = "unknown"
)

// File 是一个 geo 数据文件的现状。
type File struct {
	Name string `json:"name"`
	// Path 是 dae 实际会读到的那一份的完整路径；不存在时为空。
	Path    string     `json:"path,omitempty"`
	Present bool       `json:"present"`
	Size    int64      `json:"size,omitempty"`
	ModTime *time.Time `json:"modTime,omitempty"`
	// Shadowed 列出被 Path 遮蔽掉的同名文件。dae 只读优先级最高的那一份，
	// 其余的既占磁盘又容易让人以为"更新了却没生效"。
	Shadowed []string `json:"shadowed,omitempty"`
	// TargetPath 是下一次更新这个文件时的落盘位置。两个 Geo 文件可能由 dae
	// 从不同目录读取，因此不能再用一个公共目录代替。
	TargetPath string `json:"targetPath"`
}

type ResidualKind string

const (
	ResidualTemporary ResidualKind = "temporary"
	ResidualRollback  ResidualKind = "rollback"
)

// Residual 是异常退出后遗留的 Geo 事务文件。
type Residual struct {
	Path       string       `json:"path"`
	Kind       ResidualKind `json:"kind"`
	Size       int64        `json:"size"`
	ModTime    time.Time    `json:"modTime"`
	TargetPath string       `json:"targetPath,omitempty"`
	// Restorable 表示正式文件缺失，回滚点是可直接恢复的旧数据。
	Restorable bool `json:"restorable"`
	// Deletable 表示它不承载唯一旧数据，可在用户确认后安全清理。
	Deletable bool `json:"deletable"`
}

// Status 是 geo 数据的现状与可更新性。
type Status struct {
	// Sources 是可选的数据来源。两个来源的规则集不是同一套，必须由用户显式选择。
	Sources []upstream.GeoSourceInfo `json:"sources"`
	// DefaultSource 是界面该预选的来源：用过就沿用上次那个，否则用内置默认。
	DefaultSource upstream.GeoSource `json:"defaultSource"`
	// TargetDir 为兼容旧客户端保留；两个文件同目录时返回该目录，分目录时为空。
	TargetDir string `json:"targetDir"`
	// SearchPath 是 dae 查找 geo 的完整顺序，便于用户理解为什么写这里。
	SearchPath []string   `json:"searchPath"`
	Files      []File     `json:"files"`
	Residuals  []Residual `json:"residuals,omitempty"`
	// Updatable 为假表示还不能更新，Problem 说明原因。
	Updatable bool   `json:"updatable"`
	Problem   string `json:"problem,omitempty"`
	// Managed 记录面板上次更新到哪一版。
	Managed  *State   `json:"managed,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	// ServiceState 决定更新后是立即 reload，还是等 dae 下次启动时读取。
	ServiceState ServiceState `json:"serviceState"`
}

// State 记录面板上次把 geo 更新到了哪一版。
type State struct {
	Source upstream.GeoSource `json:"source"`
	// Repositories 如实记录当时的信任根，来源改名或换仓库时旧记录仍可读。
	Repositories []string  `json:"repositories,omitempty"`
	Tag          string    `json:"tag"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Manager struct {
	configPath string
	statePath  string
	fetcher    Fetcher
	service    ServiceController
	reloader   Reloader
	logger     *slog.Logger
	// backend 只影响"目录不可写"该怎么建议用户去修。两套 init 系统下这件事
	// 的成因完全不同，给错方向的指引比不给更浪费用户时间。
	backend host.Backend
}

// ResidualManager 是 GeoService 可选的异常事务恢复能力。
type ResidualManager interface {
	CleanupResiduals(ctx context.Context) (Status, error)
	RestoreResidual(ctx context.Context, path string) (Status, error)
}

type Options struct {
	// ConfigPath 是 dae 的入口配置，其所在目录是 geo 搜索顺序里优先级最高的默认位置。
	ConfigPath string
	StatePath  string
	Fetcher    Fetcher
	Service    ServiceController
	Reloader   Reloader
	Logger     *slog.Logger
	// ServiceBackend 留空按 host.Backend.Resolve 的规则自动探测。
	ServiceBackend host.Backend
}

func New(options Options) (*Manager, error) {
	if options.ConfigPath == "" {
		return nil, errors.New("dae 配置路径不能为空")
	}
	if options.StatePath == "" {
		return nil, errors.New("geo 状态文件路径不能为空")
	}
	if options.Fetcher == nil {
		return nil, errors.New("geo 数据取回器不能为空")
	}
	if options.Reloader == nil {
		return nil, errors.New("dae 重载器不能为空")
	}
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	backend, err := options.ServiceBackend.Resolve()
	if err != nil {
		return nil, err
	}
	return &Manager{
		configPath: options.ConfigPath,
		statePath:  options.StatePath,
		fetcher:    options.Fetcher,
		service:    options.Service,
		reloader:   options.Reloader,
		logger:     logger,
		backend:    backend,
	}, nil
}

func (m *Manager) sourceEditor() (SourceEditor, error) {
	editor, ok := m.fetcher.(SourceEditor)
	if !ok {
		return nil, errors.New("当前 geo 取回器不支持自定义来源")
	}
	return editor, nil
}

func (m *Manager) CustomSources() []upstream.CustomGeoSource {
	editor, err := m.sourceEditor()
	if err != nil {
		return nil
	}
	return editor.CustomSources()
}

func (m *Manager) CreateCustomSource(source upstream.CustomGeoSource) (upstream.CustomGeoSource, error) {
	editor, err := m.sourceEditor()
	if err != nil {
		return upstream.CustomGeoSource{}, err
	}
	return editor.CreateCustomSource(source)
}

func (m *Manager) UpdateCustomSource(id string, source upstream.CustomGeoSource) (upstream.CustomGeoSource, error) {
	editor, err := m.sourceEditor()
	if err != nil {
		return upstream.CustomGeoSource{}, err
	}
	return editor.UpdateCustomSource(id, source)
}

func (m *Manager) DeleteCustomSource(id string) error {
	editor, err := m.sourceEditor()
	if err != nil {
		return err
	}
	return editor.DeleteCustomSource(id)
}
