package transport_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hktkhub/commonsdk/http/base"
	"github.com/hktkhub/commonsdk/http/transport"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRoundTripInjectsMetadata(t *testing.T) {
	var got *http.Request
	rt := transport.New(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		got = r
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	}))

	ctx := base.WithMetadata(context.Background(), base.Metadata{
		TraceID: "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:  "00f067aa0ba902b7",
		Lane:    "lane-a",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	if got == nil {
		t.Fatal("underlying transport was not called")
	}
	if values := got.Header[base.HeaderLane]; len(values) == 0 || values[0] != "lane-a" {
		t.Fatalf("lane = %v, want lane-a under exact key %q", values, base.HeaderLane)
	}
	if _, ok := got.Header[http.CanonicalHeaderKey(base.HeaderLane)]; ok {
		t.Fatalf("unexpected canonical duplicate %q", http.CanonicalHeaderKey(base.HeaderLane))
	}
	if values := got.Header[base.HeaderTraceParent]; len(values) == 0 || values[0] == "" {
		t.Fatal("traceparent was not injected under exact key")
	}
	if req.Header.Get(base.HeaderLane) != "" {
		t.Fatal("original request must not be modified")
	}
}

func TestRoundTripCanonicalHeaderCaseWhenDisabled(t *testing.T) {
	var got *http.Request
	rt := transport.New(
		roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			got = r
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("ok")),
			}, nil
		}),
		transport.WithExactHeaderCase(false),
	)

	ctx := base.WithMetadata(context.Background(), base.Metadata{Lane: "lane-a"})
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}

	canonical := http.CanonicalHeaderKey(base.HeaderLane)
	if values := got.Header[canonical]; len(values) == 0 || values[0] != "lane-a" {
		t.Fatalf("lane = %v, want lane-a under canonical key %q", values, canonical)
	}
	if _, ok := got.Header[base.HeaderLane]; ok && base.HeaderLane != canonical {
		t.Fatalf("unexpected exact duplicate %q", base.HeaderLane)
	}
}

func TestRoundTripUserAgent(t *testing.T) {
	var got *http.Request
	rt := transport.New(
		roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			got = r
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("ok")),
			}, nil
		}),
		transport.WithUserAgent("commonsdk"),
	)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("User-Agent", "existing")
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if got.Header.Get("User-Agent") != "existing" {
		t.Fatalf("User-Agent = %q, want existing", got.Header.Get("User-Agent"))
	}
}
