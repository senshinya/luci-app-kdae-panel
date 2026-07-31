package daeinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tuoro/kdae-panel/internal/geodata"
	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

func managedUninstallFixture(t *testing.T) (*Installer, *fakeService, string, string) {
	t.Helper()
	service := &fakeService{unitFileState: "enabled"}
	installer, binaryPath := newTestInstaller(t, &fakeFetcher{}, service)
	seed(t, binaryPath, "v1")

	installer.unitDir = testDir(t)
	unitPath := filepath.Join(installer.unitDir, installer.serviceUnit())
	unit := "[Service]\nExecStart=" + binaryPath + " run -c " + installer.configPath + "\n"
	if err := os.WriteFile(unitPath, []byte(unit), unitMode); err != nil {
		t.Fatal(err)
	}
	service.unitPath = unitPath
	if err := installer.writeState(&State{
		Source: upstream.SourceOfficial,
		Ref:    "v2.0.0",
		Label:  "v2.0.0",
		SHA256: digestBytes(elf("v1")),
	}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{installer.backupPath, installer.previousStatePath(), installer.pendingBackupPath()} {
		if err := writeFileSynced(path, elf("old"), binaryMode); err != nil {
			t.Fatal(err)
		}
	}
	return installer, service, binaryPath, unitPath
}

func TestUninstallRemovesManagedDaeButKeepsUserData(t *testing.T) {
	installer, service, binaryPath, unitPath := managedUninstallFixture(t)
	geoIP := filepath.Join(filepath.Dir(installer.configPath), "geoip.dat")
	geoSite := filepath.Join(filepath.Dir(installer.configPath), "geosite.dat")
	for _, path := range []string{geoIP, geoSite} {
		if err := os.WriteFile(path, []byte("user-data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	configBefore, err := os.ReadFile(installer.configPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := installer.Uninstall(context.Background(), UninstallOptions{}); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		binaryPath, unitPath, installer.statePath, installer.backupPath,
		installer.previousStatePath(), installer.pendingBackupPath(),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("卸载后 %s 仍存在或状态异常: %v", path, err)
		}
	}
	if got, err := os.ReadFile(installer.configPath); err != nil || string(got) != string(configBefore) {
		t.Fatalf("配置不应被卸载修改: %q, %v", got, err)
	}
	for _, path := range []string{geoIP, geoSite} {
		if got, err := os.ReadFile(path); err != nil || string(got) != "user-data" {
			t.Fatalf("geo 数据不应被卸载修改: %s = %q, %v", path, got, err)
		}
	}
	want := []host.Action{host.ActionStop, host.ActionDisable, host.ActionDaemonReload}
	if !reflect.DeepEqual(service.actions, want) {
		t.Fatalf("systemd 动作 = %v，期望 %v", service.actions, want)
	}
}

func TestUninstallAppliesIndependentDataChoices(t *testing.T) {
	for _, test := range []struct {
		name       string
		options    UninstallOptions
		keepConfig bool
		keepGeo    bool
	}{
		{name: "全部保留", options: UninstallOptions{}, keepConfig: true, keepGeo: true},
		{name: "只删配置", options: UninstallOptions{PurgeConfig: true}, keepGeo: true},
		{name: "只删 geo", options: UninstallOptions{PurgeGeo: true}, keepConfig: true},
		{name: "全部删除", options: UninstallOptions{PurgeConfig: true, PurgeGeo: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			installer, _, _, _ := managedUninstallFixture(t)
			directory := filepath.Dir(installer.configPath)
			installer.geoSearchDirs = []string{directory}
			geoPaths := []string{
				filepath.Join(directory, "geoip.dat"),
				filepath.Join(directory, "geosite.dat"),
			}
			for _, path := range geoPaths {
				if err := os.WriteFile(path, []byte("geo"), geoMode); err != nil {
					t.Fatal(err)
				}
			}

			if err := installer.Uninstall(context.Background(), test.options); err != nil {
				t.Fatal(err)
			}
			assertPresence := func(path string, present bool) {
				t.Helper()
				_, err := os.Stat(path)
				if present && err != nil {
					t.Fatalf("%s 应保留: %v", path, err)
				}
				if !present && !os.IsNotExist(err) {
					t.Fatalf("%s 应删除，状态 = %v", path, err)
				}
			}
			assertPresence(installer.configPath, test.keepConfig)
			for _, path := range geoPaths {
				assertPresence(path, test.keepGeo)
			}
		})
	}
}

func TestUninstallPreservesInactiveOrDisabledServiceState(t *testing.T) {
	for _, test := range []struct {
		name      string
		active    string
		unitState string
		want      []host.Action
	}{
		{name: "已停止但已启用", active: "inactive", unitState: "enabled", want: []host.Action{host.ActionDisable, host.ActionDaemonReload}},
		{name: "运行中但未启用", active: "active", unitState: "disabled", want: []host.Action{host.ActionStop, host.ActionDaemonReload}},
	} {
		t.Run(test.name, func(t *testing.T) {
			installer, service, _, _ := managedUninstallFixture(t)
			service.activeState = test.active
			service.unitFileState = test.unitState
			if err := installer.Uninstall(context.Background(), UninstallOptions{}); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(service.actions, test.want) {
				t.Fatalf("systemd 动作 = %v，期望 %v", service.actions, test.want)
			}
		})
	}
}

func TestUninstallRefusesRuntimeEnabledUnit(t *testing.T) {
	installer, service, _, _ := managedUninstallFixture(t)
	service.unitFileState = "enabled-runtime"
	if err := installer.Uninstall(context.Background(), UninstallOptions{}); err == nil || !strings.Contains(err.Error(), "enabled-runtime") {
		t.Fatalf("临时启用状态应被拒绝: %v", err)
	}
	if len(service.actions) != 0 {
		t.Fatalf("预检失败不应控制服务: %v", service.actions)
	}
}

func TestUninstallRefusesUnmanagedOrDriftedBinary(t *testing.T) {
	for _, test := range []struct {
		name    string
		prepare func(*Installer) error
		want    string
	}{
		{name: "没有安装记录", prepare: func(i *Installer) error { return os.Remove(i.statePath) }, want: "没有面板安装记录"},
		{name: "摘要漂移", prepare: func(i *Installer) error {
			return os.WriteFile(i.binaryPath, elf("outside"), binaryMode)
		}, want: "面板之外被替换"},
	} {
		t.Run(test.name, func(t *testing.T) {
			installer, service, binaryPath, unitPath := managedUninstallFixture(t)
			if err := test.prepare(installer); err != nil {
				t.Fatal(err)
			}
			if err := installer.Uninstall(context.Background(), UninstallOptions{}); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("错误 = %v，期望包含 %q", err, test.want)
			}
			if len(service.actions) != 0 {
				t.Fatalf("预检失败不应控制服务: %v", service.actions)
			}
			for _, path := range []string{binaryPath, unitPath} {
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("预检失败不应删除 %s: %v", path, err)
				}
			}
		})
	}
}

