# http 包

团队所有出站 HTTP 请求的统一入口。业务只把链路元数据放进 `context`，由 `transport` 在 `RoundTrip` 阶段统一提取并注入 header，从而保证 trace、泳道等上下文在所有出站请求中自动透传。

同时提供可选的 gin 响应规范中间件，不注册时完全不影响现有服务。

## 目录结构

```text
http/
├── base/        契约层：元数据、header 常量、名称映射、context 存取
├── propagate/   转换层：Metadata ↔ http.Header，白名单过滤
├── transport/   拦截层：包装官方 RoundTripper，统一注入
├── codec/       编解码层：请求/响应 body 的序列化与反序列化
├── client/      门面层：团队唯一使用的 Client
├── response/    共享响应契约：Envelope[T]
└── ginmw/       可选 gin 中间件：JSON API 响应规范
```

| 目录 | 文件 | 职责 |
|---|---|---|
| `base` | `metadata.go` | header 常量、默认白名单、`Metadata`、`HeaderNames`、链式写入方法 |
| `base` | `context.go` | `WithMetadata` / `MetadataFromContext`，使用非导出 key 避免冲突 |
| `propagate` | `propagate.go` | 入站 `FromHeader`、出站 `Inject`、显式 `ToHeader`，以及 traceparent 生成/解析 |
| `transport` | `transport.go` | `http.RoundTripper` 实现，所有出站请求的统一拦截点 |
| `codec` | `codec.go` | `Codec` 接口与默认 JSON 实现 |
| `client` | `client.go` | `New` / `Do` / `Get` / `Post` / `DoJSON`、URL 解析、错误归一 |
| `client` | `option.go` | Client 配置项：BaseURL、超时、Transport、Codec、User-Agent、header 白名单与名称映射 |
| `response` | `response.go` | 服务端与客户端共用的 `Envelope[T]` 响应契约 |
| `ginmw` | `ginmw.go` | 可选 gin 中间件，把 JSON API 响应包装成 `{code,msg,data}` |

## 分层依赖

```text
client ──▶ transport ──▶ propagate ──▶ base
   │
   └────▶ codec

ginmw ──▶ response
   └────▶ gin（仅该可选包依赖）
```

- 依赖严格单向，无循环。
- `base` 不依赖任何其他 http 子包，是整个包的稳定契约。
- `propagate` 只处理 header 语义，不关心网络、URL、body。
- `transport` 只做拦截和注入，不关心 codec 和业务路径。
- `client` 只做编排，不重复实现 header 细节。
- 核心包不依赖 gin 等 Web 框架，框架适配应放在独立中间件中。

## 请求链路

### 入站

服务端中间件把请求头提取成 `Metadata` 并写入 `context`：

```go
md := propagate.FromHeader(r.Header)
ctx := base.WithMetadata(r.Context(), md)
```

header 名与默认值不同时，使用同一份 `HeaderNames` 覆盖：

```go
names := base.HeaderNames{
    TraceID: "X-Trade-Log",
    Lane:    "X-Env",
}
md := propagate.FromHeaderWith(r.Header, names)
```

### 出站

```text
client.DoJSON(ctx, ...)
  → NewRequest / resolve URL
  → http.Client.Do
  → transport.RoundTrip
      → req.Clone()                    不修改原请求
      → propagate.InjectWith(ctx, h)   从 context 提取并按白名单注入
      → 官方 RoundTripper
```

## 语义与 header 对照

| 语义 | 默认 header | Metadata 字段 |
|---|---|---|
| Trace ID / 日志 ID | `X-Log-ID` | `TraceID` |
| 泳道 / 环境 | `X-TH-Env` | `Lane` |
| W3C traceparent | `traceparent` | `TraceParent` |
| W3C tracestate | `tracestate` | `TraceState` |
| Span ID | `X-Span-ID` | `SpanID` |
| 请求 ID | `X-Request-ID` | `RequestID` |
| 租户 ID | `X-Tenant-ID` | `TenantID` |
| 用户 ID | `X-User-ID` | `UserID` |
| 其他自定义 header | 由调用方指定 | `Extra` |

说明：

- `TraceID` 对应现网叫法 `X-Log-ID`，`Lane` 对应现网叫法 `X-TH-Env`，内部保持语义化命名。
- `traceparent` 只有 `TraceID` 为 32 位十六进制且存在合法 `SpanID` 时才会自动生成；否则不影响 `X-Log-ID` / `X-TH-Env` 透传。
- `Extra` 的 key 是 header 名，写入时大小写不敏感。

### Header 大小写

吐出时默认按 `HeaderNames` 中配置的原始拼写写入，`X-Log-ID` / `X-TH-Env` 会原样发到 HTTP/1.1 线上，不会被 Go canonical 化成 `X-Log-Id` / `X-Th-Env`。

