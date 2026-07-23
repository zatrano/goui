package forms

import (
	"context"
	"strconv"

	"github.com/zatrano/goui/core"
)

// TextInput covers type=text|password|email|search|tel|url.
type TextInput struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation
	Type          string
	Value         string
	Placeholder   string
	MinLength     int
	MaxLength     int
	Pattern       string
	Size          int
	Multiple      bool // email
	List          string
	EventName     string // g-change / g-input event name (defaults to Name)
	DebounceMS    int
	OnChange      func(newValue string)
	ShowCharCount bool
	ShowStrength  bool   // password strength meter (server-side)
	HelperText    string // hint below the field
}

func (t *TextInput) Name() string         { return t.CommonAttrs.Name }
func (t *TextInput) RawValue() string     { return t.Value }
func (t *TextInput) SetRawValue(v string) { t.Value = v }

func (t *TextInput) Mount(_ context.Context) error   { return nil }
func (t *TextInput) Unmount(_ context.Context) error { return nil }

func (t *TextInput) Validate() bool {
	return t.FieldValidation.run(t.Value, t.T)
}

func (t *TextInput) inputType() string {
	if t.Type == "" {
		return "text"
	}
	return t.Type
}

func (t *TextInput) eventName() string {
	if t.EventName != "" {
		return t.EventName
	}
	return t.CommonAttrs.Name
}

func (t *TextInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = t.CommonAttrs.apply(attrs)
	attrs = t.FieldValidation.applyErrorState(attrs, classInput)
	attrs = attrs.Set("type", t.inputType())
	attrs = attrs.Set("value", t.Value)
	attrs = attrs.Set("placeholder", t.Placeholder)
	attrs = attrs.Set("pattern", t.Pattern)
	attrs = attrs.Set("list", t.List)
	attrs = attrs.SetInt("minlength", t.MinLength)
	attrs = attrs.SetInt("maxlength", t.MaxLength)
	attrs = attrs.SetInt("size", t.Size)
	attrs = attrs.SetBool("multiple", t.Multiple)
	if ev := t.eventName(); ev != "" {
		attrs = attrs.Set("g-change", ev)
		attrs = attrs.Set("g-input", ev)
		if t.DebounceMS > 0 {
			attrs = attrs.Set("g-debounce", strconv.Itoa(t.DebounceMS))
		} else if t.ShowCharCount || t.ShowStrength {
			attrs = attrs.Set("g-debounce", "100")
		}
	}
	meta := fieldMetaHTML(t.HelperText, t.ShowCharCount, t.Value, t.MaxLength, t.ShowStrength && t.inputType() == "password", t.Value, t.T)
	return "<input" + attrs.String() + ">" + meta + t.FieldValidation.errorsHTML(), nil
}

func (t *TextInput) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	if event != "change" && event != "input" && event != t.eventName() {
		return nil
	}
	val := payloadString(payload, "value")
	t.Value = val
	t.MarkDirty()
	if t.OnChange != nil {
		t.OnChange(val)
	}
	return nil
}
