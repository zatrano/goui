package main

import (
	"context"
	"html"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	gouifiber "github.com/zatrano/goui/adapters/fiber"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/forms"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/ws"
)

type Tier2EDemo struct {
	core.BaseComponent
	MD   forms.MarkdownEditor
	Rich forms.RichTextEditor
	Code forms.CodeEditor
}

func NewTier2EDemo() *Tier2EDemo {
	return &Tier2EDemo{
		MD: forms.MarkdownEditor{
			CommonAttrs: forms.CommonAttrs{Name: "md", ID: "md"},
			Value:       "# Merhaba\n\nGoUI **Markdown** önizlemesi server-side (`goldmark`).\n\n- madde 1\n- madde 2\n",
			Placeholder: "Markdown yazın...",
			EventName:   "md",
			Rows:        12,
		},
		Rich: forms.RichTextEditor{
			CommonAttrs: forms.CommonAttrs{Name: "rt", ID: "rt"},
			Value:       "<p>Rich text — <strong>Quill</strong> CDN</p>",
			EventName:   "rt",
		},
		Code: forms.CodeEditor{
			CommonAttrs: forms.CommonAttrs{Name: "code", ID: "code"},
			Value:       "function hello() {\n  return 'GoUI';\n}\n",
			Language:    "javascript",
			EventName:   "code",
		},
	}
}

func (d *Tier2EDemo) Mount(ctx context.Context) error {
	return d.MD.Mount(ctx)
}
func (d *Tier2EDemo) Unmount(_ context.Context) error { return nil }

func (d *Tier2EDemo) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	switch {
	case strings.HasPrefix(event, "md."):
		err := d.MD.HandleEvent(ctx, event, payload)
		d.MarkDirty()
		return err
	case strings.HasPrefix(event, "rt."):
		_ = d.Rich.HandleEvent(ctx, event, payload)
		// Quill owns the DOM — patching would remount and double-escape HTML.
		return core.ErrSkipRender
	case strings.HasPrefix(event, "code."):
		_ = d.Code.HandleEvent(ctx, event, payload)
		return core.ErrSkipRender
	}
	d.MarkDirty()
	return nil
}

func section(title, body string) string {
	return `<section class="demo-section"><h2>` + html.EscapeString(title) + `</h2>` + body + `</section>`
}

func (d *Tier2EDemo) Render() (string, error) {
	md, err := d.MD.Render()
	if err != nil {
		return "", err
	}
	rt, err := d.Rich.Render()
	if err != nil {
		return "", err
	}
	code, err := d.Code.Render()
	if err != nil {
		return "", err
	}
	parts := []string{
		section("1. Markdown Editor", md),
		section("2. Rich Text (Quill)", rt+`<p class="hint">İçerik server'da tutulur; UI client-owned (data-goui-ignore).</p>`),
		section("3. Code Editor (CodeMirror)", code+`<p class="hint">Yazmaya devam edilebilir; sync patch göndermez.</p>`),
	}
	return `<div class="demo"><h1>Metin Editörleri</h1><p class="lead">Markdown sunucuda render; Quill/CM CDN, içerik g-input ile sync.</p>` + strings.Join(parts, "") + `</div>`, nil
}

func main() {
	root := repoRoot()
	registry := core.NewRegistry()
	if err := registry.Register("editors", func() core.Component { return NewTier2EDemo() }); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Use("/forms", static.New(filepath.Join(root, "forms")))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(filepath.Join(root, "examples", "editors", "index.html"))
	})
	gouifiber.Register(app, gouifiber.Options{Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator())})

	log.Println("GoUI editors demo at http://localhost:3007")
	log.Fatal(app.Listen(":3007"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
