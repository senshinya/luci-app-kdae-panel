package daeconn

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
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

// interfaceAddrs 同上，注入缝。生产环境即标准库实现。
var interfaceAddrs = net.InterfaceAddrs

// tcpEstablished 是 /proc/net/tcp 的 st 列里 ESTABLISHED 的值。
const tcpEstablished = "01"

// maxInboundLegs 限制单次快照记录多少条入站腿。它直接决定响应里孤儿记录的
// 上限——孤儿数量等于 dae 持有的入站 socket 数，完全由局域网流量决定，没有
// 这道闸，一台机器开几万条连接就能让面板每 5 秒构造并编码几万条记录。
// 超出的部分仍计入 TCPSockets 总数，只是不再逐条呈现。
const maxInboundLegs = 5000

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

// Snapshot 是某一时刻 dae 进程持有的 socket 集合。
//
// dae 是 tproxy 透明代理、不做 NAT，它接受的入站 socket 在内核表里显示为
// local=原始目的地址、remote=客户端——这两个值恰好就是连接日志四元组的两端，
// 对账由此成立。区分入站腿与出站腿（dae→代理服务器）靠 local 是否为本机地址：
// 入站腿的 local 是被欺骗的远端地址，不属于任何本机接口。
type Snapshot struct {
	TakenAt time.Time
	// inbound 键为 tupleKey(tcp, 客户端, 原始目的)，值是两端地址，供孤儿展示。
	inbound map[string]inboundLeg
	// TCPSockets 是 dae 持有的 ESTABLISHED TCP socket 总数（含出站腿）。
	TCPSockets int
	// UDPSockets 是 dae 持有的 UDP socket 总数。
	UDPSockets int
	// Truncated 表示入站腿超出 maxInboundLegs、只记录了前一部分。
	// 计数仍然准确，逐条列出的记录不完整。
	Truncated bool
}

type inboundLeg struct {
	src netip.AddrPort
	dst netip.AddrPort
}

// TakeSnapshot 采集 dae 进程（mainPID）当前持有的 socket，结果在
// snapshotCacheTTL 内复用。mainPID <= 0 表示 dae 未运行，返回空快照——
// 没有进程就没有存活连接，这不是错误。/proc 读不动才返回 error，
// 调用方应把存活状态降级为"未知"。
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
	snapshot := Snapshot{TakenAt: time.Now().UTC(), inbound: map[string]inboundLeg{}}
	if mainPID <= 0 {
		return snapshot, nil
	}
	inodes, err := socketInodes(mainPID)
	if err != nil {
		return Snapshot{}, fmt.Errorf("枚举 dae 进程 socket: %w", err)
	}
	local, err := localAddresses()
	if err != nil {
		return Snapshot{}, fmt.Errorf("枚举本机接口地址: %w", err)
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
			if _, isLocal := local[row.local.Addr()]; isLocal {
				continue // local 是本机地址 ⇒ dae 主动拨出的出站腿
			}
			if len(snapshot.inbound) >= maxInboundLegs {
				snapshot.Truncated = true
				continue
			}
			leg := inboundLeg{src: row.remote, dst: row.local}
			snapshot.inbound[tupleKey("tcp", leg.src, leg.dst)] = leg
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

// liveTCP 报告某条四元组当前是否存在对应的入站腿。
func (s Snapshot) liveTCP(key string) bool {
	_, live := s.inbound[key]
	return live
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

func localAddresses() (map[netip.Addr]struct{}, error) {
	addresses, err := interfaceAddrs()
	if err != nil {
		return nil, err
	}
	local := make(map[netip.Addr]struct{}, len(addresses))
	for _, address := range addresses {
		var ip net.IP
		switch value := address.(type) {
		case *net.IPNet:
			ip = value.IP
		case *net.IPAddr:
			ip = value.IP
		default:
			continue
		}
		if parsed, ok := netip.AddrFromSlice(ip); ok {
			local[parsed.Unmap()] = struct{}{}
		}
	}
	return local, nil
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
	// 是因为大端 MIPS 是 OpenWrt 的现实目标，而搞错的症状是"存活判定全错"
	// 这种不会报错、只会给出错误结论的坏。
	for offset := 0; offset < len(raw); offset += 4 {
		binary.BigEndian.PutUint32(raw[offset:offset+4], binary.NativeEndian.Uint32(raw[offset:offset+4]))
	}
	address, ok := netip.AddrFromSlice(raw)
	if !ok {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(address.Unmap(), uint16(port)), true
}
