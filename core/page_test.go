package core

import (
	"context"
	"net/http"
	"testing"
)

func TestRequestContext(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/x?q=1", nil)
	ctx := ContextWithRequest(context.Background(), req)
	got := RequestFromContext(ctx)
	if got != req {
		t.Fatalf("RequestFromContext = %v, want same request", got)
	}
	if RequestFromContext(context.Background()) != nil {
		t.Fatal("expected nil")
	}
}

func TestPageModeString(t *testing.T) {
	if ModeLive.String() != "live" || ModeSEO.String() != "seo" || ModeStatic.String() != "static" {
		t.Fatalf("unexpected String values")
	}
}
