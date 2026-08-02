package daeconn

import (
	"testing"
	"time"
)

func TestParseConnectionEvent(t *testing.T) {
	timestamp := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	lines := []LogLine{{Timestamp: timestamp, Message: `level=info msg="192.0.2.23:4567 <-> example.com:443" dialer=tokyo ip=203.0.113.8:443 mac=00:11:22:33:44:55 network=tcp4 outbound=proxy pname=curl policy=min sniffed=example.com:443`}}
	events, dropped := Parse(lines)
	if dropped != 0 || len(events) != 1 {
		t.Fatalf("events = %+v, dropped = %d", events, dropped)
	}
	event := events[0]
	if event.Timestamp != timestamp || event.Network != "tcp4" || event.Src != "192.0.2.23:4567" || event.Target != "example.com:443" {
		t.Fatalf("基础字段解析异常: %+v", event)
	}
	if event.DstAddr != "203.0.113.8:443" || event.Dialer != "tokyo" || event.Outbound != "proxy" || event.Sniffed != "example.com:443" {
		t.Fatalf("连接元数据解析异常: %+v", event)
	}
}

func TestParseLocalhostAndMappedIPv4(t *testing.T) {
	lines := []LogLine{{Message: `level=info msg="localhost:1234 <-> target:80" ip="[::ffff:192.0.2.10]:80" network=tcp6 outbound=direct`}}
	events, dropped := Parse(lines)
	if dropped != 0 || len(events) != 1 {
		t.Fatalf("events = %+v, dropped = %d", events, dropped)
	}
	if events[0].Src != "192.0.2.10:1234" || events[0].DstAddr != "192.0.2.10:80" {
		t.Fatalf("localhost 或映射地址未归一化: %+v", events[0])
	}
}

func TestParseSkipsDebugAndCountsMalformedConnectionLines(t *testing.T) {
	lines := []LogLine{
		{Message: `level=debug msg="192.0.2.1:1 <-> example.com:53" network=udp4 outbound=proxy`},
		{Message: `level=info msg="192.0.2.1:1 <-> example.com:443" network=tcp4`},
		{Message: `level=info msg="Successfully created Netkit device pair dae0 <-> dae0-peer"`},
		{Message: `level=info msg="普通日志"`},
	}
	events, dropped := Parse(lines)
	if len(events) != 0 || dropped != 1 {
		t.Fatalf("events = %+v, dropped = %d", events, dropped)
	}
}

func TestParseDoesNotAcceptFieldsInjectedIntoQuotedMessage(t *testing.T) {
	line := LogLine{Message: `level=info msg="192.0.2.1:1 <-> node \" outbound=block:443" network=tcp4 outbound=proxy ip=203.0.113.1:443`}
	events, dropped := Parse([]LogLine{line})
	if dropped != 0 || len(events) != 1 || events[0].Outbound != "proxy" {
		t.Fatalf("引号内字段影响了解析: events=%+v dropped=%d", events, dropped)
	}
}
