// Package ratelimit 提供本地限流能力。
//
// 对外的唯一抽象是 Limiter：调用方传入维度和次数配置，限流器只负责判断放行或拒绝。
// 维度由 key 表达，可以是全局、IP、用户或它们的组合。
package ratelimit

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// GlobalKey 是全局限流使用的固定 key。
const GlobalKey = "global"

const defaultMaxKeys = 10_000

type options struct {
	maxKeys int
}

func defaultOptions() options {
	return options{maxKeys: defaultMaxKeys}
}

// Option 配置本地限流器。
type Option func(*options)

// WithMaxKeys 设置最多保留的 key 数量，默认 10000。
//
// 超过上限后按 LRU 淘汰最久未使用的 key，被淘汰 key 的配额会重置。
// n <= 0 表示不限制 key 数量，高基数场景不建议关闭。
func WithMaxKeys(n int) Option {
	return func(o *options) {
		if n < 0 {
			n = 0
		}
		o.maxKeys = n
	}
}

// Limiter 判断某个 key 当前是否允许通过。
//
// ctx 预留给分布式实现（例如 Redis 超时控制），本地实现不依赖它。
type Limiter interface {
	Allow(ctx context.Context, key string) bool
}

// Join 把多个维度拼成不会冲突的 key。
//
// 每个片段带长度前缀，例如 Join("ip", "1.2.3.4", "user", "alice")。
func Join(parts ...string) string {
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(strconv.Itoa(len(part)))
		b.WriteByte(':')
		b.WriteString(part)
	}
	return b.String()
}

func validate(limit int, window time.Duration) {
	if limit <= 0 {
		panic("ratelimit: limit must be greater than 0")
	}
	if window <= 0 {
		panic("ratelimit: window must be greater than 0")
	}
}

// cleanupSettings 返回空闲 key 的扫描间隔和过期时间。
func cleanupSettings(window time.Duration) (every, idleTTL time.Duration) {
	every = window
	if every < time.Minute {
		every = time.Minute
	}
	idleTTL = 2 * window
	if idleTTL < 10*time.Minute {
		idleTTL = 10 * time.Minute
	}
	return every, idleTTL
}
