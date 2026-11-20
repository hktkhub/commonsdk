package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSlidingWindow(t *testing.T) {
	clock := newFakeClock()
	limiter := newSlidingWindow(2, time.Second, clock.Now)
	ctx := context.Background()

	if !limiter.Allow(ctx, "key") {
		t.Fatal("first request should be allowed")
	}
	clock.Advance(400 * time.Millisecond)
	if !limiter.Allow(ctx, "key") {
		t.Fatal("second request should be allowed")
	}
	if limiter.Allow(ctx, "key") {
		t.Fatal("third request should be rejected")
	}

	clock.Advance(601 * time.Millisecond)
	if !limiter.Allow(ctx, "key") {
		t.Fatal("request should be allowed after the first timestamp expires")
	}
	if limiter.Allow(ctx, "key") {
		t.Fatal("window should still be full")
	}

	clock.Advance(399 * time.Millisecond)
	if !limiter.Allow(ctx, "key") {
		t.Fatal("second timestamp should have expired")
	}
}

func TestSlidingWindowExactBoundary(t *testing.T) {
	clock := newFakeClock()
	limiter := newSlidingWindow(1, time.Second, clock.Now)
	ctx := context.Background()

	if !limiter.Allow(ctx, "key") {
		t.Fatal("first request should be allowed")
	}
	clock.Advance(time.Second)
	if !limiter.Allow(ctx, "key") {
		t.Fatal("request at the exact window boundary should be allowed")
	}
}

func TestSlidingWindowKeysAreIndependent(t *testing.T) {
	clock := newFakeClock()
	limiter := newSlidingWindow(1, time.Second, clock.Now)
	ctx := context.Background()

	if !limiter.Allow(ctx, "a") || limiter.Allow(ctx, "a") {
		t.Fatal("key a should be limited to one request")
	}
	if !limiter.Allow(ctx, "b") {
		t.Fatal("key b should be independent")
	}
}

func TestSlidingWindowConcurrentLimit(t *testing.T) {
	clock := newFakeClock()
	limiter := newSlidingWindow(50, time.Minute, clock.Now)
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

func TestSlidingWindowCleanupIdleKeys(t *testing.T) {
	clock := newFakeClock()
	limiter := newSlidingWindow(1, time.Second, clock.Now)
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

func TestSlidingWindowEvictsLeastRecentlyUsed(t *testing.T) {
	clock := newFakeClock()
	limiter := newSlidingWindow(1, time.Minute, clock.Now, WithMaxKeys(2))
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
