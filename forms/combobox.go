package forms

import (
	"context"
	"html"
	"strings"
)

// Combobox is a searchable select that also accepts free-typed values.
type Combobox struct {
	BaseSelectField
	EventName string
	// RestrictToList when true rejects free text (Value only from Items).
	RestrictToList bool
}

func (c *Combobox) Name() string         { return c.CommonAttrs.Name }
func (c *Combobox) RawValue() string     { return c.Value }
func (c *Combobox) SetRawValue(v string) { c.Value = v }

func (c *Combobox) Mount(_ context.Context) error   { return nil }
func (c *Combobox) Unmount(_ context.Context) error { return nil }

func (c *Combobox) Validate() bool {
	return c.FieldValidation.Run(c.Value, c.T)
}

func (c *Combobox) eventName() string {
	if c.EventName != "" {
		return c.EventName
	}
	return c.CommonAttrs.Name
}

func (c *Combobox) ev(action string) string {
	base := c.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (c *Combobox) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := eventAction(event, c.eventName(), payload)
	switch action {
	case "toggle", "open":
		c.Open = true
		c.EnsureFiltered()
		c.MarkDirty()
	case "close":
		c.Open = false
		c.MarkDirty()
	case "query":
		q := payloadString(payload, "value")
		c.ApplyQuery(q)
		c.Open = true
		if !c.RestrictToList {
			c.Value = q
			if c.OnChange != nil {
				c.OnChange(q)
			}
		}
	case "select":
		val := payloadString(payload, "value")
		c.Value = val
		c.Query = displayLabel(SelectItem{Value: val, Label: c.labelFor(val)})
		c.Open = false
		c.MarkDirty()
		if c.OnChange != nil {
			c.OnChange(val)
		}
	case "commit":
		c.Open = false
		c.MarkDirty()
	}
	return nil
}

func (c *Combobox) labelFor(value string) string {
	for _, it := range c.Items {
		if it.Value == value {
			return displayLabel(it)
		}
	}
	return value
}

func (c *Combobox) Render() (string, error) {
	c.EnsureFiltered()

	attrs := Attrs{}
	attrs = c.CommonAttrs.Apply(attrs)
	attrs = c.FieldValidation.ApplyErrorState(attrs, "goui-searchable goui-combobox")

	display := c.Query
	if display == "" {
		display = c.Value
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + ` data-goui-select="combobox">`)
	b.WriteString(`<input class="goui-input border border-goui-border rounded-goui px-goui-field py-goui-field w-full" type="text"`)
	ph := c.Placeholder
	if ph == "" {
		ph = "Yazın veya seçin..."
	}
	b.WriteString(` placeholder="` + html.EscapeString(ph) + `" value="` + html.EscapeString(display) + `"`)
	b.WriteString(` g-input="` + html.EscapeString(c.ev("query")) + `" g-debounce="200"`)
	b.WriteString(` g-change="` + html.EscapeString(c.ev("commit")) + `"`)
	b.WriteString(` autocomplete="off" role="combobox" aria-expanded="`)
	if c.Open {
		b.WriteString(`true"`)
	} else {
		b.WriteString(`false"`)
	}
	b.WriteString(`>`)

	if c.Open && len(c.Filtered) > 0 {
		b.WriteString(`<div class="goui-searchable-panel border border-goui-border rounded-goui" role="listbox">`)
		b.WriteString(`<ul class="goui-searchable-list">`)
		for _, it := range c.Filtered {
			b.WriteString(`<li class="goui-searchable-option" g-click="` + html.EscapeString(c.ev("select")) + `" data-goui-value="` + html.EscapeString(it.Value) + `">`)
			b.WriteString(html.EscapeString(displayLabel(it)))
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul></div>`)
	}

	b.WriteString(`</div>`)
	b.WriteString(c.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
