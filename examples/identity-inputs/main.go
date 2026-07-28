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

type Tier2FDemo struct {
	core.BaseComponent
	OTP      forms.OTPInput
	PIN      forms.OTPInput
	Country  forms.SearchableSelect
	Language forms.SearchableSelect
	TZ       forms.SearchableSelect
	Currency forms.SearchableSelect
	Phone    *forms.PhoneInput
}

func NewTier2FDemo() *Tier2FDemo {
	return &Tier2FDemo{
		OTP: forms.OTPInput{
			CommonAttrs: forms.CommonAttrs{Name: "otp", ID: "otp"},
			Length:      6,
			EventName:   "otp",
		},
		PIN: forms.OTPInput{
			CommonAttrs: forms.CommonAttrs{Name: "pin", ID: "pin"},
			Length:      4,
			Masked:      true,
			EventName:   "pin",
		},
		Country:  forms.NewCountryPicker("country", "country"),
		Language: forms.NewLanguagePicker("lang", "lang"),
		TZ:       forms.NewTimezonePicker("tz", "tz"),
		Currency: forms.NewCurrencyPicker("cur", "cur"),
		Phone:    forms.NewPhoneInput("phone"),
	}
}

func (d *Tier2FDemo) Mount(_ context.Context) error   { return nil }
func (d *Tier2FDemo) Unmount(_ context.Context) error { return nil }

func (d *Tier2FDemo) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	var err error
	switch {
	case strings.HasPrefix(event, "otp."):
		err = d.OTP.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "pin."):
		err = d.PIN.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "country."):
		err = d.Country.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "lang."):
		err = d.Language.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "tz."):
		err = d.TZ.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "cur."):
		err = d.Currency.HandleEvent(ctx, event, payload)
	case strings.HasPrefix(event, "phone_"):
		err = d.Phone.HandleEvent(ctx, event, payload)
	}
	d.MarkDirty()
	return err
}

func section(title, body string) string {
	return `<section class="demo-section"><h2>` + html.EscapeString(title) + `</h2>` + body + `</section>`
}

func (d *Tier2FDemo) Render() (string, error) {
	otp, err := d.OTP.Render()
	if err != nil {
		return "", err
	}
	pin, err := d.PIN.Render()
	if err != nil {
		return "", err
	}
	country, err := d.Country.Render()
	if err != nil {
		return "", err
	}
	lang, err := d.Language.Render()
	if err != nil {
		return "", err
	}
	tz, err := d.TZ.Render()
	if err != nil {
		return "", err
	}
	cur, err := d.Currency.Render()
	if err != nil {
		return "", err
	}
	phone, err := d.Phone.Render()
	if err != nil {
		return "", err
	}

	parts := []string{
		section("1. OTP", otp+`<p class="hint">Kod: <strong>`+html.EscapeString(d.OTP.Value)+`</strong></p>`),
		section("2. PIN", pin+`<p class="hint">Uzunluk: `+html.EscapeString(d.PIN.Value)+` (masked)</p>`),
		section("3. Country / Language / TZ / Currency",
			`<div class="field">`+country+`</div>`+
				`<div class="field">`+lang+`</div>`+
				`<div class="field">`+tz+`</div>`+
				`<div class="field">`+cur+`</div>`+
				`<p class="hint">`+html.EscapeString(d.Country.Value+" / "+d.Language.Value+" / "+d.TZ.Value+" / "+d.Currency.Value)+`</p>`),
		section("4. Phone Input", phone+`<p class="hint">E.164 benzeri: <strong>`+html.EscapeString(d.Phone.RawValue())+`</strong></p>`),
	}
	return `<div class="demo"><h1>Kimlik/Format</h1><p class="lead">OTP client auto-advance; picker'lar SearchableSelect + data.</p>` + strings.Join(parts, "") + `</div>`, nil
}

func main() {
	root := repoRoot()
	registry := core.NewRegistry()
	if err := registry.Register("identity", func() core.Component { return NewTier2FDemo() }); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Use("/forms", static.New(filepath.Join(root, "forms")))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(filepath.Join(root, "examples", "identity-inputs", "index.html"))
	})
	gouifiber.Register(app, gouifiber.Options{Server: ws.NewServer(ws.NewHub(), registry, i18n.NewTranslator())})

	log.Println("GoUI identity-inputs demo at http://localhost:3006")
	log.Fatal(app.Listen(":3006"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