读取统一使用 `base.HeaderValue(h, name)`，大小写不敏感：

- 优先匹配原始拼写 `X-Log-ID`
- 其次匹配 canonical 形式 `X-Log-Id`
- 最后遍历所有大小写变体

所以上游发 `X-Log-ID`、`X-Log-Id`，或 HTTP/2 把 header 名压成 `x-log-id`，都能正确读取。

关闭精确大小写：

```go
client.WithExactHeaderCase(false)
```

注意：

- 精确大小写只对 HTTP/1.1 出站有效；HTTP/2 协议要求 header 名小写，会变成 `x-log-id`。
- 读取不要求精确大小写，`HeaderValue` 已处理。
- 如果底层 Transport 也是会读取 header 的注入器（例如 `lokisdk.OutboundTransport`），请只保留一个注入器，否则可能出现大小写重复 header。

## 使用方式

### 1. 创建 Client

```go
c, err := client.New(
    client.WithBaseURL("http://trade-service"),
    client.WithTimeout(5*time.Second),
    client.WithUserAgent("trade-service"),
)
if err != nil {
    return err
}
```

默认 header 名已经是 `X-Log-ID` / `X-TH-Env`，通常不需要额外配置。只有服务间命名不一致时才需要 `WithHeaderNames`。

### 2. 写入链路元数据

```go
ctx = base.WithMetadata(ctx, base.Metadata{}.
    WithTraceID("log-1").
    WithLane("gray").
    WithRequestID("req-1"))
```

### 3. 发起请求

```go
var out DemoResponse
result, err := c.DoJSON(ctx, http.MethodPost, "/api/v1/demo", in, &out)
if err != nil {
    var apiErr *client.APIError
    if errors.As(err, &apiErr) {
        log.Warn("biz error", apiErr.Code, apiErr.Msg)
    }
    return err
}

_ = result.Code       // 业务码
_ = result.Msg        // 业务消息
_ = result.HTTPStatus // 原始 HTTP 状态码，例如 200 / 503
_ = result.Header     // 响应头
```

`DoJSON` 默认按统一响应解析：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {}
}
```

行为：

- 请求体按 `Codec` 编码，默认 JSON。
- 自动设置 `Content-Type` 和 `Accept`。
- `out` 自动接收 `data`。
- 成功码默认 `0`，可用 `WithSuccessCode` 覆盖（例如 200）。
- 2xx 且业务码等于成功码：返回 `Result, nil`。
- 2xx 但业务码失败：返回 `Result, *APIError`，`Code/Msg` 可直接读。
- HTTP 状态码 >= 400：返回 `Result, *APIError`，`Result.HTTPStatus` 保留原始状态码。
- 网络错误 / context 取消：返回 `nil, error`，没有 HTTP 响应可溯源。
- 204 或空 body：正常返回 `Result`，不执行解码。
- 错误响应体最多保留 8 KiB 在 `APIError.Body`。

对接第三方裸 JSON 时关闭响应壳层解析：

```go
c, err := client.New(
    client.WithBaseURL("https://third-party.example.com"),
    client.WithResponseEnvelope(false),
)
```

需要流式或特殊响应时，可以直接使用 `Do` / `Get` / `Post` 获取原始 `*http.Response`。

### 4. 自定义 header 名

```go
names := base.HeaderNames{
    TraceID: "X-Trade-Log",
    Lane:    "X-Env",
}

c, err := client.New(
    client.WithBaseURL("http://trade-service"),
    client.WithHeaderNames(names),
)
```

入站侧使用同一份 `names` 调用 `propagate.FromHeaderWith`，保证收发一致。

未填写的字段会自动回退到 `base.DefaultHeaderNames()`；白名单未显式设置时，会根据 `HeaderNames` 自动推导。

### 5. 新增自定义 header

```go
md := base.Metadata{}.WithExtra("X-Custom", "value")
ctx = base.WithMetadata(ctx, md)
```

`Extra` 默认不放行，需要显式加入白名单：

```go
c, err := client.New(
    client.WithBaseURL("http://trade-service"),
    client.WithPropagationHeaders(
        base.HeaderTraceID,
        base.HeaderLane,
        "X-Custom",
    ),
)
```

## 服务端响应规范（可选）

`http/ginmw` 提供可选的 gin 中间件，把 JSON API 响应统一成：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {}
}
```

**不注册中间件时，gin 的默认输出完全不变。**

### 注册方式

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/hktkhub/commonsdk/http/ginmw"
)

