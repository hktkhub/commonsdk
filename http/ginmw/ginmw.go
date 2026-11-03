// Package ginmw 提供可选的 gin 中间件。
//
// 显式注册 Envelope 时才会规范 JSON API 的响应格式：
//
//	r.Use(ginmw.Envelope())
//
// 不注册时完全不改变 gin 的默认行为。流式、SSE、文件、非 JSON 响应会被跳过。
// RateLimit 提供基于 ratelimit.Limiter 的限流中间件。
package ginmw

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hktkhub/commonsdk/http/response"
)

const defaultMaxBuffer = 1 << 20

type config struct {
	successCode int
	successMsg  string
	maxBuffer   int
	skip        func(*gin.Context) bool
	errorCode   func(status int) int
}

func defaultConfig() config {
	return config{
		successCode: 0,
		successMsg:  "ok",
		maxBuffer:   defaultMaxBuffer,
		errorCode:   func(status int) int { return status },
	}
}

// Option 配置 Envelope 中间件。
type Option func(*config)

// WithSuccess 设置成功响应的业务码与消息，默认 0 / "ok"。
func WithSuccess(code int, msg string) Option {
	return func(c *config) {
		c.successCode = code
		c.successMsg = msg
	}
}

// WithMaxBuffer 设置最大缓冲字节数，超过后原样透传，默认 1 MiB。
func WithMaxBuffer(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.maxBuffer = n
		}
	}
}

// WithSkip 按请求跳过规范化。
func WithSkip(fn func(*gin.Context) bool) Option {
	return func(c *config) {
		if fn != nil {
			c.skip = fn
		}
	}
}

// WithErrorCode 将 HTTP 状态码映射为业务码，默认直接使用状态码。
func WithErrorCode(fn func(status int) int) Option {
	return func(c *config) {
		if fn != nil {
			c.errorCode = fn
		}
	}
}

// Envelope 返回把 JSON API 响应包装为 {code,msg,data} 的 gin 中间件。
func Envelope(opts ...Option) gin.HandlerFunc {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return func(c *gin.Context) {
		if cfg.skip != nil && cfg.skip(c) {
			c.Next()
			return
		}
		if c.Request.Method == http.MethodHead {
			c.Next()
			return
		}

		original := c.Writer
		w := &bufferedWriter{
			ResponseWriter: original,
			maxBuffer:      cfg.maxBuffer,
		}
		c.Writer = w
		defer func() { c.Writer = original }()

		c.Next()
		w.finish(cfg)
	}
}

type bufferedWriter struct {
	gin.ResponseWriter

	buf         bytes.Buffer
	status      int
	size        int
	written     bool
	passthrough bool
	rawDone     bool
	maxBuffer   int
}

var _ gin.ResponseWriter = (*bufferedWriter)(nil)

func (w *bufferedWriter) WriteHeader(code int) {
	if code > 0 {
		w.status = code
	}
}

func (w *bufferedWriter) WriteHeaderNow() {
	w.written = true
	if w.status == 0 {
		w.status = http.StatusOK
	}
}

func (w *bufferedWriter) Write(p []byte) (int, error) {
	if w.passthrough {
		if err := w.flushRaw(); err != nil {
			return 0, err
		}
		n, err := w.ResponseWriter.Write(p)
		w.size += n
		return n, err
	}

	w.written = true
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.maxBuffer > 0 && w.buf.Len()+len(p) > w.maxBuffer {
		w.passthrough = true
		if _, err := w.buf.Write(p); err != nil {
			return 0, err
		}
		w.size += len(p)
		if err := w.flushRaw(); err != nil {
			return 0, err
		}
		return len(p), nil
	}

	n, err := w.buf.Write(p)
	w.size += n
	return n, err
}

func (w *bufferedWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *bufferedWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *bufferedWriter) Size() int { return w.size }

func (w *bufferedWriter) Written() bool { return w.written }

func (w *bufferedWriter) Flush() {
	if w.rawDone {
		w.ResponseWriter.Flush()
		return
	}
	w.passthrough = true
	_ = w.flushRaw()
	w.ResponseWriter.Flush()
}

func (w *bufferedWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.passthrough = true
	_ = w.flushRaw()
	return w.ResponseWriter.Hijack()
}

func (w *bufferedWriter) finish(cfg config) {
	if w.rawDone {
		return
	}
	if w.passthrough {
		_ = w.flushRaw()
		return
	}

	status := w.Status()
	if status < http.StatusOK || status == http.StatusNoContent || status == http.StatusNotModified ||
		(status >= http.StatusMultipleChoices && status < http.StatusBadRequest) {
		_ = w.flushRaw()
		return
	}

	contentType := w.Header().Get("Content-Type")
	if w.size > 0 {
		if contentType != "" && !isJSON(contentType) {
			_ = w.flushRaw()
			return
		}
		if contentType == "" && !json.Valid(w.buf.Bytes()) {
			_ = w.flushRaw()
			return
		}
	} else if contentType != "" && !isJSON(contentType) {
		_ = w.flushRaw()
		return
	}

	var data any
	if w.size > 0 {
		if err := json.Unmarshal(w.buf.Bytes(), &data); err != nil {
			_ = w.flushRaw()
			return
		}
		if isEnvelope(data) {
			_ = w.flushRaw()
			return
		}
	}

	envelope := response.Envelope[any]{Data: nil}
	if status >= http.StatusBadRequest {
		envelope.Code = cfg.errorCode(status)
		envelope.Msg = errorMessage(data, status)
	} else {
		envelope.Code = cfg.successCode
		envelope.Msg = cfg.successMsg
		envelope.Data = data
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		_ = w.flushRaw()
		return
	}

	w.Header().Del("Content-Length")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.ResponseWriter.WriteHeader(status)
	w.ResponseWriter.WriteHeaderNow()
	w.rawDone = true
	_, _ = w.ResponseWriter.Write(body)
}

func (w *bufferedWriter) flushRaw() error {
	if w.rawDone {
		return nil
	}
	status := w.Status()
	w.ResponseWriter.WriteHeader(status)
	w.ResponseWriter.WriteHeaderNow()
	w.rawDone = true

	if w.buf.Len() == 0 {
		return nil
	}
	_, err := w.ResponseWriter.Write(w.buf.Bytes())
	w.buf.Reset()
	return err
}

func isJSON(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "json")
}

func isEnvelope(data any) bool {
	m, ok := data.(map[string]any)
	if !ok {
		return false
	}
	_, code := m["code"]
	_, msg := m["msg"]
	_, dataField := m["data"]
	return code && msg && dataField
}

func errorMessage(data any, status int) string {
	if m, ok := data.(map[string]any); ok {
		for _, key := range []string{"msg", "message", "error"} {
			if value, ok := m[key].(string); ok && value != "" {
				return value
			}
		}
	}
	if value, ok := data.(string); ok && value != "" {
		return value
	}
	if text := http.StatusText(status); text != "" {
		return text
	}
	return "error"
}
