package daeconn

import (
	"sort"
	"sync"
	"time"
)

// Status 是一条连接记录的存活判定结果。
type Status string

const (
	// StatusLive：四元组在当前 socket 表中有对应的入站腿。
	StatusLive Status = "live"
	// StatusClosed：socket 表里已经没有它，且没有理由怀疑判定。
	StatusClosed Status = "closed"
	// StatusUnknown：无法判定。UDP 没有存活语义；被 eBPF 卸载的 TCP 若
	// socket 未命中，宁可说不知道也不误报已结束；快照采集失败时同理。
	StatusUnknown Status = "unknown"
	// StatusOrphan：socket 存活但建立日志已滚出窗口，只有裸四元组。
	StatusOrphan Status = "orphan"
)

// Record 是对外输出的一条连接记录：事件元数据加存活判定。
type Record struct {
	Event
	Status Status `json:"status"`
	// ApproxFirstSeen 表示 firstSeen 不是日志时间戳而是面板首次观测时刻
	// （孤儿连接，或日志行没带可解析的时间）。前端用它渲染 "≥" 前缀。
	ApproxFirstSeen bool `json:"approxFirstSeen,omitempty"`

	// everLive 记录这条记录是否曾经被 socket 认领过。四元组会被后续连接
	// 复用，若那条新连接的建立日志没采到，旧记录会再次命中快照——没有这个
	// 标记就会把它从 closed 翻回 live，带着旧的出站与节点信息张冠李戴。
	everLive bool
	// settledAt 是进入 closed 的时刻，TTL 从它起算而不是从建立时刻起算，
	// 否则一条活过 TTL 的长连接会在断开的同一轮里被删掉，用户永远看不到
	// 它的"已结束"。
	settledAt time.Time
}

// Store 在面板进程内累积连接事件，把可视窗口从"系统日志缓冲区的寿命"
// 延长到"面板运行期"。有界：容量与 TTL 双重上限，重启即清空。
// 没有后台协程——只在每次 API 请求时合并，页面没人看就不做功。
type Store struct {
	mu         sync.Mutex
	records    map[string]*Record   // dedupKey → 记录
	orphanSeen map[string]time.Time // tupleKey → 孤儿首次观测时刻
	capacity   int
	ttl        time.Duration
	now        func() time.Time
}

const (
	defaultCapacity = 2000
	defaultTTL      = 24 * time.Hour
)

func NewStore() *Store {
	return &Store{
		records:    map[string]*Record{},
		orphanSeen: map[string]time.Time{},
		capacity:   defaultCapacity,
		ttl:        defaultTTL,
		now:        time.Now,
	}
}

// Reconcile 合并一批新解析的事件，并按 socket 快照重算每条记录的存活状态。
// snapshotOK 为 false 表示快照采集失败：这时不生成孤儿，TCP 一律降级为
// 未知——把"探测坏了"报成"连接全断了"是最误导人的输出。
func (s *Store) Reconcile(events []Event, snapshot Snapshot, snapshotOK bool) []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()

	for _, event := range events {
		key := event.dedupKey()
		if _, exists := s.records[key]; exists {
			continue
		}
		record := &Record{Event: event}
		if record.Timestamp.IsZero() {
			record.Timestamp = now
			record.ApproxFirstSeen = true
		}
		s.records[key] = record
	}

	// 先判存活再做淘汰：还活着的连接不能因为建立得早就被 TTL 挤掉。
	liveByTuple := map[string]*Record{}
	for _, record := range s.records {
		record.Status = s.status(record, snapshot, snapshotOK, now)
		if record.Status != StatusLive {
			continue
		}
		tuple := record.flowKey()
		// 同一四元组先后承载多条连接时，只有最新一条对应现存 socket。
		if current, exists := liveByTuple[tuple]; exists {
			older := current
			if record.Timestamp.Before(current.Timestamp) {
				older = record
			} else {
				liveByTuple[tuple] = record
			}
			older.settle(now)
			continue
		}
		liveByTuple[tuple] = record
	}

	s.evict(now)

	results := make([]Record, 0, len(s.records)+len(snapshot.inbound))
	for _, record := range s.records {
		results = append(results, *record)
	}
	// 快照失败时不动 orphanSeen：那一轮本来就不新增条目，清空只会让恢复后
	// 所有孤儿重新盖上当前时刻，界面上的"已持续"归零、排序整体跳变。
	if snapshotOK {
		results = append(results, s.orphans(snapshot, liveByTuple, now)...)
	}
	sort.Slice(results, func(left, right int) bool {
		if !results[left].Timestamp.Equal(results[right].Timestamp) {
			return results[left].Timestamp.After(results[right].Timestamp)
		}
		return results[left].Src < results[right].Src
	})
	return results
}

