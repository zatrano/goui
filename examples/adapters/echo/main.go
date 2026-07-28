package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"

	"github.com/labstack/echo/v4"

	gouiecho "github.com/zatrano/goui/adapters/echo"
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

	e := echo.New()
	e.HideBanner = true
	e.Static("/client", filepath.Join(root, "client"))
	e.File("/", filepath.Join(root, "examples", "counter", "index.html"))

	gouiecho.Register(e, gouiecho.Options{
		Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator()),
	})

	log.Println("GoUI Echo adapter example at http://localhost:3013")
	log.Fatal(e.Start(":3013"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
