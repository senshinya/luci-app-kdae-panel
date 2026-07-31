// Package daeinstall 把选定版本的 dae 可执行文件安装到位。
//
// 安装是一个带回滚的事务，顺序刻意如此：
//
//	下载并校验 sha256 → 写入同目录暂存文件 → 用暂存的新二进制跑 --version
//	→ 用它 validate 现有配置 → 备份当前二进制 → 原子替换 → 重启服务
//	→ 确认服务已起来；任一步失败都恢复备份并把服务重启回去。
//
// 先验证再替换，是为了让"新版本根本跑不起来"或"新版本不认现有配置"这两类
// 问题在磁盘被改动之前就暴露出来。
package daeinstall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/tuoro/kdae-panel/internal/atomicfile"
	"github.com/tuoro/kdae-panel/internal/dae"
	"github.com/tuoro/kdae-panel/internal/geodata"
	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

const (
	binaryMode     = 0o755
	probeTimeout   = 60 * time.Second
	restartTimeout = 150 * time.Second
	// 重启后需要连续观察一段时间:dae validate 不加载 eBPF,能通过自检的版本
	// 仍可能在真正挂载 eBPF 时崩掉,只查一次状态会把这种失败当成成功。
	healthWindow   = 10 * time.Second
	healthInterval = time.Second
	// settleWindow 是"重启已经发起，但服务还没落到终态"这段过程的上限。
	//
	// systemd 的 restart 是同步的，返回时新进程已经起来；procd 不是：
	// rc.common 对 USE_PROCD=1 把 restart 展开成 stop + start，stop 只是一次
	// ubus service delete，procd 发完 SIGTERM 就立刻返回，最多等 term_timeout
	// （默认 5 秒）才补 SIGKILL，而新实例要等旧进程真正退干净才会被拉起。
	// dae 退出时正好要卸载 eBPF/TC 挂载点，这一两秒实打实存在。
	// 于是重启后的头几秒里，"查到的还是正在退出的旧 pid"和"两个实例都不在"
	// 都是正常过程而非故障，按第一次采样定生死等于随机回滚正常版本。
	settleWindow = 15 * time.Second
)

// Probe 用某个具体路径上的 dae 可执行文件做验证。
type Probe interface {
	Inspect(ctx context.Context) dae.Report
	Validate(ctx context.Context, configPath string) error
}

// ProbeFactory 为给定路径构造探针，便于测试注入。
type ProbeFactory func(binaryPath string) Probe

// ServiceController 控制 dae 的 systemd 单元。
type ServiceController interface {
	Action(ctx context.Context, action host.Action) error
	Status(ctx context.Context) (host.Status, error)
}

// Fetcher 取回并校验指定资产，返回发布包内的全部可用物料。
type Fetcher interface {
	List(ctx context.Context, source upstream.Source, limit int) ([]upstream.Version, error)
	Resolve(ctx context.Context, source upstream.Source, ref string, platform upstream.Platform) (upstream.Asset, error)
	FetchBundle(ctx context.Context, asset upstream.Asset) (upstream.Bundle, error)
}

// State 记录当前装的是哪个版本。
// dae --version 对 CI 构建往往无法区分 commit，因此以面板自己的记录为准，
// 同时保存二进制摘要，用于发现外部手动替换。
type State struct {
	Source      upstream.Source `json:"source,omitempty"`
	Ref         string          `json:"ref,omitempty"`
	Label       string          `json:"label,omitempty"`
	Version     string          `json:"version,omitempty"`
	InstalledAt time.Time       `json:"installedAt,omitempty"`
	SHA256      string          `json:"sha256,omitempty"`
}

// Status 是对外暴露的安装状态。
// 磁盘上是什么、面板以为装了什么、服务是否跑得起来，是三件独立的事，
// 出问题时必须分别可见，不能糅成一个"版本"字段。
type Status struct {
	// BinaryPath 是 systemd 单元实际启动的可执行文件，也是替换的目标。
	BinaryPath string `json:"binaryPath,omitempty"`
	Platform   string `json:"platform"`
	// Ready 为假表示还不具备安装条件，Problem 说明原因。
	Ready bool `json:"ready"`
	// Present 为假表示目标路径上没有可执行文件。
	Present bool `json:"present"`
	// Version 是对磁盘上那个二进制实际探测到的版本。
	Version string `json:"version,omitempty"`
	Managed *State `json:"managed,omitempty"`
	// Drifted 为真表示磁盘上的二进制不是面板装的那个（被外部替换过）。
	Drifted bool `json:"drifted,omitempty"`
	// RollbackAvailable 为真表示存在可回滚的上一版本。
	RollbackAvailable bool `json:"rollbackAvailable"`
	// ServiceActive 反映替换后服务是否真的在跑。
	ServiceActive bool `json:"serviceActive"`
	// Warnings 是不阻断安装但用户应当知道的问题，如缺少 geo 数据文件。
	Warnings []string `json:"warnings,omitempty"`
	Problem  string   `json:"problem,omitempty"`
}

