// Package daeconn 从 dae 的公开输出推导连接活动：info 级别的连接建立日志
// 提供每条连接的元数据（出站、节点、嗅探域名…），/proc/net 里 dae 持有的
// socket 提供它当前扛着多少条出站连接。不读 dae 内部 eBPF Map。
//
// 这里刻意不提供"某条连接现在是否还活着"。dae 的 eBPF 数据面把被代理连接的
// 客户端侧完全留在内核：既没有 userspace socket，也不进 netfilter conntrack，
// 真机验证过没有任何公开接口能逐条判定。日志因此是"活动流水"而不是"当前状态"，
// 界面按这个精度呈现。详见 docs/architecture.md 的观测边界一节。
package daeconn

import (
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/tuoro/kdae-panel/internal/host"
	"github.com/tuoro/kdae-panel/internal/logfmt"
)

// Event 是一条从 dae 日志还原出的连接建立事件。
//
// 字段名与 dae 的 logfmt 键一一对应（control/tcp.go buildTCPLinkLogFields、
// control/udp.go 的日志块）。这不是上游承诺的稳定契约，所以解析失败的行
// 一律静默跳过、只计数，绝不让整页报错。
type Event struct {
	Timestamp time.Time `json:"at"`
	Network   string    `json:"network"`           // tcp4 / tcp6 / udp4 / udp6
	Src       string    `json:"src"`               // 客户端 ip:port（msg 左侧）
	Target    string    `json:"dst"`               // 拨号目标，可能是域名:端口（msg 右侧）
	DstAddr   string    `json:"dstAddr,omitempty"` // 原始目的 ip:port（ip 字段）
	Sniffed   string    `json:"sniffed,omitempty"`
	Outbound  string    `json:"outbound"`
	Dialer    string    `json:"dialer,omitempty"`
	Policy    string    `json:"policy,omitempty"`
	Pname     string    `json:"pname,omitempty"`
	Mac       string    `json:"mac,omitempty"`
	Offloaded bool      `json:"offloaded,omitempty"`
	// ApproxTime 表示 at 是面板观测到该行的时刻，而不是日志自带的时间戳。
	ApproxTime bool `json:"approxTime,omitempty"`

	// srcAddr、dstAddr 是归一化后的地址，仅供去重使用；解析不出时为零值。
	srcAddr netip.AddrPort
	dstAddr netip.AddrPort
}

// connectionMarker 是连接日志 msg 的固定形态 "src <-> target"。
const connectionMarker = " <-> "

// ParseEntries 从服务日志里筛出连接建立事件。dropped 是"看上去像连接日志
// 但没解析出来"的行数——上游改格式时它先变大，页面据此提示而不是装作没数据。
func ParseEntries(entries []host.LogEntry) (events []Event, dropped int) {
	for _, entry := range entries {
		raw := entry.Raw
		if raw == "" {
			raw = entry.Message
		}
		if !strings.Contains(raw, connectionMarker) {
			continue
		}
		event, outcome := parseEvent(entry.Timestamp, raw)
		switch outcome {
		case parseOK:
			events = append(events, event)
		case parseFailed:
			dropped++
		}
		// parseSkipped 不计入 dropped：那是我们主动忽略的行（非 info 级别），
		// 把它算成解析失败会在用户开着 debug 时弹出一个假的"格式已变"警报。
	}
	return events, dropped
}

// parseOutcome 区分"解析失败"与"主动忽略"。只有前者说明我们对格式的理解
// 可能已经过时，值得报给用户。
type parseOutcome int

const (
	parseOK parseOutcome = iota
	parseFailed
	parseSkipped
)

// parseEvent 解析一行原始日志。要求同时具备 network、outbound 字段和
// "src <-> target" 形态的 msg，这个组合足以排除 dae 其他带 "<->" 的输出。
func parseEvent(timestamp time.Time, raw string) (Event, parseOutcome) {
	fields, ok := logfmt.Parse(raw)
	if !ok {
		return Event{}, parseFailed
	}
	// 只认 info：连接建立恰好记在 info，而 debug 会为每个复用的 UDP 会话
	// 再记一行。放行 debug 等于让一条长期存在的 UDP 流每隔几秒就产生一条
	// 新记录，把存储和页面一起淹掉。
	if fields["level"] != "info" {
		return Event{}, parseSkipped
	}
	message, network, outbound := fields["msg"], fields["network"], fields["outbound"]
	if message == "" || outbound == "" || !validNetwork(network) {
		return Event{}, parseFailed
	}
	source, target, found := strings.Cut(message, connectionMarker)
	if !found {
		return Event{}, parseFailed
	}
	event := Event{
		Timestamp: timestamp,
		Network:   network,
		Src:       strings.TrimSpace(source),
		Target:    strings.TrimSpace(target),
		DstAddr:   fields["ip"],
		Sniffed:   fields["sniffed"],
		Outbound:  outbound,
		Dialer:    fields["dialer"],
		Policy:    fields["policy"],
		Pname:     fields["pname"],
		Mac:       fields["mac"],
		Offloaded: fields["ebpf_offload"] == "true",
	}
	if addr, ok := parseAddrPort(event.DstAddr); ok {
		event.dstAddr = addr
	}
	// 本机流量 dae 显示成 "localhost:端口"（源与目的同地址时，见 dae 的
	// RefineSourceToShow）。四元组里的真实源地址就是目的地址本身，可以还原。
	if port, found := strings.CutPrefix(event.Src, "localhost:"); found && event.dstAddr.IsValid() {
		if parsed, err := parsePort(port); err == nil {
			event.srcAddr = netip.AddrPortFrom(event.dstAddr.Addr(), parsed)
			event.Src = event.srcAddr.String()
		}
	} else if addr, ok := parseAddrPort(event.Src); ok {
		event.srcAddr = addr
		event.Src = addr.String()
	}
	if event.dstAddr.IsValid() {
		event.DstAddr = event.dstAddr.String()
	}
	return event, parseOK
}

// dedupKey 是流水去重键。同一四元组可以先后承载多条连接，所以要叠加
// 建立时刻；同一行日志跨两次轮询重复出现时，两个键完全一致。
//
// Src 与 Target 来自日志正文、可以含任意字符，用 Quote 分段拼接，避免
// 内容里的分隔符伪造出与另一条记录相同的键、把它顶掉。
func (e Event) dedupKey() string {
	return strconv.Quote(e.Network) + strconv.Quote(e.Src) + strconv.Quote(e.Target) +
		"|" + e.Timestamp.UTC().Format(time.RFC3339Nano)
}

// validNetwork 限定 dae 的四种取值。它同时是 flowKey 的协议前缀来源，
// 放任意字符串进来等于允许日志内容伪造出新的协议分区。
func validNetwork(value string) bool {
	switch value {
	case "tcp4", "tcp6", "udp4", "udp6":
		return true
	default:
		return false
	}
}

// parseAddrPort 解析 "ip:port"／"[v6]:port"，并把 v4 映射地址归一成纯 v4，
// 与 /proc/net/tcp{,6} 解析侧的归一化保持同构。
func parseAddrPort(value string) (netip.AddrPort, bool) {
	parsed, err := netip.ParseAddrPort(value)
	if err != nil {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(parsed.Addr().Unmap(), parsed.Port()), true
}

func parsePort(value string) (uint16, error) {
	parsed, err := netip.ParseAddrPort("0.0.0.0:" + value)
	if err != nil {
		return 0, err
	}
	return parsed.Port(), nil
}