func TestUninstallRefusesNonStandardUnitPath(t *testing.T) {
	installer, service, binaryPath, _ := managedUninstallFixture(t)
	service.unitPath = filepath.Join(testDir(t), "dae.service")
	if err := os.WriteFile(service.unitPath, []byte("[Service]\nExecStart="+binaryPath+" run\n"), unitMode); err != nil {
		t.Fatal(err)
	}

	err := installer.Uninstall(context.Background(), UninstallOptions{})
	if err == nil || !strings.Contains(err.Error(), "不是面板管理的标准路径") {
		t.Fatalf("非标准单元应被拒绝: %v", err)
	}
	if len(service.actions) != 0 {
		t.Fatalf("拒绝前不应控制服务: %v", service.actions)
	}
}

func TestUninstallRejectsNonFileDataBeforeStoppingService(t *testing.T) {
	installer, service, _, _ := managedUninstallFixture(t)
	installer.configPath = testDir(t)

	err := installer.Uninstall(context.Background(), UninstallOptions{PurgeConfig: true})
	if err == nil || !strings.Contains(err.Error(), "不是普通文件或符号链接") {
		t.Fatalf("目录不应被当作配置文件删除: %v", err)
	}
	if len(service.actions) != 0 {
		t.Fatalf("数据预检失败前不应控制服务: %v", service.actions)
	}
}

