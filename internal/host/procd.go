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

// clockTicksPerSecond 是 /proc/<pid>/stat 里 utime/stime/starttime 的单位
// USER_HZ。Linux 上恒为 100，即每一跳 10ms；写死比为几个展示用的数字去引
// cgo 调 sysconf 划算。
const (
	clockTicksPerSecond  = 100
	clockTickNanoseconds = 1_000_000_000 / clockTicksPerSecond
)

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
// 只有成功的查询才能证明服务没装、没跑或没有启用。ubus 超时、返回坏 JSON，
// 或 init 脚本执行异常都属于"状态未知"；把它们伪装成 inactive/disabled 会让
// 安装跳过重启、卸载跳过停止，最终在磁盘和运行进程之间制造静默分叉。
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
		unitFileState, err := m.unitFileState(ctx)
		if err != nil {
			return Status{}, err
		}
		status.UnitFileState = unitFileState
	} else if !os.IsNotExist(err) {
		return Status{}, fmt.Errorf("检查 procd init 脚本 %s: %w", script, err)
	}
	instance, found, err := m.instance(ctx)
	if err != nil {
		return Status{}, err
	}
	if found {
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
		status.UptimeSeconds = readUptimeSeconds(status.MainPID)
		status.Environment = readProcessEnvironment(status.MainPID)
	}
	return status, nil
}

// instance 取 procd 记录的第一个实例。本包写出的服务只开一个实例。
func (m *procdManager) instance(ctx context.Context) (ubusInstance, bool, error) {
	result, err := m.run(ctx, "ubus", "call", "service", "list",
		`{"name":"`+m.serviceName+`"}`)
	if err != nil {
		return ubusInstance{}, false, fmt.Errorf("读取 procd 服务状态: %s", command.Describe(err, result))
	}
	var services map[string]ubusService
	if err := json.Unmarshal([]byte(result.Stdout), &services); err != nil {
		return ubusInstance{}, false, fmt.Errorf("解析 procd 状态 JSON: %w", err)
	}
	service, ok := services[m.serviceName]
	if !ok {
		return ubusInstance{}, false, nil
	}
	// map 遍历顺序不定，但本包写出的服务只有一个实例，取到哪个都一样。
	for _, instance := range service.Instances {
		return instance, true, nil
	}
	return ubusInstance{}, false, nil
}

// unitFileState 把 `/etc/init.d/<name> enabled` 的退出码翻译成开机自启状态。
func (m *procdManager) unitFileState(ctx context.Context) (string, error) {
	result, err := m.run(ctx, m.initScript(), "enabled")
	if err == nil {
		return "enabled", nil
	}
	// rc.common 用 1 明确表达“没有 enable”；它是业务状态，不是查询故障。
	if result.ExitCode == 1 {
		return "disabled", nil
	}
	return "", fmt.Errorf("读取 procd 开机自启状态: %s", command.Describe(err, result))
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

// procStatFields 返回 /proc/<pid>/stat 从第 3 个字段（state）起的全部字段。
//
// 必须从最后一个 ')' 之后切，不能对整行做 Fields：第 2 个字段是 comm，内核
// 原样放进圆括号里，可执行文件名带空格时（"my dae"）整行的字段序号会平移，
// 之后每一个按下标取的值都取错。从 ')' 之后切开则与 comm 的内容无关。
//
// 返回切片的下标 = stat 手册里的字段号 - 3。
func procStatFields(pid int) []string {
	content, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		return nil
	}
	end := strings.LastIndex(string(content), ")")
	if end < 0 {
		return nil
	}
	return strings.Fields(string(content)[end+1:])
}

func readCPUNanoseconds(pid int) uint64 {
	fields := procStatFields(pid)
	// utime 是第 14 个字段，stime 是第 15 个。
	if len(fields) < 13 {
		return 0
	}
	utime, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return 0
	}
	stime, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return 0
	}
	return (utime + stime) * clockTickNanoseconds
}

// readUptimeSeconds 算出主进程已经跑了多久。
//
// 用"系统已开机秒数 − 进程启动时刻"，而不是"墙上时钟 − 进程启动时刻"：
// stat 里的 starttime 以开机为原点，换算成绝对时刻还要再引一次 /proc/stat
// 的 btime，而路由器开机后常会被 NTP 校时一次，btime 与那次校时之间的关系
// 并不稳定，算出来的运行时长可能是负的或凭空多出几年。两个量同以开机为原点，
// 相减就与时钟怎么跳完全无关。
func readUptimeSeconds(pid int) uint64 {
	fields := procStatFields(pid)
	// starttime 是第 22 个字段。
	if len(fields) < 20 {
		return 0
	}
	startTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0
	}
	systemUptime, ok := readSystemUptimeSeconds()
	if !ok {
		return 0
	}
	startedAt := startTicks / clockTicksPerSecond
	// 进程比系统还"老"只可能是读到了不一致的快照，此时报 0（界面显示"—"）
	// 比报一个下溢成天文数字的 uint64 强。
	if startedAt > systemUptime {
		return 0
	}
	return systemUptime - startedAt
}

