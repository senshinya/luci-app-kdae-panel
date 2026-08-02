package netprobe

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestProbeMeasuresReachableTarget(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	prober := newWithLimits((&net.Dialer{}).DialContext, defaultTimeout, defaultConcurrency)
	results, err := prober.Probe(context.Background(), []Target{{Host: "127.0.0.1", Port: port}})
	if err != nil {
		t.Fatalf("探测失败: %v", err)
	}
	if len(results) != 1 || !results[0].Reachable || results[0].LatencyMs <= 0 {
		t.Fatalf("探测结果异常: %+v", results)
	}
}

func TestProbeReportsUnreachableTarget(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	prober := newWithLimits((&net.Dialer{}).DialContext, defaultTimeout, defaultConcurrency)
	results, err := prober.Probe(context.Background(), []Target{{Host: "127.0.0.1", Port: port}})
	if err != nil {
		t.Fatalf("探测失败: %v", err)
	}
	if len(results) != 1 || results[0].Reachable || results[0].Error == "" {
		t.Fatalf("探测结果异常: %+v", results)
	}
}

func TestProbeReportsTimeout(t *testing.T) {
	prober := newWithLimits(func(ctx context.Context, _, _ string) (net.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}, 10*time.Millisecond, defaultConcurrency)
	results, err := prober.Probe(context.Background(), []Target{{Host: "10.0.0.1", Port: 443}})
	if err != nil {
		t.Fatalf("探测失败: %v", err)
	}
	if results[0].Reachable || results[0].Error != "连接超时" {
		t.Fatalf("超时结果异常: %+v", results[0])
	}
}

func TestProbeRejectsMalformedBatch(t *testing.T) {
	prober := New()
	cases := []struct {
		name    string
		targets []Target
		wantErr string
	}{
		{"空目标", nil, "不能为空"},
		{"过多目标", make([]Target, MaxTargets+1), "上限"},
	}
	for _, testCase := range cases {
		if _, err := prober.Probe(context.Background(), testCase.targets); err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
			t.Fatalf("%s: 期望包含 %q 的错误, 得到 %v", testCase.name, testCase.wantErr, err)
		}
	}
}

func TestProbeIsolatesInvalidTargets(t *testing.T) {
	var dialed atomic.Int64
	prober := newWithLimits(func(_ context.Context, _, _ string) (net.Conn, error) {
		dialed.Add(1)
		return nil, errRefused("refused")
	}, time.Second, defaultConcurrency)

	invalid := []Target{
		{Host: "", Port: 443},
		{Host: "a b", Port: 443},
		{Host: "example.com", Port: 0},
		{Host: "example.com", Port: 70000},
		{Host: strings.Repeat("a", 254), Port: 443},
		{Host: " example.com", Port: 443},
	}
	targets := append([]Target{{Host: "10.0.0.1", Port: 443}}, invalid...)
	results, err := prober.Probe(context.Background(), targets)
	if err != nil {
		t.Fatalf("单个非法目标不应让整批失败: %v", err)
	}
	if len(results) != len(targets) {
		t.Fatalf("结果数量 = %d，期望 %d", len(results), len(targets))
	}
	if results[0].Error == "" || results[0].Reachable {
		t.Fatalf("合法目标仍应被探测并返回结果: %+v", results[0])
	}
	for index, result := range results[1:] {
		if result.Reachable || result.Error == "" {
			t.Fatalf("非法目标 %d 应带错误且不可达: %+v", index, result)
		}
	}
	if dialed.Load() != 1 {
		t.Fatalf("只应对合法目标发起拨号，实际 %d 次", dialed.Load())
	}
}

func TestProbeExcludesResolutionTimeAndSamplesThreeTimes(t *testing.T) {
	var dialed atomic.Int64
	prober := newWithLimits(func(_ context.Context, _, _ string) (net.Conn, error) {
		dialed.Add(1)
		client, server := net.Pipe()
		_ = server.Close()
		return client, nil
	}, time.Second, defaultConcurrency)
	prober.resolve = func(_ context.Context, _ string) ([]net.IPAddr, error) {
		time.Sleep(100 * time.Millisecond)
		return []net.IPAddr{{IP: net.ParseIP("10.0.0.1")}}, nil
	}

	results, err := prober.Probe(context.Background(), []Target{{Host: "node.example", Port: 443}})
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].Reachable || results[0].LatencyMs >= 50 {
		t.Fatalf("解析时间不应计入入口延迟: %+v", results[0])
	}
	if dialed.Load() != probeAttempts {
		t.Fatalf("成功节点拨号次数 = %d,期望 %d", dialed.Load(), probeAttempts)
	}
	if results[0].Method != "tcp" || results[0].ResolvedIP != "10.0.0.1" {
		t.Fatalf("TCP 探测元数据异常: %+v", results[0])
	}
}

func TestProbeUsesICMPForEveryPublicAddress(t *testing.T) {
	prober := newWithLimits(func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("公网节点不应先发起可能被 dae 接管的 TCP 探测")
		return nil, nil
	}, time.Second, defaultConcurrency)
	prober.resolve = staticResolution("203.0.113.8")
	prober.ping = func(_ context.Context, address net.IP) (float64, error) {
		if address.String() != "203.0.113.8" {
			t.Fatalf("ICMP 探测地址 = %s", address)
		}
		return 131.4, nil
	}

	results, err := prober.Probe(context.Background(), []Target{{Host: "node.example", Port: 443}})
	if err != nil {
		t.Fatal(err)
	}
	result := results[0]
	if !result.Reachable || result.Method != "icmp" || result.LatencyMs != 131.4 {
		t.Fatalf("公网节点未使用 ICMP: %+v", result)
	}
}