r := gin.New()
r.Use(gin.Recovery(), ginmw.Envelope())
```

### 处理规则

| Handler 输出 | 中间件输出 |
|---|---|
| `c.JSON(200, data)` | `{"code":0,"msg":"ok","data":data}` |
| `c.JSON(400, gin.H{"error":"bad"})` | HTTP 400 + `{"code":400,"msg":"bad","data":null}` |
| `c.JSON(200, response.OK(data))` | 已是统一结构，原样透传 |
| 空响应 | `{"code":0,"msg":"ok","data":null}` |
| `c.String` / HTML / 文件 / Stream / SSE | 跳过，不改变原响应 |
| 204 / 304 / 3xx / HEAD | 跳过 |
| 响应体超过 `MaxBuffer` | 跳过，原样透传 |
| 已调用 `Flush` / `Hijack` | 跳过，原样透传 |

错误响应的 `msg` 会依次从 body 的 `msg`、`message`、`error` 字段提取，没有时使用 HTTP 状态文本；`data` 固定为 `null`。

### 配置

```go
r.Use(ginmw.Envelope(
    ginmw.WithSuccess(0, "ok"),
    ginmw.WithErrorCode(func(status int) int {
        return status // 默认 HTTP 状态码作为业务码
    }),
    ginmw.WithMaxBuffer(1<<20),
    ginmw.WithSkip(func(c *gin.Context) bool {
        return c.FullPath() == "/metrics"
    }),
))
```

### 注意事项

- 只处理 JSON API，流式、SSE、文件下载、非 JSON 响应明确跳过。
- 中间件会缓冲 JSON 响应体，默认上限 1 MiB，超过上限自动透传。
- 与 gzip 中间件一起使用时，`Envelope` 要注册在 gzip 之后（内层），否则会尝试解析压缩后的字节。
- `gin.Recovery()` 建议在 `Envelope` 外层；发生 panic 时中间件不会写入半成品 envelope。
- 已经按 `{code,msg,data}` 返回的 handler 会被识别并透传，不会二次包裹。
- `Envelope` 与 client 解析使用同一套 `{code,msg,data}` 契约。

## 接入现有服务

### 步骤 1：引入依赖

```bash
go get github.com/hktkhub/commonsdk@latest
```

按需导入：

```go
import (
    "github.com/hktkhub/commonsdk/http/base"
    "github.com/hktkhub/commonsdk/http/client"
    "github.com/hktkhub/commonsdk/http/propagate"
)
```

### 步骤 2：启动时初始化 Client

Client 内部持有连接池，必须作为单例在服务启动时创建，不要每个请求新建：

```go
type Dependencies struct {
    TradeClient *client.Client
}

func NewDependencies() (*Dependencies, error) {
    trade, err := client.New(
        client.WithBaseURL(cfg.TradeServiceURL),
        client.WithTimeout(3*time.Second),
        client.WithUserAgent("trade-service"),
    )
    if err != nil {
        return nil, err
    }
    return &Dependencies{TradeClient: trade}, nil
}
```

### 步骤 3：注册入站中间件

中间件只做一件事：把入站 header 提取成 `Metadata`，放进请求的 `context`。

gin：

```go
func TraceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        md := propagate.FromHeader(c.Request.Header)
        ctx := base.WithMetadata(c.Request.Context(), md)
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}
```

net/http：

```go
func WithTrace(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        md := propagate.FromHeader(r.Header)
        next.ServeHTTP(w, r.WithContext(base.WithMetadata(r.Context(), md)))
    })
}
```

header 名不一致时，入站和出站使用同一份 `HeaderNames`：

```go
names := base.HeaderNames{TraceID: "X-Trade-Log", Lane: "X-Env"}

md := propagate.FromHeaderWith(c.Request.Header, names)
ctx := base.WithMetadata(c.Request.Context(), md)

