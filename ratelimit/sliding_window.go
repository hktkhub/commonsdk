package ratelimit

import (
	"context"
	"sync"
	"time"
)

type slidingWindow struct {
	mu    sync.Mutex
	cache *cache[*windowEntry]

	limit  int
	period time.Duration
	now    func() time.Time
}

type windowEntry struct {
	times []int64
	head  int
	count int
}

// NewSlidingWindow 创建本地滑动窗口。
//
// 每个 key 在任意 period 窗口内最多放行 limit 次。
// 每个活跃 key 的内存占用与 limit 成正比，默认最多保留 10000 个 key。
func NewSlidingWindow(limit int, period time.Duration, opts ...Option) Limiter {
	return newSlidingWindow(limit, period, time.Now, opts...)
}

func newSlidingWindow(limit int, period time.Duration, now func() time.Time, opts ...Option) *slidingWindow {
	validate(limit, period)
	cfg := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &slidingWindow{
		cache:  newCache[*windowEntry](cfg.maxKeys, period),
		limit:  limit,
		period: period,
		now:    now,
	}
}

func (w *slidingWindow) Allow(_ context.Context, key string) bool {
	now := w.now()

	w.mu.Lock()
	defer w.mu.Unlock()

	w.cache.cleanup(now)

	entry, ok := w.cache.get(now, key)
	if !ok {
		entry = &windowEntry{times: make([]int64, w.limit)}
		w.cache.put(now, key, entry)
	}

	cutoff := now.Add(-w.period).UnixNano()
	for entry.count > 0 && entry.times[entry.head] <= cutoff {
		entry.head = (entry.head + 1) % w.limit
		entry.count--
	}

	if entry.count >= w.limit {
		return false
	}

	entry.times[(entry.head+entry.count)%w.limit] = now.UnixNano()
	entry.count++
	return true
}
