package geodata

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/tuoro/kdae-panel/internal/atomicfile"
	"github.com/tuoro/kdae-panel/internal/host"
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
//
// 结果按目录去重。同一个目录出现两次不是"多查一遍"这种无害的事：locate 会在那里
// 命中同一个文件两次，第二次记成被遮蔽的副本，界面于是给出一句自相矛盾、照做还会
// 丢数据的话——dae 只读 /etc/dae/geoip.dat，而 /etc/dae/geoip.dat 里的副本可以删掉。
// procd 部署必然踩中：dae.init 设的 DAE_LOCATION_ASSET 就是 dirname $dae_config。
func SearchPath(configPath string, environment map[string]string) []string {
	candidates := make([]string, 0, len(systemDirs)+2)
	if directory := environment[LocationAssetEnv]; directory != "" {
		candidates = append(candidates, directory)
	}
	if configPath != "" {
		candidates = append(candidates, filepath.Dir(configPath))
	}
	candidates = append(candidates, systemDirs...)

	paths := make([]string, 0, len(candidates))
	for _, directory := range candidates {
		// 环境变量里的值是用户写的，可能带尾斜杠或 ".."，先归一再比。
		directory = filepath.Clean(directory)
		if !slices.Contains(paths, directory) {
			paths = append(paths, directory)
		}
	}
	return paths
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
		if SandboxHidesHome() {
			return fmt.Sprintf("在面板可见的目录里未找到 geoip.dat / geosite.dat；"+
				"%s 受面板沙箱限制读不到，文件若在那里 dae 仍能读到。"+
				"确实缺失且路由规则用到 geosite/geoip 时，dae 将无法启动", SandboxHiddenDir)
		}
		return "未找到 geoip.dat / geosite.dat；路由规则用到 geosite/geoip 时 dae 将无法启动"
	}
	return ""
}

// SandboxHidesHome 判断 SandboxHiddenDir 是否因为沙箱而对本进程不可见。
//
// 判据是读它的上级目录得到权限错误——ProtectHome=true 正是这个症状。
// 目录不存在不算遮挡：那说明 dae 本来就没在那里放东西。
//
// 导出是因为卸载也要用同一个判据：那边要决定这一份 geo 该不该删。同一个问题
// 在两处各答一次，早晚会出现"提示说看不见、卸载却按看得见处理"这种自相矛盾。
func SandboxHidesHome() bool {
	_, err := os.ReadDir(filepath.Dir(SandboxHiddenDir))
	return errors.Is(err, fs.ErrPermission)
}

type serviceSnapshot struct {
	status     host.Status
	state      ServiceState
	problem    string
	inspectErr error
}

// unknownStateProblem 说明状态查询失败为什么足以拦下一次 geo 更新，并按后端给出
// 对得上的理由。两套后端各丢的东西不一样，写成一句通用话会把用户支到错的地方：
//
// systemd 的 Environment 来自 `systemctl show` 读到的单元声明，DAE_LOCATION_ASSET
// 可以是任意目录（dae-installer 装出来的就常指向 /usr/local/share/dae）。读不到它，
// 那个目录压根不在搜索顺序里，targetDir 退回配置目录——优先级更低，更新静默不生效
// 而接口报成功。
//
// procd 上这条不成立：dae.init 与面板 config_load 同一份 UCI，
// DAE_LOCATION_ASSET 恒等于 dirname(dae_config)，也就是搜索顺序里本来就有的那一项，
// 读不到它搜索路径一个字都不差。这里丢的是实例 PID：状态未知会退回不带 PID 的
// `dae reload`，转而依赖 dae 默认的 PID 文件。那意味着先搬几十兆文件、再在 reload
// 上失败、然后整个回滚，不如提前拒绝。
func unknownStateProblem(backend host.Backend, err error) string {
	if backend == host.BackendProcd {
		return fmt.Sprintf(
			"无法确认 dae 服务状态（%v）；拿不到实例 PID 就只能退回不带 PID 的 dae reload，"+
				"更新多半会在重载这一步失败并整个回滚，因此在状态恢复前拒绝更新", err)
	}
	return fmt.Sprintf(
		"无法确认 dae 服务状态（%v）；此时读不到单元里声明的 DAE_LOCATION_ASSET，"+
			"更新可能写进优先级更低的目录而永不生效，因此在状态恢复前拒绝更新", err)
}

