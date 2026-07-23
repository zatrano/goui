package forms

import (
	"context"
	"html"
	"strings"

	"github.com/zatrano/goui/core"
)

// SwatchColorPicker is an advanced palette + hex field (beyond native type=color).
type SwatchColorPicker struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Value     string // #rrggbb
	Swatches  []string
	EventName string
	OnChange  func(value string)
}

func (c *SwatchColorPicker) Name() string         { return c.CommonAttrs.Name }
func (c *SwatchColorPicker) RawValue() string     { return c.Value }
func (c *SwatchColorPicker) SetRawValue(v string) { c.Value = v }

func (c *SwatchColorPicker) Mount(_ context.Context) error   { return nil }
func (c *SwatchColorPicker) Unmount(_ context.Context) error { return nil }

func (c *SwatchColorPicker) Validate() bool {
	return c.FieldValidation.Run(c.Value, c.T)
}

func (c *SwatchColorPicker) eventName() string {
	if c.EventName != "" {
		return c.EventName
	}
	return c.CommonAttrs.Name
}

func (c *SwatchColorPicker) ev(action string) string {
	base := c.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (c *SwatchColorPicker) swatches() []string {
	if len(c.Swatches) > 0 {
		return c.Swatches
	}
	return []string{
		"#111827", "#dc2626", "#ea580c", "#ca8a04", "#16a34a",
		"#0891b2", "#2563eb", "#7c3aed", "#db2777", "#ffffff",
	}
}

func (c *SwatchColorPicker) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, c.eventName())
	switch action {
	case "pick", "select":
		c.Value = payloadString(payload, "value")
	case "hex", "change", "input":
		c.Value = normalizeHex(payloadString(payload, "value"))
	default:
		return nil
	}
	c.MarkDirty()
	if c.OnChange != nil {
		c.OnChange(c.Value)
	}
	return nil
}

func normalizeHex(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	if !strings.HasPrefix(v, "#") {
		v = "#" + v
	}
	return strings.ToLower(v)
}

func (c *SwatchColorPicker) Render() (string, error) {
	attrs := Attrs{}
	attrs = c.CommonAttrs.Apply(attrs)
	attrs = c.FieldValidation.ApplyErrorState(attrs, "goui-swatch-color")

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	b.WriteString(`<div class="goui-swatch-row">`)
	for _, hex := range c.swatches() {
		cls := "goui-swatch"
		if strings.EqualFold(hex, c.Value) {
			cls += " is-selected"
		}
		b.WriteString(`<button type="button" class="` + cls + `" style="background:` + html.EscapeString(hex) + `"`)
		b.WriteString(` title="` + html.EscapeString(hex) + `"`)
		b.WriteString(` g-click="` + html.EscapeString(c.ev("pick")) + `"`)
		b.WriteString(` data-goui-value="` + html.EscapeString(hex) + `"></button>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`<input type="text" class="` + classInput + `" value="` + html.EscapeString(c.Value) + `"`)
	b.WriteString(` placeholder="#000000" g-change="` + html.EscapeString(c.ev("hex")) + `"`)
	b.WriteString(` g-input="` + html.EscapeString(c.ev("hex")) + `" g-debounce="150">`)
	b.WriteString(`</div>`)
	b.WriteString(c.FieldValidation.ErrorsHTML())
	return b.String(), nil
}

// GradientPicker builds a simple CSS linear-gradient from two stops + angle.
type GradientPicker struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	From      string
	To        string
	Angle     string // e.g. "135deg"
	EventName string
	OnChange  func(css string)
}

func (g *GradientPicker) Name() string { return g.CommonAttrs.Name }

func (g *GradientPicker) CSS() string {
	from := g.From
	to := g.To
	ang := g.Angle
	if from == "" {
		from = "#2563eb"
	}
	if to == "" {
		to = "#db2777"
	}
	if ang == "" {
		ang = "135deg"
	}
	return "linear-gradient(" + ang + ", " + from + ", " + to + ")"
}

func (g *GradientPicker) RawValue() string     { return g.CSS() }
func (g *GradientPicker) SetRawValue(v string) { _ = v }

func (g *GradientPicker) Mount(_ context.Context) error   { return nil }
func (g *GradientPicker) Unmount(_ context.Context) error { return nil }

func (g *GradientPicker) Validate() bool {
	return g.FieldValidation.Run(g.RawValue(), g.T)
}

func (g *GradientPicker) eventName() string {
	if g.EventName != "" {
		return g.EventName
	}
	return g.CommonAttrs.Name
}

func (g *GradientPicker) ev(action string) string {
	base := g.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (g *GradientPicker) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, g.eventName())
	val := payloadString(payload, "value")
	switch action {
	case "from":
		g.From = normalizeHex(val)
	case "to":
		g.To = normalizeHex(val)
	case "angle":
		g.Angle = strings.TrimSpace(val)
	default:
		return nil
	}
	g.MarkDirty()
	if g.OnChange != nil {
		g.OnChange(g.CSS())
	}
	return nil
}

func (g *GradientPicker) Render() (string, error) {
	attrs := Attrs{}
	attrs = g.CommonAttrs.Apply(attrs)
	attrs = g.FieldValidation.ApplyErrorState(attrs, "goui-gradient")

	from := g.From
	to := g.To
	ang := g.Angle
	if from == "" {
		from = "#2563eb"
	}
	if to == "" {
		to = "#db2777"
	}
	if ang == "" {
		ang = "135deg"
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	b.WriteString(`<div class="goui-gradient-preview" style="background:` + html.EscapeString(g.CSS()) + `"></div>`)
	b.WriteString(`<div class="goui-gradient-fields">`)
	b.WriteString(`<label>From <input type="color" value="` + html.EscapeString(from) + `" g-change="` + html.EscapeString(g.ev("from")) + `"></label>`)
	b.WriteString(`<label>To <input type="color" value="` + html.EscapeString(to) + `" g-change="` + html.EscapeString(g.ev("to")) + `"></label>`)
	b.WriteString(`<label>Açı <input type="text" class="` + classInput + `" value="` + html.EscapeString(ang) + `" g-change="` + html.EscapeString(g.ev("angle")) + `"></label>`)
	b.WriteString(`</div>`)
	b.WriteString(`<code class="goui-gradient-css">` + html.EscapeString(g.CSS()) + `</code>`)
	b.WriteString(`</div>`)
	b.WriteString(g.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
