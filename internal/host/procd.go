package host

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tuoro/kdae-panel/internal/command"
)

// initDirectory 是 procd 读取服务定义的目录。做成变量是给测试留的缝。
var initDirectory = "/etc/init.d"

// procRoot 是 /proc 的挂载点，测试指向临时目录。
var procRoot = "/proc"

// clockTickNanoseconds 是 /proc/<pid>/stat 里 utime/stime 的单位。
// Linux 上 USER_HZ 恒为 100，即每一跳 10ms；写死比为一个展示用的数字
// 去引 cgo 调 sysconf 划算。
const clockTickNanoseconds = 10_000_000

// procdManager 通过 procd 的 init 脚本与 ubus 管理服务，适用于 OpenWrt/ImmortalWrt。
type procdManager struct {
	interfaceLister
	serviceName string
	daeBinary   string
	runner      command.Runner
	timeout     time.Duration
}

func newProcdManager(options Options) (*procdManager, error) {
	if options.Runner == nil {
		return nil, errors.New("命令执行器不能为空")
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &procdManager{
		serviceName: options.ServiceName,
		daeBinary:   options.DaeBinary,
		runner:      options.Runner,
		timeout:     timeout,
	}, nil
}

// initScript 是本服务的 procd 定义文件。
func (m *procdManager) initScript() string {
	return filepath.Join(initDirectory, m.serviceName)
}

// ubusInstance 是 `ubus call service list` 里单个实例用得上的字段。
type ubusInstance struct {
	Running bool     `json:"running"`
	PID     int      `json:"pid"`
	Command []string `json:"command"`
}

type ubusService struct {
	Instances map[string]ubusInstance `json:"instances"`
}

// Status 汇报 dae 的运行状况。
//
// 不返回错误是有意的：procd 的状态全部来自本机文件与 ubus，读不到就是"没装/没跑"，
// 不存在 systemd 那种"守护进程抽风导致查询失败"的中间态。把读不到当成错误，
// 会让 daeinstall 的预检永久卡在"无法确认是否已有 dae"。
func (m *procdManager) Status(ctx context.Context) (Status, error) {
	script := m.initScript()
	// Restarts 刻意不填：procd 不暴露重启计数器，填 0 会让 daeinstall 的
	// 崩溃循环检测（"计数没涨就算稳"）静默通过。字段带 omitempty，0 不进 JSON，
	// 仪表盘那一格显示 "—" 而不是 "0"——不知道就该说不知道。
	// 真正的替代信号是 MainPID 变化，见 daeinstall 的重启后观察窗口。
	status := Status{
		Name:        m.serviceName,
		Description: "procd service " + m.serviceName,
		ActiveState: "inactive",
		SubState:    "dead",
		UnitPath:    script,
		LoadState:   "not-found",
	}
	if _, err := os.Stat(script); err == nil {
		status.LoadState = "loaded"
		status.UnitFileState = m.unitFileState(ctx)
	}
	if instance, found := m.instance(ctx); found {
		if instance.Running {
			status.ActiveState = "active"
			status.SubState = "running"
			status.MainPID = instance.PID
		}
		if len(instance.Command) > 0 {
			status.ExecStartPath = instance.Command[0]
		}
	}
	// ExecStartPath 绝不能留空：调用方靠它判断这台机器上有没有 dae，也靠它
	// 决定把新版本写到哪。回退到面板配置的路径是安全的——init 脚本与面板的
	// --dae-binary 读的是同一份 UCI，两者不可能分叉。
	if status.ExecStartPath == "" {
		status.ExecStartPath = m.daeBinary
	}
	if status.MainPID > 0 {
		status.MemoryBytes = readMemoryBytes(status.MainPID)
		status.Tasks = readThreadCount(status.MainPID)
		status.CPUUsageNanoseconds = readCPUNanoseconds(status.MainPID)
		status.Environment = readProcessEnvironment(status.MainPID)
	}
	return status, nil
}

// instance 取 procd 记录的第一个实例。本包写出的服务只开一个实例。
func (m *procdManager) instance(ctx context.Context) (ubusInstance, bool) {
	result, err := m.run(ctx, "ubus", "call", "service", "list",
		`{"name":"`+m.serviceName+`"}`)
	if err != nil {
		return ubusInstance{}, false
	}
	var services map[string]ubusService
	if err := json.Unmarshal([]byte(result.Stdout), &services); err != nil {
		return ubusInstance{}, false
	}
	service, ok := services[m.serviceName]
	if !ok {
		return ubusInstance{}, false
	}
	// map 遍历顺序不定，但本包写出的服务只有一个实例，取到哪个都一样。
	for _, instance := range service.Instances {
		return instance, true
	}
	return ubusInstance{}, false
}

// unitFileState 把 `/etc/init.d/<name> enabled` 的退出码翻译成开机自启状态。
func (m *procdManager) unitFileState(ctx context.Context) string {
	if _, err := m.run(ctx, m.initScript(), "enabled"); err != nil {
		return "disabled"
	}
	return "enabled"
}

func (m *procdManager) run(ctx context.Context, name string, args ...string) (command.Result, error) {
	return m.runFor(ctx, m.timeout, name, args...)
}

func (m *procdManager) runFor(ctx context.Context, timeout time.Duration, name string, args ...string) (command.Result, error) {
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return m.runner.Run(commandCtx, name, args...)
}

func readMemoryBytes(pid int) uint64 {
	value := procStatusField(pid, "VmRSS:")
	value = strings.TrimSpace(strings.TrimSuffix(value, "kB"))
	kilobytes, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return kilobytes * 1024
}

// readThreadCount 供仪表盘的"任务数"格。systemd 那边取自 TasksCurrent，
// procd 没有等价物，但 /proc/<pid>/status 的 Threads 就是同一个意思。
func readThreadCount(pid int) uint64 {
	count, err := strconv.ParseUint(procStatusField(pid, "Threads:"), 10, 64)
	if err != nil {
		return 0
	}
	return count
}

func procStatusField(pid int, prefix string) string {
	content, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "status"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		if value, found := strings.CutPrefix(line, prefix); found {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func readCPUNanoseconds(pid int) uint64 {
	content, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(content))
	if len(fields) < 15 {
		return 0
	}
	utime, err := strconv.ParseUint(fields[13], 10, 64)
	if err != nil {
		return 0
	}
	stime, err := strconv.ParseUint(fields[14], 10, 64)
	if err != nil {
		return 0
	}
	return (utime + stime) * clockTickNanoseconds
}

// readProcessEnvironment 读 dae 进程实际生效的环境变量。
// geo 更新必须知道 DAE_LOCATION_ASSET——它的优先级高于所有默认搜索路径，
// 不读它就可能把新 geo 写到一个根本不生效的目录。
func readProcessEnvironment(pid int) map[string]string {
	content, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "environ"))
	if err != nil || len(content) == 0 {
		return nil
	}
	environment := map[string]string{}
	for _, entry := range strings.Split(string(content), "\x00") {
		name, value, found := strings.Cut(entry, "=")
		if !found || name == "" {
			continue
		}
		environment[name] = value
	}
	if len(environment) == 0 {
		return nil
	}
	return environment
}
