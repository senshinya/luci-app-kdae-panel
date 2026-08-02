//go:build linux

package netprobe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const icmpAttemptTimeout = time.Second

var icmpSequence atomic.Uint32

// probeICMP 使用 Linux ping socket 测量公网节点主机的网络往返时间。
// ICMP 不属于 dae 接管的 TCP/UDP 流量，因此不会测到当前代理路径产生的假延迟。
// 它只验证主机是否响应 ICMP，不验证节点端口或代理协议是否可用。
func probeICMP(ctx context.Context, address net.IP) (float64, error) {
	network, rawNetwork, listenAddress, protocol := "udp4", "ip4:icmp", "0.0.0.0", 1
	echoRequest, echoReply := icmp.Type(ipv4.ICMPTypeEcho), icmp.Type(ipv4.ICMPTypeEchoReply)
	if address.To4() == nil {
		network, rawNetwork, listenAddress, protocol = "udp6", "ip6:ipv6-icmp", "::", 58
		echoRequest, echoReply = ipv6.ICMPTypeEchoRequest, ipv6.ICMPTypeEchoReply
	}

	connection, err := icmp.ListenPacket(network, listenAddress)
	if err != nil {
		// 部分发行版默认禁止非特权 ping socket。服务单元保留 CAP_NET_RAW，
		// 因而可以回退到 raw ICMP，而不要求用户修改全局 ping_group_range。
		var rawErr error
		connection, rawErr = icmp.ListenPacket(rawNetwork, listenAddress)
		if rawErr != nil {
			return 0, fmt.Errorf("打开 ICMP 探测 Socket: %w", errors.Join(err, rawErr))
		}
		network = rawNetwork
	}
	defer connection.Close()
	stop := context.AfterFunc(ctx, func() { _ = connection.SetDeadline(time.Now()) })
	defer stop()

	var samples []float64
	var lastErr error
	for range probeAttempts {
		sequence := int(icmpSequence.Add(1) & 0xffff)
		message := icmp.Message{
			Type: echoRequest,
			Body: &icmp.Echo{ID: sequence, Seq: sequence, Data: []byte("kdae-panel")},
		}
		payload, marshalErr := message.Marshal(nil)
		if marshalErr != nil {
			return 0, fmt.Errorf("构造 ICMP 请求: %w", marshalErr)
		}
		deadline := time.Now().Add(icmpAttemptTimeout)
		if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
			deadline = contextDeadline
		}
		if err = connection.SetDeadline(deadline); err != nil {
			return 0, fmt.Errorf("设置 ICMP 超时: %w", err)
		}

		startedAt := time.Now()
		var destination net.Addr = &net.UDPAddr{IP: address}
		if network == rawNetwork {
			destination = &net.IPAddr{IP: address}
		}
		if _, err = connection.WriteTo(payload, destination); err != nil {
			lastErr = err
			continue
		}
		buffer := make([]byte, 1500)
		for {
			n, peer, readErr := connection.ReadFrom(buffer)
			if readErr != nil {
				lastErr = readErr
				break
			}
			peerIP := packetIP(peer)
			if peerIP == nil || !peerIP.Equal(address) {
				continue
			}
			reply, parseErr := icmp.ParseMessage(protocol, buffer[:n])
			if parseErr != nil || reply.Type != echoReply {
				continue
			}
			echo, ok := reply.Body.(*icmp.Echo)
			if !ok || echo.Seq != sequence {
				continue
			}
			samples = append(samples, float64(time.Since(startedAt).Microseconds())/1000)
			break
		}
		if ctx.Err() != nil {
			break
		}
	}
	if len(samples) == 0 {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		if lastErr == nil {
			lastErr = errors.New("节点没有响应 ICMP")
		}
		return 0, lastErr
	}
	sort.Float64s(samples)
	return samples[len(samples)/2], nil
}

func packetIP(address net.Addr) net.IP {
	switch typed := address.(type) {
	case *net.IPAddr:
		return typed.IP
	case *net.UDPAddr:
		return typed.IP
	default:
		return nil
	}
}
