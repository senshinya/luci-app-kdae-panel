package daeconn

import (
	"sort"
	"sync"
	"time"
)

// Store 在面板进程内累积连接事件，把可视窗口从"系统日志缓冲区的寿命"
// 延长到"面板运行期"。有界：容量与 TTL 双重上限，重启即清零。
// 没有后台协程——只在每次 API 请求时合并，页面没人看就不做功。
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
		events:   map[string]*Event{},
		capacity: defaultCapacity,
		ttl:      defaultTTL,
		now:      time.Now,
	}
}

// Merge 合并一批新解析的事件，返回按时间倒序的全量流水。
// 同一行日志跨多次轮询重复出现时只保留一份。
func (s *Store) Merge(events []Event) []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()

	for _, event := range events {
		key := event.dedupKey()
		if _, exists := s.events[key]; exists {
			continue
		}
		stored := event
		if stored.Timestamp.IsZero() {
			// 日志行没带可解析的时间，用观测时刻兜底并如实标注。
			stored.Timestamp = now
			stored.ApproxTime = true
		}
		s.events[key] = &stored
	}

	s.evict(now)

	results := make([]Event, 0, len(s.events))
	for _, event := range s.events {
		results = append(results, *event)
	}
	sort.Slice(results, func(left, right int) bool {
		if !results[left].Timestamp.Equal(results[right].Timestamp) {
			return results[left].Timestamp.After(results[right].Timestamp)
		}
		return results[left].Src < results[right].Src
	})
	return results
}

// evict 执行 TTL 与容量双重淘汰，最旧的先走。
func (s *Store) evict(now time.Time) {
	for key, event := range s.events {
		if now.Sub(event.Timestamp) > s.ttl {
			delete(s.events, key)
		}
	}
	if len(s.events) <= s.capacity {
		return
	}
	type aged struct {
		key       string
		timestamp time.Time
	}
	candidates := make([]aged, 0, len(s.events))
	for key, event := range s.events {
		candidates = append(candidates, aged{key: key, timestamp: event.Timestamp})
	}
	sort.Slice(candidates, func(left, right int) bool {
		return candidates[left].timestamp.Before(candidates[right].timestamp)
	})
	for _, candidate := range candidates {
		if len(s.events) <= s.capacity {
			break
		}
		delete(s.events, candidate.key)
	}
}
