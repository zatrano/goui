package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	gouifiber "github.com/zatrano/goui/adapters/fiber"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/i18n"
	gouitemplate "github.com/zatrano/goui/template"
	"github.com/zatrano/goui/ws"
)

type Counter struct {
	core.BaseComponent
	Count int
}

func (c *Counter) Mount(_ context.Context) error { return nil }

func (c *Counter) View() string { return "counter" }

// Render satisfies core.Component; template.Wrap intercepts ViewComponent and
// never calls this when the factory is registered via Wrap.
func (c *Counter) Render() (string, error) {
	return "", gouitemplate.ErrViewRenderDirect
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
	exampleDir := filepath.Join(root, "examples", "counter-view")

	tmplReg, err := gouitemplate.NewRegistry(gouitemplate.Config{
		Root: filepath.Join(exampleDir, "views"),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer tmplReg.Close()

	registry := core.NewRegistry()
	if err := registry.Register("counter", gouitemplate.Wrap(tmplReg, func() core.Component {
		return &Counter{}
	})); err != nil {
		log.Fatal(err)
	}

	translator := i18n.NewTranslator()
	hub := ws.NewHub()

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(filepath.Join(exampleDir, "index.html"))
	})

	gouifiber.Register(app, gouifiber.Options{Server: ws.NewServer(hub, registry, translator)})

	log.Println("GoUI counter-view example at http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
