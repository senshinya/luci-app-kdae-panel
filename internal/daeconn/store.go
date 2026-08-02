package daeconn

import (
	"sort"
	"sync"
	"time"
)

// Store 在面板进程内累积连接建立事件，把可视窗口从系统日志缓冲区延长到
// 面板运行期。容量和 TTL 都有硬上限，面板重启后重新开始积累。
type Store struct {
	mu       sync.Mutex
	events   map[string]*Event
	capacity int
	ttl      time.Duration
	now      func() time.Time
}

const (
	defaultCapacity = 2000
	defaultTTL      = 24 * time.Hour
)

func NewStore() *Store {
	return &Store{
		events:   make(map[string]*Event),
		capacity: defaultCapacity,
		ttl:      defaultTTL,
		now:      time.Now,
	}
}

// Merge 合并新解析的事件并返回按建立时间倒序的完整窗口。journald 的同一行
// 会被多次轮询到，dedupKey 保证它只进入一次。
func (store *Store) Merge(events []Event) (result []Event, truncated bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	now := store.now().UTC()

	for _, event := range events {
		key := event.dedupKey()
		if _, exists := store.events[key]; exists {
			continue
		}
		stored := event
		if stored.Timestamp.IsZero() {
			stored.Timestamp = now
			stored.ApproxTime = true
		}
		store.events[key] = &stored
	}

	for key, event := range store.events {
		if now.Sub(event.Timestamp) > store.ttl {
			delete(store.events, key)
		}
	}
	if len(store.events) > store.capacity {
		truncated = true
		store.evictOldest(len(store.events) - store.capacity)
	}

	result = make([]Event, 0, len(store.events))
	for _, event := range store.events {
		result = append(result, *event)
	}
	sort.Slice(result, func(left, right int) bool {
		if !result[left].Timestamp.Equal(result[right].Timestamp) {
			return result[left].Timestamp.After(result[right].Timestamp)
		}
		return result[left].Src < result[right].Src
	})
	return result, truncated
}

func (store *Store) evictOldest(count int) {
	type candidate struct {
		key       string
		timestamp time.Time
	}
	candidates := make([]candidate, 0, len(store.events))
	for key, event := range store.events {
		candidates = append(candidates, candidate{key: key, timestamp: event.Timestamp})
	}
	sort.Slice(candidates, func(left, right int) bool {
		return candidates[left].timestamp.Before(candidates[right].timestamp)
	})
	for _, candidate := range candidates[:count] {
		delete(store.events, candidate.key)
	}
}
