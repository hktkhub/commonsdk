package propagate

import (
	"context"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/hktkhub/commonsdk/http/base"
)

// FromHeader 使用默认 header 名从入站请求头提取元数据。
// allow 为空时使用默认白名单；不在白名单内的 header 会被忽略，
// 白名单内的非核心 header 会写入 Metadata.Extra。
func FromHeader(h http.Header, allow ...string) base.Metadata {
	return FromHeaderWith(h, base.DefaultHeaderNames(), allow...)
}

// FromHeaderWith 按 names 映射提取元数据，供 header 名不同的服务覆盖默认值。
// allow 为空时使用 names.All()。
func FromHeaderWith(h http.Header, names base.HeaderNames, allow ...string) base.Metadata {
	names = names.WithDefaults()
	if len(allow) == 0 {
		allow = names.All()
	}
	setters := fieldSetters(names)

	var md base.Metadata
	for _, name := range allow {
		value := strings.TrimSpace(h.Get(name))
		if value == "" {
			continue
		}
		key := http.CanonicalHeaderKey(name)
		if set, ok := setters[key]; ok {
			set(&md, value)
			continue
		}
		if md.Extra == nil {
			md.Extra = make(map[string]string)
		}
		md.Extra[key] = value
	}
	if md.TraceParent != "" {
		if traceID, spanID, ok := parseTraceParent(md.TraceParent); ok {
			if md.TraceID == "" {
				md.TraceID = traceID
			}
			if md.SpanID == "" {
				md.SpanID = spanID
			}
		}
	}
	return md
}

type fieldSetter func(*base.Metadata, string)

func fieldSetters(names base.HeaderNames) map[string]fieldSetter {
	setters := make(map[string]fieldSetter, 8)
	add := func(name string, set fieldSetter) {
		if name == "" {
			return
		}
		key := http.CanonicalHeaderKey(name)
		if _, exists := setters[key]; exists {
			return
		}
		setters[key] = set
	}
	add(names.TraceParent, func(m *base.Metadata, v string) { m.TraceParent = v })
	add(names.TraceState, func(m *base.Metadata, v string) { m.TraceState = v })
	add(names.TraceID, func(m *base.Metadata, v string) { m.TraceID = v })
	add(names.SpanID, func(m *base.Metadata, v string) { m.SpanID = v })
	add(names.Lane, func(m *base.Metadata, v string) { m.Lane = v })
	add(names.RequestID, func(m *base.Metadata, v string) { m.RequestID = v })
	add(names.TenantID, func(m *base.Metadata, v string) { m.TenantID = v })
	add(names.UserID, func(m *base.Metadata, v string) { m.UserID = v })
	return setters
}

// ToHeader 使用默认 header 名将元数据写入 header。
func ToHeader(md base.Metadata, h http.Header) {
	ToHeaderWith(md, h, base.DefaultHeaderNames())
}

// ToHeaderWith 按 names 映射将元数据写入 header。
// TraceParent 缺失时由 TraceID/SpanID 生成。
func ToHeaderWith(md base.Metadata, h http.Header, names base.HeaderNames) {
	if h == nil {
		return
	}
	for name, value := range headerValues(md, names) {
		h.Set(name, value)
	}
}

// Extract 从 context 读取元数据。
func Extract(ctx context.Context) base.Metadata {
	md, _ := base.MetadataFromContext(ctx)
	return md
}

// Inject 使用默认 header 名从 context 提取元数据并按白名单写入 header。
// allow 为空时使用默认白名单。
func Inject(ctx context.Context, h http.Header, allow ...string) {
	InjectWith(ctx, h, base.DefaultHeaderNames(), allow...)
}

// InjectWith 按 names 映射写入 header，供 header 名不同的服务覆盖默认值。
// allow 为空时使用 names.All()。
func InjectWith(ctx context.Context, h http.Header, names base.HeaderNames, allow ...string) {
	if h == nil {
		return
	}
	md, ok := base.MetadataFromContext(ctx)
	if !ok {
		return
	}

	names = names.WithDefaults()
	if len(allow) == 0 {
		allow = names.All()
	}
	allowed := make(map[string]struct{}, len(allow))
	for _, name := range allow {
		allowed[http.CanonicalHeaderKey(name)] = struct{}{}
	}

	for name, value := range headerValues(md, names) {
		if _, ok := allowed[http.CanonicalHeaderKey(name)]; !ok {
			continue
		}
		h.Set(name, value)
	}
}

func headerValues(md base.Metadata, names base.HeaderNames) map[string]string {
	names = names.WithDefaults()
	out := make(map[string]string, len(names.All())+len(md.Extra))
	put := func(name, value string) {
		if value != "" {
			out[http.CanonicalHeaderKey(name)] = value
		}
	}

	if md.TraceParent != "" {
		put(names.TraceParent, md.TraceParent)
	} else if tp, ok := buildTraceParent(md.TraceID, md.SpanID); ok {
		put(names.TraceParent, tp)
	}
	put(names.TraceState, md.TraceState)
	put(names.TraceID, md.TraceID)
	put(names.SpanID, md.SpanID)
	put(names.Lane, md.Lane)
	put(names.RequestID, md.RequestID)
	put(names.TenantID, md.TenantID)
	put(names.UserID, md.UserID)

	for name, value := range md.Extra {
		key := http.CanonicalHeaderKey(name)
		if key == "" || value == "" {
			continue
		}
		if _, exists := out[key]; exists {
			continue
		}
		out[key] = value
	}
	return out
}

func buildTraceParent(traceID, spanID string) (string, bool) {
	if !validHex(traceID, 32) || !validHex(spanID, 16) {
		return "", false
	}
	return "00-" + traceID + "-" + spanID + "-01", true
}

func parseTraceParent(value string) (string, string, bool) {
	parts := strings.Split(value, "-")
	if len(parts) != 4 {
		return "", "", false
	}
	if !validHex(parts[0], 2) || !validHex(parts[1], 32) || !validHex(parts[2], 16) || !validHex(parts[3], 2) {
		return "", "", false
	}
	if strings.Trim(parts[1], "0") == "" || strings.Trim(parts[2], "0") == "" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func validHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
