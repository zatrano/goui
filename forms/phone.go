package forms

import (
	"context"
	"html"
	"strings"
)

// PhoneInput composes a dial-code SearchableSelect + national number TextInput.
// Not a separate control family — wiring helper for Alt-Grup F.
type PhoneInput struct {
	CommonAttrs
	FieldValidation

	Dial   SearchableSelect
	Number TextInput
}

func NewPhoneInput(name string) *PhoneInput {
	dialEvent := name + "_dial"
	numEvent := name + "_num"
	p := &PhoneInput{
		CommonAttrs: CommonAttrs{Name: name, ID: name},
		Dial: SearchableSelect{
			BaseSelectField: BaseSelectField{
				CommonAttrs: CommonAttrs{Name: dialEvent, ID: dialEvent},
				Placeholder: "Kod",
				Items:       DialCodeItems(),
				Value:       "+90",
			},
			EventName: dialEvent,
		},
		Number: TextInput{
			CommonAttrs: CommonAttrs{Name: numEvent, ID: numEvent},
			Type:        "tel",
			Placeholder: "5xx xxx xx xx",
			EventName:   numEvent,
			DebounceMS:  100,
		},
	}
	return p
}

func (p *PhoneInput) Name() string { return p.CommonAttrs.Name }

func (p *PhoneInput) RawValue() string {
	num := strings.TrimSpace(p.Number.Value)
	if num == "" {
		return p.Dial.Value
	}
	return strings.TrimSpace(p.Dial.Value + " " + num)
}

func (p *PhoneInput) SetRawValue(v string) {
	v = strings.TrimSpace(v)
	if v == "" {
		p.Dial.Value = ""
		p.Number.Value = ""
		return
	}
	parts := strings.SplitN(v, " ", 2)
	p.Dial.Value = parts[0]
	if len(parts) > 1 {
		p.Number.Value = parts[1]
	}
}

func (p *PhoneInput) Validate() bool {
	ok := p.FieldValidation.Run(p.RawValue(), p.Number.T)
	if !p.Number.Validate() {
		ok = false
	}
	return ok
}

func (p *PhoneInput) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	switch {
	case strings.HasPrefix(event, p.Dial.EventName):
		return p.Dial.HandleEvent(ctx, event, payload)
	case event == p.Number.EventName || strings.HasPrefix(event, p.Number.EventName):
		return p.Number.HandleEvent(ctx, event, payload)
	}
	return nil
}

func (p *PhoneInput) Render() (string, error) {
	dialHTML, err := p.Dial.Render()
	if err != nil {
		return "", err
	}
	numHTML, err := p.Number.Render()
	if err != nil {
		return "", err
	}
	attrs := Attrs{}
	attrs = p.CommonAttrs.Apply(attrs)
	attrs = p.FieldValidation.ApplyErrorState(attrs, "goui-phone")
	return `<div` + attrs.String() + `><div class="goui-phone-dial">` + dialHTML + `</div><div class="goui-phone-num">` + numHTML + `</div></div>` +
		p.FieldValidation.ErrorsHTML() + hintE164(p.RawValue()), nil
}

func hintE164(v string) string {
	if v == "" || v == "+90" {
		return ""
	}
	return `<p class="goui-helper-text text-sm">` + html.EscapeString(v) + `</p>`
}
