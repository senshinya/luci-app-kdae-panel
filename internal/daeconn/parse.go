// Package daeconn 从 dae 的公开输出推导连接活动。
//
// info 日志提供连接建立流水；procfs 只提供 dae 进程当前持有的出站 socket。
// dae 的客户端侧连接留在 eBPF 数据面，没有可用于逐条存活判定的 userspace
// socket，因此本包不猜测每条连接是否仍然存活。
package daeconn

import (
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/tuoro/kdae-panel/internal/logfmt"
)

// LogLine 是连接解析所需的最小日志契约。
type LogLine struct {
	Timestamp time.Time
	Message   string
}

// Event 是 dae 在连接建立时输出的元数据。
type Event struct {
	Timestamp  time.Time `json:"at"`
	Network    string    `json:"network"`
	Src        string    `json:"src"`
	Target     string    `json:"dst"`
	DstAddr    string    `json:"dstAddr,omitempty"`
	Sniffed    string    `json:"sniffed,omitempty"`
	Outbound   string    `json:"outbound,omitempty"`
	Dialer     string    `json:"dialer,omitempty"`
	Policy     string    `json:"policy,omitempty"`
	Pname      string    `json:"pname,omitempty"`
	Mac        string    `json:"mac,omitempty"`
	Offloaded  bool      `json:"offloaded,omitempty"`
	ApproxTime bool      `json:"approxTime,omitempty"`
}

const connectionMarker = " <-> "

// Parse 从日志中筛出 info 级别的连接建立事件。dropped 只统计形似连接事件、
// 但不符合当前格式的行，便于在上游日志格式变化时暴露兼容性问题。
func Parse(lines []LogLine) (events []Event, dropped int) {
	for _, line := range lines {
		fields, ok := logfmt.Parse(line.Message)
		if !ok || !connectionCandidate(fields) {
			continue
		}
		event, outcome := parseEvent(line.Timestamp, fields)
		switch outcome {
		case parseOK:
			events = append(events, event)
		case parseFailed:
			dropped++
		}
	}
	return events, dropped
}

func connectionCandidate(fields map[string]string) bool {
	if !strings.Contains(fields["msg"], connectionMarker) {
		return false
	}
	// kdae 还会用 <-> 描述 Netkit 设备对。只有同时带连接元数据的日志才进入
	// 严格校验，普通接口生命周期日志既不是连接，也不应制造格式变化告警。
	return fields["network"] != "" || fields["outbound"] != "" || fields["dialer"] != "" ||
		fields["ip"] != "" || fields["sniffed"] != ""
}

type parseOutcome uint8

const (
	parseOK parseOutcome = iota
	parseFailed
	parseSkipped
)

func parseEvent(timestamp time.Time, fields map[string]string) (Event, parseOutcome) {
	// dae 会为复用的 UDP 会话输出 debug 行；只接收连接建立时的 info 行，
	// 否则一条 UDP 流会反复制造新记录。
	if fields["level"] != "info" {
		return Event{}, parseSkipped
	}
	network := fields["network"]
	if !validNetwork(network) || fields["outbound"] == "" {
		return Event{}, parseFailed
	}
	source, target, found := strings.Cut(fields["msg"], connectionMarker)
	if !found || strings.TrimSpace(source) == "" || strings.TrimSpace(target) == "" {
		return Event{}, parseFailed
	}

	event := Event{
		Timestamp: timestamp,
		Network:   network,
		Src:       strings.TrimSpace(source),
		Target:    strings.TrimSpace(target),
		DstAddr:   fields["ip"],
		Sniffed:   fields["sniffed"],
		Outbound:  fields["outbound"],
		Dialer:    fields["dialer"],
		Policy:    fields["policy"],
		Pname:     fields["pname"],
		Mac:       fields["mac"],
		Offloaded: fields["ebpf_offload"] == "true",
	}
	destination, destinationValid := parseAddrPort(event.DstAddr)
	if destinationValid {
		event.DstAddr = destination.String()
	}
	if port, localhost := strings.CutPrefix(event.Src, "localhost:"); localhost && destinationValid {
		if parsed, err := strconv.ParseUint(port, 10, 16); err == nil {
			event.Src = netip.AddrPortFrom(destination.Addr(), uint16(parsed)).String()
		}
	} else if address, valid := parseAddrPort(event.Src); valid {
		event.Src = address.String()
	}
	return event, parseOK
}

func (event Event) dedupKey() string {
	return strconv.Quote(event.Network) + strconv.Quote(event.Src) + strconv.Quote(event.Target) +
		strconv.Quote(event.Outbound) + strconv.Quote(event.Dialer) + "|" +
		event.Timestamp.UTC().Format(time.RFC3339Nano)
}

func validNetwork(value string) bool {
	switch value {
	case "tcp4", "tcp6", "udp4", "udp6":
		return true
	default:
		return false
	}
}

func parseAddrPort(value string) (netip.AddrPort, bool) {
	address, err := netip.ParseAddrPort(value)
	if err != nil {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(address.Addr().Unmap(), address.Port()), true
}
