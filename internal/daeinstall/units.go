package daeinstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// unitDetection 是"这台机器上已经有 dae 了吗"的判定结果。
type unitDetection struct {
	Installed bool
	// Blocker 非空表示不能首次安装，它直接作为拒绝理由呈现给用户。
	Blocker string
	// Notes 是不阻断安装、但用户应当知道的情况。
	Notes []string
}

// unitProvisioner 抽掉"服务定义"在不同 init 系统上的差异。
//
// systemd 下服务定义是 /etc/systemd/system/dae.service，由面板从发布包渲染并
// 写入，卸载时删除；procd 下它是 ipk 自带的 /etc/init.d/dae，面板只校验、
// 从不改写，也从不删除——那个文件属于软件包，不属于某一次 dae 安装。
type unitProvisioner interface {
	// Path 是服务定义文件的位置，回报给界面。
	Path() string
	// WritableDirs 是首次安装需要写入、因而必须提前预检的目录。
	WritableDirs() []string
	// RemovablePaths 是卸载时应当一并删除的服务定义文件。
	RemovablePaths() []string
	// VerifyRemovable 在卸载前确认服务定义确实归面板管理。
	// binaryPath 是本次将要删除的可执行文件。
	VerifyRemovable(status host.Status, binaryPath string) error
	// Detect 判定机器上是否已有 dae 服务。
	Detect(ctx context.Context, status host.Status) unitDetection
	// Plan 在动任何文件之前决定服务定义要不要写、写什么。
	// inPlace 为真表示已经就位，Commit 不必写盘。
	Plan(bundle upstream.Bundle) (content string, inPlace bool, err error)
	// Commit 落地服务定义并让 init 系统认识它。
	Commit(ctx context.Context, content string, inPlace bool) error
}

// systemdUnits 管理 systemd 的服务单元。
//
// 刻意不自带目录字段：`Installer.unitDir` 与 `unitDirectory()` / `serviceUnit()`
// 原地不动，现有测试有十余处直接读写它们，把这份状态挪个位置只会制造一批
// 与本次改造无关的测试改动。
type systemdUnits struct {
	installer *Installer
}

func (u *systemdUnits) Path() string {
	return filepath.Join(u.installer.unitDirectory(), u.installer.serviceUnit())
}

func (u *systemdUnits) WritableDirs() []string {
	return []string{u.installer.unitDirectory()}
}

func (u *systemdUnits) RemovablePaths() []string {
	return []string{u.Path()}
}

// VerifyRemovable 确认要删的单元正是面板写下的那一个。
//
// 三道关缺一不可：单元必须位于面板管理的标准路径（别人放在 /usr/lib/systemd
// 下的单元不归面板删）、必须是普通文件、它的 ExecStart 必须与本次要删的
// 可执行文件一致。少了最后一条，一个指向别处的同名单元会被连坐删除。
func (u *systemdUnits) VerifyRemovable(status host.Status, binaryPath string) error {
	// enabled-runtime 是 systemd 独有的临时启用态，面板无法无损恢复它。
	if status.UnitFileState == "enabled-runtime" {
		return errors.New("dae 使用临时启用状态 enabled-runtime，面板无法无损恢复该状态，请先执行 systemctl disable dae")
	}
	if status.UnitPath == "" {
		return errors.New("没有找到 dae 的服务单元")
	}
	unitPath, err := filepath.Abs(status.UnitPath)
	if err != nil {
		return fmt.Errorf("解析 dae 服务单元路径: %w", err)
	}
	expectedUnit, err := filepath.Abs(u.Path())
	if err != nil {
		return fmt.Errorf("解析面板服务单元路径: %w", err)
	}
	if unitPath != expectedUnit {
		return fmt.Errorf(
			"dae 服务单元位于 %s，不是面板管理的标准路径 %s；请用原安装方式卸载", unitPath, expectedUnit)
	}
	if err := regularFile(unitPath, "dae 服务单元"); err != nil {
		return err
	}
	unit, err := os.ReadFile(unitPath)
	if err != nil {
		return fmt.Errorf("读取 dae 服务单元: %w", err)
	}
	if exec := execStartBinary(unitExecStart(string(unit))); exec != binaryPath {
		return fmt.Errorf("服务单元实际启动 %s，与服务状态报告的 %s 不一致，拒绝卸载", exec, binaryPath)
	}
	return nil
}

