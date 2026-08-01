package host

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tuoro/kdae-panel/internal/command"
)

const (
	defaultTimeout = 30 * time.Second
	actionTimeout  = 150 * time.Second
	maxLogLines    = 500
)

type systemdManager struct {
	interfaceLister
	serviceName string
	systemctl   string
	journalctl  string
	runner      command.Runner
	timeout     time.Duration
}

type Status struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	LoadState      string `json:"loadState,omitempty"`
	ActiveState    string `json:"activeState,omitempty"`
	SubState       string `json:"subState,omitempty"`
	UnitFileState  string `json:"unitFileState,omitempty"`
	MainPID        int    `json:"mainPid,omitempty"`
	ExecMainStatus int    `json:"execMainStatus,omitempty"`
	// ActiveSince、StartedAt 一律是 RFC 3339。两个后端拿到的原始格式完全不同
	// （systemd 是 "Mon 2026-08-01 10:00:00 CST"，procd 只有主进程已运行的秒数），
	// 各自归一到同一种格式后，前端才能对两个后端用同一段解析代码。
	ActiveSince string `json:"activeSince,omitempty"`
	StartedAt   string `json:"startedAt,omitempty"`
	MemoryBytes uint64 `json:"memoryBytes,omitempty"`
	Tasks       uint64 `json:"tasks,omitempty"`
	// Restarts 只有 systemd 填：procd 不暴露重启计数器。字段带 omitempty，
	// procd 上不进 JSON，读到 0 的地方要当"不知道"而不是"没重启过"。
	Restarts uint64 `json:"restarts,omitempty"`
	UnitPath string `json:"unitPath,omitempty"`
	// ExecStartPath 是单元实际启动的可执行文件。安装新版本时必须替换这个路径，
	// 否则会出现"替换成功但服务仍在跑旧二进制"的静默失败。
	ExecStartPath string `json:"execStartPath,omitempty"`
	// Environment 是单元里声明的环境变量。dae 用 DAE_LOCATION_ASSET 指定
	// geo 数据目录，它的优先级高于所有默认搜索路径；不读它就无法确定
	// dae 究竟从哪里读 geo，更新会写到一个根本不生效的地方。
	Environment map[string]string `json:"-"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Priority  int       `json:"priority"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Unit      string    `json:"unit,omitempty"`
	PID       string    `json:"pid,omitempty"`
}

type Action string

const (
	ActionStart   Action = "start"
	ActionStop    Action = "stop"
	ActionRestart Action = "restart"
	// Enable/Disable 只供 dae 卸载事务维护原有的开机启动状态；
	// 对外的服务控制 API 仍只开放 start/stop/restart。
	ActionEnable  Action = "enable"
	ActionDisable Action = "disable"
	// EnableNow/DisableNow 只用于用户主动控制服务：把当前运行状态同步成
	// systemd 的开机状态。安装与版本切换仍使用普通 start/stop，避免改写原策略。
	ActionEnableNow  Action = "enable-now"
	ActionDisableNow Action = "disable-now"
	// ActionDaemonReload 让 systemd 重新读取单元文件。首次安装写入 dae.service
	// 之后必须执行它，否则 systemd 看不到新单元。它不作用于具体服务，
	// 因此不接受服务名参数。
	ActionDaemonReload Action = "daemon-reload"
)

func (m *systemdManager) Status(ctx context.Context) (Status, error) {
	properties := strings.Join([]string{
		"Id",
		"Description",
		"LoadState",
		"ActiveState",
		"SubState",
		"UnitFileState",
		"MainPID",
		"ExecMainStatus",
		"ActiveEnterTimestamp",
		"ExecMainStartTimestamp",
		"MemoryCurrent",
		"TasksCurrent",
		"NRestarts",
		"FragmentPath",
		"ExecStart",
		"Environment",
	}, ",")
	result, err := m.run(ctx, m.systemctl, "show", m.serviceName, "--no-page", "--property="+properties)
	if err != nil {
		return Status{}, fmt.Errorf("读取 systemd 服务状态: %s", command.Describe(err, result))
	}
	values := parseProperties(result.Stdout)
	status := Status{
		Name:           valueOr(values, "Id", m.serviceName),
		Description:    values["Description"],
		LoadState:      values["LoadState"],
		ActiveState:    values["ActiveState"],
		SubState:       values["SubState"],
		UnitFileState:  values["UnitFileState"],
		MainPID:        parseInt(values["MainPID"]),
		ExecMainStatus: parseInt(values["ExecMainStatus"]),
		ActiveSince:    normalizeSystemdTimestamp(values["ActiveEnterTimestamp"]),
		StartedAt:      normalizeSystemdTimestamp(values["ExecMainStartTimestamp"]),
		MemoryBytes:    parseUint(values["MemoryCurrent"]),
		Tasks:          parseUint(values["TasksCurrent"]),
		Restarts:       parseUint(values["NRestarts"]),
		UnitPath:       values["FragmentPath"],
		ExecStartPath:  parseExecStartPath(values["ExecStart"]),
		Environment:    parseEnvironment(values["Environment"]),
	}
	return status, nil
}

