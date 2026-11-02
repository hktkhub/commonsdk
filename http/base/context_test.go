package base_test

import (
	"context"
	"testing"

	"github.com/hktkhub/commonsdk/http/base"
)

func TestMetadataFromContextAcceptsExtraOnly(t *testing.T) {
	ctx := base.WithMetadata(context.Background(), base.Metadata{
		Extra: map[string]string{"X-Custom": "custom-value"},
	})

	md, ok := base.MetadataFromContext(ctx)
	if !ok {
		t.Fatal("metadata not found")
	}
	if got := md.Extra["X-Custom"]; got != "custom-value" {
		t.Fatalf("X-Custom = %q, want custom-value", got)
	}
}

func TestWithMetadataClonesExtra(t *testing.T) {
	extra := map[string]string{"X-Custom": "custom-value"}
	ctx := base.WithMetadata(context.Background(), base.Metadata{Extra: extra})

	extra["X-Custom"] = "mutated"

	md, _ := base.MetadataFromContext(ctx)
	if got := md.Extra["X-Custom"]; got != "custom-value" {
		t.Fatalf("X-Custom = %q, want custom-value", got)
	}
}
