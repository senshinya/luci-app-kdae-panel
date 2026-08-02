package daeconn

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseHexAddrPort(t *testing.T) {
	for input, want := range map[string]string{
		"0100007F:1F90":                         "127.0.0.1:8080",
		"00000000000000000000000001000000:01BB": "[::1]:443",
		"0000000000000000FFFF00000100007F:0035": "127.0.0.1:53",
	} {
		got, ok := parseHexAddrPort(input)
		if !ok || got.String() != want {
			t.Fatalf("parseHexAddrPort(%q) = %q, %v，期望 %q", input, got, ok, want)
		}
	}
	for _, input := range []string{"", "broken", "0100007F:xxxx", "01:0001"} {
		if got, ok := parseHexAddrPort(input); ok {
			t.Fatalf("非法地址 %q 被解析为 %s", input, got)
		}
	}
}

func TestProcSnapshotterGroupsOutboundEndpointsAndCaches(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "42", "fd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "net"), 0o700); err != nil {
		t.Fatal(err)
	}
	links := map[string]string{"1": "socket:[111]", "2": "socket:[112]", "3": "socket:[113]", "4": "socket:[222]"}
	for name := range links {
		if err := os.WriteFile(filepath.Join(root, "42", "fd", name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	tcp := strings.Join([]string{
		"sl local_address rem_address st tx_queue tr tm->when retrnsmt uid timeout inode",
		"0: 0201A8C0:9C40 08080808:01BB 01 0:0 00:0 0 0 0 111",
		"1: 0201A8C0:9C41 08080808:01BB 01 0:0 00:0 0 0 0 112",
		"2: 0201A8C0:9C42 01010101:0050 01 0:0 00:0 0 0 0 113",
		"3: 0100007F:0001 0100007F:0002 01 0:0 00:0 0 0 0 999",
	}, "\n") + "\n"
	udp := "sl local_address rem_address st tx_queue tr tm->when retrnsmt uid timeout inode\n" +
		"0: 00000000:0035 00000000:0000 07 0:0 00:0 0 0 0 222\n"
	writeProcTable(t, root, "tcp", tcp)
	writeProcTable(t, root, "tcp6", "header\n")
	writeProcTable(t, root, "udp", udp)
	writeProcTable(t, root, "udp6", "header\n")

	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	snapshotter := NewProcSnapshotter()
	snapshotter.procRoot = root
	snapshotter.now = func() time.Time { return now }
	snapshotter.readlink = func(path string) (string, error) {
		return links[filepath.Base(path)], nil
	}

	snapshot, err := snapshotter.Snapshot(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.OutboundTCP != 3 || snapshot.UDPSockets != 1 || snapshot.SampledTCPPeak != 3 || snapshot.SampledUDPPeak != 1 ||
		snapshot.Endpoints["8.8.8.8:443"] != 2 || snapshot.Endpoints["1.1.1.1:80"] != 1 {
		t.Fatalf("快照统计异常: %+v", snapshot)
	}

	writeProcTable(t, root, "tcp", "header\n")
	cached, err := snapshotter.Snapshot(context.Background(), 42)
	if err != nil || cached.OutboundTCP != 3 {
		t.Fatalf("缓存未复用: %+v, %v", cached, err)
	}
	now = now.Add(defaultCacheTTL + time.Millisecond)
	refreshed, err := snapshotter.Snapshot(context.Background(), 42)
	if err != nil || refreshed.OutboundTCP != 0 || refreshed.SampledTCPPeak != 3 || refreshed.SampledUDPPeak != 1 {
		t.Fatalf("缓存过期后未重新采集: %+v, %v", refreshed, err)
	}
	writeProcTable(t, root, "udp", "header\n")
	now = now.Add(RecentSampleWindow + time.Millisecond)
	expired, err := snapshotter.Snapshot(context.Background(), 42)
	if err != nil || expired.SampledTCPPeak != 0 || expired.SampledUDPPeak != 0 {
		t.Fatalf("过期采样仍计入峰值: %+v, %v", expired, err)
	}
}

func TestProcSnapshotterCapsEndpointGroupsWithoutLosingSocketCount(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "7", "fd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "net"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"1", "2"} {
		if err := os.WriteFile(filepath.Join(root, "7", "fd", name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeProcTable(t, root, "tcp", "header\n0: 0100007F:1 01010101:1 01 0:0 0:0 0 0 0 1\n1: 0100007F:2 02020202:2 01 0:0 0:0 0 0 0 2\n")
	for _, name := range []string{"tcp6", "udp", "udp6"} {
		writeProcTable(t, root, name, "header\n")
	}
	snapshotter := NewProcSnapshotter()
	snapshotter.procRoot = root
	snapshotter.maxEndpoints = 1
	snapshotter.readlink = func(path string) (string, error) { return "socket:[" + filepath.Base(path) + "]", nil }
	snapshot, err := snapshotter.Snapshot(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.OutboundTCP != 2 || len(snapshot.Endpoints) != 1 || !snapshot.Truncated {
		t.Fatalf("端点上限未正确执行: %+v", snapshot)
	}
}

func TestProcSnapshotterClearsSampledPeakWhenPIDChanges(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	snapshotter := NewProcSnapshotter()
	snapshotter.now = func() time.Time { return now }
	snapshotter.cachedPID = 42
	snapshotter.observed = []socketObservation{{at: now, tcp: 8, udp: 3}}

	snapshot, err := snapshotter.Snapshot(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SampledTCPPeak != 0 || snapshot.SampledUDPPeak != 0 {
		t.Fatalf("旧 dae PID 的峰值泄漏到停止状态: %+v", snapshot)
	}
}

func writeProcTable(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "net", name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