// parseEnvironment 解析 systemd 的 Environment 属性。
// 形如：FOO=1 DAE_LOCATION_ASSET=/etc/dae BAR=2
//
// systemd 对含空格的值会加引号，这里只做朴素的空格切分：面板唯一关心的
// DAE_LOCATION_ASSET 是一个目录路径，带空格的目录本就极罕见，宁可解析不出
// 也不去猜——解析不出的后果是少一条提示，猜错的后果是把 geo 写到错误的地方。
func parseEnvironment(value string) map[string]string {
	if value == "" {
		return nil
	}
	environment := map[string]string{}
	for _, entry := range strings.Fields(value) {
		name, content, found := strings.Cut(entry, "=")
		if !found || name == "" {
			continue
		}
		environment[name] = content
	}
	if len(environment) == 0 {
		return nil
	}
	return environment
}

// parseExecStartPath 从 systemd 的 ExecStart 属性里取出可执行文件路径。
// 形如：{ path=/usr/local/bin/dae ; argv[]=/usr/local/bin/dae run ; ... }
func parseExecStartPath(value string) string {
	_, rest, found := strings.Cut(value, "path=")
	if !found {
		return ""
	}
	path, _, _ := strings.Cut(rest, ";")
	return strings.TrimSpace(path)
}

func (m *systemdManager) Action(ctx context.Context, action Action) error {
	var args []string
	switch action {
	case ActionStart, ActionStop, ActionRestart, ActionEnable, ActionDisable:
		args = []string{string(action), m.serviceName}
	case ActionEnableNow:
		args = []string{"enable", "--now", m.serviceName}
	case ActionDisableNow:
		args = []string{"disable", "--now", m.serviceName}
	case ActionDaemonReload:
		// daemon-reload 是全局动作，不带服务名。
		result, err := m.runFor(ctx, actionTimeout, m.systemctl, "daemon-reload")
		if err != nil {
			return fmt.Errorf("执行 systemd daemon-reload: %s", command.Describe(err, result))
		}
		return nil
	default:
		return fmt.Errorf("不支持的服务动作 %q", action)
	}
	result, err := m.runFor(ctx, actionTimeout, m.systemctl, args...)
	if err != nil {
		return fmt.Errorf("执行 systemd %s: %s", strings.Join(args[:len(args)-1], " "), command.Describe(err, result))
	}
	return nil
}

// normalizeSystemdTimestamp 把 systemd 的本地化时间转成浏览器可可靠解析的 RFC 3339。
// 直接把 "CST" 交给浏览器会产生歧义：它既可能指中国标准时间，也可能指北美中部时间。
func normalizeSystemdTimestamp(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "n/a" {
		return ""
	}
	for _, layout := range []string{
		"Mon 2006-01-02 15:04:05 -07",
		"Mon 2006-01-02 15:04:05 -0700",
		"Mon 2006-01-02 15:04:05 -07:00",
	} {
		// MST 布局也会接受 "+08"，但会把它当作未知名称并赋予零偏移。
		// 数字时区必须先按偏移解析，结果才不依赖宿主机的本地时区。
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Format(time.RFC3339)
		}
	}
	parsed, err := time.ParseInLocation("Mon 2006-01-02 15:04:05 MST", value, time.Local)
	if err == nil {
		return parsed.Format(time.RFC3339)
	}
	return ""
}

// PanelUnit 是面板自身的 systemd 单元名。
const PanelUnit = "kdae-panel.service"