// procd 部署没有 ProtectHome，/root/.local/share/dae 对面板完全可见。
// 无条件跳过它，"删除 geo 数据"就会悄悄漏掉那一份，而确认框承诺的是
// 删掉所有面板可见的副本。判据必须与提示文案用的那个一致。
func TestUninstallDeletesHomeGeoWhenVisible(t *testing.T) {
	installer, service, _, _ := managedUninstallFixture(t)
	visible := filepath.Join(testDir(t), "dae")
	if err := os.MkdirAll(visible, 0o755); err != nil {
		t.Fatal(err)
	}
	previous := geodata.SandboxHiddenDir
	geodata.SandboxHiddenDir = visible
	t.Cleanup(func() { geodata.SandboxHiddenDir = previous })

	geoIP := filepath.Join(visible, "geoip.dat")
	if err := os.WriteFile(geoIP, []byte("geo"), geoMode); err != nil {
		t.Fatal(err)
	}
	installer.geoSearchDirs = []string{visible}

	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	paths, err := installer.uninstallDataPaths(status, UninstallOptions{PurgeGeo: true})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(paths, filepath.Clean(geoIP)) {
		t.Fatalf("面板看得见的那份 geo 应当在删除清单里，实际 %v", paths)
	}
}

func TestUninstallRollbackRestoresFilesAndServiceState(t *testing.T) {
	installer, service, binaryPath, unitPath := managedUninstallFixture(t)
	directory := filepath.Dir(installer.configPath)
	installer.geoSearchDirs = []string{directory}
	geoPaths := []string{
		filepath.Join(directory, "geoip.dat"),
		filepath.Join(directory, "geosite.dat"),
	}
	for _, path := range geoPaths {
		if err := os.WriteFile(path, []byte("geo-before"), geoMode); err != nil {
			t.Fatal(err)
		}
	}
	configBefore, err := os.ReadFile(installer.configPath)
	if err != nil {
		t.Fatal(err)
	}
	service.actionErrors = map[host.Action][]error{
		host.ActionDaemonReload: {errors.New("第一次 daemon-reload 失败"), nil},
	}

	err = installer.Uninstall(context.Background(), UninstallOptions{PurgeConfig: true, PurgeGeo: true})
	if err == nil || !strings.Contains(err.Error(), "daemon-reload") {
		t.Fatalf("应报告 daemon-reload 失败: %v", err)
	}
	for _, path := range []string{binaryPath, unitPath, installer.statePath, installer.backupPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("回滚后 %s 应恢复: %v", path, err)
		}
	}
	if got, err := os.ReadFile(installer.configPath); err != nil || string(got) != string(configBefore) {
		t.Fatalf("回滚后配置应恢复: %q, %v", got, err)
	}
	for _, path := range geoPaths {
		if got, err := os.ReadFile(path); err != nil || string(got) != "geo-before" {
			t.Fatalf("回滚后 %s 应恢复: %q, %v", path, got, err)
		}
	}
	want := []host.Action{
		host.ActionStop,
		host.ActionDisable,
		host.ActionDaemonReload,
		host.ActionDaemonReload,
		host.ActionEnable,
		host.ActionStart,
	}
	if !reflect.DeepEqual(service.actions, want) {
		t.Fatalf("恢复动作 = %v，期望 %v", service.actions, want)
	}
}

func TestUninstallStopFailureChangesNothing(t *testing.T) {
	installer, service, binaryPath, unitPath := managedUninstallFixture(t)
	service.actionErrors = map[host.Action][]error{
		host.ActionStop: {errors.New("stop failed")},
	}

	if err := installer.Uninstall(context.Background(), UninstallOptions{}); err == nil || !strings.Contains(err.Error(), "停止 dae") {
		t.Fatalf("应报告停止失败: %v", err)
	}
	for _, path := range []string{binaryPath, unitPath, installer.statePath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("停止失败不应修改 %s: %v", path, err)
		}
	}
	if want := []host.Action{host.ActionStop, host.ActionStart}; !reflect.DeepEqual(service.actions, want) {
		t.Fatalf("systemd 动作 = %v，期望 %v", service.actions, want)
	}
}

func TestUninstallDisableFailureRestoresEnabledAndActiveState(t *testing.T) {
	installer, service, binaryPath, unitPath := managedUninstallFixture(t)
	service.actionErrors = map[host.Action][]error{
		host.ActionDisable: {errors.New("disable partially failed")},
	}

	if err := installer.Uninstall(context.Background(), UninstallOptions{}); err == nil || !strings.Contains(err.Error(), "禁用 dae") {
		t.Fatalf("应报告禁用失败: %v", err)
	}
	for _, path := range []string{binaryPath, unitPath, installer.statePath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("禁用失败不应修改 %s: %v", path, err)
		}
	}
	want := []host.Action{host.ActionStop, host.ActionDisable, host.ActionEnable, host.ActionStart}
	if !reflect.DeepEqual(service.actions, want) {
		t.Fatalf("恢复动作 = %v，期望 %v", service.actions, want)
	}
}
