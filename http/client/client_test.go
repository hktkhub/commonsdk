package client_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hktkhub/commonsdk/http/base"
	"github.com/hktkhub/commonsdk/http/client"
)

func TestDoJSONPropagatesMetadata(t *testing.T) {
	var gotLane, gotTraceParent, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLane = r.Header.Get(base.HeaderLane)
		gotTraceParent = r.Header.Get(base.HeaderTraceParent)
		gotContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": map[string]string{"hello": "world"},
		})
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	ctx := base.WithMetadata(context.Background(), base.Metadata{
		TraceID: "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:  "00f067aa0ba902b7",
		Lane:    "lane-a",
	})

	var out map[string]string
	result, err := c.DoJSON(ctx, http.MethodPost, "/api/demo", map[string]string{"a": "b"}, &out)
	if err != nil {
		t.Fatal(err)
	}

	if gotLane != "lane-a" {
		t.Fatalf("lane = %q, want lane-a", gotLane)
	}
	if gotTraceParent == "" {
		t.Fatal("traceparent was not propagated")
	}
	if gotContentType != "application/json" {
		t.Fatalf("content-type = %q, want application/json", gotContentType)
	}
	if out["hello"] != "world" {
		t.Fatalf("response = %v, want hello=world", out)
	}
	if result.Code != 0 || result.Msg != "ok" {
		t.Fatalf("result = code %d msg %q, want 0/ok", result.Code, result.Msg)
	}
	if result.HTTPStatus != http.StatusOK {
		t.Fatalf("http status = %d, want 200", result.HTTPStatus)
	}
	if result.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("result header content-type = %q", result.Header.Get("Content-Type"))
	}
}

func TestDoJSONWithCustomHeaderNames(t *testing.T) {
	var gotTraceID, gotLane string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = r.Header.Get("X-Trade-Log")
		gotLane = r.Header.Get("X-Env")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	names := base.DefaultHeaderNames()
	names.TraceID = "X-Trade-Log"
	names.Lane = "X-Env"

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithHeaderNames(names),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := base.WithMetadata(context.Background(), base.Metadata{
		TraceID: "log-9",
		Lane:    "blue",
	})
	result, err := c.DoJSON(ctx, http.MethodGet, "/", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.HTTPStatus != http.StatusNoContent {
		t.Fatalf("http status = %d, want 204", result.HTTPStatus)
	}

	if gotTraceID != "log-9" {
		t.Fatalf("X-Trade-Log = %q, want log-9", gotTraceID)
	}
	if gotLane != "blue" {
		t.Fatalf("X-Env = %q, want blue", gotLane)
	}
}

func TestDoJSONEmitsExactHeaderCase(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	rawReq := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		br := bufio.NewReader(conn)
		var b strings.Builder
		for {
			line, err := br.ReadString('\n')
			b.WriteString(line)
			if err != nil || line == "\r\n" {
				break
			}
		}
		rawReq <- b.String()
		_, _ = conn.Write([]byte("HTTP/1.1 204 No Content\r\n\r\n"))
	}()

	c, err := client.New(
		client.WithBaseURL("http://"+ln.Addr().String()),
		client.WithTransport(&http.Transport{Proxy: nil}),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ctx = base.WithMetadata(ctx, base.Metadata{
		TraceID: "log-1",
		Lane:    "gray",
	})
	if _, err := c.DoJSON(ctx, http.MethodGet, "/", nil, nil); err != nil {
		t.Fatal(err)
	}

	select {
	case raw := <-rawReq:
		if !strings.Contains(raw, "X-Log-ID: log-1\r\n") {
			t.Fatalf("request did not contain exact X-Log-ID:\n%s", raw)
		}
		if !strings.Contains(raw, "X-TH-Env: gray\r\n") {
			t.Fatalf("request did not contain exact X-TH-Env:\n%s", raw)
		}
		if strings.Contains(raw, "X-Log-Id:") {
			t.Fatalf("request contained canonical X-Log-Id:\n%s", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for raw request")
	}
}

func TestDoJSONReturnsAPIErrorForHTTPStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "busy", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	result, err := c.DoJSON(context.Background(), http.MethodGet, "/", nil, nil)
	if result == nil || result.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("result = %v, want http 503", result)
	}

	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("http status = %d, want 503", apiErr.HTTPStatus)
	}
	if !strings.Contains(string(apiErr.Body), "busy") {
		t.Fatalf("body = %q, want busy", apiErr.Body)
	}
}

func TestDoJSONReturnsAPIErrorForBusinessCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 1001,
			"msg":  "quota exceeded",
			"data": nil,
		})
	}))
	defer srv.Close()

	c, err := client.New(client.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	result, err := c.DoJSON(context.Background(), http.MethodGet, "/", nil, nil)
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.HTTPStatus != http.StatusOK {
		t.Fatalf("http status = %d, want 200", result.HTTPStatus)
	}
	if result.Code != 1001 || result.Msg != "quota exceeded" {
		t.Fatalf("result = code %d msg %q", result.Code, result.Msg)
	}

	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.Code != 1001 || apiErr.Msg != "quota exceeded" {
		t.Fatalf("api error = code %d msg %q", apiErr.Code, apiErr.Msg)
	}
}

func TestDoJSONWithoutEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"hello": "world"})
	}))
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithResponseEnvelope(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]string
	result, err := c.DoJSON(context.Background(), http.MethodGet, "/", nil, &out)
	if err != nil {
		t.Fatal(err)
	}
	if out["hello"] != "world" {
		t.Fatalf("response = %v, want hello=world", out)
	}
	if result.Code != 0 || result.Msg != "" {
		t.Fatalf("result = code %d msg %q, want zeros", result.Code, result.Msg)
	}
}

func TestDoJSONCustomSuccessCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 200,
			"msg":  "ok",
			"data": nil,
		})
	}))
	defer srv.Close()

	c, err := client.New(
		client.WithBaseURL(srv.URL),
		client.WithSuccessCode(200),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := c.DoJSON(context.Background(), http.MethodGet, "/", nil, nil)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if result.Code != 200 {
		t.Fatalf("code = %d, want 200", result.Code)
	}
}

func TestNewRejectsRelativeBaseURL(t *testing.T) {
	if _, err := client.New(client.WithBaseURL("/relative")); err == nil {
		t.Fatal("expected error for relative base url")
	}
}
