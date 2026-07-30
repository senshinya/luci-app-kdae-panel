package geodata

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/tuoro/kdae-panel/internal/atomicfile"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// Names 是 dae 会查找的两个 geo 数据文件。
var Names = []string{upstream.GeoIPName, upstream.GeoSiteName}

// SandboxHiddenDir 是 dae 搜索顺序里面板可能看不到的那一位。
//
// dae 以 root 运行时会读 $HOME/.local/share/dae。systemd 单元设了
// ProtectHome=true 时，/root 被换成一个空且不可访问的目录，面板也没有
// CAP_DAC_OVERRIDE 可以绕；procd 部署则没有这层遮挡。仍然把它列进搜索顺序，
// 是因为 dae 确实读这里。做成变量是给测试留的缝。
var SandboxHiddenDir = "/root/.local/share/dae"

// systemDirs 是 dae 搜索顺序里排在配置目录之后的固定系统目录。
//
// 独立成变量是给测试留的缝：这些绝对路径在跑测试的机器上可能真的存在
// （开发者装过 dae 的 /usr/local/share/dae；Windows 上还会按当前盘符解析），
// 测试必须能把它们清空——否则"就地更新实际生效的那一份"会把开发者机器上
// 真实的 geo 文件当成更新目标，go test 一跑就把它们覆写掉。
var systemDirs = []string{
	SandboxHiddenDir,
	"/usr/local/share/dae",
	"/usr/share/dae",
}

// SearchPath 复刻 dae 查找 geo 数据文件的顺序。
//
// 最高优先级是 DAE_LOCATION_ASSET 指定的目录，其次是配置文件所在目录
// （dae 用 filepath.Dir(cfgFile) 作为 externDirs），之后才轮到那几个系统目录。
// 顺序错了后果很实际：往低优先级目录写的更新永远不会生效，而检查却显示已就位。
//
// environment 是 dae 单元声明的环境变量，可以为 nil。
func SearchPath(configPath string, environment map[string]string) []string {
	paths := []string{}
	if directory := environment[LocationAssetEnv]; directory != "" {
		paths = append(paths, directory)
	}
	if configPath != "" {
		paths = append(paths, filepath.Dir(configPath))
	}
	return append(paths, systemDirs...)
}

// MissingWarning 在面板可见的目录里都找不到 geo 数据时提醒，找得到就返回空。
//
// 必须提醒：dae 只在路由规则用到 geosite/geoip 时才读它们，但一旦用到而文件
// 不在，dae 会直接启动失败，且 dae validate 完全察觉不到——它只读配置文件。
//
// 措辞取决于面板能不能读到 SandboxHiddenDir。读不到时留余地：文件可能就在
// 那里而 dae 读得好好的，说死"未找到"会把一个正常运行的系统报成故障。
// 读得到时就该直说——对着 procd 部署的用户念 ProtectHome 只会让人困惑。
func MissingWarning(searchPath []string) string {
	for _, file := range locate(searchPath, Names) {
		if file.Present {
			continue
		}
		if sandboxHidesHome() {
			return fmt.Sprintf("在面板可见的目录里未找到 geoip.dat / geosite.dat；"+
				"%s 受面板沙箱限制读不到，文件若在那里 dae 仍能读到。"+
				"确实缺失且路由规则用到 geosite/geoip 时，dae 将无法启动", SandboxHiddenDir)
		}
		return "未找到 geoip.dat / geosite.dat；路由规则用到 geosite/geoip 时 dae 将无法启动"
	}
	return ""
}

// sandboxHidesHome 判断 SandboxHiddenDir 是否因为沙箱而对本进程不可见。
//
// 判据是读它的上级目录得到权限错误——ProtectHome=true 正是这个症状。
// 目录不存在不算遮挡：那说明 dae 本来就没在那里放东西。
func sandboxHidesHome() bool {
	_, err := os.ReadDir(filepath.Dir(SandboxHiddenDir))
	return errors.Is(err, fs.ErrPermission)
}

