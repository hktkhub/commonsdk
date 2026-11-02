package transport

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/hktkhub/commonsdk/http/base"
	"github.com/hktkhub/commonsdk/http/propagate"
)

// Transport 是所有出站请求的统一拦截点：
// 在 RoundTrip 中从 context 提取元数据并注入 trace/泳道 header。
type Transport struct {
	base      http.RoundTripper
	userAgent string
	whitelist []string
	names     base.HeaderNames
	exactCase bool
}

// Option 配置 Transport。
type Option func(*Transport)

// WithUserAgent 设置缺省 User-Agent，调用方已设置时不覆盖。
func WithUserAgent(ua string) Option {
	return func(t *Transport) { t.userAgent = ua }
}

// WithWhitelist 限定允许透传的 header，为空时使用 HeaderNames 推导出的白名单。
func WithWhitelist(names ...string) Option {
	return func(t *Transport) { t.whitelist = append([]string(nil), names...) }
}

// WithHeaderNames 覆盖语义字段对应的实际 header 名。
// 未设置的字段回退到 base.DefaultHeaderNames()。
func WithHeaderNames(names base.HeaderNames) Option {
	return func(t *Transport) { t.names = names }
}

// WithExactHeaderCase 控制是否按 HeaderNames 中的原始大小写写入 header。
// 默认开启；关闭后使用 net/http 的 canonical 形式。
func WithExactHeaderCase(enabled bool) Option {
	return func(t *Transport) { t.exactCase = enabled }
}

// New 包装官方 RoundTripper。next 为空时使用 http.DefaultTransport。
func New(next http.RoundTripper, opts ...Option) *Transport {
	if next == nil {
		next = http.DefaultTransport
	}
	t := &Transport{base: next, names: base.DefaultHeaderNames(), exactCase: true}
	for _, opt := range opts {
		if opt != nil {
			opt(t)
		}
	}
	return t
}

// RoundTrip 实现 http.RoundTripper，不修改调用方传入的请求。
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("http/transport: nil request")
	}

	clone := req.Clone(req.Context())
	if clone.Header == nil {
		clone.Header = make(http.Header)
	}
	propagate.InjectWith(clone.Context(), clone.Header, t.names, t.whitelist...)
	if t.userAgent != "" && clone.Header.Get("User-Agent") == "" {
		clone.Header.Set("User-Agent", t.userAgent)
	}
	if t.exactCase {
		normalizeHeaderCase(clone.Header, t.exactHeaderNames(clone.Context()))
	}
	return t.base.RoundTrip(clone)
}

func (t *Transport) exactHeaderNames(ctx context.Context) []string {
	names := t.names.WithDefaults().All()
	md, ok := base.MetadataFromContext(ctx)
	if !ok || len(md.Extra) == 0 {
		return names
	}

	seen := make(map[string]struct{}, len(names)+len(md.Extra))
	for _, name := range names {
		seen[http.CanonicalHeaderKey(name)] = struct{}{}
	}
	for name := range md.Extra {
		if name == "" {
			continue
		}
		key := http.CanonicalHeaderKey(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, name)
	}
	return names
}

// normalizeHeaderCase 删除大小写变体，并按 names 中的原始拼写重新写入。
func normalizeHeaderCase(h http.Header, names []string) {
	for _, name := range names {
		if name == "" {
			continue
		}
		var values []string
		found := false
		for key, v := range h {
			if !strings.EqualFold(key, name) {
				continue
			}
			if !found {
				values = append([]string(nil), v...)
				found = true
			}
			delete(h, key)
		}
		if found {
			h[name] = values
		}
	}
}
