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
	"github.com/zatrano/goui/page"
	"github.com/zatrano/goui/ws"
)

// Landing is a public SEO page: full HTML on GET, then live over WebSocket.
type Landing struct {
	core.BaseComponent
	Visits int
}

func (l *Landing) Mount(_ context.Context) error { return nil }

func (l *Landing) Head() core.Head {
	return core.Head{
		Title:         "GoUI — SEO landing",
		Description:   "Server-rendered first paint, then WebSocket interactivity.",
		OGTitle:       "GoUI SEO landing",
		OGDescription: "Hybrid ModeSEO example",
	}
}

func (l *Landing) Render() (string, error) {
	return core.RenderTemplate(`<main class="landing">
<h1>GoUI SEO</h1>
<p>This HTML was in the first HTTP response (crawlers see it).</p>
<p>Visits this session: <strong>{{.Visits}}</strong></p>
<button type="button" g-click="visit">+1 visit</button>
</main>`, l)
}

func (l *Landing) HandleEvent(_ context.Context, event string, _ map[string]any) error {
	if event == "visit" {
		l.Visits++
		l.MarkDirty()
	}
	return nil
}

func (l *Landing) Unmount(_ context.Context) error { return nil }

// About is static HTML only — no WebSocket client.
type About struct {
	core.BaseComponent
}

func (a *About) Mount(_ context.Context) error { return nil }

func (a *About) Head() core.Head {
	return core.Head{
		Title:       "About",
		Description: "Static ModeStatic page",
	}
}

func (a *About) Render() (string, error) {
	return `<main><h1>About</h1><p>No WebSocket on this page.</p></main>`, nil
}

func (a *About) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}
func (a *About) Unmount(_ context.Context) error { return nil }

// Admin stays ModeLive (empty shell + WS), typical for panels.
type Admin struct {
	core.BaseComponent
	N int
}

func (a *Admin) Mount(_ context.Context) error { return nil }

func (a *Admin) Render() (string, error) {
	return core.RenderTemplate(`<div class="admin">
<h1>Admin</h1>
<p>Count: {{.N}}</p>
<button type="button" g-click="inc">+</button>
</div>`, a)
}

func (a *Admin) HandleEvent(_ context.Context, event string, _ map[string]any) error {
	if event == "inc" {
		a.N++
		a.MarkDirty()
	}
	return nil
}

func (a *Admin) Unmount(_ context.Context) error { return nil }

func main() {
	root := repoRoot()
	registry := core.NewRegistry()
	_ = registry.RegisterPage("landing", func() core.Component { return &Landing{} }, core.ModeSEO)
	_ = registry.RegisterPage("about", func() core.Component { return &About{} }, core.ModeStatic)
	_ = registry.RegisterPage("admin", func() core.Component { return &Admin{} }, core.ModeLive)

	translator := i18n.NewTranslator()
	hub := ws.NewHub()
	server := ws.NewServer(hub, registry, translator)
	renderer := page.NewRenderer(page.Options{
		Registry:   registry,
		Translator: translator,
		Styles:     []string{"/styles.css"},
	})

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Get("/styles.css", func(c fiber.Ctx) error {
		c.Type("css")
		return c.SendString(`body{font-family:system-ui,sans-serif;max-width:36rem;margin:2rem auto;padding:0 1rem}
button{font-size:1rem;padding:.4rem .8rem;cursor:pointer}`)
	})

	gouifiber.Register(app, gouifiber.Options{
		Server: server,
		Page:   renderer,
		Routes: []page.Route{
			{Path: "/", Component: "landing"},
			{Path: "/about", Component: "about"},
			{Path: "/admin", Component: "admin"},
		},
	})

	log.Println("SEO example: http://localhost:3005/  /about  /admin")
	log.Fatal(app.Listen(":3005"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
