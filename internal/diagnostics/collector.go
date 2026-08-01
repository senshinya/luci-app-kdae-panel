// Package diagnostics 汇总 dae 的公开运行信息，生成可操作的故障诊断报告。
package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tuoro/kdae-panel/internal/configstore"
	"github.com/tuoro/kdae-panel/internal/dae"
	"github.com/tuoro/kdae-panel/internal/geodata"
	"github.com/tuoro/kdae-panel/internal/host"
)

type Level string

const (
	LevelOK      Level = "ok"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
	LevelUnknown Level = "unknown"
)

type Item struct {
	ID         string   `json:"id"`
	Category   string   `json:"category"`
	Title      string   `json:"title"`
	Level      Level    `json:"level"`
	Summary    string   `json:"summary"`
	Details    []string `json:"details,omitempty"`
	Suggestion string   `json:"suggestion,omitempty"`
}

type Counts struct {
	OK      int `json:"ok"`
	Warning int `json:"warning"`
	Error   int `json:"error"`
	Unknown int `json:"unknown"`
}

type Report struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Overall     Level     `json:"overall"`
	Counts      Counts    `json:"counts"`
	Items       []Item    `json:"items"`
}

type DaeInspector interface {
	Inspect(ctx context.Context) dae.Report
}

type Configuration interface {
	Read(ctx context.Context) (configstore.Document, error)
	Validate(ctx context.Context, content string) error
}

type Host interface {
	Status(ctx context.Context) (host.Status, error)
	Logs(ctx context.Context, limit int) ([]host.LogEntry, error)
	Interfaces(ctx context.Context) ([]host.NetworkInterface, error)
}

type Geo interface {
	Status(ctx context.Context) geodata.Status
}

type SystemProbe interface {
	Snapshot(ctx context.Context) SystemSnapshot
}

type Options struct {
	Dae           DaeInspector
	Configuration Configuration
	Host          Host
	Geo           Geo
	System        SystemProbe
	Now           func() time.Time
	// Backend 决定报告里的证据标签与修复建议怎么说。留空按 systemd 措辞，
	// 理由见 wording.go。
	Backend host.Backend
}

type Collector struct {
	dae           DaeInspector
	configuration Configuration
	host          Host
	geo           Geo
	system        SystemProbe
	now           func() time.Time
	words         wording
}

