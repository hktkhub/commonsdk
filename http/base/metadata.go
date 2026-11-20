package base

import "net/http"

// 出站请求统一传播的 header。
// HeaderTraceID / HeaderLane 使用现网已经在跑的 header 名，内部仍按 TraceID / Lane 语义处理。
const (
	HeaderTraceParent = "traceparent"
	HeaderTraceState  = "tracestate"

	HeaderTraceID = "X-Log-ID"
	HeaderSpanID  = "X-Span-ID"

	HeaderLane = "X-TH-Env"

	HeaderRequestID = "X-Request-ID"
	HeaderTenantID  = "X-Tenant-ID"
	HeaderUserID    = "X-User-ID"
)

var propagationHeaders = []string{
	HeaderTraceParent,
	HeaderTraceState,
	HeaderTraceID,
	HeaderSpanID,
	HeaderLane,
	HeaderRequestID,
	HeaderTenantID,
	HeaderUserID,
}

// PropagationHeaders 返回默认透传 header 白名单的副本。
func PropagationHeaders() []string {
	out := make([]string, len(propagationHeaders))
	copy(out, propagationHeaders)
	return out
}

// HeaderNames 定义语义字段到实际 header 名的映射。
// 不同服务可在各自入口覆盖字段，零值字段回退到 DefaultHeaderNames。
type HeaderNames struct {
	TraceParent string
	TraceState  string
	TraceID     string
	SpanID      string
	Lane        string
	RequestID   string
	TenantID    string
	UserID      string
}

// DefaultHeaderNames 返回包内默认的 header 映射。
func DefaultHeaderNames() HeaderNames {
	return HeaderNames{
		TraceParent: HeaderTraceParent,
		TraceState:  HeaderTraceState,
		TraceID:     HeaderTraceID,
		SpanID:      HeaderSpanID,
		Lane:        HeaderLane,
		RequestID:   HeaderRequestID,
		TenantID:    HeaderTenantID,
		UserID:      HeaderUserID,
	}
}

// WithDefaults 返回补齐零值字段后的副本。
func (n HeaderNames) WithDefaults() HeaderNames {
	d := DefaultHeaderNames()
	if n.TraceParent != "" {
		d.TraceParent = n.TraceParent
	}
	if n.TraceState != "" {
		d.TraceState = n.TraceState
	}
	if n.TraceID != "" {
		d.TraceID = n.TraceID
	}
	if n.SpanID != "" {
		d.SpanID = n.SpanID
	}
	if n.Lane != "" {
		d.Lane = n.Lane
	}
	if n.RequestID != "" {
		d.RequestID = n.RequestID
	}
	if n.TenantID != "" {
		d.TenantID = n.TenantID
	}
	if n.UserID != "" {
		d.UserID = n.UserID
	}
	return d
}

// All 返回补齐默认值后的全部 header 名，按大小写不敏感去重。
func (n HeaderNames) All() []string {
	n = n.WithDefaults()
	names := []string{
		n.TraceParent, n.TraceState, n.TraceID, n.SpanID,
		n.Lane, n.RequestID, n.TenantID, n.UserID,
	}
	out := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		key := http.CanonicalHeaderKey(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

// Metadata 是一次调用链上需要跨进程传播的元数据。
// TraceID 对应现网 X-Log-ID，Lane 对应现网 X-TH-Env。
// 核心字段提供类型安全和固定语义；Extra 承载后续新增的自定义 header，
// key 为 header 名，注入时同样受白名单约束。
// TraceParent 存在时优先透传原始值。
type Metadata struct {
	TraceParent string
	TraceState  string
	TraceID     string
	SpanID      string
	Lane        string
	RequestID   string
	TenantID    string
	UserID      string
	Extra       map[string]string
}

// WithTraceID 返回设置了链路 ID 的副本。
func (m Metadata) WithTraceID(value string) Metadata {
	m.TraceID = value
	return m
}

// WithLane 返回设置了泳道的副本。
func (m Metadata) WithLane(value string) Metadata {
	m.Lane = value
	return m
}

// WithRequestID 返回设置了请求 ID 的副本。
func (m Metadata) WithRequestID(value string) Metadata {
	m.RequestID = value
	return m
}

// WithTenantID 返回设置了租户 ID 的副本。
func (m Metadata) WithTenantID(value string) Metadata {
	m.TenantID = value
	return m
}

// WithUserID 返回设置了用户 ID 的副本。
func (m Metadata) WithUserID(value string) Metadata {
	m.UserID = value
	return m
}

// WithExtra 返回追加了自定义 header 的副本，不会修改原 map。
func (m Metadata) WithExtra(name, value string) Metadata {
	extra := make(map[string]string, len(m.Extra)+1)
	for key, v := range m.Extra {
		extra[key] = v
	}
	extra[name] = value
	m.Extra = extra
	return m
}

// IsZero 判断元数据是否为空。
func (m Metadata) IsZero() bool {
	if m.TraceParent != "" || m.TraceState != "" || m.TraceID != "" ||
		m.SpanID != "" || m.Lane != "" || m.RequestID != "" ||
		m.TenantID != "" || m.UserID != "" {
		return false
	}
	for _, value := range m.Extra {
		if value != "" {
			return false
		}
	}
	return true
}