type Installer struct {
	binaryPath  string
	configPath  string
	statePath   string
	backupPath  string
	serviceName string
	cache       *versionCache
	// unitDir 是 systemd 单元的落地目录，留空即用系统默认，测试会覆盖它。
	unitDir string
	// units 抽掉不同 init 系统在"服务定义"上的差异。
	units unitProvisioner
	// geoSearchDirs 只供卸载测试把搜索范围收进临时目录；生产环境留空，
	// 始终使用 dae 的真实 geo 搜索顺序。
	geoSearchDirs []string
	fetcher       Fetcher
	newProbe      ProbeFactory
	service       ServiceController
	logger        *slog.Logger
	// health/interval 是重启后的健康观察窗口与采样间隔，测试会调短它们。
	health   time.Duration
	interval time.Duration
	// settle 是观察窗口之前那段"等服务落到终态"的上限，与 health 分开计时。
	settle time.Duration
}

type Options struct {
	BinaryPath  string
	ConfigPath  string
	StatePath   string
	ServiceName string
	Fetcher     Fetcher
	NewProbe    ProbeFactory
	Service     ServiceController
	Logger      *slog.Logger
	// ServiceBackend 决定服务定义由哪套 init 系统承载，留空按
	// host.Backend.Resolve 的规则自动探测（有 /sbin/procd 则判为 procd，
	// 否则 systemd），并非无条件落到 systemd。
	ServiceBackend host.Backend
}

func New(options Options) (*Installer, error) {
	if options.BinaryPath == "" {
		return nil, errors.New("dae 可执行文件路径不能为空")
	}
	if options.StatePath == "" {
		return nil, errors.New("安装状态文件路径不能为空")
	}
	if options.Fetcher == nil {
		return nil, errors.New("上游取回器不能为空")
	}
	if options.Service == nil {
		return nil, errors.New("服务控制器不能为空")
	}
	binaryPath, err := resolveBinaryPath(options.BinaryPath)
	if err != nil {
		return nil, fmt.Errorf("解析 dae 路径: %w", err)
	}
	newProbe := options.NewProbe
	if newProbe == nil {
		newProbe = func(path string) Probe { return dae.NewClient(path) }
	}
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	installer := &Installer{
		binaryPath:  binaryPath,
		configPath:  options.ConfigPath,
		statePath:   options.StatePath,
		backupPath:  options.StatePath + ".previous-dae",
		serviceName: options.ServiceName,
		cache:       newVersionCache(options.StatePath),
		fetcher:     options.Fetcher,
		newProbe:    newProbe,
		service:     options.Service,
		logger:      logger,
		health:      healthWindow,
		interval:    healthInterval,
		settle:      settleWindow,
	}
	backend, err := options.ServiceBackend.Resolve()
	if err != nil {
		return nil, err
	}
	if backend == host.BackendProcd {
		installer.units = &procdUnits{installer: installer}
	} else {
		installer.units = &systemdUnits{installer: installer}
	}
	return installer, nil
}

// resolveBinaryPath 把配置里的 dae 路径解析成绝对路径。
//
// 默认值是裸名 "dae"，靠 PATH 查找即可满足探测。但首次安装要往这个路径写文件，
// 而 filepath.Abs 会把裸名解析成进程 cwd 下的位置——systemd 启动的面板 cwd 是
// 根目录，于是预检会一本正经地建议把 / 加进 ReadWritePaths，安装则把 dae 装到
// /dae。裸名必须先按 PATH 查，查不到再退回约定俗成的 /usr/bin。
func resolveBinaryPath(configured string) (string, error) {
	if configured != filepath.Base(configured) {
		return filepath.Abs(configured)
	}
	if found, err := exec.LookPath(configured); err == nil {
		return filepath.Abs(found)
	}
	return filepath.Join("/usr/bin", configured), nil
}

func (i *Installer) Versions(ctx context.Context, source upstream.Source, limit int) ([]Version, error) {
	platform, err := upstream.DetectPlatform()
	if err != nil {
		return nil, err
	}
	cached, cacheErr := i.cache.list(source, platform.Name)
	if cacheErr != nil {
		i.logger.Warn("读取 dae 本地版本列表时跳过了无效条目", "error", cacheErr)
	}
	remote, upstreamErr := i.fetcher.List(ctx, source, limit)
	if upstreamErr != nil && len(cached) == 0 {
		return nil, upstreamErr
	}
	if upstreamErr != nil {
		i.logger.Warn("上游版本列表不可用，仅返回本地版本", "source", source, "error", upstreamErr)
	}

	versions := make([]Version, 0, len(remote)+len(cached))
	byRef := make(map[string]int, len(remote))
	for _, item := range remote {
		byRef[item.Ref] = len(versions)
		versions = append(versions, Version{Version: item})
	}
	for _, item := range cached {
		cachedAt := item.CachedAt
		if index, exists := byRef[item.Ref]; exists {
			versions[index].Cached = true
			versions[index].CachedAt = &cachedAt
			versions[index].CachedBytes = item.Size
			// 上游产物即使已经过期，本地副本仍可安装。
			versions[index].Installable = true
			continue
		}
		description := "仅保留在本机"
		if upstreamErr != nil {
			description = "上游暂不可用，仅显示本机缓存"
		}
		versions = append(versions, Version{
			Version: upstream.Version{
				Source:      item.Source,
				Ref:         item.Ref,
				Label:       item.Label,
				Description: description,
				PublishedAt: item.CachedAt,
				Installable: true,
			},
			Cached:      true,
			CachedOnly:  true,
			CachedAt:    &cachedAt,
			CachedBytes: item.Size,
		})
	}
	return versions, nil
}

