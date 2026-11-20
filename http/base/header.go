package base

import (
	"net/http"
	"strings"
)

// HeaderValue 按 header 名读取值，大小写不敏感。
// 优先匹配传入的原始拼写，其次 canonical 形式，最后遍历所有大小写变体。
func HeaderValue(h http.Header, name string) string {
	if h == nil || name == "" {
		return ""
	}
	if values := h[name]; len(values) > 0 {
		return values[0]
	}
	if canonical := http.CanonicalHeaderKey(name); canonical != name {
		if values := h[canonical]; len(values) > 0 {
			return values[0]
		}
	}
	for key, values := range h {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
