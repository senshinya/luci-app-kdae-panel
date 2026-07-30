package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// PanelInitScript 是面板自身的 procd 初始化脚本。
const PanelInitScript = "/etc/init.d/kdae-panel"

func (m *procdManager) Action(ctx context.Context, action Action) error {
	switch action {
	case ActionStart, ActionStop, ActionRestart, ActionEnable, ActionDisable:
	case ActionDaemonReload:
		// procd 每次执行 init 脚本都重新读取服务定义，没有需要"让它重新认识
		// 单元文件"这一步。静默成功而不是报错：首次安装与卸载事务都会调用它。
		return nil
	default:
		return fmt.Errorf("不支持的服务动作 %q", action)
	}
	result, err := m.runFor(ctx, actionTimeout, m.initScript(), string(action))
	if err != nil {
		return fmt.Errorf("执行 %s %s: %s", m.initScript(), action, command.Describe(err, result))
	}
	return nil
}

// RestartSelf 请求 procd 重启面板自身。
//
// setsid 不能省。这条命令是面板的子进程，与面板同属一个会话；procd 停掉面板
// 实例时会把它一并杀掉，于是重启命令先于重启本身死亡，面板停在旧版本上而
// 调用方看到的是"命令被信号终止"。setsid 让它脱离面板的会话与进程组，
// 后台化则让本进程立刻拿回控制权去回复 HTTP 请求。
func (m *procdManager) RestartSelf(ctx context.Context) error {
	script := "setsid " + PanelInitScript + " restart >/dev/null 2>&1 &"
	result, err := m.runFor(ctx, actionTimeout, "/bin/sh", "-c", script)
	if err != nil {
		return fmt.Errorf("请求重启面板: %s", command.Describe(err, result))
	}
	return nil
}

// Logs 读系统日志。本包写出的 init 脚本把服务的 stdout/stderr 交给 procd，
// procd 再转投 syslog，因此 logread 就是全部日志的所在。
func (m *procdManager) Logs(ctx context.Context, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > maxLogLines {
		limit = maxLogLines
	}
	result, err := m.run(ctx, "logread", "-e", m.serviceName)
	if err != nil {
		return nil, fmt.Errorf("读取 logread 日志: %s", command.Describe(err, result))
	}
	entries := parseLogread(result.Stdout, m.serviceName)
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, nil
}

func parseLogread(output, serviceName string) []LogEntry {
	entries := make([]LogEntry, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if entry, ok := parseLogreadLine(line, serviceName); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

// parseLogreadLine 解析一行 logread 输出。两种前缀格式都要兼容：
//
//	Fri Jul 31 01:02:03 2026 daemon.info dae[4321]: level=info msg="…"   (ubox)
//	Jul 31 01:02:03 router dae[7]: 裸消息                                  (busybox)
//
// 唯一可靠的锚点是 "<服务名>["，因此从它切开：之前是时间戳与 facility.level，
// 之后是 pid 和消息体。解析不出的部分一律退化而不丢弃整行——用户看日志
// 正是因为出了问题，这时候少一行比格式难看严重得多。
func parseLogreadLine(line, serviceName string) (LogEntry, bool) {
	tag := serviceName + "["
	index := strings.Index(line, tag)
	if index < 0 {
		return LogEntry{}, false
	}
	entry := LogEntry{Unit: serviceName, Level: "info", Priority: 6}
	prefix := strings.TrimSpace(line[:index])
	entry.Timestamp = parseLogreadTimestamp(prefix)
	if level, ok := logreadLevel(prefix); ok {
		entry.Level = level
		entry.Priority = levelPriority(level)
	}
	pid, message, found := strings.Cut(line[index+len(tag):], "]")
	if !found {
		return entry, true
	}
	entry.PID = pid
	entry.Message = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(message), ":"))
	// dae 自己输出 logfmt。把 level/msg 提到结构化字段上，日志页的级别筛选
	// 才对 dae 的日志同样有效，而不是只对 procd 的封装层有效。
	if level, text, ok := parseLogfmt(entry.Message); ok {
		entry.Level = level
		entry.Priority = levelPriority(level)
		entry.Message = text
	}
	return entry, true
}

// logreadTimestampLayouts 覆盖 ubox 与 busybox 两种 logread 的时间格式。
var logreadTimestampLayouts = []string{
	"Mon Jan _2 15:04:05 2006",
	"2006-01-02 15:04:05",
	"Jan _2 15:04:05",
}

// parseLogreadTimestamp 解析行首时间戳，解析不出返回零值。
//
// 刻意不退回 time.Now()：那会把一批来历不明的旧日志全部标成"刚刚"，
// 比空时间戳更容易误导。零值与 systemd 后端解析失败时的行为一致。
func parseLogreadTimestamp(prefix string) time.Time {
	fields := strings.Fields(prefix)
	for count := len(fields); count > 0; count-- {
		candidate := strings.Join(fields[:count], " ")
		for _, layout := range logreadTimestampLayouts {
			if parsed, err := time.Parse(layout, candidate); err == nil {
				return parsed.UTC()
			}
		}
	}
	return time.Time{}
}

// logreadLevel 从前缀里找 "facility.level" 形式的 token 并取出 level。
func logreadLevel(prefix string) (string, bool) {
	fields := strings.Fields(prefix)
	for index := len(fields) - 1; index >= 0; index-- {
		_, level, found := strings.Cut(fields[index], ".")
		if !found {
			continue
		}
		if levelPriority(level) >= 0 {
			return level, true
		}
	}
	return "", false
}

// parseLogfmt 从 dae 的 `level=… msg="…"` 里取出级别与正文。
// 两者缺一就当作不是 logfmt，保留原始整行。
func parseLogfmt(message string) (string, string, bool) {
	var level, text string
	var foundLevel, foundText bool
	rest := strings.TrimSpace(message)
	for rest != "" {
		name, remainder, found := strings.Cut(rest, "=")
		if !found {
			break
		}
		name = strings.TrimSpace(name)
		if strings.ContainsAny(name, " \t") {
			break
		}
		var value string
		if strings.HasPrefix(remainder, `"`) {
			quoted, tail, closed := strings.Cut(remainder[1:], `"`)
			if !closed {
				break
			}
			value, rest = quoted, strings.TrimSpace(tail)
		} else {
			value, rest, _ = strings.Cut(remainder, " ")
			rest = strings.TrimSpace(rest)
		}
		switch name {
		case "level":
			level, foundLevel = value, true
		case "msg":
			text, foundText = value, true
		}
	}
	if !foundLevel || !foundText {
		return "", "", false
	}
	if levelPriority(level) < 0 {
		return "", "", false
	}
	return level, text, true
}

// levelPriority 把日志级别名映射到 syslog 优先级；未知级别返回 -1。
// 同时收 syslog 的写法（err、crit）与 dae 的写法（error、fatal）。
func levelPriority(level string) int {
	switch level {
	case "emerg":
		return 0
	case "alert":
		return 1
	case "crit", "critical", "fatal":
		return 2
	case "err", "error":
		return 3
	case "warning", "warn":
		return 4
	case "notice":
		return 5
	case "info":
		return 6
	case "debug":
		return 7
	default:
		return -1
	}
}

var _ Manager = (*procdManager)(nil)
