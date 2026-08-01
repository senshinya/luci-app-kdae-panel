package diagnostics

import "github.com/tuoro/kdae-panel/internal/host"

// 诊断中心报的是"你该去查什么、该动哪个文件"，而这两样在两套后端上根本不是同一
// 组东西：systemd 部署查 `systemctl status dae`、看 journald、改 dae.service；
// procd 部署上这三样一样都没有——服务状态来自 ubus，日志来自内核日志缓冲区，
// 服务定义是软件包自带的 /etc/init.d/dae。
//
// 别处的文案说错了顶多让人多看一眼，诊断页说错了是把人直接支到不存在的地方去，
// 而会点开诊断页的用户恰恰是已经出了问题、最缺判断依据的那批。所以这里不写通用
// 措辞（"检查服务管理器"这类话两边都不像人话），而是各后端给各自那一套。
//
// 检查逻辑本身一行不分叉：判定条件、级别、字段全部共用，分叉的只有名字与建议。
type wording struct {
	// stateUnreadableSummary 是状态读不出来时的结论。
	stateUnreadableSummary string
	// stateUnreadableSuggestion 指向本后端真正该确认的那几样东西。
	stateUnreadableSuggestion string
	// unitLabel 是服务定义文件那一行证据的名字。
	unitLabel string
	// brokenPIDSuggestion 是"显示运行中但拿不到主进程号"时的排查入口。
	brokenPIDSuggestion string
	// geoUnwritableSuggestion 说明 geo 目录写不进去该往哪查。systemd 下第一嫌疑
	// 是单元自己的 ReadWritePaths 白名单；procd 下面板以 root 跑且没有沙箱，
	// 那句话在这台机器上无从执行。
	geoUnwritableSuggestion string
	// logsUnreadableSummary 点名读不到的是哪套日志。
	logsUnreadableSummary string
	// missingBTFSuggestion 是缺 BTF 时的出路。通用机器换内核即可；OpenWrt 的内核
	// 由固件决定，BTF 是编译期开关，装什么包都补不出来，只能换固件。
	missingBTFSuggestion string
}

var systemdWording = wording{
	stateUnreadableSummary:    "无法读取 systemd 状态",
	stateUnreadableSuggestion: "确认 systemd 与 dae.service 可用，并检查面板服务权限",
	unitLabel:                 "单元文件",
	brokenPIDSuggestion:       "重启 dae，并检查 systemctl status dae",
	geoUnwritableSuggestion:   "按提示修正目录权限或 kdae-panel.service 的 ReadWritePaths",
	logsUnreadableSummary:     "无法读取 journald 日志",
	missingBTFSuggestion:      "升级或更换带 BTF 的 Linux 内核后再启动 dae",
}

var procdWording = wording{
	stateUnreadableSummary:    "无法读取 procd 服务状态",
	stateUnreadableSuggestion: "确认 ubus 可用、/etc/init.d/dae 存在，并检查面板进程权限",
	unitLabel:                 "init 脚本",
	brokenPIDSuggestion:       `重启 dae，并检查 ubus call service list '{"name":"dae"}' 与 logread -e dae`,
	geoUnwritableSuggestion:   "按提示修正目录权限与可用空间；面板以 root 运行，写不进去通常是只读挂载或空间不足",
	logsUnreadableSummary:     "无法读取 logread 日志",
	missingBTFSuggestion:      "OpenWrt 的 BTF 是内核编译期开关（CONFIG_KERNEL_DEBUG_INFO_BTF），装软件包补不出来，只能换用带 BTF 的固件",
}

// wordingFor 在后端未知时退回 systemd 措辞：那是上游的原生部署，也是所有文案的
// 原始写法。宁可对 procd 用户少说一句，也不要对 systemd 用户改口。
func wordingFor(backend host.Backend) wording {
	if backend == host.BackendProcd {
		return procdWording
	}
	return systemdWording
}
