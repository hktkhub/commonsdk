package propagate_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hktkhub/commonsdk/http/base"
	"github.com/hktkhub/commonsdk/http/propagate"
)

const (
	traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	spanID  = "00f067aa0ba902b7"
)

func TestFromHeaderParsesTraceParent(t *testing.T) {
	h := http.Header{}
	h.Set(base.HeaderTraceParent, "00-"+traceID+"-"+spanID+"-01")
	h.Set(base.HeaderLane, "lane-a")

	md := propagate.FromHeader(h)

	if md.TraceID != traceID {
		t.Fatalf("TraceID = %q, want %q", md.TraceID, traceID)
	}
	if md.SpanID != spanID {
		t.Fatalf("SpanID = %q, want %q", md.SpanID, spanID)
	}
	if md.Lane != "lane-a" {
		t.Fatalf("Lane = %q, want lane-a", md.Lane)
	}
}

func TestInjectBuildsTraceParent(t *testing.T) {
	ctx := base.WithMetadata(context.Background(), base.Metadata{
		TraceID:   traceID,
		SpanID:    spanID,
		Lane:      "lane-a",
		RequestID: "req-1",
	})

	h := http.Header{}
	propagate.Inject(ctx, h)

	if got, want := h.Get(base.HeaderTraceParent), "00-"+traceID+"-"+spanID+"-01"; got != want {
		t.Fatalf("traceparent = %q, want %q", got, want)
	}
	if got := h.Get(base.HeaderLane); got != "lane-a" {
		t.Fatalf("lane = %q, want lane-a", got)
	}
	if got := h.Get(base.HeaderRequestID); got != "req-1" {
		t.Fatalf("request id = %q, want req-1", got)
	}
}

func TestInjectHonorsWhitelist(t *testing.T) {
	ctx := base.WithMetadata(context.Background(), base.Metadata{
		TraceID: traceID,
		SpanID:  spanID,
		Lane:    "lane-a",
	})

	h := http.Header{}
	propagate.Inject(ctx, h, base.HeaderLane)

	if got := h.Get(base.HeaderLane); got != "lane-a" {
		t.Fatalf("lane = %q, want lane-a", got)
	}
	if got := h.Get(base.HeaderTraceParent); got != "" {
		t.Fatalf("traceparent = %q, want empty", got)
	}
}

func TestInjectWithoutMetadataIsNoop(t *testing.T) {
	h := http.Header{}
	propagate.Inject(context.Background(), h)
	if len(h) != 0 {
		t.Fatalf("header = %v, want empty", h)
	}
}

func TestFromHeaderReadsExactCaseKey(t *testing.T) {
	h := http.Header{}
	h[base.HeaderTraceID] = []string{"log-1"}
	h[base.HeaderLane] = []string{"gray"}

	md := propagate.FromHeader(h)

	if md.TraceID != "log-1" {
		t.Fatalf("TraceID = %q, want log-1", md.TraceID)
	}
	if md.Lane != "gray" {
		t.Fatalf("Lane = %q, want gray", md.Lane)
	}
}

func TestFromHeaderMapsLogIDAndEnvToCoreFields(t *testing.T) {
	h := http.Header{}
	h.Set(base.HeaderTraceID, "log-1")
	h.Set(base.HeaderLane, "gray")

	md := propagate.FromHeader(h)

	if md.TraceID != "log-1" {
		t.Fatalf("TraceID = %q, want log-1", md.TraceID)
	}
	if md.Lane != "gray" {
		t.Fatalf("Lane = %q, want gray", md.Lane)
	}
	if len(md.Extra) != 0 {
		t.Fatalf("Extra = %v, want empty", md.Extra)
	}
}

func TestInjectWritesLogIDAndEnvHeaders(t *testing.T) {
	ctx := base.WithMetadata(context.Background(), base.Metadata{
		TraceID: "log-1",
		Lane:    "gray",
	})

	h := http.Header{}
	propagate.Inject(ctx, h)

	if got := h.Get("X-Log-ID"); got != "log-1" {
		t.Fatalf("X-Log-ID = %q, want log-1", got)
	}
	if got := h.Get("X-TH-Env"); got != "gray" {
		t.Fatalf("X-TH-Env = %q, want gray", got)
	}
}

func TestInjectCustomExtraHeader(t *testing.T) {
	ctx := base.WithMetadata(context.Background(), base.Metadata{
		Extra: map[string]string{"X-Custom": "custom-value"},
	})

	h := http.Header{}
	propagate.Inject(ctx, h, "X-Custom")

	if got := h.Get("X-Custom"); got != "custom-value" {
		t.Fatalf("X-Custom = %q, want custom-value", got)
	}
}

func TestFromHeaderWithCustomNames(t *testing.T) {
	names := base.DefaultHeaderNames()
	names.TraceID = "X-Trade-Log"
	names.Lane = "X-Env"

	h := http.Header{}
	h.Set("X-Trade-Log", "log-9")
	h.Set("X-Env", "blue")

	md := propagate.FromHeaderWith(h, names)

	if md.TraceID != "log-9" {
		t.Fatalf("TraceID = %q, want log-9", md.TraceID)
	}
	if md.Lane != "blue" {
		t.Fatalf("Lane = %q, want blue", md.Lane)
	}
}

func TestInjectWithCustomNames(t *testing.T) {
	names := base.DefaultHeaderNames()
	names.TraceID = "X-Trade-Log"
	names.Lane = "X-Env"

	ctx := base.WithMetadata(context.Background(), base.Metadata{
		TraceID: "log-9",
		Lane:    "blue",
	})

	h := http.Header{}
	propagate.InjectWith(ctx, h, names)

	if got := h.Get("X-Trade-Log"); got != "log-9" {
		t.Fatalf("X-Trade-Log = %q, want log-9", got)
	}
	if got := h.Get("X-Env"); got != "blue" {
		t.Fatalf("X-Env = %q, want blue", got)
	}
	if got := h.Get(base.HeaderTraceID); got != "" {
		t.Fatalf("%s = %q, want empty", base.HeaderTraceID, got)
	}
}

func TestMetadataSetters(t *testing.T) {
	md := base.Metadata{}.
		WithTraceID("log-1").
		WithLane("gray").
		WithRequestID("req-1").
		WithTenantID("tenant-1").
		WithUserID("user-1").
		WithExtra("X-Custom", "custom-value")

	ctx := base.WithMetadata(context.Background(), md)
	h := http.Header{}
	propagate.Inject(ctx, h, append(base.PropagationHeaders(), "X-Custom")...)

	want := map[string]string{
		base.HeaderTraceID:   "log-1",
		base.HeaderLane:      "gray",
		base.HeaderRequestID: "req-1",
		base.HeaderTenantID:  "tenant-1",
		base.HeaderUserID:    "user-1",
		"X-Custom":           "custom-value",
	}
	for name, value := range want {
		if got := h.Get(name); got != value {
			t.Fatalf("%s = %q, want %q", name, got, value)
		}
	}
}

func TestInjectDropsExtraOutsideWhitelist(t *testing.T) {
	ctx := base.WithMetadata(context.Background(), base.Metadata{
		Extra: map[string]string{"Authorization": "Bearer secret"},
	})

	h := http.Header{}
	propagate.Inject(ctx, h, base.HeaderLane)

	if got := h.Get("Authorization"); got != "" {
		t.Fatalf("authorization = %q, want empty", got)
	}
}