// target 以 systemd 单元实际启动的可执行文件为准。
//
// 这一点至关重要：配置项 KDAE_PANEL_DAE_BINARY 只用于探测，若它与单元的
// ExecStart 不是同一个文件，替换它不会影响真正运行的进程——事务会全绿，
// 而 dae 仍跑着旧二进制。宁可拒绝安装，也不做这种静默的假成功。
func (i *Installer) target(ctx context.Context) (string, bool, error) {
	status, err := i.service.Status(ctx)
	if err != nil {
		return "", false, fmt.Errorf("读取 dae 服务状态失败，无法确定要替换哪个文件：%w", err)
	}
	if status.ExecStartPath == "" {
		return "", false, errors.New("机器上找不到 dae 的服务定义，面板只能升级或切换已有的 dae")
	}
	path, err := filepath.Abs(status.ExecStartPath)
	if err != nil {
		return "", false, fmt.Errorf("解析服务启动路径 %q: %w", status.ExecStartPath, err)
	}
	return path, status.ActiveState == "active", nil
}

func (i *Installer) Status(ctx context.Context) Status {
	platform, platformErr := upstream.DetectPlatform()
	status := Status{Platform: platform.Name}
	if platformErr != nil {
		status.Problem = platformErr.Error()
		return status
	}
	if _, err := os.Stat(i.backupPath); err == nil {
		status.RollbackAvailable = true
	}

	target, active, err := i.target(ctx)
	if err != nil {
		status.Problem = err.Error()
		return status
	}
	status.BinaryPath = target
	status.ServiceActive = active

	digest, err := i.fileDigest(target)
	if err != nil {
		if os.IsNotExist(err) {
			status.Problem = fmt.Sprintf("服务指向的 %s 不存在", target)
		} else {
			status.Problem = err.Error()
		}
		return status
	}
	status.Present = true
	status.Ready = true
	// 备份与磁盘上完全相同时，回滚是个空操作，不该在界面上亮出按钮。
	if status.RollbackAvailable {
		if backup, err := i.fileDigest(i.backupPath); err == nil && backup == digest {
			status.RollbackAvailable = false
		}
	}

	if report := i.newProbe(target).Inspect(ctx); report.Available {
		status.Version = report.Version
	} else {
		status.Problem = report.Problem
	}
	if state, err := i.readState(); err == nil && state != nil {
		status.Managed = state
		status.Drifted = state.SHA256 != "" && state.SHA256 != digest
	}
	if i.binaryPath != target {
		status.Warnings = append(status.Warnings, fmt.Sprintf(
			"面板配置的 dae 路径是 %s，而服务实际启动的是 %s；安装会替换后者", i.binaryPath, target))
	}
	// 暂存备份只在事务进行中存在，结算时必然被提升或删除。它还留在磁盘上，
	// 说明上一次安装是被打断的（面板重启、进程被杀），当时磁盘处于哪一步无从
	// 得知，必须让用户自己核对版本，而不是假装什么都没发生。
	if _, err := os.Stat(i.pendingBackupPath()); err == nil {
		status.Warnings = append(status.Warnings,
			"发现上一次安装留下的暂存备份，说明它在中途被打断；请核对上面的运行版本是否符合预期")
	}
	status.Warnings = append(status.Warnings, i.geoWarnings(ctx)...)
	return status
}

// geoSearchPath 取回 dae 查找 geo 数据文件的顺序。
// 顺序本身由 geodata 包定义，这里只补上本机的环境变量——两处各写一份的话，
// 早晚会出现"安装认为已就位、更新却写去了别处"这种自相矛盾。
func (i *Installer) geoSearchPath(ctx context.Context) []string {
	var environment map[string]string
	if status, err := i.service.Status(ctx); err == nil {
		environment = status.Environment
	}
	return geodata.SearchPath(i.configPath, environment)
}

// geoWarnings 检查 dae 运行所需的 geo 数据文件。
func (i *Installer) geoWarnings(ctx context.Context) []string {
	if warning := geodata.MissingWarning(i.geoSearchPath(ctx)); warning != "" {
		return []string{warning}
	}
	return nil
}

