package forms

import (
	"context"

	"github.com/zatrano/goui/core"
)

// DateTimeInput covers type=date|time|datetime-local|month|week.
type DateTimeInput struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation
	Type      string
	Value     string
	Min       string
	Max       string
	Step      string
	EventName string
	OnChange  func(newValue string)
}

func (d *DateTimeInput) Name() string         { return d.CommonAttrs.Name }
func (d *DateTimeInput) RawValue() string     { return d.Value }
func (d *DateTimeInput) SetRawValue(v string) { d.Value = v }

func (d *DateTimeInput) Mount(_ context.Context) error   { return nil }
func (d *DateTimeInput) Unmount(_ context.Context) error { return nil }

func (d *DateTimeInput) Validate() bool {
	return d.FieldValidation.run(d.Value, d.T)
}

func (d *DateTimeInput) inputType() string {
	if d.Type == "" {
		return "date"
	}
	return d.Type
}

func (d *DateTimeInput) eventName() string {
	if d.EventName != "" {
		return d.EventName
	}
	return d.CommonAttrs.Name
}

func (d *DateTimeInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = d.CommonAttrs.apply(attrs)
	attrs = d.FieldValidation.applyErrorState(attrs, classInput)
	attrs = attrs.Set("type", d.inputType())
	attrs = attrs.Set("value", d.Value)
	attrs = attrs.Set("min", d.Min)
	attrs = attrs.Set("max", d.Max)
	attrs = attrs.Set("step", d.Step)
	if ev := d.eventName(); ev != "" {
		attrs = attrs.Set("g-change", ev)
	}
	return "<input" + attrs.String() + ">" + d.FieldValidation.errorsHTML(), nil
}

func (d *DateTimeInput) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	if event != "change" && event != "input" && event != d.eventName() {
		return nil
	}
	val := payloadString(payload, "value")
	d.Value = val
	d.MarkDirty()
	if d.OnChange != nil {
		d.OnChange(val)
	}
	return nil
}
