package daeinstall

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// procdInitDir 是 procd 读取服务定义的目录。init 脚本按服务名落在这里，
// 与 host.procdManager 的 initScript() 是同一个约定。
const procdInitDir = "/etc/init.d"

// defaultProcdServiceName 是没有配置服务名时的取值，与 host 包一致。
const defaultProcdServiceName = "dae"

// procdUnits 管理 procd 的服务定义。
//
// 与 systemd 分支最大的不同：这个文件属于 kdae-panel 软件包，不属于任何一次
// dae 安装。面板只校验它在不在、从不写它，卸载 dae 时也不删它——删了之后
// 用户就再也没法从面板重新装回 dae，而修复手段是重装整个软件包。
type procdUnits struct {
	installer *Installer
	// path 是 init 脚本的位置，留空即用默认，测试会覆盖它。
	path string
}

// Path 按服务名拼出 init 脚本位置。
//
// 不能写死 /etc/init.d/dae：服务名来自 UCI 的 service_name，host.procdManager
// 用的就是 filepath.Join("/etc/init.d", serviceName)。两处从同一个配置项读出
// 两个答案时，面板会对着一个不存在的脚本报"软件包被破坏"，而服务本身好好的。
func (u *procdUnits) Path() string {
	if u.path != "" {
		return u.path
	}
	name := u.installer.serviceName
	if name == "" {
		name = defaultProcdServiceName
	}
	return filepath.Join(procdInitDir, name)
}

// WritableDirs 为空：面板从不写 init 脚本所在的目录。
func (u *procdUnits) WritableDirs() []string { return nil }

// RemovablePaths 为空：init 脚本归软件包所有。
func (u *procdUnits) RemovablePaths() []string { return nil }

// VerifyRemovable 无事可做：没有要删的服务定义，也就没有"删对了没有"可校验。
// dae 可执行文件本身的归属校验在 uninstallTarget 里按摘要账本完成，与后端无关。
func (u *procdUnits) VerifyRemovable(host.Status, string) error { return nil }

// Detect 以磁盘上有没有 dae 可执行文件为准。
//
// 刻意不看 status.ExecStartPath：procd 在服务停止时拿不到命令行，那个字段
// 会回退成面板自己配置的路径，据此判断"已安装"就恒为真，首次安装从此不可达。
func (u *procdUnits) Detect(_ context.Context, _ host.Status) unitDetection {
	detection := unitDetection{}
	info, err := os.Stat(u.installer.binaryPath)
	if err != nil {
		return detection
	}
	if !info.Mode().IsRegular() {
		detection.Installed = true
		detection.Blocker = fmt.Sprintf("%s 已存在且不是普通文件，面板拒绝替换它", u.installer.binaryPath)
		return detection
	}
	detection.Installed = true
	detection.Blocker = fmt.Sprintf(
		"已存在 dae（%s），请使用版本切换而不是首次安装", u.installer.binaryPath)
	return detection
}

// Plan 只校验 init 脚本在不在。
//
// 必须赶在二进制被替换之前查：脚本缺失时装上 dae 也起不来，而那时候
// 错误现场离真正的原因（软件包被破坏）已经很远了。
func (u *procdUnits) Plan(upstream.Bundle) (string, bool, error) {
	path := u.Path()
	info, err := os.Stat(path)
	if err != nil {
		return "", false, fmt.Errorf(
			"找不到 dae 的启动脚本 %s：%v；它由 kdae-panel 软件包提供，请重新安装该软件包", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("%s 不是普通文件，面板拒绝据此安装 dae", path)
	}
	return "", true, nil
}

// Commit 什么都不做：脚本已经在位，procd 每次执行它都会重读定义。
func (u *procdUnits) Commit(context.Context, string, bool) error { return nil }

var _ unitProvisioner = (*procdUnits)(nil)
