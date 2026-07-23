package forms

import (
	"context"
	"strings"

	"github.com/zatrano/goui/core"
)

// DateRangePicker holds start/end dates (YYYY-MM-DD) on the server.
type DateRangePicker struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Start     string
	End       string
	Min       string
	Max       string
	EventName string
	OnChange  func(start, end string)
}

func (d *DateRangePicker) Name() string { return d.CommonAttrs.Name }

func (d *DateRangePicker) RawValue() string {
	if d.Start == "" && d.End == "" {
		return ""
	}
	return d.Start + "/" + d.End
}

func (d *DateRangePicker) SetRawValue(v string) {
	parts := strings.SplitN(v, "/", 2)
	if len(parts) == 0 {
		d.Start, d.End = "", ""
		return
	}
	d.Start = parts[0]
	if len(parts) > 1 {
		d.End = parts[1]
	}
}

func (d *DateRangePicker) Mount(_ context.Context) error   { return nil }
func (d *DateRangePicker) Unmount(_ context.Context) error { return nil }

func (d *DateRangePicker) Validate() bool {
	ok := d.FieldValidation.Run(d.RawValue(), d.T)
	if d.Start != "" && d.End != "" && d.End < d.Start {
		msg := d.T("forms.date_range.invalid")
		if msg == "" || msg == "forms.date_range.invalid" {
			msg = "Bitiş, başlangıçtan önce olamaz"
		}
		d.Errors = append(d.Errors, msg)
		return false
	}
	return ok
}

func (d *DateRangePicker) eventName() string {
	if d.EventName != "" {
		return d.EventName
	}
	return d.CommonAttrs.Name
}

func (d *DateRangePicker) ev(action string) string {
	base := d.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (d *DateRangePicker) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, d.eventName())
	val := payloadString(payload, "value")
	switch action {
	case "start":
		d.Start = val
	case "end":
		d.End = val
	default:
		return nil
	}
	d.MarkDirty()
	if d.OnChange != nil {
		d.OnChange(d.Start, d.End)
	}
	return nil
}

func (d *DateRangePicker) Render() (string, error) {
	attrs := Attrs{}
	attrs = d.CommonAttrs.Apply(attrs)
	attrs = d.FieldValidation.ApplyErrorState(attrs, "goui-date-range")

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	b.WriteString(rangeDateInput("start", d.ev("start"), d.Start, d.Min, d.Max, d.End))
	b.WriteString(`<span class="goui-range-sep" aria-hidden="true">–</span>`)
	b.WriteString(rangeDateInput("end", d.ev("end"), d.End, maxStr(d.Min, d.Start), d.Max, ""))
	b.WriteString(`</div>`)
	b.WriteString(d.FieldValidation.ErrorsHTML())
	return b.String(), nil
}

func rangeDateInput(which, event, value, min, max, _ string) string {
	attrs := Attrs{}
	attrs = attrs.Set("type", "date")
	attrs = attrs.Set("class", classInput+" goui-range-input")
	attrs = attrs.Set("value", value)
	attrs = attrs.Set("min", min)
	attrs = attrs.Set("max", max)
	attrs = attrs.Set("data-goui-range", which)
	if event != "" {
		attrs = attrs.Set("g-change", event)
	}
	return "<input" + attrs.String() + ">"
}

func maxStr(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if b > a {
		return b
	}
	return a
}

// TimeRangePicker holds start/end times (HH:MM) on the server.
type TimeRangePicker struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Start     string
	End       string
	Min       string
	Max       string
	Step      string
	EventName string
	OnChange  func(start, end string)
}

func (t *TimeRangePicker) Name() string { return t.CommonAttrs.Name }

func (t *TimeRangePicker) RawValue() string {
	if t.Start == "" && t.End == "" {
		return ""
	}
	return t.Start + "/" + t.End
}

func (t *TimeRangePicker) SetRawValue(v string) {
	parts := strings.SplitN(v, "/", 2)
	if len(parts) == 0 {
		t.Start, t.End = "", ""
		return
	}
	t.Start = parts[0]
	if len(parts) > 1 {
		t.End = parts[1]
	}
}

func (t *TimeRangePicker) Mount(_ context.Context) error   { return nil }
func (t *TimeRangePicker) Unmount(_ context.Context) error { return nil }

func (t *TimeRangePicker) Validate() bool {
	ok := t.FieldValidation.Run(t.RawValue(), t.T)
	if t.Start != "" && t.End != "" && t.End < t.Start {
		msg := t.T("forms.time_range.invalid")
		if msg == "" || msg == "forms.time_range.invalid" {
			msg = "Bitiş saati, başlangıçtan önce olamaz"
		}
		t.Errors = append(t.Errors, msg)
		return false
	}
	return ok
}

func (t *TimeRangePicker) eventName() string {
	if t.EventName != "" {
		return t.EventName
	}
	return t.CommonAttrs.Name
}

func (t *TimeRangePicker) ev(action string) string {
	base := t.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (t *TimeRangePicker) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, t.eventName())
	val := payloadString(payload, "value")
	switch action {
	case "start":
		t.Start = val
	case "end":
		t.End = val
	default:
		return nil
	}
	t.MarkDirty()
	if t.OnChange != nil {
		t.OnChange(t.Start, t.End)
	}
	return nil
}

func (t *TimeRangePicker) Render() (string, error) {
	attrs := Attrs{}
	attrs = t.CommonAttrs.Apply(attrs)
	attrs = t.FieldValidation.ApplyErrorState(attrs, "goui-time-range")

	step := t.Step
	if step == "" {
		step = "60"
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	b.WriteString(rangeTimeInput(t.ev("start"), t.Start, t.Min, t.Max, step))
	b.WriteString(`<span class="goui-range-sep" aria-hidden="true">–</span>`)
	b.WriteString(rangeTimeInput(t.ev("end"), t.End, maxStr(t.Min, t.Start), t.Max, step))
	b.WriteString(`</div>`)
	b.WriteString(t.FieldValidation.ErrorsHTML())
	return b.String(), nil
}

func rangeTimeInput(event, value, min, max, step string) string {
	attrs := Attrs{}
	attrs = attrs.Set("type", "time")
	attrs = attrs.Set("class", classInput+" goui-range-input")
	attrs = attrs.Set("value", value)
	attrs = attrs.Set("min", min)
	attrs = attrs.Set("max", max)
	attrs = attrs.Set("step", step)
	if event != "" {
		attrs = attrs.Set("g-change", event)
	}
	return "<input" + attrs.String() + ">"
}
