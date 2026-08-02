package geodata

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tuoro/kdae-panel/internal/atomicfile"
)

const (
	geoTempPattern = ".kdae-panel-geo-*"
	geoTempPrefix  = ".kdae-panel-geo-"
	rollbackSuffix = ".kdae-panel-previous"
	// 跨进程实例不共享面板的操作锁；只有超过一次 Geo 更新最大时长数倍的
	// 专属暂存文件，才能确定不是另一个实例正在写入。
	geoTempStaleAfter = time.Hour
)

// findResiduals 只认 Geo 专属暂存前缀和固定回滚后缀。
// 历史版本使用通用的 .kdae-panel-*，它也可能属于配置或二进制事务，无法安全归属，
// 因此不能在 Geo 页面里猜测并删除。
func findResiduals(searchPath []string) []Residual {
	var residuals []Residual
	seen := make(map[string]bool)
	for _, directory := range searchPath {
		directory = filepath.Clean(directory)
		if directory == SandboxHiddenDir || seen[directory] {
			continue
		}
		seen[directory] = true

		entries, err := os.ReadDir(directory)
		if err == nil {
			staleBefore := nowUTC().Add(-geoTempStaleAfter)
			for _, entry := range entries {
				if !strings.HasPrefix(entry.Name(), geoTempPrefix) {
					continue
				}
				path := filepath.Join(directory, entry.Name())
				info, infoErr := os.Lstat(path)
				if infoErr != nil {
					continue
				}
				if info.Mode().IsRegular() && info.ModTime().After(staleBefore) {
					continue
				}
				residuals = append(residuals, Residual{
					Path: path, Kind: ResidualTemporary, Size: info.Size(), ModTime: info.ModTime().UTC(),
					Deletable: info.Mode().IsRegular(),
				})
			}
		}

		for _, name := range Names {
			target := filepath.Join(directory, name)
			backup := target + rollbackSuffix
			info, err := os.Lstat(backup)
			if err != nil {
				continue
			}
			targetInfo, targetErr := os.Lstat(target)
			targetMissing := os.IsNotExist(targetErr)
			regular := info.Mode().IsRegular()
			residuals = append(residuals, Residual{
				Path: backup, Kind: ResidualRollback, Size: info.Size(), ModTime: info.ModTime().UTC(),
				TargetPath: target, Restorable: regular && targetMissing,
				Deletable: regular && targetErr == nil && targetInfo.Mode().IsRegular(),
			})
		}
	}
	sort.Slice(residuals, func(i, j int) bool { return residuals[i].Path < residuals[j].Path })
	return residuals
}

func cleanupTemporaryResiduals(residuals []Residual) error {
	var failures []error
	for _, residual := range residuals {
		if residual.Kind != ResidualTemporary || !residual.Deletable {
			continue
		}
		if err := removeResidualFile(residual.Path, geoTempPrefix); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// CleanupResiduals 删除不承载唯一旧数据的残留：Geo 暂存文件，以及正式文件仍在时
// 的回滚点。正式文件缺失时，回滚点必须走 RestoreResidual，不能在这里删除。
func (m *Manager) CleanupResiduals(ctx context.Context) (Status, error) {
	status := m.Status(ctx)
	var failures []error
	for _, residual := range status.Residuals {
		if !residual.Deletable {
			continue
		}
		prefix := geoTempPrefix
		if residual.Kind == ResidualRollback {
			prefix = ""
		}
		if err := removeResidualFile(residual.Path, prefix); err != nil {
			failures = append(failures, err)
		}
	}
	if err := errors.Join(failures...); err != nil {
		return m.Status(ctx), err
	}
	return m.Status(ctx), nil
}

// RestoreResidual 把正式文件缺失时的回滚点原子地放回原位。
func (m *Manager) RestoreResidual(ctx context.Context, path string) (Status, error) {
	status := m.Status(ctx)
	var selected *Residual
	for index := range status.Residuals {
		candidate := &status.Residuals[index]
		if candidate.Path == path && candidate.Kind == ResidualRollback && candidate.Restorable {
			selected = candidate
			break
		}
	}
	if selected == nil {
		return status, errors.New("指定的 Geo 回滚点不存在、不可恢复，或正式文件已经存在")
	}
	if _, err := os.Lstat(selected.TargetPath); !os.IsNotExist(err) {
		return status, fmt.Errorf("恢复目标 %s 已存在，拒绝覆盖", selected.TargetPath)
	}
	info, err := os.Lstat(selected.Path)
	if err != nil || !info.Mode().IsRegular() {
		return status, fmt.Errorf("回滚点 %s 已变化，拒绝恢复", selected.Path)
	}
	if err := atomicfile.Replace(selected.Path, selected.TargetPath); err != nil {
		return status, fmt.Errorf("恢复 %s: %w", selected.TargetPath, err)
	}

	service := m.inspectService(ctx)
	if _, err := m.reload(ctx, service); err != nil {
		return m.Status(ctx), fmt.Errorf("旧 Geo 文件已恢复，但 dae 重新加载失败: %w", err)
	}
	return m.Status(ctx), nil
}

func removeResidualFile(path, requiredPrefix string) error {
	if requiredPrefix != "" && !strings.HasPrefix(filepath.Base(path), requiredPrefix) {
		return fmt.Errorf("拒绝清理无法确认归属的文件 %s", path)
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("拒绝清理非普通文件 %s", path)
	}
	return os.Remove(path)
}
