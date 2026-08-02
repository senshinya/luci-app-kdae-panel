package daeconn

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// procRoot 是 /proc 的挂载点。做成变量与 host、diagnostics 包同一个理由：
// 给将来的测试留注入缝，生产路径永远是 /proc。
var procRoot = "/proc"

// tcpEstablished 是 /proc/net/tcp 的 st 列里 ESTABLISHED 的值。
const tcpEstablished = "01"

// maxOutboundEndpoints 限制分组结果的条目数。远端数量由 dae 的节点数决定，
// 正常只有个位数；设闸是防止异常情况下把响应撑大。
const maxOutboundEndpoints = 200

// snapshotCacheTTL 是快照的复用窗口。页面每 5 秒轮询一次，多开几个标签页或
// 连点刷新就会叠加成并发全量扫描，而一次扫描要读完 /proc/net/tcp 并对 dae
// 的每个 fd 做一次 readlink——连接数上万时这不是可以忽略的开销。
const snapshotCacheTTL = 2 * time.Second

var snapshotCache struct {
	mu       sync.Mutex
	takenAt  time.Time
	pid      int
	snapshot Snapshot
	err      error
}

// Snapshot 是某一时刻 dae 进程持有的 socket 概况。
//
// 只统计 dae 自己发起的出站连接——这是它在 /proc 里唯一可见的连接形态。
// 客户端一侧不在这里：dae 的 eBPF 数据面把被代理连接的客户端侧完全留在内核，
// 既不产生 userspace socket，也不进 netfilter conntrack，因此没有任何公开
// 接口能逐条判断某条客户端连接是否还活着。详见 docs/architecture.md。
type Snapshot struct {
	TakenAt time.Time
	// Outbound 是 dae 当前持有的 ESTABLISHED 出站连接，按远端 ip:port 分组。
	// 远端通常就是代理节点的地址，所以这张表等价于"每个节点当前扛着多少连接"。
	Outbound map[string]int
	// TCPSockets、UDPSockets 是 dae 持有的 socket 总数。
	TCPSockets int
	UDPSockets int
	// Truncated 表示分组条目超过上限、只保留了一部分。计数仍然准确。
	Truncated bool
}

// TakeSnapshot 采集 dae 进程（mainPID）当前持有的 socket，结果在
// snapshotCacheTTL 内复用。mainPID <= 0 表示 dae 未运行，返回空快照——
// 没有进程就没有连接，这不是错误。/proc 读不动才返回 error。
func TakeSnapshot(mainPID int) (Snapshot, error) {
	snapshotCache.mu.Lock()
	defer snapshotCache.mu.Unlock()
	if snapshotCache.pid == mainPID && !snapshotCache.takenAt.IsZero() &&
		time.Since(snapshotCache.takenAt) < snapshotCacheTTL {
		return snapshotCache.snapshot, snapshotCache.err
	}
	snapshot, err := takeSnapshot(mainPID)
	snapshotCache.takenAt = time.Now()
	snapshotCache.pid = mainPID
	snapshotCache.snapshot = snapshot
	snapshotCache.err = err
	return snapshot, err
}

func takeSnapshot(mainPID int) (Snapshot, error) {
	snapshot := Snapshot{TakenAt: time.Now().UTC(), Outbound: map[string]int{}}
	if mainPID <= 0 {
		return snapshot, nil
	}
	inodes, err := socketInodes(mainPID)
	if err != nil {
		return Snapshot{}, fmt.Errorf("枚举 dae 进程 socket: %w", err)
	}
	for _, name := range []string{"tcp", "tcp6"} {
		rows, err := parseSocketTable(filepath.Join(procRoot, "net", name))
		if err != nil {
			if os.IsNotExist(err) {
				continue // 单栈内核没有对应协议族的表
			}
			return Snapshot{}, err
		}
		for _, row := range rows {
			if row.state != tcpEstablished {
				continue
			}
			if _, held := inodes[row.inode]; !held {
				continue
			}
			snapshot.TCPSockets++
			endpoint := row.remote.String()
			if _, known := snapshot.Outbound[endpoint]; !known && len(snapshot.Outbound) >= maxOutboundEndpoints {
				snapshot.Truncated = true
				continue
			}
			snapshot.Outbound[endpoint]++
		}
	}
	for _, name := range []string{"udp", "udp6"} {
		rows, err := parseSocketTable(filepath.Join(procRoot, "net", name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Snapshot{}, err
		}
		for _, row := range rows {
			if _, held := inodes[row.inode]; held {
				snapshot.UDPSockets++
			}
		}
	}
	return snapshot, nil
}

// socketInodes 收集 /proc/<pid>/fd 里所有 socket 的 inode。
// 个别 fd 在遍历间隙被关闭属于常态，单条 readlink 失败直接跳过。
func socketInodes(pid int) (map[string]struct{}, error) {
	fdDir := filepath.Join(procRoot, strconv.Itoa(pid), "fd")
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return nil, err
	}
	inodes := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join(fdDir, entry.Name()))
		if err != nil {
			continue
		}
		if inode, found := strings.CutPrefix(target, "socket:["); found {
			inodes[strings.TrimSuffix(inode, "]")] = struct{}{}
		}
	}
	return inodes, nil
}

type socketRow struct {
	local  netip.AddrPort
	remote netip.AddrPort
	state  string
	inode  string
}

// parseSocketTable 解析 /proc/net/tcp{,6}、udp{,6}。列布局固定：
//
//	sl local_address rem_address st tx:rx tr:when retrnsmt uid timeout inode …
//
// 逐行流式读取：连接数上万时这张表有十几 MB，没有必要整份驻留内存。
func parseSocketTable(path string) ([]socketRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	rows := make([]socketRow, 0, 128)
	scanner := bufio.NewScanner(file)
	for lineNumber := 0; scanner.Scan(); lineNumber++ {
		if lineNumber == 0 {
			continue // 首行是表头
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		local, okLocal := parseHexAddrPort(fields[1])
		remote, okRemote := parseHexAddrPort(fields[2])
		if !okLocal || !okRemote {
			continue
		}
		rows = append(rows, socketRow{local: local, remote: remote, state: fields[3], inode: fields[9]})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 %s: %w", path, err)
	}
	return rows, nil
}

// parseHexAddrPort 解析 "0100007F:1F90" 这样的地址列。
func parseHexAddrPort(value string) (netip.AddrPort, bool) {
	addressPart, portPart, found := strings.Cut(value, ":")
	if !found {
		return netip.AddrPort{}, false
	}
	port, err := strconv.ParseUint(portPart, 16, 16)
	if err != nil {
		return netip.AddrPort{}, false
	}
	raw, err := hex.DecodeString(addressPart)
	if err != nil || (len(raw) != 4 && len(raw) != 16) {
		return netip.AddrPort{}, false
	}
	// 内核把地址按 32 位字用主机字节序打印。按主机序读、按大端写，就还原成
	// 网络字节序——小端机上等价于逐 4 字节反转，大端机上是空操作。不写死反转
	// 是因为大端 MIPS 是 OpenWrt 的现实目标，而搞错的症状是"地址全错"这种
	// 不会报错、只会给出错误结论的坏。
	for offset := 0; offset < len(raw); offset += 4 {
		binary.BigEndian.PutUint32(raw[offset:offset+4], binary.NativeEndian.Uint32(raw[offset:offset+4]))
	}
	address, ok := netip.AddrFromSlice(raw)
	if !ok {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(address.Unmap(), uint16(port)), true
}
