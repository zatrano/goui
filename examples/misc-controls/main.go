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

type Tier2GDemo struct {
	core.BaseComponent
	Emoji    forms.SearchableSelect
	Icon     forms.SearchableSelect
	Font     forms.SearchableSelect
	Color    forms.SwatchColorPicker
	Gradient forms.GradientPicker
	Mention  forms.MentionTextarea
	Sign     forms.SignaturePad
}

func NewTier2GDemo() *Tier2GDemo {
	users := forms.MentionUsers()
	mu := make([]forms.MentionUser, 0, len(users))
	for _, u := range users {
		mu = append(mu, forms.MentionUser{ID: u.Value, Label: u.Label})
	}
	return &Tier2GDemo{
		Emoji:    forms.NewEmojiPicker("emoji", "emoji"),
		Icon:     forms.NewIconPicker("icon", "icon"),
		Font:     forms.NewFontPicker("font", "font"),
		Color:    forms.SwatchColorPicker{CommonAttrs: forms.CommonAttrs{Name: "color", ID: "color"}, Value: "#2563eb", EventName: "color"},
		Gradient: forms.GradientPicker{CommonAttrs: forms.CommonAttrs{Name: "grad", ID: "grad"}, From: "#2563eb", To: "#db2777", Angle: "135deg", EventName: "grad"},
		Mention: forms.MentionTextarea{
			CommonAttrs: forms.CommonAttrs{Name: "mention", ID: "mention"},
			Placeholder: "@ ile birini etiketle...",
			Users:       mu,
			EventName:   "mention",
			Rows:        4,
		},
		Sign: forms.SignaturePad{CommonAttrs: forms.CommonAttrs{Name: "sig", ID: "sig"}, EventName: "sig"},
	}
}

func (d *Tier2GDemo) Mount(_ context.Context) error   { return nil }
func (d *Tier2GDemo) Unmount(_ context.Context) error { return nil }

func (d *Tier2GDemo) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	var err error
	switch {
	case strings.HasPrefix(event, "emoji."):
		err = d.Emoji.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "icon."):
		err = d.Icon.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "font."):
		err = d.Font.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "color."):
		err = d.Color.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "grad."):
		err = d.Gradient.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "mention."):
		err = d.Mention.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "sig."):
		err = d.Sign.HandleEvent(ctx, event, payload)
	}
	d.MarkDirty()
	return err
}

func section(title, body string) string {
	return `<section class="demo-section"><h2>` + html.EscapeString(title) + `</h2>` + body + `</section>`
}

func (d *Tier2GDemo) Render() (string, error) {
	emoji, err := d.Emoji.Render()
	if err != nil {
		return "", err
	}
	icon, err := d.Icon.Render()
	if err != nil {
		return "", err
	}
	font, err := d.Font.Render()
	if err != nil {
		return "", err
	}
	color, err := d.Color.Render()
	if err != nil {
		return "", err
	}
	grad, err := d.Gradient.Render()
	if err != nil {
		return "", err
	}
	mention, err := d.Mention.Render()
	if err != nil {
		return "", err
	}
	sig, err := d.Sign.Render()
	if err != nil {
		return "", err
	}

	fontSample := `<p class="hint" style="font-family:` + html.EscapeString(d.Font.Value) + `">Örnek: The quick brown fox — ` + html.EscapeString(d.Font.SelectedLabel()) + `</p>`
	parts := []string{
		section("1. Emoji / Icon / Font", emoji+icon+font+fontSample+`<p class="hint">Seçili: `+html.EscapeString(d.Emoji.Value+" / "+d.Icon.Value)+`</p>`),
		section("2. Color (swatch) + Gradient", color+grad),
		section("3. Mention (@)", mention),
		section("4. Signature Pad", sig+`<p class="hint">Çiz → Kaydet → /goui/upload → server ref.</p>`),
	}
	return `<div class="demo"><h1>Misc controls</h1><p class="lead">Picker data + mention + imza. Char count / strength zaten G-küçük’te.</p>` + strings.Join(parts, "") + `</div>`, nil
}

func main() {
	root := repoRoot()
	store, err := upload.NewLocalStore(filepath.Join(root, ".goui-uploads"), "/goui/files", 8<<20)
	if err != nil {
		log.Fatal(err)
	}

	registry := core.NewRegistry()
	if err := registry.Register("misc", func() core.Component { return NewTier2GDemo() }); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Use("/forms", static.New(filepath.Join(root, "forms")))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(filepath.Join(root, "examples", "misc-controls", "index.html"))
	})
	gouifiber.Register(app, gouifiber.Options{
		Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator()),
		Store:  store,
	})

	log.Println("GoUI misc-controls demo at http://localhost:3009")
	log.Fatal(app.Listen(":3009"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
