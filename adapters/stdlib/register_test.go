package stdlib_test

import (
	"net/http"
	"testing"

	gouistdlib "github.com/zatrano/goui/adapters/stdlib"
	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/ws"
)

func TestRegister_MountsWSPath(t *testing.T) {
	mux := http.NewServeMux()
	server := ws.NewServer(ws.NewHub(), core.NewRegistry(), i18n.NewTranslator())
	defer server.Hub.Stop()

	gouistdlib.Register(mux, gouistdlib.Options{Server: server})

	req, _ := http.NewRequest(http.MethodGet, ws.Path, nil)
	_, pattern := mux.Handler(req)
	if pattern != ws.Path {
		t.Fatalf("pattern = %q, want %q", pattern, ws.Path)
	}
}
