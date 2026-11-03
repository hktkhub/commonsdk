package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Unix(0, 0)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

func TestTokenBucketBurstAndRefill(t *testing.T) {
	clock := newFakeClock()
	limiter := newTokenBucket(2, time.Second, clock.Now)
	ctx := context.Background()

	if !limiter.Allow(ctx, "key") || !limiter.Allow(ctx, "key") {
		t.Fatal("first two requests should be allowed")
	}
	if limiter.Allow(ctx, "key") {
		t.Fatal("third request should be rejected")
	}

	clock.Advance(500 * time.Millisecond)
	if !limiter.Allow(ctx, "key") {
		t.Fatal("request should be allowed after refill")
	}
	if limiter.Allow(ctx, "key") {
		t.Fatal("bucket should be empty again")
	}

	clock.Advance(time.Second)
	if !limiter.Allow(ctx, "key") || !limiter.Allow(ctx, "key") {
		t.Fatal("bucket should be full after a full window")
	}
}

func TestTokenBucketKeysAreIndependent(t *testing.T) {
	clock := newFakeClock()
	limiter := newTokenBucket(1, time.Second, clock.Now)
	ctx := context.Background()

	if !limiter.Allow(ctx, "a") {
		t.Fatal("key a should be allowed")
	}
	if limiter.Allow(ctx, "a") {
		t.Fatal("key a should be rejected")
	}
	if !limiter.Allow(ctx, "b") {
		t.Fatal("key b should be independent")
	}
}

func TestTokenBucketConcurrentLimit(t *testing.T) {
	clock := newFakeClock()
	limiter := newTokenBucket(50, time.Minute, clock.Now)
	ctx := context.Background()

	var allowed atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow(ctx, "key") {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := allowed.Load(); got != 50 {
		t.Fatalf("allowed = %d, want 50", got)
	}
}

func TestTokenBucketCleanupIdleKeys(t *testing.T) {
	clock := newFakeClock()
	limiter := newTokenBucket(1, time.Second, clock.Now)
	ctx := context.Background()

	limiter.Allow(ctx, "old")
	clock.Advance(11 * time.Minute)
	limiter.Allow(ctx, "new")

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if _, ok := limiter.cache.entries["old"]; ok {
		t.Fatal("idle key was not cleaned up")
	}
}

func TestTokenBucketEvictsLeastRecentlyUsed(t *testing.T) {
	clock := newFakeClock()
	limiter := newTokenBucket(1, time.Minute, clock.Now, WithMaxKeys(2))
	ctx := context.Background()

	limiter.Allow(ctx, "a")
	limiter.Allow(ctx, "b")
	limiter.Allow(ctx, "a")
	limiter.Allow(ctx, "c")

	limiter.mu.Lock()
	_, hasA := limiter.cache.entries["a"]
	_, hasB := limiter.cache.entries["b"]
	_, hasC := limiter.cache.entries["c"]
	limiter.mu.Unlock()

	if !hasA || hasB || !hasC {
		t.Fatalf("cache keys a=%v b=%v c=%v, want a and c", hasA, hasB, hasC)
	}
	if !limiter.Allow(ctx, "b") {
		t.Fatal("evicted key should get a fresh quota")
	}
}
