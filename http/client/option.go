package client

import (
	"net/http"
	"time"

	"github.com/hktkhub/commonsdk/http/base"
	"github.com/hktkhub/commonsdk/http/codec"
)

type config struct {
	baseURL     string
	timeout     time.Duration
	transport   http.RoundTripper
	codec       codec.Codec
	userAgent   string
	propagation []string
	headerNames base.HeaderNames
	successCode int
	envelope    bool
}

func defaultConfig() *config {
	return &config{
		timeout:     30 * time.Second,
		codec:       codec.Default,
		successCode: 0,
		envelope:    true,
	}
}

// Option 配置 Client。
type Option func(*config)

// WithBaseURL 设置相对路径的基准地址，必须是绝对 URL。
func WithBaseURL(raw string) Option {
	return func(c *config) { c.baseURL = raw }
}

// WithTimeout 设置整体请求超时，默认 30s。
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// WithTransport 设置底层 RoundTripper，仍会被统一注入逻辑包装。
func WithTransport(rt http.RoundTripper) Option {
	return func(c *config) { c.transport = rt }
}

// WithCodec 设置编解码器，默认 JSON。
func WithCodec(cdc codec.Codec) Option {
	return func(c *config) { c.codec = cdc }
}

// WithUserAgent 设置缺省 User-Agent。
func WithUserAgent(ua string) Option {
	return func(c *config) { c.userAgent = ua }
}

// WithPropagationHeaders 覆盖默认透传 header 白名单。
func WithPropagationHeaders(names ...string) Option {
	return func(c *config) { c.propagation = append([]string(nil), names...) }
}

// WithHeaderNames 覆盖语义字段对应的实际 header 名。
// 未设置的字段回退到 base.DefaultHeaderNames()。
func WithHeaderNames(names base.HeaderNames) Option {
	return func(c *config) { c.headerNames = names }
}

// WithSuccessCode 设置统一响应体中表示成功的业务码，默认 0。
func WithSuccessCode(code int) Option {
	return func(c *config) { c.successCode = code }
}

// WithResponseEnvelope 控制是否按 {code,msg,data} 解析响应，默认开启。
// 对接第三方裸 JSON 时关闭。
func WithResponseEnvelope(enabled bool) Option {
	return func(c *config) { c.envelope = enabled }
}
