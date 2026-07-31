package daeinstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tuoro/kdae-panel/internal/atomicfile"
	"github.com/tuoro/kdae-panel/internal/geodata"
	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// removedFile 是卸载事务里被移到同目录暂存位的文件。
// 同目录 rename 既是原子的，也不会碰到 PrivateTmp 带来的跨文件系统问题。
type removedFile struct {
	original string
	staged   string
}

// UninstallOptions 是用户对 dae 数据的明确处置选择。
// 零值必须安全：旧客户端或空请求一律保留配置与 geo。
type UninstallOptions struct {
	PurgeConfig bool `json:"purgeConfig"`
	PurgeGeo    bool `json:"purgeGeo"`
}

// Uninstall 删除面板管理的 dae 可执行文件、服务单元与版本账本。
// 配置和 geo 默认保留，只有 options 显式要求时才进入同一个删除事务。
func (i *Installer) Uninstall(ctx context.Context, options UninstallOptions) error {
	status, target, err := i.uninstallTarget(ctx)
	if err != nil {
		return err
	}
	dataPaths, err := i.uninstallDataPaths(status, options)
	if err != nil {
		return err
	}
	// required 里的文件必须存在，缺一个就说明状态与账本对不上，宁可中止；
	// 其余的允许缺失。procd 下服务定义属于软件包而非某次安装，因此
	// RemovablePaths 为空——卸载 dae 不该删掉 ipk 装的 /etc/init.d/dae。
	required := append([]string{target}, i.units.RemovablePaths()...)
	paths := append([]string{}, required...)
	paths = append(paths,
		i.statePath,
		i.previousStatePath(),
		i.backupPath,
		i.pendingBackupPath(),
	)
	paths = append(paths, dataPaths...)
	for index, path := range paths {
		if _, err := os.Lstat(path); err != nil {
			if index >= len(required) && os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("检查待删除文件 %s: %w", path, err)
		}
		if err := atomicfile.Writable(filepath.Dir(path)); err != nil {
			return fmt.Errorf("面板无法删除 %s：%w", path, err)
		}
	}

	wasActive := status.ActiveState == "active"
	wasEnabled := status.UnitFileState == "enabled"
	if wasActive {
		if err := i.service.Action(ctx, host.ActionStop); err != nil {
			return i.recoverAfterControlFailure(ctx, false, true, fmt.Errorf("停止 dae 服务: %w", err))
		}
	}
	if wasEnabled {
		if err := i.service.Action(ctx, host.ActionDisable); err != nil {
			// systemctl 可能删掉部分 wants 链接后才报错；重新 enable 是幂等的，
			// 因此按原状态恢复，不能假定失败就代表什么都没改。
			return i.recoverAfterControlFailure(ctx, true, wasActive, fmt.Errorf("禁用 dae 服务: %w", err))
		}
	}

	removed := make([]removedFile, 0, len(paths))
	for index, path := range paths {
		file, err := stageRemoval(path, index >= len(required))
		if err != nil {
			return i.rollbackUninstall(ctx, removed, wasEnabled, wasActive,
				fmt.Errorf("暂存待删除文件 %s: %w", path, err))
		}
		if file != nil {
			removed = append(removed, *file)
		}
	}

	if err := i.service.Action(ctx, host.ActionDaemonReload); err != nil {
		// 这里的调用点本身与后端无关：两套 init 系统都会走到这一行。
		// procdManager 对 ActionDaemonReload 硬编码 return nil，今天这个分支
		// 在 procd 下走不到，但那只是 procd.go 里的一个实现选择，不是这里能
		// 保证的不变量；措辞中性化，不必像 provision.go 的写权限提示那样按
		// 后端分支给不同建议——两套后端的修法本来就是同一句话："重试卸载"。
		return i.rollbackUninstall(ctx, removed, wasEnabled, wasActive,
			fmt.Errorf("重新加载服务配置: %w", err))
	}

	for _, file := range removed {
		if err := os.Remove(file.staged); err != nil && !os.IsNotExist(err) {
			// 生效文件已经移走，残留的隐藏暂存位不会影响运行；卸载本身不应因此
			// 被报告成失败，但必须留日志供运维清理。
			i.logger.Warn("清理 dae 卸载暂存文件失败", "path", file.staged, "error", err)
		}
	}
	i.logger.Info("已卸载 dae", "binary", target, "unit", i.units.Path(),
		"purge_config", options.PurgeConfig, "purge_geo", options.PurgeGeo)
	return nil
}

