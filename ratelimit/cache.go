package ratelimit

import (
	"container/list"
	"time"
)

// cache 是按最近访问排序的 key 容器。
// 它不是并发安全的，调用方需要自己持有锁。
type cache[V any] struct {
	entries map[string]*list.Element
	order   *list.List

	maxKeys      int
	cleanupEvery time.Duration
	idleTTL      time.Duration
	lastCleanup  time.Time
}

type cacheEntry[V any] struct {
	key      string
	value    V
	lastSeen time.Time
}

func newCache[V any](maxKeys int, window time.Duration) *cache[V] {
	every, ttl := cleanupSettings(window)
	return &cache[V]{
		entries:      make(map[string]*list.Element),
		order:        list.New(),
		maxKeys:      maxKeys,
		cleanupEvery: every,
		idleTTL:      ttl,
	}
}

func (c *cache[V]) get(now time.Time, key string) (V, bool) {
	element, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	entry := element.Value.(*cacheEntry[V])
	entry.lastSeen = now
	c.order.MoveToFront(element)
	return entry.value, true
}

func (c *cache[V]) put(now time.Time, key string, value V) {
	if element, ok := c.entries[key]; ok {
		entry := element.Value.(*cacheEntry[V])
		entry.value = value
		entry.lastSeen = now
		c.order.MoveToFront(element)
		return
	}

	element := c.order.PushFront(&cacheEntry[V]{
		key:      key,
		value:    value,
		lastSeen: now,
	})
	c.entries[key] = element
	c.evict()
}

// evict 从最近最少使用的 key 开始淘汰，保证内存有上限。
func (c *cache[V]) evict() {
	if c.maxKeys <= 0 {
		return
	}
	for len(c.entries) > c.maxKeys {
		element := c.order.Back()
		if element == nil {
			return
		}
		entry := element.Value.(*cacheEntry[V])
		delete(c.entries, entry.key)
		c.order.Remove(element)
	}
}

// cleanup 从 LRU 尾部删除空闲 key，复杂度与删除数量成正比。
func (c *cache[V]) cleanup(now time.Time) {
	if now.Sub(c.lastCleanup) < c.cleanupEvery {
		return
	}
	c.lastCleanup = now

	for {
		element := c.order.Back()
		if element == nil {
			return
		}
		entry := element.Value.(*cacheEntry[V])
		if now.Sub(entry.lastSeen) < c.idleTTL {
			return
		}
		delete(c.entries, entry.key)
		c.order.Remove(element)
	}
}

func (c *cache[V]) len() int { return len(c.entries) }