// readSystemUptimeSeconds 读 /proc/uptime 的第一个字段（开机至今的秒数）。
func readSystemUptimeSeconds() (uint64, bool) {
	content, err := os.ReadFile(filepath.Join(procRoot, "uptime"))
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(content))
	if len(fields) == 0 {
		return 0, false
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || seconds < 0 {
		return 0, false
	}
	return uint64(seconds), true
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

// logreadTimestampLayoutsWithYear 是自带年份的时间格式（ubox 的 logread），
// 解析成功即为完整时间，不需要任何补全。
var logreadTimestampLayoutsWithYear = []string{
	"Mon Jan _2 15:04:05 2006",
	"2006-01-02 15:04:05",
}

// logreadTimestampLayoutsWithoutYear 是不带年份的时间格式（busybox 的
// logread）。解析成功后还要靠 timeNow 补年份，见 withInferredYear。
var logreadTimestampLayoutsWithoutYear = []string{
	"Jan _2 15:04:05",
}

// timeNow 是"现在"的取值口，测试覆盖它以获得确定的结果。
var timeNow = time.Now

// parseLogreadTimestamp 解析行首时间戳，解析不出返回零值。
//
// logread 打印的是本机墙上时间，不带时区信息——用 time.ParseInLocation 配
// time.Local 才是对它的正确解释：数字本身就是本地时间，不是 UTC 只是"标签
// 恰好没写"。两组布局解析都统一走本地时区，最后再各自转 UTC 交给上层，
// 与 systemd 后端返回 UTC 的约定一致。
//
// 零值只在"这段前缀本就不像时间戳"时出现——这时候编造一个日期比说"不知道"
// 更容易误导，与 systemd 后端解析失败时的行为一致。但 busybox 格式没有年份，
// 这不属于"解析不出"：月/日/时间是真实信息，直接丢给零值太可惜，因此单独
// 用 withInferredYear 补全，而不是并入失败路径。
func parseLogreadTimestamp(prefix string) time.Time {
	fields := strings.Fields(prefix)
	for count := len(fields); count > 0; count-- {
		candidate := strings.Join(fields[:count], " ")
		for _, layout := range logreadTimestampLayoutsWithYear {
			if parsed, err := time.ParseInLocation(layout, candidate, time.Local); err == nil {
				return parsed.UTC()
			}
		}
		for _, layout := range logreadTimestampLayoutsWithoutYear {
			if parsed, err := time.ParseInLocation(layout, candidate, time.Local); err == nil {
				return withInferredYear(parsed)
			}
		}
	}
	return time.Time{}
}

// withInferredYear 给不带年份的时间戳补年份。
//
// 全程留在本地参照系里比较，出口才转 UTC：parsed 是"贴着本地时区标签的
// 墙上时间"，timeNow() 同样是本地参照系下的一个真实时刻，两者才可比——
// 之前的实现把 parsed 的裸数字硬套上 time.UTC 标签去跟 timeNow().UTC()
// （一次真实的时区换算）比较，在 UTC+8（面向国内用户的部署最常见的时区）
// 上会让 8 小时以内的日志全部被判成"来自未来"进而误回拨一年。
func withInferredYear(parsed time.Time) time.Time {
	now := timeNow()
	guess := dateInYear(now.Year(), parsed)
	// 回拨要求"晚于现在超过 24 小时"而不是"晚于现在"：几秒到几分钟的偏差是
	// 路由器 RTC 与日志时间戳之间常见的时钟误差，为此回拨一整年是灾难性的
	// 误判；真正的跨年场景（元旦读上一年 12 月的日志）与"现在"差着将近
	// 一年，24 小时的宽限足以把两者分开，不会误伤时钟误差。
	if guess.Sub(now) > 24*time.Hour {
		guess = dateInYear(guess.Year()-1, parsed)
	}
	return guess.UTC()
}

// dateInYear 用给定年份重建 parsed 的月/日/时刻，落在 time.Local 下。
//
// 只有 2 月 29 日会让目标年份"没有这一天"——time.Date 对此不报错，而是把
// 溢出静默归一化成 3 月 1 日，月和日一起被改掉，不校验就会把闰年才有的
// 日志显示成 3 月的日志。校验不通过就退到更早的年份重建，直到找到真正
// 拥有这一天的年份；8 次封顶只是给异常输入（parsed 根本不是真实存在过的
// 日期）一个退出条件，正常情况下最近的闰年最多退 4 年就能找到。
func dateInYear(year int, parsed time.Time) time.Time {
	for attempt := 0; attempt < 8; attempt++ {
		guess := time.Date(year-attempt, parsed.Month(), parsed.Day(),
			parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.Local)
		if guess.Month() == parsed.Month() && guess.Day() == parsed.Day() {
			return guess
		}
	}
	return time.Date(year, parsed.Month(), parsed.Day(),
		parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.Local)
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
