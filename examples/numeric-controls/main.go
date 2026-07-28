package main

import (
	"context"
	"html"
	"log"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	gouifiber "github.com/zatrano/goui/adapters/fiber"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/forms"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/ws"
)

type Tier2CDemo struct {
	core.BaseComponent
	Price forms.CurrencyInput
	VAT   forms.PercentageInput
	Score forms.Rating
}

func NewTier2CDemo() *Tier2CDemo {
	max := 100.0
	min := 0.0
	return &Tier2CDemo{
		Price: forms.CurrencyInput{
			CommonAttrs: forms.CommonAttrs{Name: "price", ID: "price"},
			Currency:    "TRY",
			Locale:      "tr",
			Value:       1250.5,
			EventName:   "price",
		},
		VAT: forms.PercentageInput{
			CommonAttrs: forms.CommonAttrs{Name: "vat", ID: "vat"},
			Locale:      "tr",
			Value:       20,
			Min:         &min,
			Max:         &max,
			EventName:   "vat",
		},
		Score: forms.Rating{
			CommonAttrs: forms.CommonAttrs{Name: "score", ID: "score"},
			Value:       3,
			Max:         5,
			EventName:   "score",
		},
	}
}

func (d *Tier2CDemo) Mount(_ context.Context) error   { return nil }
func (d *Tier2CDemo) Unmount(_ context.Context) error { return nil }

func (d *Tier2CDemo) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	var err error
	switch {
	case strings.HasPrefix(event, "price."):
		err = d.Price.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "vat."):
		err = d.VAT.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "score."):
		err = d.Score.HandleEvent(ctx, event, payload)
	}
	d.MarkDirty()
	return err
}

func section(title, body string) string {
	return `<section class="demo-section"><h2>` + html.EscapeString(title) + `</h2>` + body + `</section>`
}

func (d *Tier2CDemo) Render() (string, error) {
	priceL, _ := (&forms.Label{For: "price", Text: "Tutar (Currency)"}).Render()
	price, err := d.Price.Render()
	if err != nil {
		return "", err
	}
	vatL, _ := (&forms.Label{For: "vat", Text: "KDV (Percentage)"}).Render()
	vat, err := d.VAT.Render()
	if err != nil {
		return "", err
	}
	scoreL, _ := (&forms.Label{For: "score", Text: "Puan (Rating)"}).Render()
	score, err := d.Score.Render()
	if err != nil {
		return "", err
	}

	parts := []string{
		section("1. Currency Input", priceL+price+`<p class="hint">Server float64: <strong>`+html.EscapeString(strconv.FormatFloat(d.Price.Value, 'f', -1, 64))+`</strong> — blur/change ile TR format (`+html.EscapeString(d.Price.RawValue())+` ₺)</p>`),
		section("2. Percentage Input", vatL+vat+`<p class="hint">Değer: <strong>`+html.EscapeString(strconv.FormatFloat(d.VAT.Value, 'f', -1, 64))+`</strong>%</p>`),
		section("3. Rating", scoreL+score+`<p class="hint">Seçili: <strong>`+strconv.Itoa(d.Score.Value)+`</strong> / `+strconv.Itoa(d.Score.Max)+`</p>`),
	}
	return `<div class="demo"><h1>Sayısal &amp; Değerlendirme</h1><p class="lead">Formatlama server-side (tr: 1.234,56). Rating native g-click.</p>` + strings.Join(parts, "") + `</div>`, nil
}

func main() {
	root := repoRoot()
	registry := core.NewRegistry()
	if err := registry.Register("numeric", func() core.Component { return NewTier2CDemo() }); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Use("/forms", static.New(filepath.Join(root, "forms")))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(filepath.Join(root, "examples", "numeric-controls", "index.html"))
	})
	gouifiber.Register(app, gouifiber.Options{Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator())})

	log.Println("GoUI numeric-controls demo at http://localhost:3003")
	log.Fatal(app.Listen(":3003"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
