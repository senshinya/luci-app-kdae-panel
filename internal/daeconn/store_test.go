package daeconn

import (
	"sync"
	"testing"
	"time"
)

func TestStoreMergesDeduplicatesAndSortsEvents(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	store.now = func() time.Time { return now }
	older := testEvent(now.Add(-time.Minute), "192.0.2.1:1")
	newer := testEvent(now, "192.0.2.1:2")
	result, truncated := store.Merge([]Event{older, newer, older})
	if truncated || len(result) != 2 || result[0].Src != newer.Src || result[1].Src != older.Src {
		t.Fatalf("合并结果异常: %+v, truncated=%v", result, truncated)
	}
}

func TestStoreUsesObservationTimeAndExpiresOldEvents(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	store.ttl = time.Hour
	store.now = func() time.Time { return now }
	withoutTime := testEvent(time.Time{}, "192.0.2.1:1")
	result, _ := store.Merge([]Event{withoutTime})
	if len(result) != 1 || !result[0].ApproxTime || !result[0].Timestamp.Equal(now) {
		t.Fatalf("缺失时间未使用观测时刻: %+v", result)
	}
	now = now.Add(time.Hour + time.Second)
	result, _ = store.Merge(nil)
	if len(result) != 0 {
		t.Fatalf("过期记录未清理: %+v", result)
	}
}

func TestStoreCapacityIsHardLimit(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	store.capacity = 2
	store.now = func() time.Time { return now }
	events := []Event{
		testEvent(now.Add(-3*time.Minute), "192.0.2.1:1"),
		testEvent(now.Add(-2*time.Minute), "192.0.2.1:2"),
		testEvent(now.Add(-time.Minute), "192.0.2.1:3"),
	}
	result, truncated := store.Merge(events)
	if len(result) != 2 || !truncated || result[1].Src != "192.0.2.1:2" {
		t.Fatalf("容量上限异常: %+v, truncated=%v", result, truncated)
	}
}

func TestStoreConcurrentMerge(t *testing.T) {
	store := NewStore()
	event := testEvent(time.Now().UTC(), "192.0.2.1:1")
	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			store.Merge([]Event{event})
		}()
	}
	wait.Wait()
	result, _ := store.Merge(nil)
	if len(result) != 1 {
		t.Fatalf("并发去重后记录数 = %d", len(result))
	}
}

func testEvent(timestamp time.Time, source string) Event {
	return Event{
		Timestamp: timestamp,
		Network:   "tcp4",
		Src:       source,
		Target:    "example.com:443",
		Outbound:  "proxy",
		Dialer:    "tokyo",
	}
}