trade, _ := client.New(
    client.WithBaseURL(cfg.TradeServiceURL),
    client.WithHeaderNames(names),
)
```

### 步骤 4：出站调用替换

业务代码只拿 `context` 调用 Client，不手工拼 header：

```go
func (s *Service) GetOrder(ctx context.Context, id string) (*Order, error) {
    var out Order
    _, err := s.deps.TradeClient.DoJSON(ctx, http.MethodGet, "/api/v1/orders/"+id, nil, &out)
    if err != nil {
        return nil, err
    }
    return &out, nil
}
```

只要 `ctx` 来自请求链路，trace / 泳道会自动透传，业务代码不需要感知。

替换清单：

```bash
grep -R "http.Get\|http.Post\|http.DefaultClient\|&http.Client{" --include="*.go" .
```

这些调用应逐步替换为 `client.Client`。

### 步骤 5：入口服务补充元数据

网关、BFF 等链路起点如果没有上游 header，需要自己生成并写入 `context`：

```go
md, _ := base.MetadataFromContext(ctx)
if md.TraceID == "" {
    md = md.WithTraceID(newLogID()) // 复用现有日志 ID 生成逻辑
}
if md.Lane == "" {
    md = md.WithLane(defaultLane())
}
ctx = base.WithMetadata(ctx, md)
```

中间服务不需要这一步，入站中间件已经把上游值放进 `context`。

### 步骤 6：接入检查

- [ ] 服务启动时只创建一个 `client.Client`
- [ ] 入站中间件已注册，且在所有业务 handler 之前执行
- [ ] 出站调用统一走 `client.Client`，不再直接使用 `http.Get` / `http.Post` / `http.DefaultClient`
- [ ] 请求链路中的 `ctx` 一路传递，没有被 `context.Background()` 截断
- [ ] header 名不一致的服务，入站和出站配置了同一份 `HeaderNames`
- [ ] CI 增加静态检查，禁止新增裸 `http.Client` 出站调用

静态检查示例（golangci-lint `forbidigo`）：

```yaml
linters-settings:
  forbidigo:
    forbid:
      - p: '^http\.(Get|Post|PostForm|Head)$'
      - p: '^http\.DefaultClient$'
      - p: '^http\.Client\{'
```

### 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 下游收不到 trace | 中间件未注册，或业务用了 `context.Background()` | 确保使用请求 `context` |
| 手工设置的 header 被覆盖 | context 中的元数据优先 | 统一从 context 取值 |
| 自定义 header 没透传 | 不在白名单 | 加入 `WithPropagationHeaders` |
| 不同服务 header 名不一致 | 未配置 `HeaderNames` | 入站、出站使用同一份映射 |
| 连接数持续增长 | 每个请求新建 Client | 改为启动时单例 |

## 安全约束

- **只透传白名单内的 header**，`Authorization`、`Cookie` 等敏感头默认不会带出。
- **context 优先**：同一 header 同时被手工设置和 context 提供时，以 context 为准，保证链路值唯一。
- **不修改原请求**：`transport` 先 `Clone` 再注入，调用方请求和重试逻辑不受影响。
- **`Extra` 不是后门**：放入 `Extra` 只表示“有这么一个值”，仍需白名单放行。
- **User-Agent 例外**：只有调用方未设置时才补默认值。

## 错误处理

| 场景 | 返回 |
|---|---|
| 网络错误、context 取消 | `nil, 官方 Go error`，保持原错误链 |
| HTTP >= 400 且 body 是统一响应 | `Result, *client.APIError`，Code/Msg 取 body |
| HTTP >= 400 且 body 不是统一响应 | `Result, *client.APIError`，Code=0，Msg 为 HTTP 状态摘要 |
| 2xx 但业务码非成功 | `Result, *client.APIError`，HTTPStatus 仍为 200 |
| 2xx 且业务码成功 | `Result, nil` |
| 3xx | 交由 `http.Client` 的重定向策略处理 |
| 请求编码失败 | `nil, http/client: encode request` 包装错误 |
| 响应解码失败 | `Result, http/client: decode response` 包装错误 |

两个状态码要区分：

- `Result.HTTPStatus` / `APIError.HTTPStatus`：HTTP 协议状态，来自 `http.Response.StatusCode`。
- `Result.Code` / `APIError.Code`：业务响应体里的 `code`。

所以 HTTP 200 也可能返回业务错误；HTTP 503 也可以从 `Result.HTTPStatus` 溯源。

后续如与统一错误码包整合，可以把 `APIError.Code` 换成 `error/base.Code`，并让 `APIError` 实现 `Code()` 契约。

## 扩展点

- **重试 / 熔断 / 指标 / 日志**：继续以 `http.RoundTripper` 装饰器叠加，建议组织为 `retry(inject(base))`，保证每次重试都重新注入 header。
- **入站中间件**：为 gin / net/http 提供适配，把 `FromHeader` 结果写入 `context`，避免核心包依赖框架。
- **Codec**：新增 form、protobuf 等实现，无需修改 Client 编排。
- **错误整合**：把 `APIError` 接入统一错误码体系。

## 当前边界

已实现：

- 出站统一入口与 header 注入
- 默认白名单与按服务覆盖 header 名
- W3C traceparent 解析与补齐
- 相对 URL、整体超时、JSON 编解码
- 统一响应 `{code,msg,data}` 解析与 `Result` / `APIError`
- 非 2xx 与业务错误归一
- 可选 gin 响应规范中间件 `http/ginmw`

未实现：

- trace / 泳道入站中间件（当前只提供提取函数和接入示例）
- 自动生成 TraceID / SpanID
- 重试、熔断、指标、日志
- 流式 / SSE 专用 API

## 验证

```bash
go vet ./...
go test ./...
```
