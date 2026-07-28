package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"

	"github.com/gin-gonic/gin"

	gouigin "github.com/zatrano/goui/adapters/gin"
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

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Static("/client", filepath.Join(root, "client"))
	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(root, "examples", "counter", "index.html"))
	})

	gouigin.Register(r, gouigin.Options{
		Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator()),
	})

	log.Println("GoUI Gin adapter example at http://localhost:3012")
	log.Fatal(r.Run(":3012"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
