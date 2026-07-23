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
	"github.com/zatrano/goui/forms"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/validation"
	"github.com/zatrano/goui/ws"
)

// Landing is a lightweight home view used to demonstrate prefetch → activate.
type Landing struct {
	core.BaseComponent
}

func NewLanding() *Landing { return &Landing{} }

func (l *Landing) Mount(_ context.Context) error   { return nil }
func (l *Landing) Unmount(_ context.Context) error { return nil }

func (l *Landing) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

func (l *Landing) Render() (string, error) {
	return `<div class="landing">
  <p class="lede">Linkin üzerine gelince form arka planda hazırlanır; tıklayınca aktive edilir.</p>
  <p><a href="#" data-goui-prefetch="contact" data-goui-activate="contact">İletişim formuna git</a></p>
</div>`, nil
}

type ContactForm struct {
	core.BaseComponent
	Name      forms.TextInput
	Email     forms.TextInput
	Country   forms.Select
	Message   forms.Textarea
	Subscribe forms.ChoiceInput
	Submitted bool
	Summary   string
}

func NewContactForm(tr *i18n.Translator) *ContactForm {
	c := &ContactForm{
		Name: forms.TextInput{
			CommonAttrs:     forms.CommonAttrs{Name: "name", ID: "name", Required: true},
			Type:            "text",
			Placeholder:     "Adınız",
			FieldValidation: forms.FieldValidation{Rules: []validation.Rule{validation.Required()}},
		},
		Email: forms.TextInput{
			CommonAttrs:     forms.CommonAttrs{Name: "email", ID: "email", Required: true},
			Type:            "email",
			Placeholder:     "ornek@mail.com",
			FieldValidation: forms.FieldValidation{Rules: []validation.Rule{validation.Required(), validation.Email()}},
		},
		Country: forms.Select{
			CommonAttrs: forms.CommonAttrs{Name: "country", ID: "country"},
			Options: []forms.Option{
				{Value: "", Label: "Ülke seçin"},
				{Value: "tr", Label: "Türkiye"},
				{Value: "de", Label: "Almanya"},
				{Value: "us", Label: "United States"},
			},
		},
		Message: forms.Textarea{
			CommonAttrs: forms.CommonAttrs{Name: "message", ID: "message"},
			Rows:        4,
			Placeholder: "Mesajınız",
		},
		Subscribe: forms.ChoiceInput{
			CommonAttrs: forms.CommonAttrs{Name: "subscribe", ID: "subscribe"},
			Type:        "checkbox",
			Value:       "yes",
			LabelText:   "Bültene abone ol",
		},
	}
	c.SetTranslator(tr)
	c.Name.SetTranslator(tr)
	c.Email.SetTranslator(tr)
	c.Country.SetTranslator(tr)
	c.Message.SetTranslator(tr)
	c.Subscribe.SetTranslator(tr)
	return c
}

func (c *ContactForm) Mount(_ context.Context) error   { return nil }
func (c *ContactForm) Unmount(_ context.Context) error { return nil }

func (c *ContactForm) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	switch event {
	case "name":
		return c.Name.HandleEvent(ctx, event, payload)
	case "email":
		return c.Email.HandleEvent(ctx, event, payload)
	case "country":
		return c.Country.HandleEvent(ctx, event, payload)
	case "message":
		return c.Message.HandleEvent(ctx, event, payload)
	case "subscribe":
		return c.Subscribe.HandleEvent(ctx, event, payload)
	case "save":
		if !forms.ValidateAll(&c.Name, &c.Email, &c.Country, &c.Message, &c.Subscribe) {
			c.Submitted = false
			c.Summary = ""
			c.MarkDirty()
			return nil
		}
		c.Submitted = true
		sub := "hayır"
		if c.Subscribe.Checked {
			sub = "evet"
		}
		c.Summary = c.Name.Value + " | " + c.Email.Value + " | " + c.Country.Value + " | " + c.Message.Value + " | abone:" + sub
		c.Name.Errors = nil
		c.Email.Errors = nil
		c.ToastT("success", "contact.submit_success")
		c.MarkDirty()
	}
	return nil
}

func (c *ContactForm) Render() (string, error) {
	nameL, _ := (&forms.Label{For: "name", Text: "Ad"}).Render()
	nameI, _ := c.Name.Render()
	emailL, _ := (&forms.Label{For: "email", Text: "E-posta"}).Render()
	emailI, _ := c.Email.Render()
	countryL, _ := (&forms.Label{For: "country", Text: "Ülke"}).Render()
	countryI, _ := c.Country.Render()
	msgL, _ := (&forms.Label{For: "message", Text: "Mesaj"}).Render()
	msgI, _ := c.Message.Render()
	subI, _ := c.Subscribe.Render()
	btn, _ := (&forms.Button{Type: "button", Text: "Gönder", EventName: "save"}).Render()

	out := ""
	if c.Submitted {
		o, _ := (&forms.Output{
			CommonAttrs: forms.CommonAttrs{Name: "summary", Class: "goui-output"},
			Text:        c.Summary,
		}).Render()
		out = `<div class="result">` + o + `</div>`
	}

	inner := forms.JoinHTML(
		`<div class="field">`, nameL, nameI, `</div>`,
		`<div class="field">`, emailL, emailI, `</div>`,
		`<div class="field">`, countryL, countryI, `</div>`,
		`<div class="field">`, msgL, msgI, `</div>`,
		`<div class="field choice">`, subI, `</div>`,
		`<div class="actions">`, btn, `</div>`,
		out,
	)
	form := &forms.Form{Method: "post", OnSubmit: "save", InnerHTML: inner}
	html, err := form.Render()
	if err != nil {
		return "", err
	}
	c.ResetDirty()
	return html, nil
}

func main() {
	root := repoRoot()

	tr := i18n.NewTranslator()
	_ = tr.LoadLocale("tr", filepath.Join(root, "i18n", "locales", "tr.json"))
	_ = tr.LoadLocale("en", filepath.Join(root, "i18n", "locales", "en.json"))

	registry := core.NewRegistry()
	if err := registry.Register("landing", func() core.Component { return NewLanding() }); err != nil {
		log.Fatal(err)
	}
	if err := registry.Register("contact", func() core.Component { return NewContactForm(tr) }); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()
	app.Use("/client", static.New(filepath.Join(root, "client")))
	app.Use("/forms", static.New(filepath.Join(root, "forms")))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile(filepath.Join(root, "examples", "contact-form", "index.html"))
	})

	wsHub := ws.NewHub()
	gouifiber.Register(app, gouifiber.Options{Server: ws.NewServer(wsHub, registry, tr)})

	app.Get("/admin/broadcast", func(c fiber.Ctx) error {
		text := c.Query("text")
		if text == "" {
			text = "Sunucudan duyuru"
		}
		kind := c.Query("kind")
		if kind == "" {
			kind = "info"
		}
		wsHub.Broadcast(ws.PushMessage{Kind: kind, Text: text})
		return c.JSON(fiber.Map{"ok": true, "kind": kind, "text": text})
	})

	log.Println("GoUI contact form at http://localhost:3001")
	log.Println("Broadcast: GET http://localhost:3001/admin/broadcast?text=Merhaba&kind=info")
	log.Fatal(app.Listen(":3001"))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
