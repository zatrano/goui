package forms

import (
	"context"
	"html"
	"strings"

	"github.com/zatrano/goui/core"
)

// CalendarDatePicker is a visual date picker.
// Month navigation is client-side (UI-only); the selected YYYY-MM-DD lives on the server.
type CalendarDatePicker struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Value       string // YYYY-MM-DD
	Min         string
	Max         string
	Open        bool
	Placeholder string
	EventName   string
	OnChange    func(value string)
}

func (c *CalendarDatePicker) Name() string         { return c.CommonAttrs.Name }
func (c *CalendarDatePicker) RawValue() string     { return c.Value }
func (c *CalendarDatePicker) SetRawValue(v string) { c.Value = v }

func (c *CalendarDatePicker) Mount(_ context.Context) error   { return nil }
func (c *CalendarDatePicker) Unmount(_ context.Context) error { return nil }

func (c *CalendarDatePicker) Validate() bool {
	return c.FieldValidation.Run(c.Value, c.T)
}

func (c *CalendarDatePicker) eventName() string {
	if c.EventName != "" {
		return c.EventName
	}
	return c.CommonAttrs.Name
}

func (c *CalendarDatePicker) ev(action string) string {
	base := c.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (c *CalendarDatePicker) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, c.eventName())
	switch action {
	case "toggle":
		c.Open = !c.Open
		c.MarkDirty()
	case "close":
		c.Open = false
		c.MarkDirty()
	case "select":
		c.Value = payloadString(payload, "value")
		c.Open = false
		c.MarkDirty()
		if c.OnChange != nil {
			c.OnChange(c.Value)
		}
	case "clear":
		c.Value = ""
		c.MarkDirty()
		if c.OnChange != nil {
			c.OnChange(c.Value)
		}
	}
	return nil
}

func (c *CalendarDatePicker) displayLabel() string {
	if c.Value != "" {
		return c.Value
	}
	if c.Placeholder != "" {
		return c.Placeholder
	}
	return "Tarih seçin"
}

func (c *CalendarDatePicker) Render() (string, error) {
	attrs := Attrs{}
	attrs = c.CommonAttrs.Apply(attrs)
	attrs = c.FieldValidation.ApplyErrorState(attrs, "goui-calendar")
	attrs = attrs.Set("data-goui-calendar", "1")

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	b.WriteString(`<button type="button" class="goui-calendar-trigger border border-goui-border rounded-goui px-goui-field py-goui-field w-full" g-click="` + html.EscapeString(c.ev("toggle")) + `">`)
	b.WriteString(html.EscapeString(c.displayLabel()))
	b.WriteString(`</button>`)
	if c.Open {
		b.WriteString(`<div class="goui-searchable-backdrop" g-click="` + html.EscapeString(c.ev("close")) + `"></div>`)
		b.WriteString(`<div class="goui-calendar-panel border border-goui-border rounded-goui"`)
		b.WriteString(` data-goui-calendar-mount`)
		b.WriteString(` data-selected="` + html.EscapeString(c.Value) + `"`)
		b.WriteString(` data-min="` + html.EscapeString(c.Min) + `"`)
		b.WriteString(` data-max="` + html.EscapeString(c.Max) + `"`)
		b.WriteString(` data-select-event="` + html.EscapeString(c.ev("select")) + `"`)
		b.WriteString(`>`)
		// Client module fills the month grid (UI-only month state).
		b.WriteString(`<div class="goui-calendar-placeholder">Takvim yükleniyor…</div>`)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(c.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