// Acquire 优先读取并重新校验本地版本；requireBundle 为真时仍取完整发布包，
// 因为首次安装还需要服务单元、种子配置与 geo，而缓存只保留可执行文件。
func (i *Installer) Acquire(ctx context.Context, source upstream.Source, ref, label string,
	requireBundle bool) (upstream.Bundle, bool, error) {
	platform, err := upstream.DetectPlatform()
	if err != nil {
		return upstream.Bundle{}, false, err
	}
	if !requireBundle {
		content, metadata, err := i.cache.load(source, ref, platform.Name)
		switch {
		case err == nil:
			i.logger.Info("使用已校验的 dae 本地版本", "source", source, "ref", ref,
				"cached_at", metadata.CachedAt, "bytes", len(content))
			return upstream.Bundle{Binary: content}, true, nil
		case errors.Is(err, ErrCachedVersionNotFound):
		case errors.Is(err, errInvalidVersionCache):
			i.logger.Warn("dae 本地版本已损坏，将重新下载", "source", source, "ref", ref, "error", err)
			if removeErr := i.cache.discardInvalid(source, ref, platform.Name); removeErr != nil {
				i.logger.Warn("清理损坏的 dae 本地版本失败", "source", source, "ref", ref, "error", removeErr)
			}
		default:
			return upstream.Bundle{}, false, fmt.Errorf("读取 dae 本地版本: %w", err)
		}
	}
	asset, err := i.fetcher.Resolve(ctx, source, ref, platform)
	if err != nil {
		return upstream.Bundle{}, false, err
	}
	bundle, err := i.fetcher.FetchBundle(ctx, asset)
	if err != nil {
		return upstream.Bundle{}, false, err
	}
	if err := i.cache.store(source, ref, label, platform.Name, bundle.Binary); err != nil {
		return upstream.Bundle{}, false, fmt.Errorf("保存 dae 本地版本: %w", err)
	}
	i.logger.Info("已取得并校验 dae 发布包",
		"source", source, "ref", ref, "asset", asset.Filename, "bytes", len(bundle.Binary))
	return bundle, false, nil
}

// DeleteCached 删除指定版本的本地副本，不触碰当前运行文件与事务回滚点。
func (i *Installer) DeleteCached(source upstream.Source, ref string) error {
	platform, err := upstream.DetectPlatform()
	if err != nil {
		return err
	}
	return i.cache.delete(source, ref, platform.Name)
}

// Install 把已下载的内容装上去。调用方应在持有全局控制锁时调用它。
func (i *Installer) Install(ctx context.Context, binary []byte, source upstream.Source, ref, label string) (Status, error) {
	return i.applyBinary(ctx, binary, &State{
		Source:      source,
		Ref:         ref,
		Label:       label,
		InstalledAt: nowUTC(),
		SHA256:      digestBytes(binary),
	})
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

// Rollback 恢复上一次安装前备份的二进制。
func (i *Installer) Rollback(ctx context.Context) (Status, error) {
	binary, err := os.ReadFile(i.backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Status{}, errors.New("没有可回滚的上一版本")
		}
		return Status{}, fmt.Errorf("读取备份的 dae: %w", err)
	}
	previous, _ := i.readPreviousState()
	if previous == nil {
		previous = &State{}
	}
	previous.InstalledAt = time.Now().UTC()
	previous.SHA256 = digestBytes(binary)
	return i.applyBinary(ctx, binary, previous)
}