func New(options Options) *Collector {
	system := options.System
	if system == nil {
		system = nativeSystemProbe{}
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Collector{
		dae: options.Dae, configuration: options.Configuration, host: options.Host,
		geo: options.Geo, system: system, now: now, words: wordingFor(options.Backend),
	}
}

func (c *Collector) Report(ctx context.Context) Report {
	service, serviceKnown := c.serviceItem(ctx)
	system := c.system.Snapshot(ctx)
	groups := make([][]Item, 6)
	checks := []func() []Item{
		func() []Item { return []Item{c.daeItem(ctx)} },
		func() []Item { return []Item{c.configurationItem(ctx)} },
		func() []Item { return []Item{c.geoItem(ctx)} },
		func() []Item { return c.networkItems(ctx, system) },
		func() []Item { return c.systemItems(system, serviceKnown) },
		func() []Item { return c.logItem(ctx, serviceKnown) },
	}
	var wait sync.WaitGroup
	for index, check := range checks {
		wait.Add(1)
		go func() {
			defer wait.Done()
			groups[index] = check()
		}()
	}
	wait.Wait()
	items := []Item{service}
	for _, group := range groups {
		items = append(items, group...)
	}

	report := Report{GeneratedAt: c.now().UTC(), Overall: LevelOK, Items: items}
	for _, item := range items {
		switch item.Level {
		case LevelOK:
			report.Counts.OK++
		case LevelWarning:
			report.Counts.Warning++
		case LevelError:
			report.Counts.Error++
		case LevelUnknown:
			report.Counts.Unknown++
		}
		if levelWeight(item.Level) > levelWeight(report.Overall) {
			report.Overall = item.Level
		}
	}
	return report
}

func (c *Collector) serviceItem(ctx context.Context) (Item, *host.Status) {
	item := Item{ID: "service", Category: "服务", Title: "dae 服务"}
	if c.host == nil {
		item.Level, item.Summary = LevelUnknown, "主机服务接口未初始化"
		return item, nil
	}
	status, err := c.host.Status(ctx)
	if err != nil {
		item.Level, item.Summary = LevelUnknown, c.words.stateUnreadableSummary
		item.Details = []string{err.Error()}
		item.Suggestion = c.words.stateUnreadableSuggestion
		return item, nil
	}
	item.Details = appendNonEmpty(nil,
		fmt.Sprintf("状态：%s/%s", status.ActiveState, status.SubState),
		fmt.Sprintf("主进程 PID：%d", status.MainPID),
		fmt.Sprintf("开机状态：%s", status.UnitFileState),
		fmt.Sprintf("%s：%s", c.words.unitLabel, status.UnitPath),
	)
	switch status.ActiveState {
	case "active":
		if status.MainPID <= 0 {
			item.Level, item.Summary = LevelWarning, "服务显示运行中，但主进程号无效"
			item.Suggestion = c.words.brokenPIDSuggestion
		} else if status.UnitFileState == "disabled" {
			item.Level, item.Summary = LevelWarning, "服务正在运行，但没有设为随系统启动"
			item.Suggestion = "在运行概览停止后重新启动一次，面板会同步开机状态"
		} else {
			item.Level, item.Summary = LevelOK, "服务运行正常"
		}
	case "failed":
		item.Level, item.Summary = LevelError, "服务启动失败"
		item.Suggestion = "先处理下面的配置、Geo 与近期日志故障，再重新启动 dae"
	default:
		item.Level = LevelWarning
		item.Summary = fmt.Sprintf("服务当前为 %s/%s", valueOr(status.ActiveState, "未知"), valueOr(status.SubState, "未知"))
		item.Suggestion = "如果需要透明代理，请在运行概览启动 dae"
	}
	return item, &status
}

func (c *Collector) daeItem(ctx context.Context) Item {
	item := Item{ID: "dae-binary", Category: "运行环境", Title: "dae 可执行文件"}
	if c.dae == nil {
		item.Level, item.Summary = LevelUnknown, "dae 探测接口未初始化"
		return item
	}
	report := c.dae.Inspect(ctx)
	item.Details = appendNonEmpty(nil, "路径："+report.Binary, "版本："+report.Version)
	if !report.Available {
		item.Level, item.Summary = LevelError, "dae 无法执行"
		item.Details = appendNonEmpty(item.Details, report.Problem)
		item.Suggestion = "在 dae 版本页安装与当前架构和系统兼容的版本"
		return item
	}
	if report.Problem != "" {
		item.Level, item.Summary = LevelWarning, "dae 可以执行，但部分能力探测失败"
		item.Details = append(item.Details, report.Problem)
		item.Suggestion = "检查 dae 版本是否完整支持当前面板使用的公开命令"
		return item
	}
	item.Level, item.Summary = LevelOK, "dae 可执行文件与公开命令可用"
	return item
}

func (c *Collector) configurationItem(ctx context.Context) Item {
	item := Item{ID: "configuration", Category: "配置", Title: "当前配置"}
	if c.configuration == nil {
		item.Level, item.Summary = LevelUnknown, "配置管理接口未初始化"
		return item
	}
	document, err := c.configuration.Read(ctx)
	if err != nil {
		item.Level = LevelError
		if errors.Is(err, configstore.ErrNotFound) {
			item.Summary = "入口配置不存在"
		} else {
			item.Summary = "无法读取入口配置"
		}
		item.Details = []string{err.Error()}
		item.Suggestion = "在配置管理或代理编排中创建并保存完整配置"
		return item
	}
	item.Details = []string{
		"路径：" + document.Path,
		fmt.Sprintf("大小：%d 字节", document.Size),
		"摘要：" + shortHash(document.Hash),
	}
	if err := c.configuration.Validate(ctx, document.Content); err != nil {
		item.Level, item.Summary = LevelError, "当前配置未通过 dae validate"
		item.Details = append(item.Details, err.Error())
		item.Suggestion = "根据校验错误修改配置；必要时从配置历史恢复已验证的存档"
		return item
	}
	item.Level, item.Summary = LevelOK, "当前配置已通过 dae validate"
	return item
}

func (c *Collector) geoItem(ctx context.Context) Item {
	item := Item{ID: "geo", Category: "数据", Title: "Geo 数据"}
	if c.geo == nil {
		item.Level, item.Summary = LevelUnknown, "Geo 管理接口未初始化"
		return item
	}
	status := c.geo.Status(ctx)
	if status.Problem != "" {
		item.Level, item.Summary = LevelError, "Geo 数据目录不可维护"
		item.Details = appendNonEmpty(status.Warnings, status.Problem)
		item.Suggestion = c.words.geoUnwritableSuggestion
		return item
	}
	if len(status.Files) == 0 {
		item.Level, item.Summary = LevelUnknown, "Geo 服务没有返回文件状态"
		item.Suggestion = "在 Geo 数据页刷新状态；若仍为空，请检查面板启动日志"
		return item
	}
	missing := make([]string, 0, len(status.Files))
	for _, file := range status.Files {
		if file.Present {
			item.Details = append(item.Details, fmt.Sprintf("%s：%s（%d 字节）", file.Name, file.Path, file.Size))
		} else {
			missing = append(missing, file.Name)
		}
	}
	item.Details = append(item.Details, status.Warnings...)
	if len(missing) > 0 {
		item.Level, item.Summary = LevelWarning, "面板可见目录缺少 "+strings.Join(missing, "、")
		item.Suggestion = "如果路由使用 geosite/geoip，请先到 Geo 数据页更新数据"
		return item
	}
	item.Level, item.Summary = LevelOK, "Geo 数据文件已就位"
	return item
}

func (c *Collector) networkItems(ctx context.Context, snapshot SystemSnapshot) []Item {
	interfaces := Item{ID: "interfaces", Category: "网络", Title: "网络接口"}
	if c.host == nil {
		interfaces.Level, interfaces.Summary = LevelUnknown, "主机网络接口不可用"
	} else if values, err := c.host.Interfaces(ctx); err != nil {
		interfaces.Level, interfaces.Summary = LevelUnknown, "无法枚举网络接口"
		interfaces.Details = []string{err.Error()}
	} else {
		for _, value := range values {
			if value.Name == "lo" {
				continue
			}
			interfaces.Details = append(interfaces.Details,
				fmt.Sprintf("%s：%s", value.Name, valueOr(strings.Join(value.Addresses, "、"), "无地址")))
		}
		if len(interfaces.Details) == 0 {
			interfaces.Level, interfaces.Summary = LevelWarning, "没有发现可用的非回环接口"
			interfaces.Suggestion = "检查虚拟机网卡与网络命名空间"
		} else {
			interfaces.Level, interfaces.Summary = LevelOK, fmt.Sprintf("发现 %d 个非回环接口", len(interfaces.Details))
		}
	}

	routes := Item{ID: "default-route", Category: "网络", Title: "默认路由"}
	if snapshot.RouteError != "" {
		routes.Level, routes.Summary = LevelUnknown, "无法读取系统默认路由"
		routes.Details = []string{snapshot.RouteError}
	} else if len(snapshot.DefaultRoutes) == 0 {
		routes.Level, routes.Summary = LevelWarning, "没有发现 IPv4 或 IPv6 默认路由"
		routes.Suggestion = "检查网关、DHCP/静态地址和主机路由表"
	} else {
		routes.Level, routes.Summary = LevelOK, "系统默认路由已就位"
		routes.Details = snapshot.DefaultRoutes
	}
	return []Item{interfaces, routes}
}

func (c *Collector) systemItems(snapshot SystemSnapshot, service *host.Status) []Item {
	kernel := Item{
		ID: "kernel", Category: "运行环境", Title: "Linux 内核",
		Details: appendNonEmpty(nil, "系统："+snapshot.OS, "架构："+snapshot.Architecture, "内核："+snapshot.Kernel),
	}
	if snapshot.KernelError != "" {
		kernel.Level, kernel.Summary = LevelUnknown, "无法完整读取内核信息"
		kernel.Details = append(kernel.Details, snapshot.KernelError)
	} else if snapshot.OS != "linux" {
		kernel.Level, kernel.Summary = LevelError, "dae 透明代理需要 Linux"
		kernel.Suggestion = "请部署到支持 eBPF 的 Linux 主机"
	} else {
		kernel.Level, kernel.Summary = LevelOK, "Linux 内核信息可读"
	}

	bpf := Item{ID: "bpf", Category: "运行环境", Title: "eBPF 基础能力"}
	if service != nil && service.ActiveState == "active" && service.MainPID > 0 {
		bpf.Level, bpf.Summary = LevelOK, "dae 正在运行，eBPF 基础能力已由实际运行验证"
		if !snapshot.BTFPresent || !snapshot.BPFFSMounted {
			bpf.Details = append(bpf.Details, "静态探测存在缺项，但不覆盖正在运行的事实")
		}
	} else if snapshot.BTFPresent && snapshot.BPFFSMounted {
		bpf.Level, bpf.Summary = LevelOK, "内核 BTF 与 bpffs 已就位"
	} else if snapshot.BTFPresent {
		bpf.Level, bpf.Summary = LevelWarning, "内核 BTF 已就位，但 /sys/fs/bpf 未挂载 bpffs"
		bpf.Suggestion = "若 dae 启动报 BPF 文件系统错误，再挂载 bpffs；不要仅凭此项判定 dae 必然不可用"
	} else {
		bpf.Level = LevelError
		bpf.Summary = "缺少 /sys/kernel/btf/vmlinux"
		bpf.Suggestion = c.words.missingBTFSuggestion
	}
	bpf.Details = append(bpf.Details, snapshot.BPFErrors...)
	return []Item{kernel, bpf}
}

func (c *Collector) logItem(ctx context.Context, status *host.Status) []Item {
	item := Item{ID: "recent-logs", Category: "日志", Title: "近期异常日志"}
	if c.host == nil {
		item.Level, item.Summary = LevelUnknown, "日志接口未初始化"
		return []Item{item}
	}
	entries, err := c.host.Logs(ctx, 200)
	if err != nil {
		item.Level, item.Summary = LevelUnknown, c.words.logsUnreadableSummary
		item.Details = []string{err.Error()}
		return []Item{item}
	}
	cutoff := c.now().Add(-30 * time.Minute)
	if status != nil {
		if started, err := time.Parse(time.RFC3339, status.StartedAt); err == nil && started.After(cutoff) {
			cutoff = started
		}
	}
	var abnormal []host.LogEntry
	for _, entry := range entries {
		if !entry.Timestamp.IsZero() && entry.Timestamp.Before(cutoff) {
			continue
		}
		if entry.Level == "warning" && routineDaeWarning(entry.Message) {
			continue
		}
		if entry.Level == "critical" || entry.Level == "error" || entry.Level == "warning" {
			abnormal = append(abnormal, entry)
		}
	}
	if len(abnormal) == 0 {
		item.Level, item.Summary = LevelOK, "当前服务周期或最近 30 分钟没有警告与错误"
		return []Item{item}
	}
	sort.Slice(abnormal, func(left, right int) bool { return abnormal[left].Timestamp.After(abnormal[right].Timestamp) })
	errorsSeen := 0
	for index, entry := range abnormal {
		if entry.Level == "critical" || entry.Level == "error" {
			errorsSeen++
		}
		if index < 12 {
			item.Details = append(item.Details, fmt.Sprintf("%s [%s] %s",
				entry.Timestamp.Local().Format("2006-01-02 15:04:05"), entry.Level, limitText(entry.Message, 500)))
		}
	}
	if errorsSeen > 0 {
		item.Level, item.Summary = LevelError, fmt.Sprintf("发现 %d 条错误、%d 条警告", errorsSeen, len(abnormal)-errorsSeen)
	} else {
		item.Level, item.Summary = LevelWarning, fmt.Sprintf("发现 %d 条警告", len(abnormal))
	}
	item.Suggestion = "在运行日志页查看完整上下文；优先处理时间最新的错误"
	return []Item{item}
}

func routineDaeWarning(message string) bool {
	for _, fragment := range []string{
		"[Reload] Received reload signal",
		"[Reload] Load new config",
		"[Reload] Prepare staged same-port handoff",
		"[Reload] Serve",
		"[Reload] Retiring old control plane",
		"[Reload] Finished",
		"[Reload] Retired old control plane",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func levelWeight(level Level) int {
	switch level {
	case LevelError:
		return 3
	case LevelWarning:
		return 2
	case LevelUnknown:
		return 1
	default:
		return 0
	}
}

func appendNonEmpty(target []string, values ...string) []string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" && !strings.HasSuffix(value, "：") {
			target = append(target, value)
		}
	}
	return target
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func shortHash(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func limitText(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}