// RestartSelf 请求 systemd 重启面板自身的单元。
//
// 必须带 --no-block：不带的话 systemctl 会等重启完成，而它是面板的子进程、
// 与面板同属一个 cgroup——systemd 停止该单元时会把这个 systemctl 一起杀掉，
// 于是调用方看到的是"命令被信号终止"，而重启其实已经在进行。
// --no-block 把作业排进队列后立即返回，重启的实际发生与本进程的死亡解耦。
//
// 刻意不复用 Action：那条路径操作的是 dae 的单元，而这里的目标固定是面板
// 自己；把两者混在一个入口里，只要服务名参数传错一次，就会变成"想升级面板
// 却重启了 dae"。
func (m *systemdManager) RestartSelf(ctx context.Context) error {
	result, err := m.runFor(ctx, actionTimeout, m.systemctl, "restart", "--no-block", PanelUnit)
	if err != nil {
		return fmt.Errorf("请求重启 %s: %s", PanelUnit, command.Describe(err, result))
	}
	return nil
}

func (m *systemdManager) Logs(ctx context.Context, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > maxLogLines {
		limit = maxLogLines
	}
	result, err := m.run(
		ctx,
		m.journalctl,
		"--unit", m.serviceName,
		"--no-pager",
		"--output", "json",
		"--lines", strconv.Itoa(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("读取 journald 日志: %s", command.Describe(err, result))
	}
	return parseJournal(result.Stdout)
}

func (m *systemdManager) run(ctx context.Context, name string, args ...string) (command.Result, error) {
	return m.runFor(ctx, m.timeout, name, args...)
}

func (m *systemdManager) runFor(ctx context.Context, timeout time.Duration, name string, args ...string) (command.Result, error) {
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return m.runner.Run(commandCtx, name, args...)
}

func parseProperties(output string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if found {
			values[key] = value
		}
	}
	return values
}

func parseJournal(output string) ([]LogEntry, error) {
	entries := make([]LogEntry, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("解析 journald JSON: %w", err)
		}
		message := rawString(raw["MESSAGE"])
		priority, level := logLevel(message, parsePriority(rawString(raw["PRIORITY"])))
		entries = append(entries, LogEntry{
			Timestamp: journalTimestamp(rawString(raw["__REALTIME_TIMESTAMP"])),
			Priority:  priority,
			Level:     level,
			Message:   message,
			Unit:      rawString(raw["_SYSTEMD_UNIT"]),
			PID:       rawString(raw["_PID"]),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 journald 输出: %w", err)
	}
	return entries, nil
}

// logLevel 优先采用 dae 正文开头的 level 字段。
// dae 把日志写到标准输出时，journald 通常会把所有行都记成 info；只看 PRIORITY
// 会把真实的 debug / warning / error 全部误判，前端按级别筛选也就失去意义。
func logLevel(message string, journalPriority int) (int, string) {
	const prefix = "level="
	trimmed := strings.TrimSpace(message)
	if strings.HasPrefix(trimmed, prefix) {
		fields := strings.Fields(trimmed[len(prefix):])
		if len(fields) == 0 {
			return journalPriority, priorityLevel(journalPriority)
		}
		value := strings.Trim(fields[0], `"`)
		if priority, level, ok := canonicalLevel(value); ok {
			return priority, level
		}
	}
	return journalPriority, priorityLevel(journalPriority)
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	var bytesValue []byte
	if err := json.Unmarshal(raw, &bytesValue); err == nil {
		return string(bytesValue)
	}
	return ""
}

func journalTimestamp(value string) time.Time {
	microseconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.UnixMicro(microseconds).UTC()
}

func priorityLevel(priority int) string {
	levels := []string{"emerg", "alert", "critical", "error", "warning", "notice", "info", "debug"}
	if priority < 0 || priority >= len(levels) {
		return "unknown"
	}
	return levels[priority]
}

func parsePriority(value string) int {
	priority, err := strconv.Atoi(value)
	if err != nil || priority < 0 || priority > 7 {
		return -1
	}
	return priority
}

func validUnitName(value string) bool {
	if len(value) > 255 || strings.HasPrefix(value, "-") {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("_.@:-", char) {
			continue
		}
		return false
	}
	return value != ""
}

func valueOr(values map[string]string, key, fallback string) string {
	if value := values[key]; value != "" {
		return value
	}
	return fallback
}

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(value)
	return parsed
}

func parseUint(value string) uint64 {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return parsed
}

var _ Manager = (*systemdManager)(nil)
