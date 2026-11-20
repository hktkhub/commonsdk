# ratelimit

本地限流包。对外只暴露一个判断接口，算法和维度解耦：

- 算法：令牌桶、滑动窗口
- 维度：由调用方传入 `key`
- 条件：由调用方决定什么时候调用 `Allow`

当前只提供本地实现，分布式 Redis 版本后续按需增加。

## 核心接口

```go
type Limiter interface {
    Allow(ctx context.Context, key string) bool
}
```

构造时传限流次数和时间窗口：

```go
func NewTokenBucket(limit int, window time.Duration, opts ...Option) Limiter
func NewSlidingWindow(limit int, window time.Duration, opts ...Option) Limiter
```

参数含义：`limit` 是窗口 `window` 内的允许次数。两个算法都按这个配置构造，但令牌桶允许突发，长期平均速率是 `limit/window`；滑动窗口严格保证任意滚动窗口内不超过 `limit` 次。

默认最多保留 10000 个 key，超过后按 LRU 淘汰最久未使用的 key：

```go
limiter := ratelimit.NewSlidingWindow(
    100,
    time.Second,
    ratelimit.WithMaxKeys(50_000),
)
```

`WithMaxKeys(0)` 表示不限制 key 数量，高基数场景不建议关闭。

## 使用

### 全局限流

```go
limiter := ratelimit.NewSlidingWindow(100, time.Second)

if !limiter.Allow(ctx, ratelimit.GlobalKey) {
    // 返回 429
}
```

### 按 IP + 用户限流

```go
limiter := ratelimit.NewTokenBucket(20, time.Second)

key := ratelimit.Join("ip", ip, "user", userID)
if !limiter.Allow(ctx, key) {
    // 返回 429
}
```

`Join` 使用长度前缀编码，避免简单拼接产生 key 冲突。

### 只对某类请求限流

条件判断留在业务代码里，不需要额外的规则层：

```go
if isHighCostRequest(req) && !limiter.Allow(ctx, key) {
    // 返回 429
}
```

### 按请求属性限流

把稳定的请求属性拼进 key：

```go
key := ratelimit.Join(
    "tenant", tenantID,
    "path", c.FullPath(),
    "method", c.Request.Method,
)
```

注意：把 request ID 作为 key 等于每个请求一个独立桶，起不到跨请求限流效果。

## gin 中间件

```go
limiter := ratelimit.NewSlidingWindow(100, time.Second)

engine.Use(
    middleware.RequestContext(),
    middleware.Response(),
    ginmw.RateLimit(limiter, func(c *gin.Context) string {
        return c.ClientIP()
    }),
)
```

按 IP + 用户限流：

```go
ginmw.RateLimit(limiter, func(c *gin.Context) string {
    return ratelimit.Join(
        "ip", c.ClientIP(),
        "user", c.GetString("userID"),
    )
})
```

- 超限返回 `429` + `{code:429,msg:"too many requests",data:null}`
- 与 `ginmw.Envelope()` 兼容，不会二次包裹
- 只对部分路由限流时，把中间件挂在对应的 `gin.Group` 上
- 需要自定义超限响应时，不用中间件，直接调用 `limiter.Allow(...)`

## 两种算法

| 算法 | 特点 | 适用 |
|---|---|---|
| `NewTokenBucket` | 桶容量为 `limit`，按 `limit/window` 匀速补充，允许突发 | 允许短时突发、长期速率受限 |
| `NewSlidingWindow` | 任意 `window` 内严格不超过 `limit` 次 | 对外配额、严格 QPS、计费接口 |

同样的 `limit` 和 `window`，两者行为不同：

- 令牌桶允许一开始打满 `limit` 次突发；
- 滑动窗口在任意滚动窗口内都不会超过 `limit` 次。

## 并发与内存

- `Allow` 线程安全，内部使用 `sync.Mutex`。
- 不使用后台 goroutine，令牌补充和窗口清理都是惰性执行。
- 每个 key 独立计数；key 数量通过 LRU 控制，默认上限 10000。
- 空闲 key 从 LRU 尾部删除，复杂度与删除数量成正比：
  - 扫描间隔：`max(window, 1m)`
  - 空闲过期：`max(2*window, 10m)`
- LRU 淘汰会重置被淘汰 key 的配额。本地限流可以接受；如果要求严格配额，应使用 Redis 分布式限流。
- 滑动窗口每个活跃 key 的内存占用约为 `limit * 8` 字节，总上限约为 `maxKeys * limit * 8`；`limit` 很大时要相应调小 `maxKeys`。

## 参数校验

以下情况会在构造时 panic，避免配置错误静默生效：

- `limit <= 0`
- `window <= 0`

## 分布式限流

当前不实现。后续如果需要，可以在同一 `Limiter` 接口下增加 Redis 实现：

```text
ratelimit/redis
```

需要注意：

- 用 Lua 脚本保证原子性；
- 使用 Redis `TIME` 而不是应用时钟；
- 设置 key TTL；
- 明确 Redis 超时/不可用时 fail-open 还是 fail-closed。

本地实现和 Redis 实现共用 `Allow(ctx, key)`，调用方不需要改业务代码。

## 验证

```bash
go test ./ratelimit/...
```
