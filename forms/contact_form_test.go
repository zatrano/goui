package forms_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/diff"
	"github.com/zatrano/goui/forms"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/validation"
)

// ContactForm composes Tier 1 form controls — mirrors examples/contact-form.
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
			Placeholder:     "E-posta",
			FieldValidation: forms.FieldValidation{Rules: []validation.Rule{validation.Required(), validation.Email()}},
		},
		Country: forms.Select{
			CommonAttrs: forms.CommonAttrs{Name: "country", ID: "country"},
			Options: []forms.Option{
				{Value: "", Label: "Ülke seçin"},
				{Value: "tr", Label: "Türkiye"},
				{Value: "de", Label: "Almanya"},
			},
		},
		Message: forms.Textarea{CommonAttrs: forms.CommonAttrs{Name: "message", ID: "message"}, Rows: 4, Placeholder: "Mesajınız"},
		Subscribe: forms.ChoiceInput{
			CommonAttrs: forms.CommonAttrs{Name: "subscribe", ID: "subscribe"},
			Type:        "checkbox",
			Value:       "yes",
			LabelText:   "Bültene abone ol",
		},
	}
	if tr != nil {
		c.SetTranslator(tr)
		c.Name.SetTranslator(tr)
		c.Email.SetTranslator(tr)
		c.Country.SetTranslator(tr)
		c.Message.SetTranslator(tr)
		c.Subscribe.SetTranslator(tr)
	}
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
		c.MarkDirty()
		c.ToastT("success", "contact.submit_success")
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
		o, _ := (&forms.Output{CommonAttrs: forms.CommonAttrs{Name: "summary", Class: "goui-output"}, Text: c.Summary}).Render()
		out = o
	}

	inner := forms.JoinHTML(nameL, nameI, emailL, emailI, countryL, countryI, msgL, msgI, subI, btn, out)
	form := &forms.Form{Method: "post", OnSubmit: "save", InnerHTML: inner}
	return form.Render()
}

func loadTR(t *testing.T) *i18n.Translator {
	t.Helper()
	tr := i18n.NewTranslator()
	if err := tr.LoadLocale("tr", filepath.Join("..", "i18n", "locales", "tr.json")); err != nil {
		t.Fatalf("LoadLocale: %v", err)
	}
	return tr
}

func TestContactForm_EndToEnd(t *testing.T) {
	ctx := context.Background()
	form := NewContactForm(loadTR(t))
	if err := form.Mount(ctx); err != nil {
		t.Fatal(err)
	}

	html, err := form.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"type=\"text\"", "type=\"email\"", "<select", "<textarea", "type=\"checkbox\"", "g-submit=\"save\""} {
		if !strings.Contains(html, want) {
			t.Fatalf("initial render missing %q", want)
		}
	}

	_ = form.HandleEvent(ctx, "name", map[string]any{"value": "Serhan"})
	_ = form.HandleEvent(ctx, "email", map[string]any{"value": "s@example.com"})
	_ = form.HandleEvent(ctx, "country", map[string]any{"value": "tr"})
	_ = form.HandleEvent(ctx, "message", map[string]any{"value": "Merhaba"})
	_ = form.HandleEvent(ctx, "subscribe", map[string]any{"checked": true, "value": "yes"})
	_ = form.HandleEvent(ctx, "save", nil)

	if !form.Submitted {
		t.Fatal("expected Submitted")
	}
	if !strings.Contains(form.Summary, "Serhan") || !strings.Contains(form.Summary, "s@example.com") || !strings.Contains(form.Summary, "abone:evet") {
		t.Fatalf("bad summary: %q", form.Summary)
	}

	html, err = form.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<output") || !strings.Contains(html, "Serhan") {
		t.Fatalf("post-submit render missing output: %s", html)
	}
}

func TestContactForm_InvalidSubmit_PreservesValidFields(t *testing.T) {
	ctx := context.Background()
	form := NewContactForm(loadTR(t))

	_ = form.HandleEvent(ctx, "name", map[string]any{"value": "Serhan"})
	_ = form.HandleEvent(ctx, "email", map[string]any{"value": "not-an-email"})
	_ = form.HandleEvent(ctx, "save", nil)

	if form.Submitted {
		t.Fatal("invalid submit must not mark Submitted")
	}
	if form.Name.Value != "Serhan" {
		t.Fatalf("name state lost: %q", form.Name.Value)
	}
	if len(form.Email.Errors) == 0 {
		t.Fatal("expected email errors")
	}

	html, err := form.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `value="Serhan"`) {
		t.Fatalf("name value missing from render: %s", html)
	}
	if !strings.Contains(html, `value="not-an-email"`) {
		t.Fatalf("invalid email value missing from render: %s", html)
	}
	if !strings.Contains(html, "Geçerli bir e-posta adresi girin") {
		t.Fatalf("email error message missing: %s", html)
	}
	if !strings.Contains(html, `aria-invalid="true"`) {
		t.Fatalf("aria-invalid missing: %s", html)
	}
}

func TestContactForm_InvalidSubmit_MinimalPatch(t *testing.T) {
	ctx := context.Background()
	form := NewContactForm(loadTR(t))

	_ = form.HandleEvent(ctx, "name", map[string]any{"value": "Serhan"})
	_ = form.HandleEvent(ctx, "email", map[string]any{"value": "not-an-email"})

	beforeHTML, err := form.Render()
	if err != nil {
		t.Fatal(err)
	}

	_ = form.HandleEvent(ctx, "save", nil)

	afterHTML, err := form.Render()
	if err != nil {
		t.Fatal(err)
	}

	before, err := diff.ParseHTML(beforeHTML)
	if err != nil {
		t.Fatal(err)
	}
	after, err := diff.ParseHTML(afterHTML)
	if err != nil {
		t.Fatal(err)
	}

	patches := diff.Diff(before, after)
	if len(patches) == 0 {
		t.Fatal("expected patches for validation errors")
	}

	// Name value must remain unchanged in the HTML snapshot.
	if !strings.Contains(afterHTML, `value="Serhan"`) {
		t.Fatal("name value not preserved after invalid submit")
	}

	for _, p := range patches {
		if p.Op == diff.OpUpdateText && p.Text == "Serhan" {
			t.Fatalf("unexpected name text patch: %+v", p)
		}
		if p.Op == diff.OpSetAttr && p.Attr == "value" && p.Value == "Serhan" {
			t.Fatalf("unexpected name value attr patch: %+v", p)
		}
	}

	hasErrorSignal := false
	for _, p := range patches {
		if p.Op == diff.OpSetAttr && (p.Attr == "aria-invalid" || strings.Contains(p.Value, "border-goui-error")) {
			hasErrorSignal = true
		}
		if p.Op == diff.OpInsert && strings.Contains(p.HTML, "goui-field-error") {
			hasErrorSignal = true
		}
		if p.Op == diff.OpUpdateText && strings.Contains(p.Text, "e-posta") {
			hasErrorSignal = true
		}
		if p.Op == diff.OpReplace && strings.Contains(p.HTML, "goui-field-error") {
			hasErrorSignal = true
		}
	}
	if !hasErrorSignal {
		t.Fatalf("expected error-related patches, got %#v", patches)
	}
}
