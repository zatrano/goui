package forms

import (
	"context"
	"html"

	"github.com/zatrano/goui/core"
)

// ChoiceInput covers type=checkbox|radio.
type ChoiceInput struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation
	Type      string // checkbox | radio
	Value     string // submitted value when checked
	Checked   bool
	EventName string
	LabelText string // optional adjacent label text (not a separate Label component)
	OnChange  func(checked bool, value string)
}

func (c *ChoiceInput) Name() string { return c.CommonAttrs.Name }

func (c *ChoiceInput) RawValue() string {
	if c.Checked {
		if c.Value != "" {
			return c.Value
		}
		return "on"
	}
	return ""
}

func (c *ChoiceInput) SetRawValue(v string) {
	c.Checked = v != "" && v != "false" && v != "0"
	if c.Checked && v != "on" && v != "true" && v != "1" {
		c.Value = v
	}
}

func (c *ChoiceInput) Mount(_ context.Context) error   { return nil }
func (c *ChoiceInput) Unmount(_ context.Context) error { return nil }

func (c *ChoiceInput) Validate() bool {
	return c.FieldValidation.run(c.RawValue(), c.T)
}

func (c *ChoiceInput) inputType() string {
	if c.Type == "" {
		return "checkbox"
	}
	return c.Type
}

func (c *ChoiceInput) eventName() string {
	if c.EventName != "" {
		return c.EventName
	}
	return c.CommonAttrs.Name
}

func (c *ChoiceInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = c.CommonAttrs.apply(attrs)
	attrs = c.FieldValidation.applyErrorState(attrs, classChoice)
	attrs = attrs.Set("type", c.inputType())
	attrs = attrs.Set("value", c.Value)
	attrs = attrs.SetBool("checked", c.Checked)
	if ev := c.eventName(); ev != "" {
		attrs = attrs.Set("g-change", ev)
	}
	htmlOut := "<input" + attrs.String() + ">"
	if c.LabelText != "" {
		htmlOut += " <span>" + html.EscapeString(c.LabelText) + "</span>"
	}
	htmlOut += c.FieldValidation.errorsHTML()
	return htmlOut, nil
}

func (c *ChoiceInput) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	if event != "change" && event != c.eventName() {
		return nil
	}
	c.Checked = payloadBool(payload, "checked")
	if v := payloadString(payload, "value"); v != "" {
		c.Value = v
	}
	c.MarkDirty()
	if c.OnChange != nil {
		c.OnChange(c.Checked, c.Value)
	}
	return nil
}

// CheckboxInput is an alias constructor helper naming for clarity in parent forms.
type CheckboxInput = ChoiceInput

// RadioInput is an alias for radio-typed ChoiceInput.
type RadioInput = ChoiceInput
