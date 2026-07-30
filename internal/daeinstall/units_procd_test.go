package daeinstall

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

func newProcdUnitsForTest(t *testing.T, binaryPath, initPath string) *procdUnits {
	t.Helper()
	return &procdUnits{
		installer: &Installer{binaryPath: binaryPath},
		path:      initPath,
	}
}

// init 脚本由 ipk 提供，面板只校验存在性，永远不写。
func TestProcdUnitsPlanAcceptsExistingScript(t *testing.T) {
	dir := t.TempDir()
	initPath := filepath.Join(dir, "dae")
	if err := os.WriteFile(initPath, []byte("#!/bin/sh /etc/rc.common\n"), 0o755); err != nil {
		t.Fatalf("写入 init 脚本: %v", err)
	}
	units := newProcdUnitsForTest(t, "/usr/bin/dae", initPath)

	content, inPlace, err := units.Plan(upstream.Bundle{})
	if err != nil {
		t.Fatalf("Plan 返回错误: %v", err)
	}
	if !inPlace {
		t.Fatal("inPlace 应为真：init 脚本由软件包提供，面板不写")
	}
	if content != "" {
		t.Fatalf("content = %q，期望空", content)
	}
}

// 脚本不在说明软件包坏了或被删。必须在动任何文件之前拒绝，
// 而不是装完二进制才发现服务起不来。
func TestProcdUnitsPlanRejectsMissingScript(t *testing.T) {
	units := newProcdUnitsForTest(t, "/usr/bin/dae", filepath.Join(t.TempDir(), "dae"))

	if _, _, err := units.Plan(upstream.Bundle{}); err == nil {
		t.Fatal("init 脚本缺失时 Plan 应当报错")
	} else if !strings.Contains(err.Error(), "kdae-panel") {
		t.Fatalf("错误信息 = %q，应当指引重装软件包", err.Error())
	}
}

// Commit 什么都不做，尤其不能调 daemon-reload 之外的任何写操作。
func TestProcdUnitsCommitIsNoop(t *testing.T) {
	dir := t.TempDir()
	initPath := filepath.Join(dir, "dae")
	units := newProcdUnitsForTest(t, "/usr/bin/dae", initPath)

	if err := units.Commit(context.Background(), "", true); err != nil {
		t.Fatalf("Commit 返回错误: %v", err)
	}
	if _, err := os.Stat(initPath); !os.IsNotExist(err) {
		t.Fatal("Commit 不应创建任何文件")
	}
}

// procd 下卸载 dae 不该删掉 ipk 装的 init 脚本。
func TestProcdUnitsRemovesNothing(t *testing.T) {
	units := newProcdUnitsForTest(t, "/usr/bin/dae", "/etc/init.d/dae")

	if paths := units.RemovablePaths(); len(paths) != 0 {
		t.Fatalf("RemovablePaths = %v，期望空", paths)
	}
	if dirs := units.WritableDirs(); len(dirs) != 0 {
		t.Fatalf("WritableDirs = %v，期望空：init 脚本目录面板从不写", dirs)
	}
	// 卸载 dae 不该因为"单元校验不过"而被拦下——procd 下根本没有要校验的单元。
	if err := units.VerifyRemovable(host.Status{}, "/usr/bin/dae"); err != nil {
		t.Fatalf("VerifyRemovable 返回错误: %v", err)
	}
}

// 二进制在即已安装。不看 ExecStartPath：procd 下服务停止时它来自回退链，
// 回退值恒等于面板配置的路径，据此判断会永远为真。
func TestProcdUnitsDetectUsesBinaryPresence(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "dae")
	units := newProcdUnitsForTest(t, binary, filepath.Join(dir, "init"))

	detection := units.Detect(context.Background(), host.Status{ExecStartPath: binary})
	if detection.Installed {
		t.Fatal("二进制不存在时不应判为已安装")
	}
	if detection.Blocker != "" {
		t.Fatalf("Blocker = %q，期望空", detection.Blocker)
	}

	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatalf("写入假二进制: %v", err)
	}
	detection = units.Detect(context.Background(), host.Status{ExecStartPath: binary})
	if !detection.Installed {
		t.Fatal("二进制存在时应判为已安装")
	}
	if !strings.Contains(detection.Blocker, "版本切换") {
		t.Fatalf("Blocker = %q，应当引导用户走版本切换", detection.Blocker)
	}
}

// procd 部署的用户看不懂 systemd 词汇，文案里不能出现它们。
func TestProcdUnitsMessagesAvoidSystemdVocabulary(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "dae")
	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatalf("写入假二进制: %v", err)
	}
	units := newProcdUnitsForTest(t, binary, filepath.Join(dir, "init"))

	_, _, planErr := units.Plan(upstream.Bundle{})
	detection := units.Detect(context.Background(), host.Status{})
	texts := []string{planErr.Error(), detection.Blocker}
	for _, forbidden := range []string{"systemd", "systemctl", "journalctl", "ReadWritePaths", ".service"} {
		for _, text := range texts {
			if strings.Contains(text, forbidden) {
				t.Fatalf("文案 %q 含 systemd 词汇 %q", text, forbidden)
			}
		}
	}
}