// inspectService 同时提供 geo 搜索路径所需的环境变量，以及 reload 所需的 PID。
func (m *Manager) inspectService(ctx context.Context) serviceSnapshot {
	if m.service == nil {
		return serviceSnapshot{state: ServiceStateUnknown}
	}
	status, err := m.service.Status(ctx)
	if err != nil {
		return serviceSnapshot{
			state:      ServiceStateUnknown,
			problem:    unknownStateProblem(m.backend, err),
			inspectErr: err,
		}
	}
	if status.ActiveState == "active" && status.MainPID > 0 {
		return serviceSnapshot{status: status, state: ServiceStateActive}
	}
	if status.ActiveState == "active" {
		return serviceSnapshot{
			status: status,
			state:  ServiceStateUnknown,
			// 上游原文点名 systemd。procd 上同样够得到这条：procdManager.Status
			// 在 ubus 报 running 但没给出 PID 时，正是 active + MainPID==0。
			// 措辞保持后端中性，否则 OpenWrt 用户会去查一个不存在的 systemd。
			problem: "dae 服务显示为 active，但服务管理器没有提供有效 MainPID；更新时将使用 dae 默认的 PID 文件",
		}
	}
	return serviceSnapshot{status: status, state: ServiceStateInactive}
}

// locate 沿搜索顺序找出每个文件实际生效的那一份，以及被它遮蔽的其余副本。
func locate(searchPath []string, names []string) []File {
	files := make([]File, 0, len(names))
	for _, name := range names {
		file := File{Name: name}
		var effective os.FileInfo
		for _, directory := range searchPath {
			candidate := filepath.Join(directory, name)
			info, err := os.Stat(candidate)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			if !file.Present {
				modTime := info.ModTime().UTC()
				file.Present, file.Path, file.Size, file.ModTime = true, candidate, info.Size(), &modTime
				effective = info
				continue
			}
			// 两条路径指向同一个 inode（目录是符号链接，或搜索顺序里重复列了同一个
			// 目录）不算副本。告警的措辞是"可以删掉"，照着删掉的会是唯一生效的那份。
			if os.SameFile(effective, info) {
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
func targetDir(searchPath []string, files []File, fallback string) string {
	for _, directory := range searchPath {
		for _, file := range files {
			if file.Present && filepath.Clean(filepath.Dir(file.Path)) == filepath.Clean(directory) {
				return directory
			}
		}
	}
	return fallback
}

// Status 汇报 geo 数据的现状与可更新性。
func (m *Manager) Status(ctx context.Context) Status {
	return m.status(m.inspectService(ctx))
}

// status 使用一次已经取得的服务快照构造结果。Apply 也复用同一份快照，避免
// 状态查询在“选目标目录”和“决定 reload PID”之间变化，产生 TOCTOU 偏差。
func (m *Manager) status(service serviceSnapshot) Status {
	search := SearchPath(m.configPath, service.status.Environment)
	files := locate(search, Names)
	target := targetDir(search, files, filepath.Dir(m.configPath))

	status := Status{
		Sources:       m.fetcher.Sources(),
		DefaultSource: upstream.GeoSourceLoyalsoldier,
		TargetDir:     target,
		SearchPath:    search,
		Files:         files,
		ServiceState:  service.state,
	}
	if service.problem != "" {
		status.Warnings = append(status.Warnings, service.problem)
	}
	if state, err := m.readState(); err == nil && state != nil {
		status.Managed = state
		// 用过哪个就沿用哪个：换来源会改变 geosite: 规则的含义，
		// 每次都把选择重置回默认值等于诱导用户反复来回切。
		if state.Source != "" {
			status.DefaultSource = state.Source
			if !slices.ContainsFunc(status.Sources, func(info upstream.GeoSourceInfo) bool {
				return info.Source == state.Source
			}) {
				status.Warnings = append(status.Warnings,
					fmt.Sprintf("上次使用的 geo 来源 %s 已不存在；自动更新会保持失败而不会静默切换规则集，请先选择一个现有来源手动更新", state.Source))
			}
		}
	}
	if service.inspectErr != nil {
		status.Problem = service.problem
		return status
	}

	if err := atomicfile.Writable(target); err != nil {
		status.Problem = unwritableProblem(m.backend, target, err)
		return status
	}
	status.Updatable = true
	status.Warnings = append(status.Warnings, warnings(files, target, filepath.Dir(m.configPath))...)
	return status
}

// unwritableProblem 说明目录为什么写不进去，并按后端给出对得上的修法。
//
// 原先无条件让用户去改 kdae-panel.service 的 ReadWritePaths。那个单元在
// OpenWrt 上根本不存在——面板由 procd 从 /etc/init.d/kdae-panel 拉起，
// 没有任何沙箱挡在中间，写不进去就是这个目录本身的权限或挂载有问题。
// 照着一个不存在的文件去排查，用户只会白费一轮时间。
func unwritableProblem(backend host.Backend, target string, err error) string {
	if backend == host.BackendProcd {
		return fmt.Sprintf(
			"面板无法写入 %s：%v；请确认该目录存在、所在分区可写且未被挂载为只读", target, err)
	}
	return fmt.Sprintf(
		"面板无法写入 %s：%v；请在 kdae-panel.service 的 ReadWritePaths 中加入该目录", target, err)
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
