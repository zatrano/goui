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
	"github.com/zatrano/goui/upload"
	"github.com/zatrano/goui/ws"
)

type Tier2DDemo struct {
	core.BaseComponent
	Docs   forms.DragDropUpload
	Images forms.DragDropUpload
	Avatar forms.AvatarUpload
}

func NewTier2DDemo() *Tier2DDemo {
	img := forms.NewImageUpload("images", "images")
	return &Tier2DDemo{
		Docs: forms.DragDropUpload{
			CommonAttrs: forms.CommonAttrs{Name: "docs", ID: "docs"},
			Multiple:    true,
			Accept:      ".pdf,.txt,.png,.jpg",
			ShowThumbs:  true, // görsel yüklenirse thumbnail; diğerleri yalnız listede
			EventName:   "docs",
		},
		Images: img,
		Avatar: forms.AvatarUpload{
			CommonAttrs: forms.CommonAttrs{Name: "avatar", ID: "avatar"},
			EventName:   "avatar",
		},
	}
}

func (d *Tier2DDemo) Mount(_ context.Context) error   { return nil }
func (d *Tier2DDemo) Unmount(_ context.Context) error { return nil }

func (d *Tier2DDemo) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	var err error
	switch {
	case strings.HasPrefix(event, "docs."):
		err = d.Docs.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "images."):
		err = d.Images.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "avatar."):
		err = d.Avatar.HandleEvent(ctx, event, payload)
	}
	d.MarkDirty()
	return err
}

func section(title, body string) string {
	return `<section class="demo-section"><h2>` + html.EscapeString(title) + `</h2>` + body + `</section>`
}

func (d *Tier2DDemo) Render() (string, error) {
	docs, err := d.Docs.Render()
	if err != nil {
		return "", err
	}
	images, err := d.Images.Render()
	if err != nil {
		return "", err
	}
	avatar, err := d.Avatar.Render()
	if err != nil {
		return "", err
	}
	parts := []string{
		section("1. Drag & Drop Upload", docs+`<p class="hint">Binary → POST /goui/upload; WS yalnızca ref (id).</p>`),
		section("2. Image Upload", images+`<p class="hint">accept=image/*, önizleme thumbnail.</p>`),
		section("3. Avatar + minimal crop", avatar+`<p class="hint">Seç → sürükle kaydır → kırp & yükle (1:1 PNG).</p>`),
	}
	return `<div class="demo"><h1>Dosya &amp; Medya</h1><p class="lead">Upload HTTP; state server'da file ID ile.</p>` + strings.Join(parts, "") + `</div>`, nil
}

func main() {
	root := repoRoot()
	store, err := upload.NewLocalStore(filepath.Join(root, ".goui-uploads"), "/goui/files", 8<<20)
	if err != nil {
		log.Fatal(err)
	}

	registry := core.NewRegistry()
	if err := registry.Register("media", func() core.Component { return NewTier2DDemo() }); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Use("/forms", static.New(filepath.Join(root, "forms")))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(filepath.Join(root, "examples", "media-upload", "index.html"))
	})
	gouifiber.Register(app, gouifiber.Options{
		Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator()),
		Store:  store,
	})

	log.Println("GoUI media-upload demo at http://localhost:3008")
	log.Fatal(app.Listen(":3008"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
