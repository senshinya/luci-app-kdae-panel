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
	Name                string `json:"name"`
	Description         string `json:"description,omitempty"`
	LoadState           string `json:"loadState,omitempty"`
	ActiveState         string `json:"activeState,omitempty"`
	SubState            string `json:"subState,omitempty"`
	UnitFileState       string `json:"unitFileState,omitempty"`
	MainPID             int    `json:"mainPid,omitempty"`
	ExecMainStatus      int    `json:"execMainStatus,omitempty"`
	ActiveSince         string `json:"activeSince,omitempty"`
	StartedAt           string `json:"startedAt,omitempty"`
	MemoryBytes         uint64 `json:"memoryBytes,omitempty"`
	CPUUsageNanoseconds uint64 `json:"cpuUsageNanoseconds,omitempty"`
	Tasks               uint64 `json:"tasks,omitempty"`
	Restarts            uint64 `json:"restarts,omitempty"`
	// UptimeSeconds 是主进程已经跑了多久，只有 procd 后端填。
	//
	// 与 Restarts 正好互补：procd 不暴露重启计数器，systemd 暴露。仪表盘那一格
	// 因此在 procd 上常年空着——不是显示不出来，而是这台机器上真的没有这个数。
	// 主进程的存活时长在 /proc 里是现成的，两个后端各自填自己拿得到的那一个，
	// 比让一格恒为"—"诚实，也比编一个 0 出来安全。
	//
	// 不复用 ActiveSince/StartedAt：那两个是 systemd 原样透传的时间戳字符串
	// （"Wed 2026-07-31 10:00:00 UTC"），往里塞另一种格式，等于让读 API 的人
	// 去猜这个字段今天是哪一套格式。
	UptimeSeconds uint64 `json:"uptimeSeconds,omitempty"`
	UnitPath      string `json:"unitPath,omitempty"`
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
		"CPUUsageNSec",
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
		Name:                valueOr(values, "Id", m.serviceName),
		Description:         values["Description"],
		LoadState:           values["LoadState"],
		ActiveState:         values["ActiveState"],
		SubState:            values["SubState"],
		UnitFileState:       values["UnitFileState"],
		MainPID:             parseInt(values["MainPID"]),
		ExecMainStatus:      parseInt(values["ExecMainStatus"]),
		ActiveSince:         values["ActiveEnterTimestamp"],
		StartedAt:           values["ExecMainStartTimestamp"],
		MemoryBytes:         parseUint(values["MemoryCurrent"]),
		CPUUsageNanoseconds: parseUint(values["CPUUsageNSec"]),
		Tasks:               parseUint(values["TasksCurrent"]),
		Restarts:            parseUint(values["NRestarts"]),
		UnitPath:            values["FragmentPath"],
		ExecStartPath:       parseExecStartPath(values["ExecStart"]),
		Environment:         parseEnvironment(values["Environment"]),
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
	switch action {
	case ActionStart, ActionStop, ActionRestart, ActionEnable, ActionDisable:
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
	result, err := m.runFor(ctx, actionTimeout, m.systemctl, string(action), m.serviceName)
	if err != nil {
		return fmt.Errorf("执行 systemd %s: %s", action, command.Describe(err, result))
	}
	return nil
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
		priority := parsePriority(rawString(raw["PRIORITY"]))
		entries = append(entries, LogEntry{
			Timestamp: journalTimestamp(rawString(raw["__REALTIME_TIMESTAMP"])),
			Priority:  priority,
			Level:     priorityLevel(priority),
			Message:   rawString(raw["MESSAGE"]),
			Unit:      rawString(raw["_SYSTEMD_UNIT"]),
			PID:       rawString(raw["_PID"]),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 journald 输出: %w", err)
	}
	return entries, nil
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