// applyBinary 是共用的替换事务：验证 → 备份 → 替换 → 重启 → 失败回滚。
func (i *Installer) applyBinary(ctx context.Context, binary []byte, state *State) (Status, error) {
	// 这些字节马上就要以 root 落到 ExecStart 指向的位置并被执行。
	// 首次安装会做这项检查，升级同样不能少：上游若改了打包方式，
	// 按条目名挑出来的可能根本不是可执行文件。
	if err := assertELF(binary); err != nil {
		return Status{}, err
	}
	target, _, err := i.target(ctx)
	if err != nil {
		return Status{}, err
	}
	// 只升级或切换已有安装：目标不存在时不去猜该往哪装，也不代为创建 dae 服务。
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return Status{}, fmt.Errorf("服务指向的 %s 不存在，请先用官方安装器完成 dae 的首次安装", target)
		}
		return Status{}, err
	}
	// ExecStart 来自系统上的服务单元。若它指向的不是 dae，覆盖它就是在破坏
	// 一个无关的程序，因此宁可拒绝也不猜。
	if !info.Mode().IsRegular() {
		return Status{}, fmt.Errorf("服务指向的 %s 不是普通文件，拒绝替换", target)
	}
	if filepath.Base(target) != upstream.BinaryName {
		return Status{}, fmt.Errorf("服务启动的是 %s，文件名不是 %s，为避免覆盖无关程序已拒绝替换",
			target, upstream.BinaryName)
	}
	// 文件名可以骗人：ExecStart 完全可能指向一个名叫 dae 的启动包装脚本。
	// 覆盖它会毁掉运维的包装，而 dae 仍带着原参数起来，事务全绿却已经出错。
	if err := assertExecutable(target); err != nil {
		return Status{}, err
	}

	staged, cleanup, err := i.stage(binary, target, info.Mode().Perm())
	if err != nil {
		return Status{}, err
	}
	// 暂存文件在成功改名后已经不存在，用标志位区分，避免 defer 捕获到旧闭包。
	committed := false
	defer func() {
		if !committed {
			cleanup()
		}
	}()

	// 先用暂存的新二进制自证：能报版本，且认得现有配置。
	probeCtx, cancelProbe := context.WithTimeout(ctx, probeTimeout)
	report := i.newProbe(staged).Inspect(probeCtx)
	cancelProbe()
	if !report.Available {
		return Status{}, fmt.Errorf("新版本无法运行：%s", report.Problem)
	}
	state.Version = report.Version
	if i.configPath != "" {
		if _, err := os.Stat(i.configPath); err == nil {
			validateCtx, cancelValidate := context.WithTimeout(ctx, probeTimeout)
			err := i.newProbe(staged).Validate(validateCtx, i.configPath)
			cancelValidate()
			if err != nil {
				return Status{}, fmt.Errorf("新版本拒绝当前配置，已中止安装：%w", err)
			}
		}
	}

	pending, err := i.backupCurrent(target)
	if err != nil {
		return Status{}, err
	}
	// 暂存的回滚点必须恰好结算一次：磁盘留下新版本就提升它，
	// 退回旧版本就丢弃它。早退路径（如替换失败）磁盘没变，回滚点原样保留。
	backupSettled := false
	defer func() {
		if !backupSettled {
			i.discardBackup()
		}
	}()

	if err := atomicfile.Replace(staged, target); err != nil {
		return Status{}, fmt.Errorf("替换 dae 可执行文件: %w", err)
	}
	committed = true

	if err := i.restart(ctx); err != nil {
		// 换上去起不来，必须退回原样。回滚用独立的 context：安装用的那个
		// 预算已被下载和这次失败的重启耗掉大半，甚至可能已经取消——
		// 而回滚恰恰是最不能因为超时或取消而半途而废的一步。
		restoreCtx, cancelRestore := context.WithTimeout(
			context.WithoutCancel(ctx), restartTimeout+i.health)
		outcome := i.restorePrevious(restoreCtx, target)
		cancelRestore()

		backupSettled = true
		if outcome.RestoreErr == nil {
			i.discardBackup() // 磁盘已退回旧版本，原有回滚点仍然有效
		} else {
			i.commitBackup(pending) // 磁盘上留着新版本，暂存的旧版本正是它的上一版
		}
		failure := outcome.RestoreErr
		if failure == nil {
			failure = outcome.RestartErr
		}
		return Status{}, &ApplyError{
			Cause:            err,
			RolledBack:       outcome.RestoreErr == nil,
			ServiceRecovered: outcome.RestoreErr == nil && outcome.RestartErr == nil,
			RestoreErr:       failure,
		}
	}
	i.commitBackup(pending)
	backupSettled = true

	// 账本要在读状态之前写，否则 Status 拿到的还是上一版的记录。
	stateErr := i.writeState(state)
	if stateErr != nil {
		i.logger.Warn("记录 dae 安装状态失败", "error", stateErr)
	}
	status := i.Status(ctx)
	if stateErr != nil {
		// 安装本身已经成功，不能因此报错；但账本没写上，后续状态查询会把这次
		// 安装误报成"在面板之外被替换过"，必须让用户当场看见。
		status.Warnings = append(status.Warnings, fmt.Sprintf(
			"新版本已装上并运行，但安装记录写入失败（%v）；面板会把它显示为在面板之外替换过", stateErr))
	}
	return status, nil
}

// ApplyError 表示二进制已替换但服务未能起来。
//
// RolledBack 只表示"磁盘上的二进制已还原"。旧版本能否重新跑起来是另一回事，
// 两者合成一个布尔值会让用户误以为一切已恢复原状，因此分开表达。
type ApplyError struct {
	Cause error
	// RolledBack 为真表示磁盘文件已还原为旧版本。
	RolledBack bool
	// ServiceRecovered 为真表示还原之后服务确实重新起来了。
	ServiceRecovered bool
	RestoreErr       error
}

func (e *ApplyError) Error() string {
	switch {
	case !e.RolledBack:
		return fmt.Sprintf("%v；磁盘文件未能还原：%v", e.Cause, e.RestoreErr)
	case !e.ServiceRecovered:
		return fmt.Sprintf("%v；已还原为原版本，但服务仍未恢复：%v", e.Cause, e.RestoreErr)
	default:
		return fmt.Sprintf("%v；已回滚到原版本且服务已恢复", e.Cause)
	}
}

func (e *ApplyError) Unwrap() error {
	return e.Cause
}