func (u *systemdUnits) Detect(_ context.Context, status host.Status) unitDetection {
	detection := unitDetection{}
	if status.ExecStartPath == "" {
		return detection
	}
	if _, err := os.Stat(status.ExecStartPath); err == nil {
		detection.Installed = true
		detection.Blocker = fmt.Sprintf(
			"已存在 dae 服务（启动 %s），请使用版本切换而不是首次安装", status.ExecStartPath)
		return detection
	}
	// 单元在、可执行文件不在。升级路径会说"目标不存在"，首次安装若也以
	// "已有服务"为由拒绝，面板就再没有任何办法修好这台机器。只要单元指向的
	// 正是面板要写的位置，就按首次安装把它补齐。
	if filepath.Clean(status.ExecStartPath) != u.installer.binaryPath {
		detection.Installed = true
		detection.Blocker = fmt.Sprintf(
			"服务单元指向的 %s 不存在，而面板配置的 dae 路径是 %s；"+
				"请把 KDAE_PANEL_DAE_BINARY 改成前者后重试",
			status.ExecStartPath, u.installer.binaryPath)
		return detection
	}
	detection.Notes = append(detection.Notes, fmt.Sprintf(
		"服务单元已存在，但它启动的 %s 不见了，本次安装会补齐这个文件", u.installer.binaryPath))
	return detection
}

// Plan 渲染出最终要落盘的单元，并判定它是否已经就位——但不写盘。
//
// 拆成"先算后写"是为了让冲突在事务的最前面暴露：这是唯一一处可能因为机器上
// 已有用户自建单元而中止的检查，必须赶在二进制被替换之前完成。
//
// 已存在的单元一律不覆盖，除非它与本次将要写入的内容逐字节相同——那说明它正是
// 上一轮安装留下的。少了这个例外，一旦 daemon-reload 失败，重试就会被自己写下
// 的单元永久挡住：systemd 还不认识它，所以预检仍认为没装，而写入又拒绝覆盖。
func (u *systemdUnits) Plan(bundle upstream.Bundle) (string, bool, error) {
	if len(bundle.Unit) == 0 {
		return "", false, errors.New("发布包内没有 dae.service，无法创建服务单元")
	}
	rendered, err := u.render(string(bundle.Unit))
	if err != nil {
		return "", false, err
	}
	path := u.Path()
	switch existing, err := os.ReadFile(path); {
	case err == nil && string(existing) == rendered:
		return rendered, true, nil
	case err == nil:
		// 内容不同，但它启动的已经是面板要装的那个文件——官方安装器写的单元、
		// 用户自己调过的单元都属于这种。它能把新装的二进制起起来，就没有理由
		// 为了统一格式去覆盖别人的文件。
		if execStartBinary(unitExecStart(string(existing))) == u.installer.binaryPath {
			return string(existing), true, nil
		}
		return "", false, fmt.Errorf("%s 已存在且启动的不是 %s，面板不覆盖既有服务单元",
			path, u.installer.binaryPath)
	case !os.IsNotExist(err):
		return "", false, err
	}
	return rendered, false, nil
}

func (u *systemdUnits) Commit(ctx context.Context, content string, inPlace bool) error {
	if !inPlace {
		if err := writeFileSynced(u.Path(), []byte(content), unitMode); err != nil {
			return fmt.Errorf("写入服务单元: %w", err)
		}
	}
	if err := u.installer.service.Action(ctx, host.ActionDaemonReload); err != nil {
		return fmt.Errorf("重新加载 systemd 配置: %w", err)
	}
	return nil
}

// render 生成最终落盘的单元内容，并确认改写确实生效。
//
// 替换靠的是上游单元里那两个字面量默认值。上游若换了默认路径，替换会悄无声息
// 地不生效，写出一个指向别处的单元——那样 dae 起不来，而错误现场离真正的原因
// 很远。宁可在这里直接拒绝，把原因说清楚。
func (u *systemdUnits) render(unit string) (string, error) {
	rendered := retargetUnit(unit, u.installer.binaryPath, u.installer.configPath)
	execStart := unitExecStart(rendered)
	if execStart == "" {
		return "", errors.New("发布包内的 dae.service 没有 ExecStart，无法安装")
	}
	if !strings.HasPrefix(execStart, u.installer.binaryPath+" ") && execStart != u.installer.binaryPath {
		return "", fmt.Errorf(
			"发布包内的 dae.service 启动的是 %q，面板无法把它改写为 %s；"+
				"上游可能变更了默认路径，请手动创建服务单元", execStart, u.installer.binaryPath)
	}
	return rendered, nil
}

var _ unitProvisioner = (*systemdUnits)(nil)