// uninstallDataPaths 返回用户明确要求一并删除的数据文件。
// geo 要删除所有面板可见副本，否则高优先级副本移走后，被遮蔽的旧副本会重新生效。
func (i *Installer) uninstallDataPaths(status host.Status, options UninstallOptions) ([]string, error) {
	paths := make([]string, 0, 3)
	seen := make(map[string]struct{})
	appendPath := func(path, description string) error {
		path = filepath.Clean(path)
		if _, exists := seen[path]; exists {
			return nil
		}
		info, err := os.Lstat(path)
		switch {
		case os.IsNotExist(err):
			return nil
		case err != nil:
			return fmt.Errorf("检查%s %s: %w", description, path, err)
		case !info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0:
			return fmt.Errorf("%s %s 不是普通文件或符号链接，拒绝删除", description, path)
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
		return nil
	}
	if options.PurgeConfig && i.configPath != "" {
		if err := appendPath(i.configPath, "dae 主配置文件"); err != nil {
			return nil, err
		}
	}
	if !options.PurgeGeo {
		return paths, nil
	}

	search := i.geoSearchDirs
	if search == nil {
		search = geodata.SearchPath(i.configPath, status.Environment)
	}
	for _, directory := range search {
		// 只在这个目录确实被沙箱挡住时才跳过。ProtectHome=true 下读它会拿到
		// EACCES，跳过比把权限错误冒充"不存在"更诚实，确认框也明确限定为
		// 面板可见副本；而 procd 部署没有这层遮挡，/root/.local/share/dae 对
		// 面板完全可见——无条件跳过会让"删除 geo 数据"悄悄漏掉那一份，
		// 用户看到的承诺与实际行为对不上。判据与 geodata.MissingWarning 统一。
		if filepath.Clean(directory) == filepath.Clean(geodata.SandboxHiddenDir) &&
			geodata.SandboxHidesHome() {
			continue
		}
		for _, name := range geodata.Names {
			path := filepath.Join(directory, name)
			if err := appendPath(path, "geo 数据"); err != nil {
				return nil, err
			}
		}
	}
	return paths, nil
}

// uninstallTarget 把所有破坏性操作前的安全检查集中在一起。
func (i *Installer) uninstallTarget(ctx context.Context) (host.Status, string, error) {
	status, err := i.service.Status(ctx)
	if err != nil {
		return host.Status{}, "", fmt.Errorf("读取 dae 服务状态: %w", err)
	}
	// UnitPath 的校验交给 units：procd 下服务定义归软件包，不该在这里挡路。
	if status.ExecStartPath == "" {
		return host.Status{}, "", errors.New("没有找到可卸载的 dae 服务")
	}

	target, err := filepath.Abs(status.ExecStartPath)
	if err != nil {
		return host.Status{}, "", fmt.Errorf("解析 dae 可执行文件路径: %w", err)
	}
	if filepath.Base(target) != upstream.BinaryName {
		return host.Status{}, "", fmt.Errorf("服务启动的是 %s，文件名不是 %s，拒绝卸载", target, upstream.BinaryName)
	}
	if err := regularFile(target, "dae 可执行文件"); err != nil {
		return host.Status{}, "", err
	}
	if err := i.assertExecutable(target); err != nil {
		return host.Status{}, "", err
	}

	state, err := i.readState()
	if err != nil {
		return host.Status{}, "", fmt.Errorf("读取 dae 安装记录: %w", err)
	}
	if state == nil || state.SHA256 == "" {
		return host.Status{}, "", errors.New("当前 dae 没有面板安装记录，为避免删除外部安装，已拒绝卸载")
	}
	digest, err := i.fileDigest(target)
	if err != nil {
		return host.Status{}, "", err
	}
	if digest != state.SHA256 {
		return host.Status{}, "", errors.New("dae 二进制已在面板之外被替换，为避免删除未知文件，已拒绝卸载")
	}

	if err := i.units.VerifyRemovable(status, target); err != nil {
		return host.Status{}, "", err
	}
	return status, target, nil
}

func regularFile(path, description string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("读取%s %s: %w", description, path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%s %s 不是普通文件，拒绝卸载", description, path)
	}
	return nil
}

// stageRemoval 把文件原子移到同目录的随机暂存位。optional 允许账本与回滚点不存在。
func stageRemoval(path string, optional bool) (*removedFile, error) {
	if _, err := os.Lstat(path); err != nil {
		if optional && os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".kdae-panel-uninstall-*")
	if err != nil {
		return nil, err
	}
	staged := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(staged)
		return nil, err
	}
	if err := os.Remove(staged); err != nil {
		return nil, err
	}
	// 这里不走 atomicfile.Replace：它在 rename 成功后还会 fsync 目录，fsync
	// 若失败，返回值会说失败而文件其实已经移走，调用方因而漏掉回滚。
	// 暂存位稍后还会结算，单次 rename 的明确成败比中间态持久化更重要。
	if err := os.Rename(path, staged); err != nil {
		return nil, err
	}
	return &removedFile{original: path, staged: staged}, nil
}

func restoreRemoved(files []removedFile) error {
	var failures []error
	for index := len(files) - 1; index >= 0; index-- {
		file := files[index]
		if err := os.Rename(file.staged, file.original); err != nil {
			failures = append(failures, fmt.Errorf("恢复 %s: %w", file.original, err))
		}
	}
	return errors.Join(failures...)
}

func (i *Installer) rollbackUninstall(ctx context.Context, files []removedFile, wasEnabled, wasActive bool, cause error) error {
	restoreCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), restartTimeout)
	defer cancel()

	restoreErr := restoreRemoved(files)
	if len(files) >= 2 {
		restoreErr = errors.Join(restoreErr, i.service.Action(restoreCtx, host.ActionDaemonReload))
	}
	return i.recoverService(restoreCtx, wasEnabled, wasActive, errors.Join(cause, restoreErr))
}

func (i *Installer) recoverService(ctx context.Context, wasEnabled, wasActive bool, cause error) error {
	var failures []error
	if wasEnabled {
		if err := i.service.Action(ctx, host.ActionEnable); err != nil {
			failures = append(failures, fmt.Errorf("恢复 dae 开机启动: %w", err))
		}
	}
	if wasActive {
		if err := i.service.Action(ctx, host.ActionStart); err != nil {
			failures = append(failures, fmt.Errorf("恢复 dae 运行状态: %w", err))
		}
	}
	if recovery := errors.Join(failures...); recovery != nil {
		return fmt.Errorf("%w；卸载失败后的恢复也未完成：%v", cause, recovery)
	}
	return cause
}

func (i *Installer) recoverAfterControlFailure(ctx context.Context, wasEnabled, wasActive bool, cause error) error {
	restoreCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), restartTimeout)
	defer cancel()
	return i.recoverService(restoreCtx, wasEnabled, wasActive, cause)
}