// stage 把新二进制写到目标同目录。
//
// 必须同目录，不能用 /tmp：面板单元开了 PrivateTmp，跨文件系统 rename 会 EXDEV；
// 而且 /tmp 常以 noexec 挂载，放那里的新二进制连自检都跑不了。
func (i *Installer) stage(binary []byte, target string, mode os.FileMode) (string, func(), error) {
	if len(binary) == 0 {
		return "", func() {}, errors.New("新版本内容为空")
	}
	// 沿用被替换文件原有的权限位；取不到时退回保守的 0755。
	if mode.Perm() == 0 {
		mode = binaryMode
	}
	return atomicfile.Stage(filepath.Dir(target), binary, mode)
}

// backupCurrent 把当前版本写进暂存回滚位，而不是直接覆盖正式回滚点。
//
// 正式回滚点的含义始终是"磁盘上这一版的前一版"。替换可能失败，重启失败还要把
// 磁盘退回原样——那些情况下磁盘上仍是旧版本，此时若已经覆盖了正式回滚点，
// 用户真正想回去的那一版就被本次安装永久删掉了。因此先写暂存位，
// 等结算时才知道该提升它还是丢弃它。
// pendingBackup 是一次事务里暂存的回滚材料。
// 账本只在内存里留到结算，没必要为它也在磁盘上开一个暂存文件。
type pendingBackup struct {
	// state 是被顶掉那一版的账本；为空表示它不是面板装的，没有记录可继承。
	state []byte
}

func (i *Installer) backupCurrent(target string) (*pendingBackup, error) {
	current, err := os.ReadFile(target)
	if err != nil {
		return nil, fmt.Errorf("读取当前 dae: %w", err)
	}
	if err := writeFileSynced(i.pendingBackupPath(), current, binaryMode); err != nil {
		return nil, fmt.Errorf("备份当前 dae: %w", err)
	}
	pending := &pendingBackup{}
	// 一并记下旧版本的账本，回滚后才能如实显示回到了哪一版。
	if state, err := i.readState(); err == nil && state != nil {
		pending.state, _ = json.Marshal(state)
		// 只有账本摘要与当前文件一致时才能用该账本给缓存命名；否则它可能是
		// 面板外替换的未知二进制，绝不能冒充已知版本。
		if state.Source != "" && state.Ref != "" && state.SHA256 != "" && state.SHA256 == digestBytes(current) {
			if platform, err := upstream.DetectPlatform(); err == nil {
				if err := i.cache.store(state.Source, state.Ref, state.Label, platform.Name, current); err != nil {
					i.logger.Warn("保留当前 dae 本地版本失败", "source", state.Source, "ref", state.Ref, "error", err)
				}
			}
		}
	}
	return pending, nil
}

// commitBackup 在磁盘上确实留下新版本后，把暂存位提升为正式回滚点。
func (i *Installer) commitBackup(pending *pendingBackup) {
	if err := os.Rename(i.pendingBackupPath(), i.backupPath); err != nil {
		i.logger.Warn("提交 dae 回滚点失败", "error", err)
		return
	}
	// 上一版可能是面板之外装的，没有账本；此时必须清掉更旧的那份，
	// 否则回滚后会显示一个根本不对应的版本。
	if len(pending.state) == 0 {
		_ = os.Remove(i.previousStatePath())
		return
	}
	if err := writeFileSynced(i.previousStatePath(), pending.state, 0o600); err != nil {
		i.logger.Warn("记录上一版本的安装信息失败", "error", err)
	}
}

// discardBackup 在磁盘已退回原样时丢弃暂存位，原封不动保住既有回滚点。
func (i *Installer) discardBackup() {
	_ = os.Remove(i.pendingBackupPath())
}

// restoreOutcome 区分"磁盘已还原"与"旧版本重新跑起来了"两件事。
// 把两者合成一个 error，调用方就无法分辨，只能对着成功的回滚报告失败。
type restoreOutcome struct {
	// RestoreErr 非空表示磁盘上的二进制没能还原。
	RestoreErr error
	// RestartErr 非空表示文件已还原，但旧版本没能重新起来。
	RestartErr error
}

func (i *Installer) restorePrevious(ctx context.Context, target string) restoreOutcome {
	// 要还原的是本次事务开始前的那一份（暂存位），不是历史回滚点。
	binary, err := os.ReadFile(i.pendingBackupPath())
	if err != nil {
		return restoreOutcome{RestoreErr: err}
	}
	mode := os.FileMode(binaryMode)
	if info, err := os.Stat(target); err == nil {
		mode = info.Mode().Perm()
	}
	staged, cleanup, err := i.stage(binary, target, mode)
	if err != nil {
		return restoreOutcome{RestoreErr: err}
	}
	committed := false
	defer func() {
		if !committed {
			cleanup()
		}
	}()
	if err := atomicfile.Replace(staged, target); err != nil {
		return restoreOutcome{RestoreErr: err}
	}
	committed = true
	return restoreOutcome{RestartErr: i.restart(ctx)}
}