func (s *Store) status(record *Record, snapshot Snapshot, snapshotOK bool, now time.Time) Status {
	if protocol(record.Network) == "udp" {
		return StatusUnknown
	}
	if !snapshotOK {
		return StatusUnknown
	}
	tuple := record.flowKey()
	if tuple == "" {
		return StatusUnknown
	}
	if snapshot.liveTCP(tuple) {
		// 已经判过结束的记录不复活：再次命中说明四元组被一条新连接复用了，
		// 而那条新连接的日志没被采到。它的归属由孤儿分支如实呈现，比给旧
		// 记录套上一副新连接的躯壳好。
		if record.Status == StatusClosed && record.settled() {
			return StatusClosed
		}
		record.everLive = true
		return StatusLive
	}
	if record.Offloaded {
		// 被 eBPF 卸载的连接尚未在真机上验证 userspace socket 是否保留，
		// 未命中时不下"已结束"的结论。
		return StatusUnknown
	}
	record.settle(now)
	return StatusClosed
}

// settle 标记记录进入终态，并把 TTL 的起算点定在此刻。重复调用只记第一次。
func (r *Record) settle(now time.Time) {
	r.Status = StatusClosed
	if r.settledAt.IsZero() {
		r.settledAt = now
	}
}

// settled 报告这条记录是否已经被判过结束。曾经存活过、又被判结束的记录，
// 才是"四元组被复用"的可靠信号；从未存活过的记录（建立日志晚于快照一轮）
// 允许在下一轮转成 live。
func (r *Record) settled() bool {
	return r.everLive && !r.settledAt.IsZero()
}

// orphans 把快照里没有任何存活记录认领的入站腿补成孤儿记录，
// 并维护它们的首次观测时刻：同一条腿跨轮询要保持稳定的 firstSeen，
// 排序才不会每次刷新都跳。
func (s *Store) orphans(snapshot Snapshot, liveByTuple map[string]*Record, now time.Time) []Record {
	seen := make(map[string]time.Time, len(snapshot.inbound))
	results := make([]Record, 0, len(snapshot.inbound))
	for tuple, leg := range snapshot.inbound {
		if _, claimed := liveByTuple[tuple]; claimed {
			continue
		}
		firstSeen, known := s.orphanSeen[tuple]
		if !known {
			firstSeen = now
		}
		seen[tuple] = firstSeen
		results = append(results, Record{
			Event: Event{
				Timestamp: firstSeen,
				Network:   "tcp",
				Src:       leg.src.String(),
				Target:    leg.dst.String(),
				DstAddr:   leg.dst.String(),
			},
			Status:          StatusOrphan,
			ApproxFirstSeen: true,
		})
	}
	s.orphanSeen = seen // 已消失或被认领的腿顺带清掉，防止无限增长
	return results
}

// evict 执行 TTL 与容量双重淘汰。存活记录不参与：它们的数量受真实
// socket 数约束，且正是页面的主角。
func (s *Store) evict(now time.Time) {
	for key, record := range s.records {
		if record.Status == StatusLive {
			continue
		}
		// 判过结束的记录从结束时刻起算，其余（未知、无法对账）从建立时刻起算。
		since := record.Timestamp
		if !record.settledAt.IsZero() {
			since = record.settledAt
		}
		if now.Sub(since) > s.ttl {
			delete(s.records, key)
		}
	}
	if len(s.records) <= s.capacity {
		return
	}
	type aged struct {
		key       string
		timestamp time.Time
	}
	candidates := make([]aged, 0, len(s.records))
	for key, record := range s.records {
		if record.Status != StatusLive {
			candidates = append(candidates, aged{key: key, timestamp: record.Timestamp})
		}
	}
	sort.Slice(candidates, func(left, right int) bool {
		return candidates[left].timestamp.Before(candidates[right].timestamp)
	})
	for _, candidate := range candidates {
		if len(s.records) <= s.capacity {
			break
		}
		delete(s.records, candidate.key)
	}
}
