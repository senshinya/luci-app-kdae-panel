// Package atomicfile 提供"要么完整落盘，要么什么都没变"的写入。
//
// 面板替换的都是 dae 正在使用的文件——可执行文件、配置、geo 数据。半个文件
// 落在磁盘上比写入失败糟糕得多：dae 可能在任意时刻读它，而错误现场离原因很远。
// 因此一律走"同目录临时文件 → fsync → 原子改名"，并 fsync 父目录让改名本身持久化。
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Writable 通过实际建删一个临时文件判断目录可写。
//
// 只看权限位不够：ProtectSystem=strict 下 root 对未列入 ReadWritePaths 的目录
// 同样写不进去，而那正是这个面板最常见的失败原因。
//
// 探测本身不创建目录：它会被界面轮询反复调用，一个用于展示的检查不该在文件系统上
// 留下痕迹。目录尚不存在时改为探测最近的已存在祖先——那正是安装时真正要写入的地方。
func Writable(directory string) error {
	existing := directory
	for {
		info, err := os.Stat(existing)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("%s 不是目录", existing)
			}
			break
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return fmt.Errorf("找不到 %s 的任何已存在上级目录", directory)
		}
		existing = parent
	}
	file, err := os.CreateTemp(existing, ".kdae-panel-probe-*")
	if err != nil {
		return err
	}
	name := file.Name()
	_ = file.Close()
	return os.Remove(name)
}

// Write 把内容原子地写入 path。
// 目录由 Stage 建——临时文件必须与目标同目录，Stage 成功即意味着目录已就绪。
func Write(path string, content []byte, mode os.FileMode) error {
	staged, cleanup, err := Stage(filepath.Dir(path), content, mode)
	if err != nil {
		return err
	}
	defer cleanup()
	return Replace(staged, path)
}

// Stage 在 directory 下写一个带内容的临时文件，返回它的路径与清理函数。
//
// 必须与目标同目录：跨文件系统改名会 EXDEV，而面板单元开了 PrivateTmp，
// /tmp 也常以 noexec 挂载，放那里的可执行文件连自检都跑不了。
//
// 临时名由 os.CreateTemp 生成而非固定后缀：固定名字会让两个并发写者
// 踩同一个文件，后者的内容可能被前者的清理删掉。
func Stage(directory string, content []byte, mode os.FileMode) (string, func(), error) {
	return StagePattern(directory, ".kdae-panel-*", content, mode)
}

// StagePattern 与 Stage 相同，但允许调用方使用专属的临时文件前缀。
// 只有能明确归属的异常残留才可以安全清理，不能把配置、Geo 和二进制暂存文件
// 全都混在同一个通配符下。
func StagePattern(directory, pattern string, content []byte, mode os.FileMode) (string, func(), error) {
	noop := func() {}
	if mode.Perm() == 0 {
		mode = 0o600
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", noop, fmt.Errorf("创建目录 %s: %w", directory, err)
	}
	file, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return "", noop, fmt.Errorf("创建暂存文件: %w", err)
	}
	path := file.Name()
	cleanup := func() { _ = os.Remove(path) }

	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		cleanup()
		return "", noop, fmt.Errorf("写入暂存文件: %w", err)
	}
	// CreateTemp 一律建成 0600，权限要显式设回去；UMask 不影响 Chmod。
	if err := file.Chmod(mode.Perm()); err != nil {
		_ = file.Close()
		cleanup()
		return "", noop, fmt.Errorf("设置权限: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		cleanup()
		return "", noop, fmt.Errorf("同步暂存文件: %w", err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", noop, fmt.Errorf("关闭暂存文件: %w", err)
	}
	return path, cleanup, nil
}