// restart 重启服务，然后在一个观察窗口内反复确认它稳住了。
// 替换二进制后必须整体重启：dae 的 eBPF 程序要重新挂载，reload 不足以生效。
func (i *Installer) restart(ctx context.Context) error {
	restartCtx, cancel := context.WithTimeout(ctx, restartTimeout)
	defer cancel()
	// 先记下重启前的主进程号。procd 的 restart 立刻返回，之后一两秒里查到的
	// 可能仍是正在退出的旧实例；不记下这个号，就会把旧 pid 当成新实例的基线，
	// 等真正的新 pid 出现时反过来判成"服务在观察窗口内挂过"——回滚一个
	// 完全正常的版本。读不到就当没有，判据会自动退回到"连续两次相同"。
	previousPID := 0
	if status, err := i.service.Status(restartCtx); err == nil {
		previousPID = status.MainPID
	}
	if err := i.service.Action(restartCtx, host.ActionRestart); err != nil {
		return err
	}
	return i.waitHealthy(restartCtx, previousPID)
}

// waitHealthy 是重启后的观察循环，与真正发起重启的那一步分开，
// 便于测试直接驱动观察窗口而不依赖 ServiceController.Action 的行为。
//
// 分成稳定期和观察期两段，各自独立计时。稳定期负责等服务落到终态并取得基线，
// 观察期从基线成立那一刻才开始算——合在一起计时的话，重启越慢真正用于观察的
// 时间越短，而重启慢恰恰是最该多看两眼的情形。
func (i *Installer) waitHealthy(ctx context.Context, previousPID int) error {
	baseline, err := i.settleAfterRestart(ctx, previousPID)
	if err != nil {
		return err
	}
	return i.observeAfterRestart(ctx, baseline)
}

// settleAfterRestart 轮询到服务达到终态为止，返回作为观察基线的那次采样。
//
// 终态的判据有两条，都要满足：ActiveState 为 active 且 MainPID 连续两次相同，
// 以及这个 MainPID 不再是重启前的那一个。原先的实现只看第一次采样，
// 那建立在 systemctl restart 的同步语义上，procd 不满足（见 settleWindow），
// 结果是每次版本切换都可能回滚一个完全正常的版本，理由还是编造的。
//
// 第二条不能省。procd 的重启中间态有两种，只挡住其一等于没挡：新实例尚未拉起时
// 查到的是 inactive，"连续两次相同"能挡住；而旧实例还在退出时查到的是
// active + 旧 pid，dae 卸载 eBPF 挂载点要一两秒，一秒一采就足以连续两次
// 采到同一个旧 pid——把它当基线，等新实例真的起来就会被判成"观察窗口内挂过"。
//
// 稳定期不会放过真正的崩溃循环，两种情形都有出口：
//   - 崩溃周期长于采样间隔：能凑出连续两次相同的 pid，基线成立，随后 pid 变化
//     会在观察期里被抓到；
//   - 崩溃周期短于采样间隔：每次采样都是新 pid，"连续两次相同"永远不成立，
//     稳定期到点即失败，不会被误判成起来了。
//
// 对 systemd 无害：它返回时新进程已经起稳且 pid 必然与重启前不同，
// 第二次采样必然与第一次相同，代价只是多花一个采样间隔。
func (i *Installer) settleAfterRestart(ctx context.Context, previousPID int) (host.Status, error) {
	deadline := time.Now().Add(i.settle)
	var previous host.Status
	first := true
	for {
		select {
		case <-time.After(i.interval):
		case <-ctx.Done():
			return host.Status{}, ctx.Err()
		}
		status, err := i.service.Status(ctx)
		if err != nil {
			return host.Status{}, fmt.Errorf("重启后无法读取服务状态: %w", err)
		}
		if first {
			// 第一次采样没有可比较的对象，只能记下来。稳定期到点的判定也一并
			// 推迟到下一轮，否则 settle 被调到 0 的测试连比较都做不了一次。
			previous, first = status, false
			continue
		}
		stale := previousPID != 0 && status.MainPID == previousPID
		if status.ActiveState == "active" && previous.ActiveState == "active" &&
			status.MainPID == previous.MainPID && !stale {
			return status, nil
		}
		if !time.Now().Before(deadline) {
			return host.Status{}, settleTimeout(i.settle, previous, status, stale)
		}
		previous = status
	}
}

// settleTimeout 说明服务到底卡在哪一步。三种停不下来的原因对用户是三件事：
// 根本没起来、旧实例赖着不退、起来就崩——含糊成一句"没稳定"等于让人无从下手。
func settleTimeout(window time.Duration, previous, last host.Status, stale bool) error {
	if last.ActiveState != "active" {
		return fmt.Errorf("重启后 %s 内服务没有起来，最后一次查到的状态是 %s/%s",
			window, last.ActiveState, last.SubState)
	}
	if stale {
		return fmt.Errorf("重启后 %s 内主进程号仍是重启前的 %d，服务没有换成新实例",
			window, last.MainPID)
	}
	return fmt.Errorf(
		"重启后 %s 内服务的主进程号一直在变（先是 %d，接着是 %d），说明它起来就崩、反复被拉起，新版本很可能起不稳",
		window, previous.MainPID, last.MainPID)
}

