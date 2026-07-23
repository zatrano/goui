package forms

import (
	"context"
	"html"
	"strconv"

	"github.com/zatrano/goui/core"
)

// Textarea is a multi-line text control.
type Textarea struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation
	Value         string
	Placeholder   string
	Rows          int
	Cols          int
	Wrap          string
	MinLength     int
	MaxLength     int
	EventName     string
	DebounceMS    int
	OnChange      func(newValue string)
	ShowCharCount bool
	HelperText    string
}

func (t *Textarea) Name() string         { return t.CommonAttrs.Name }
func (t *Textarea) RawValue() string     { return t.Value }
func (t *Textarea) SetRawValue(v string) { t.Value = v }

func (t *Textarea) Mount(_ context.Context) error   { return nil }
func (t *Textarea) Unmount(_ context.Context) error { return nil }

func (t *Textarea) Validate() bool {
	return t.FieldValidation.run(t.Value, t.T)
}

func (t *Textarea) eventName() string {
	if t.EventName != "" {
		return t.EventName
	}
	return t.CommonAttrs.Name
}

func (t *Textarea) Render() (string, error) {
	attrs := Attrs{}
	attrs = t.CommonAttrs.apply(attrs)
	attrs = t.FieldValidation.applyErrorState(attrs, classTextarea)
	attrs = attrs.Set("placeholder", t.Placeholder)
	attrs = attrs.Set("wrap", t.Wrap)
	attrs = attrs.SetInt("rows", t.Rows)
	attrs = attrs.SetInt("cols", t.Cols)
	attrs = attrs.SetInt("minlength", t.MinLength)
	attrs = attrs.SetInt("maxlength", t.MaxLength)
	if ev := t.eventName(); ev != "" {
		attrs = attrs.Set("g-change", ev)
		attrs = attrs.Set("g-input", ev)
		if t.DebounceMS > 0 {
			attrs = attrs.Set("g-debounce", strconv.Itoa(t.DebounceMS))
		} else if t.ShowCharCount {
			attrs = attrs.Set("g-debounce", "100")
		}
	}
	meta := fieldMetaHTML(t.HelperText, t.ShowCharCount, t.Value, t.MaxLength, false, "", t.T)
	return "<textarea" + attrs.String() + ">" + html.EscapeString(t.Value) + "</textarea>" + meta + t.FieldValidation.errorsHTML(), nil
}

func (t *Textarea) HandleEvent(_ context.Context, event string, payload map[string]any) error {
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
