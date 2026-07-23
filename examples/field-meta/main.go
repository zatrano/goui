package main

import (
	"context"
	"html"
	"log"
	"path/filepath"
	"runtime"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	gouifiber "github.com/zatrano/goui/adapters/fiber"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/forms"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/ws"
)

type Tier2GSmallDemo struct {
	core.BaseComponent
	Bio forms.Textarea
	PW  forms.TextInput
}

func NewTier2GSmallDemo(tr *i18n.Translator) *Tier2GSmallDemo {
	d := &Tier2GSmallDemo{
		Bio: forms.Textarea{
			CommonAttrs:   forms.CommonAttrs{Name: "bio", ID: "bio"},
			Rows:          4,
			MaxLength:     120,
			Placeholder:   "Kendinizi tanıtın...",
			ShowCharCount: true,
			HelperText:    "En fazla 120 karakter",
			EventName:     "bio",
			DebounceMS:    100,
		},
		PW: forms.TextInput{
			CommonAttrs: forms.CommonAttrs{
				Name:  "pw",
				ID:    "pw",
				Title: "En az 8 karakter, büyük/küçük harf ve rakam önerilir",
			},
			Type:         "password",
			Placeholder:  "Şifre",
			ShowStrength: true,
			HelperText:   "Tooltip için alana hover (title)",
			EventName:    "pw",
			DebounceMS:   100,
		},
	}
	d.SetTranslator(tr)
	d.Bio.SetTranslator(tr)
	d.PW.SetTranslator(tr)
	return d
}

func (d *Tier2GSmallDemo) Mount(_ context.Context) error   { return nil }
func (d *Tier2GSmallDemo) Unmount(_ context.Context) error { return nil }

func (d *Tier2GSmallDemo) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	switch event {
	case "bio":
		_ = d.Bio.HandleEvent(ctx, event, payload)
	case "pw":
		_ = d.PW.HandleEvent(ctx, event, payload)
	}
	d.MarkDirty()
	return nil
}

func (d *Tier2GSmallDemo) Render() (string, error) {
	bioL, _ := (&forms.Label{For: "bio", Text: "Bio (Character Counter)"}).Render()
	bio, err := d.Bio.Render()
	if err != nil {
		return "", err
	}
	pwL, _ := (&forms.Label{For: "pw", Text: "Şifre (Password Strength)"}).Render()
	pw, err := d.PW.Render()
	if err != nil {
		return "", err
	}
	level := forms.PasswordStrength(d.PW.Value)
	return `<div class="demo"><h1>Field meta</h1><p class="lead">Character counter + password strength — server-side meta.</p>` +
		`<section class="demo-section"><h2>1. Character Counter</h2>` + bioL + bio + `</section>` +
		`<section class="demo-section"><h2>2. Password Strength</h2>` + pwL + pw +
		`<p class="hint">Seviye: <strong>` + html.EscapeString(level.LabelKey()) + `</strong></p></section>` +
		`</div>`, nil
}

func main() {
	root := repoRoot()
	tr := i18n.NewTranslator()
	_ = tr.LoadLocale("tr", filepath.Join(root, "i18n", "locales", "tr.json"))
	_ = tr.LoadLocale("en", filepath.Join(root, "i18n", "locales", "en.json"))

	registry := core.NewRegistry()
	if err := registry.Register("meta", func() core.Component { return NewTier2GSmallDemo(tr) }); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Use("/forms", static.New(filepath.Join(root, "forms")))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(filepath.Join(root, "examples", "field-meta", "index.html"))
	})
	gouifiber.Register(app, gouifiber.Options{Server: ws.NewServer(ws.NewHub(), registry, tr)})

	log.Println("GoUI field-meta demo at http://localhost:3004")
	log.Fatal(app.Listen(":3004"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
