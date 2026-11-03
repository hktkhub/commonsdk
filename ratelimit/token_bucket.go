package ratelimit

import (
	"context"
	"sync"
	"time"
)

const tokenEpsilon = 1e-9

type tokenBucket struct {
	mu    sync.Mutex
	cache *cache[*tokenBucketEntry]

	limit float64
	rate  float64
	now   func() time.Time
}

type tokenBucketEntry struct {
	tokens float64
	last   time.Time
}

// NewTokenBucket 创建本地令牌桶。
//
// 桶容量为 limit，按 limit/window 的速率补充，允许最多 limit 次突发。
// 默认最多保留 10000 个 key，超过后淘汰最久未使用的 key。
func NewTokenBucket(limit int, window time.Duration, opts ...Option) Limiter {
	return newTokenBucket(limit, window, time.Now, opts...)
}

func newTokenBucket(limit int, window time.Duration, now func() time.Time, opts ...Option) *tokenBucket {
	validate(limit, window)
	cfg := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &tokenBucket{
		cache: newCache[*tokenBucketEntry](cfg.maxKeys, window),
		limit: float64(limit),
		rate:  float64(limit) / float64(window),
		now:   now,
	}
}

func (b *tokenBucket) Allow(_ context.Context, key string) bool {
	now := b.now()

	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.cleanup(now)

	entry, ok := b.cache.get(now, key)
	if !ok {
		entry = &tokenBucketEntry{
			tokens: b.limit,
			last:   now,
		}
		b.cache.put(now, key, entry)
	} else if elapsed := now.Sub(entry.last); elapsed > 0 {
		entry.tokens += float64(elapsed) * b.rate
		if entry.tokens > b.limit {
			entry.tokens = b.limit
		}
		entry.last = now
	} else if elapsed < 0 {
		entry.last = now
	}

	if entry.tokens+tokenEpsilon < 1 {
		return false
	}
	entry.tokens--
	return true
}
