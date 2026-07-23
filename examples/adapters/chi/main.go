package main

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/go-chi/chi/v5"

	gouistdlib "github.com/zatrano/goui/adapters/stdlib"
	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/ws"
)

type Counter struct {
	core.BaseComponent
	Count int
}

func (c *Counter) Mount(_ context.Context) error { return nil }

func (c *Counter) Render() (string, error) {
	html, err := core.RenderTemplate(`<div class="counter">
<span class="count">{{.Count}}</span>
<button type="button" g-click="increment">+</button>
<button type="button" g-click="decrement">-</button>
</div>`, c)
	if err != nil {
		return "", err
	}
	c.ResetDirty()
	return html, nil
}

func (c *Counter) HandleEvent(_ context.Context, event string, _ map[string]any) error {
	switch event {
	case "increment":
		c.Count++
		c.MarkDirty()
	case "decrement":
		c.Count--
		c.MarkDirty()
	}
	return nil
}

func (c *Counter) Unmount(_ context.Context) error { return nil }

func main() {
	root := repoRoot()
	registry := core.NewRegistry()
	_ = registry.Register("counter", func() core.Component { return &Counter{} })

	r := chi.NewRouter()
	r.Handle("/client/*", http.StripPrefix("/client/", http.FileServer(http.Dir(filepath.Join(root, "client")))))
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, filepath.Join(root, "examples", "counter", "index.html"))
	})

	gouistdlib.Mount(r, gouistdlib.Options{
		Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator()),
	})

	log.Println("GoUI Chi adapter example at http://localhost:3011")
	log.Fatal(http.ListenAndServe(":3011", r))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
