package core

import (
	"context"
	"net/http"
)

type requestContextKey struct{}

// ContextWithRequest stores the inbound *http.Request for Mount/Render
// (used by ModeSEO / ModeStatic page handlers).
func ContextWithRequest(ctx context.Context, r *http.Request) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil {
		return ctx
	}
	return context.WithValue(ctx, requestContextKey{}, r)
}

// RequestFromContext returns the *http.Request stored by ContextWithRequest.
func RequestFromContext(ctx context.Context) *http.Request {
	if ctx == nil {
		return nil
	}
	r, _ := ctx.Value(requestContextKey{}).(*http.Request)
	return r
}