// searchPath 取回本机当前的搜索顺序。
// 读不到服务状态就当没设置环境变量：少一条提示，好过据此把 geo 写到错误的地方。
func (m *Manager) searchPath(ctx context.Context) []string {
	var environment map[string]string
	if m.service != nil {
		if status, err := m.service.Status(ctx); err == nil {
			environment = status.Environment
		}
	}
	return SearchPath(m.configPath, environment)
}

// locate 沿搜索顺序找出每个文件实际生效的那一份，以及被它遮蔽的其余副本。
func locate(searchPath []string, names []string) []File {
	files := make([]File, 0, len(names))
	for _, name := range names {
		file := File{Name: name}
		for _, directory := range searchPath {
			candidate := filepath.Join(directory, name)
			info, err := os.Stat(candidate)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			if !file.Present {
				modTime := info.ModTime().UTC()
				file.Present, file.Path, file.Size, file.ModTime = true, candidate, info.Size(), &modTime
				continue
			}
			// dae 只读优先级最高的那一份，其余的既占磁盘，又会让人以为
			// "我明明更新了却没生效"——必须显式列出来。
			file.Shadowed = append(file.Shadowed, candidate)
		}
		files = append(files, file)
	}
	return files
}

// targetDir 选出本次更新要写入的目录。
//
// 规则是"就地更新实际生效的那一份"，而不是无脑写死某个目录：dae-installer 把
// geo 装在 /usr/local/share/dae，若面板改往配置目录写，会生成一份优先级更高的
// 副本，从此用户跑上游更新脚本将毫无效果且没有任何提示。
//
// 两个文件都不存在时才退回配置目录——它在搜索顺序里优先级最高（仅次于
// DAE_LOCATION_ASSET），且本来就在面板的 ReadWritePaths 里，不必放宽沙箱。
func targetDir(files []File, fallback string) string {
	for _, file := range files {
		if file.Present {
			return filepath.Dir(file.Path)
		}
	}
	return fallback
}

// Status 汇报 geo 数据的现状与可更新性。
func (m *Manager) Status(ctx context.Context) Status {
	search := m.searchPath(ctx)
	files := locate(search, Names)
	target := targetDir(files, filepath.Dir(m.configPath))

	status := Status{
		Sources:       m.fetcher.Sources(),
		DefaultSource: upstream.GeoSourceLoyalsoldier,
		TargetDir:     target,
		SearchPath:    search,
		Files:         files,
	}
	if state, err := m.readState(); err == nil && state != nil {
		status.Managed = state
		// 用过哪个就沿用哪个：换来源会改变 geosite: 规则的含义，
		// 每次都把选择重置回默认值等于诱导用户反复来回切。
		if state.Source != "" {
			status.DefaultSource = state.Source
		}
	}

	if err := atomicfile.Writable(target); err != nil {
		status.Problem = fmt.Sprintf(
			"面板无法写入 %s：%v；请在 kdae-panel.service 的 ReadWritePaths 中加入该目录", target, err)
		return status
	}
	status.Updatable = true
	status.Warnings = warnings(files, target, filepath.Dir(m.configPath))
	return status
}

// warnings 说明那些"更新会成功、但结果可能出乎意料"的情况。
func warnings(files []File, target, configDir string) []string {
	var result []string
	for _, file := range files {
		if len(file.Shadowed) > 0 {
			result = append(result, fmt.Sprintf(
				"%s 同时存在于多个目录，dae 只读 %s；%v 里的副本不会生效，可以删掉",
				file.Name, file.Path, file.Shadowed))
		}
		if file.Present && filepath.Dir(file.Path) != target {
			result = append(result, fmt.Sprintf(
				"%s 目前在 %s，本次更新会写到优先级更高的 %s；此后它以新位置为准",
				file.Name, file.Path, target))
		}
	}
	if target == configDir {
		for _, file := range files {
			if !file.Present {
				result = append(result, fmt.Sprintf(
					"%s 尚未安装，将写入 %s（dae 搜索顺序里优先级最高的可写目录）", file.Name, configDir))
			}
		}
	}
	return result
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