// observeAfterRestart 在观察窗口里盯住已经稳定下来的服务。
// dae validate 不加载 eBPF，能通过自检的版本仍可能在真正挂载 eBPF 时崩掉，
// 因此稳定之后还要多看一段时间。
func (i *Installer) observeAfterRestart(ctx context.Context, baseline host.Status) error {
	deadline := time.Now().Add(i.health)
	for {
		select {
		case <-time.After(i.interval):
		case <-ctx.Done():
			return ctx.Err()
		}
		status, err := i.service.Status(ctx)
		if err != nil {
			return fmt.Errorf("重启后无法读取服务状态: %w", err)
		}
		// 服务已经稳定过一次，此后任何一次不是 active 都是真的掉了，
		// 不必等到窗口结束再看最后一眼。
		if status.ActiveState != "active" {
			return fmt.Errorf("重启后服务状态为 %s/%s", status.ActiveState, status.SubState)
		}
		// 只看 ActiveState 会漏掉采样间隔内跑完的崩溃-重启循环：
		// 两次采样都是 active，中间其实已经挂掉并被拉起来过。
		// systemd 的 NRestarts 单调递增，是最直接的证据。
		if status.Restarts > baseline.Restarts {
			return fmt.Errorf("重启后服务在观察窗口内又重启了 %d 次，新版本很可能起不稳",
				status.Restarts-baseline.Restarts)
		}
		// procd 不暴露重启计数器，NRestarts 恒为 0，上面那条永远不成立。
		// 但重新拉起必然换主进程号，pid 变了就等于中间挂过一次。
		if baseline.MainPID != 0 && status.MainPID != 0 && status.MainPID != baseline.MainPID {
			return fmt.Errorf("重启后服务的主进程号从 %d 变成 %d，说明它在观察窗口内挂掉并被重新拉起，新版本很可能起不稳",
				baseline.MainPID, status.MainPID)
		}
		if !time.Now().Before(deadline) {
			return nil
		}
	}
}

func (i *Installer) previousStatePath() string {
	return i.statePath + ".previous"
}

// pendingBackupPath 是回滚点二进制的暂存位，事务结算时才决定提升还是丢弃。
// 进程中途退出留下的暂存文件无害：下一次 backupCurrent 会直接覆盖它。
// 账本不落暂存文件，它在事务期间只存在于内存里的 pendingBackup.state。
func (i *Installer) pendingBackupPath() string {
	return i.backupPath + ".pending"
}

func (i *Installer) readState() (*State, error) {
	return readStateFile(i.statePath)
}

func (i *Installer) readPreviousState() (*State, error) {
	return readStateFile(i.previousStatePath())
}

func readStateFile(path string) (*State, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state State
	if err := json.Unmarshal(content, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (i *Installer) writeState(state *State) error {
	content, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return writeFileSynced(i.statePath, content, 0o600)
}

func writeFileSynced(path string, content []byte, mode os.FileMode) error {
	return atomicfile.Write(path, content, mode)
}

// elfMagic 是 ELF 文件的前四个字节。
const elfMagic = "\x7fELF"

// assertELF 确认内容是原生可执行文件。
func assertELF(content []byte) error {
	if len(content) == 0 {
		return errors.New("发布包内没有可执行文件")
	}
	if len(content) < len(elfMagic) || string(content[:len(elfMagic)]) != elfMagic {
		return errors.New("发布包内取出的文件不是 ELF 可执行文件，上游打包方式可能已变更")
	}
	return nil
}

// assertExecutable 确认目标是原生可执行文件而不是脚本。
// 面板只替换 dae 本体；ExecStart 指向包装脚本时应当由运维自己处理。
func assertExecutable(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	header := make([]byte, len(elfMagic))
	if _, err := io.ReadFull(file, header); err != nil {
		return fmt.Errorf("读取 %s 的文件头: %w", path, err)
	}
	if string(header) != elfMagic {
		return fmt.Errorf("服务启动的 %s 不是 ELF 可执行文件，看起来是启动脚本；"+
			"面板只替换 dae 本体，请把 ExecStart 指向 dae 可执行文件本身", path)
	}
	return nil
}

// fileDigest 每次都实打实地读文件算 sha256。
//
// 刻意不按 (路径, 大小, 修改时间) 缓存：那个键对它唯一的用途——发现二进制被
// 外部替换过——并不成立。tar 解包、cp -p、rsync -t 都会原样保留 mtime，
// 同尺寸的替换因此拿到同一个键，摘要会永久停在旧值，漂移检测直接失效。
// 而这点开销本来也不显眼：状态查询每轮已经要起两个子进程（systemctl show 与
// dae --version），它们比对一个已在页缓存里的文件做 sha256 贵得多。
func (i *Installer) fileDigest(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return digestBytes(content), nil
}

func digestBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