func TestProbeDoesNotFallBackToTCPWhenPublicICMPUnavailable(t *testing.T) {
	prober := newWithLimits(func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("ICMP 失败后不应回退到可能被 dae 接管的 TCP")
		return nil, nil
	}, time.Second, defaultConcurrency)
	prober.resolve = staticResolution("203.0.113.9")
	prober.ping = func(context.Context, net.IP) (float64, error) {
		return 0, errors.New("permission denied")
	}

	results, err := prober.Probe(context.Background(), []Target{{Host: "node.example", Port: 443}})
	if err != nil {
		t.Fatal(err)
	}
	result := results[0]
	if result.Reachable || result.LatencyMs != 0 || !strings.Contains(result.Error, "无法得到可信的网络延迟") {
		t.Fatalf("ICMP 探测失败后仍返回了假延迟: %+v", result)
	}
}

func TestProbeKeepsLegitimatePrivateSubMillisecondLatency(t *testing.T) {
	prober := newWithLimits(immediateConnection, time.Second, defaultConcurrency)
	prober.resolve = staticResolution("192.168.31.2")
	prober.ping = func(context.Context, net.IP) (float64, error) {
		t.Fatal("内网亚毫秒延迟不应触发 ICMP 探测")
		return 0, nil
	}

	results, err := prober.Probe(context.Background(), []Target{{Host: "lan-node", Port: 443}})
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].Reachable || results[0].Method != "tcp" {
		t.Fatalf("内网延迟被误判: %+v", results[0])
	}
}

func TestProbeRunsTargetsConcurrently(t *testing.T) {
	prober := newWithLimits(func(_ context.Context, _, address string) (net.Conn, error) {
		time.Sleep(50 * time.Millisecond)
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errRefused(address)}
	}, time.Second, defaultConcurrency)
	targets := make([]Target, MaxTargets)
	for index := range targets {
		targets[index] = Target{Host: "10.0.0." + strconv.Itoa(index+1), Port: 443}
	}
	startedAt := time.Now()
	results, err := prober.Probe(context.Background(), targets)
	if err != nil {
		t.Fatalf("探测失败: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 2*time.Second {
		t.Fatalf("并发探测耗时过长: %v", elapsed)
	}
	for _, result := range results {
		if result.Reachable || result.Error == "" {
			t.Fatalf("探测结果异常: %+v", result)
		}
	}
}

func TestProbeSharesConcurrencyLimitAcrossCalls(t *testing.T) {
	var inFlight, peak atomic.Int64
	prober := newWithLimits(func(_ context.Context, _, _ string) (net.Conn, error) {
		current := inFlight.Add(1)
		for {
			observed := peak.Load()
			if current <= observed || peak.CompareAndSwap(observed, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		inFlight.Add(-1)
		return nil, errRefused("refused")
	}, time.Second, 4)

	targets := make([]Target, 8)
	for index := range targets {
		targets[index] = Target{Host: "10.0.1." + strconv.Itoa(index+1), Port: 443}
	}
	done := make(chan struct{}, 3)
	for range 3 {
		go func() {
			_, _ = prober.Probe(context.Background(), targets)
			done <- struct{}{}
		}()
	}
	for range 3 {
		<-done
	}
	if peak.Load() > 4 {
		t.Fatalf("并发拨号峰值 = %d，超过实例上限 4", peak.Load())
	}
}

func TestProbeStopsWhenContextCancelled(t *testing.T) {
	release := make(chan struct{})
	prober := newWithLimits(func(ctx context.Context, _, _ string) (net.Conn, error) {
		<-release
		return nil, ctx.Err()
	}, time.Minute, 1)

	ctx, cancel := context.WithCancel(context.Background())
	targets := make([]Target, 8)
	for index := range targets {
		targets[index] = Target{Host: "10.0.2." + strconv.Itoa(index+1), Port: 443}
	}
	finished := make(chan []Result, 1)
	go func() {
		results, _ := prober.Probe(ctx, targets)
		finished <- results
	}()

	cancel()
	close(release)
	select {
	case results := <-finished:
		for _, result := range results {
			if result.Reachable {
				t.Fatalf("取消后不应有可达结果: %+v", result)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("取消后 Probe 未及时返回，可能存在 goroutine 泄漏")
	}
}

func TestProberFallsBackToSafeLimits(t *testing.T) {
	prober := newWithLimits((&net.Dialer{}).DialContext, 0, 0)
	if cap(prober.semaphore) != defaultConcurrency || prober.timeout != defaultTimeout {
		t.Fatalf("无效上限未回退: concurrency=%d timeout=%v", cap(prober.semaphore), prober.timeout)
	}
}

type errRefused string

func (e errRefused) Error() string { return "connection refused: " + string(e) }

func immediateConnection(context.Context, string, string) (net.Conn, error) {
	client, server := net.Pipe()
	_ = server.Close()
	return client, nil
}

func staticResolution(address string) func(context.Context, string) ([]net.IPAddr, error) {
	return func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP(address)}}, nil
	}
}
