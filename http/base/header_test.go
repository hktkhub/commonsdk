package base_test

import (
	"net/http"
	"testing"

	"github.com/hktkhub/commonsdk/http/base"
)

func TestHeaderValueIsCaseInsensitive(t *testing.T) {
	h := http.Header{}
	h["X-Log-ID"] = []string{"exact"}
	h["X-Th-Env"] = []string{"canonical"}

	if got := base.HeaderValue(h, base.HeaderTraceID); got != "exact" {
		t.Fatalf("X-Log-ID = %q, want exact", got)
	}
	if got := base.HeaderValue(h, base.HeaderLane); got != "canonical" {
		t.Fatalf("X-TH-Env = %q, want canonical", got)
	}
	if got := base.HeaderValue(h, "x-log-id"); got != "exact" {
		t.Fatalf("x-log-id = %q, want exact", got)
	}
}

func TestHeaderValueEmpty(t *testing.T) {
	if got := base.HeaderValue(nil, base.HeaderTraceID); got != "" {
		t.Fatalf("nil header = %q, want empty", got)
	}
}
