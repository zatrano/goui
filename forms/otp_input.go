package forms

import (
	"context"
	"html"
	"strconv"
	"strings"
	"unicode"

	"github.com/zatrano/goui/core"
)

// OTPInput is N single-character boxes; combined Value lives on the server.
type OTPInput struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Length    int // default 6
	Value     string
	Masked    bool // password-style cells (PIN)
	EventName string
	OnChange  func(value string)
}

// PINInput is an alias; set Masked: true for PIN UX.
type PINInput = OTPInput

func (o *OTPInput) Name() string         { return o.CommonAttrs.Name }
func (o *OTPInput) RawValue() string     { return o.Value }
func (o *OTPInput) SetRawValue(v string) { o.Value = o.normalize(v) }

func (o *OTPInput) Mount(_ context.Context) error   { return nil }
func (o *OTPInput) Unmount(_ context.Context) error { return nil }

func (o *OTPInput) Validate() bool {
	ok := o.FieldValidation.Run(o.Value, o.T)
	if len([]rune(o.Value)) != o.len() {
		msg := o.T("forms.otp.incomplete")
		if msg == "" || msg == "forms.otp.incomplete" {
			msg = "Kod eksik"
		}
		o.Errors = append(o.Errors, msg)
		return false
	}
	return ok
}

func (o *OTPInput) len() int {
	if o.Length <= 0 {
		return 6
	}
	return o.Length
}

func (o *OTPInput) eventName() string {
	if o.EventName != "" {
		return o.EventName
	}
	return o.CommonAttrs.Name
}

func (o *OTPInput) ev(action string) string {
	base := o.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (o *OTPInput) normalize(v string) string {
	var b strings.Builder
	for _, r := range v {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
		if utf8Len(b.String()) >= o.len() {
			break
		}
	}
	return b.String()
}

func utf8Len(s string) int { return len([]rune(s)) }

func (o *OTPInput) cells() []string {
	runes := []rune(o.Value)
	out := make([]string, o.len())
	for i := 0; i < o.len() && i < len(runes); i++ {
		out[i] = string(runes[i])
	}
	return out
}

func (o *OTPInput) setAt(index int, ch string) {
	n := o.len()
	if index < 0 || index >= n {
		return
	}
	slots := make([]rune, n)
	for i, r := range []rune(o.Value) {
		if i >= n {
			break
		}
		slots[i] = r
	}
	ch = strings.TrimSpace(ch)
	if ch == "" {
		slots[index] = 0
	} else {
		slots[index] = []rune(ch)[0]
	}
	var b strings.Builder
	for _, r := range slots {
		if r == 0 {
			break
		}
		b.WriteRune(r)
	}
	o.Value = b.String()
}

func (o *OTPInput) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, o.eventName())
	switch action {
	case "digit":
		o.setAt(payloadInt(payload, "index"), payloadString(payload, "value"))
	case "commit", "paste", "change", "input", o.eventName():
		o.Value = o.normalize(payloadString(payload, "value"))
	default:
		return nil
	}
	o.MarkDirty()
	if o.OnChange != nil {
		o.OnChange(o.Value)
	}
	return nil
}

func (o *OTPInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = o.CommonAttrs.Apply(attrs)
	attrs = o.FieldValidation.ApplyErrorState(attrs, "goui-otp")
	attrs = attrs.Set("data-goui-otp", "1")
	attrs = attrs.Set("data-goui-otp-commit", o.ev("commit"))

	inputType := "text"
	if o.Masked {
		inputType = "password"
	}
	cells := o.cells()

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	for i := 0; i < o.len(); i++ {
		b.WriteString(`<input class="goui-otp-cell border border-goui-border rounded-goui" type="` + inputType + `"`)
		b.WriteString(` inputmode="numeric" maxlength="1" autocomplete="one-time-code"`)
		b.WriteString(` value="` + html.EscapeString(cells[i]) + `"`)
		b.WriteString(` data-goui-index="` + strconv.Itoa(i) + `"`)
		b.WriteString(` g-input="` + html.EscapeString(o.ev("digit")) + `"`)
		b.WriteString(` g-debounce="50"`)
		b.WriteString(`>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(o.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
