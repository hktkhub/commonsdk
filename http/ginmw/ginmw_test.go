package ginmw_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hktkhub/commonsdk/http/ginmw"
	"github.com/hktkhub/commonsdk/http/response"
	"github.com/hktkhub/commonsdk/ratelimit"
)

func newRouter(opts ...ginmw.Option) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginmw.Envelope(opts...))
	return r
}

func serve(t *testing.T, r http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func decodeEnvelope(t *testing.T, body []byte) response.Envelope[map[string]any] {
	t.Helper()
	var env response.Envelope[map[string]any]
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v, body=%s", err, body)
	}
	return env
}

func TestEnvelopeWrapsJSONSuccess(t *testing.T) {
	r := newRouter()
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"hello": "world"})
	})

	w := serve(t, r, http.MethodGet, "/ok")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type = %q", w.Header().Get("Content-Type"))
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 0 || env.Msg != "ok" {
		t.Fatalf("envelope = code %d msg %q", env.Code, env.Msg)
	}
	if env.Data["hello"] != "world" {
		t.Fatalf("data = %v, want hello=world", env.Data)
	}
}

func TestEnvelopeWrapsJSONError(t *testing.T) {
	r := newRouter()
	r.GET("/err", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	w := serve(t, r, http.MethodGet, "/err")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", env.Code)
	}
	if env.Msg != "bad request" {
		t.Fatalf("msg = %q, want bad request", env.Msg)
	}
	if env.Data != nil {
		t.Fatalf("data = %v, want nil", env.Data)
	}
}

func TestEnvelopeKeepsExistingEnvelope(t *testing.T) {
	r := newRouter()
	r.GET("/wrapped", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.OK(gin.H{"hello": "world"}))
	})

	w := serve(t, r, http.MethodGet, "/wrapped")
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 0 || env.Msg != "ok" {
		t.Fatalf("envelope = code %d msg %q", env.Code, env.Msg)
	}
	if env.Data["hello"] != "world" {
		t.Fatalf("data = %v, want hello=world", env.Data)
	}
}

type codedError struct {
	code int64
	msg  string
}

func (e codedError) Error() string { return e.msg }

func (e codedError) Code() int64 { return e.code }

func TestEnvelopeReadsCodedError(t *testing.T) {
	r := newRouter()
	r.GET("/coded", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, response.FromError(
			codedError{code: 30043, msg: "quota exceeded"},
		))
	})

	w := serve(t, r, http.MethodGet, "/coded")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 30043 || env.Msg != "quota exceeded" {
		t.Fatalf("envelope = code %d msg %q", env.Code, env.Msg)
	}
}

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginmw.Envelope())
	r.Use(ginmw.RateLimit(
		ratelimit.NewSlidingWindow(1, time.Minute),
		func(c *gin.Context) string {
			return ratelimit.Join("user", c.GetHeader("X-User"))
		},
	))

	called := 0
	r.GET("/limited", func(c *gin.Context) {
		called++
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	serveUser := func(user string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/limited", nil)
		req.Header.Set("X-User", user)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	first := serveUser("alice")
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", first.Code)
	}
	second := serveUser("alice")
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429", second.Code)
	}
	env := decodeEnvelope(t, second.Body.Bytes())
	if env.Code != http.StatusTooManyRequests || env.Msg != "too many requests" {
		t.Fatalf("envelope = code %d msg %q", env.Code, env.Msg)
	}
	if env.Data != nil {
		t.Fatalf("data = %v, want nil", env.Data)
	}

	other := serveUser("bob")
	if other.Code != http.StatusOK {
		t.Fatalf("other user status = %d, want 200", other.Code)
	}
	if called != 2 {
		t.Fatalf("handler called %d times, want 2", called)
	}
}

func TestEnvelopeSkipsNonJSON(t *testing.T) {
	r := newRouter()
	r.GET("/text", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	w := serve(t, r, http.MethodGet, "/text")
	if w.Body.String() != "hello" {
		t.Fatalf("body = %q, want hello", w.Body.String())
	}
}

func TestEnvelopeSkipsNoContent(t *testing.T) {
	r := newRouter()
	r.GET("/none", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := serve(t, r, http.MethodGet, "/none")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", w.Body.String())
	}
}

func TestEnvelopeWrapsEmptySuccess(t *testing.T) {
	r := newRouter()
	r.GET("/empty", func(c *gin.Context) {})

	w := serve(t, r, http.MethodGet, "/empty")
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 0 || env.Msg != "ok" || env.Data != nil {
		t.Fatalf("envelope = %+v", env)
	}
}

func TestEnvelopeWithSkip(t *testing.T) {
	r := newRouter(ginmw.WithSkip(func(c *gin.Context) bool {
		return c.Request.URL.Path == "/raw"
	}))
	r.GET("/raw", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"hello": "world"})
	})

	w := serve(t, r, http.MethodGet, "/raw")
	var data map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if data["hello"] != "world" {
		t.Fatalf("body = %s, want raw object", w.Body.String())
	}
	if _, ok := data["code"]; ok {
		t.Fatalf("body was wrapped: %s", w.Body.String())
	}
}

func TestEnvelopeCustomCodes(t *testing.T) {
	r := newRouter(
		ginmw.WithSuccess(200, "success"),
		ginmw.WithErrorCode(func(status int) int { return 5000 + status }),
	)
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"x": 1})
	})
	r.GET("/err", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid"})
	})

	ok := decodeEnvelope(t, serve(t, r, http.MethodGet, "/ok").Body.Bytes())
	if ok.Code != 200 || ok.Msg != "success" {
		t.Fatalf("ok envelope = code %d msg %q", ok.Code, ok.Msg)
	}

	failed := decodeEnvelope(t, serve(t, r, http.MethodGet, "/err").Body.Bytes())
	if failed.Code != 5400 || failed.Msg != "invalid" {
		t.Fatalf("error envelope = code %d msg %q", failed.Code, failed.Msg)
	}
}

func TestEnvelopePassthroughWhenBufferExceeded(t *testing.T) {
	r := newRouter(ginmw.WithMaxBuffer(8))
	r.GET("/large", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"hello": "world"})
	})

	w := serve(t, r, http.MethodGet, "/large")
	if strings.Contains(w.Body.String(), `"code"`) {
		t.Fatalf("large response should be passthrough: %s", w.Body.String())
	}
}
