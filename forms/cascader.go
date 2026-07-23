package forms

import (
	"context"
	"html"
	"strconv"
	"strings"
)

// CascaderLevel is one column in a cascader.
type CascaderLevel struct {
	Items    []SelectItem
	Selected string
}

// Cascader loads child options when a parent value is chosen (server-driven).
type Cascader struct {
	BaseSelectField
	EventName string
	// Levels[0] is root options; deeper levels filled via LoadChildren.
	Levels []CascaderLevel
	// LoadChildren returns child items for a selected parent value at level index.
	LoadChildren func(level int, parentValue string) []SelectItem
}

func (c *Cascader) Name() string { return c.CommonAttrs.Name }

func (c *Cascader) RawValue() string {
	parts := make([]string, 0, len(c.Levels))
	for _, lvl := range c.Levels {
		if lvl.Selected != "" {
			parts = append(parts, lvl.Selected)
		}
	}
	return strings.Join(parts, "/")
}

func (c *Cascader) SetRawValue(v string) {
	c.Value = v
}

func (c *Cascader) Mount(_ context.Context) error {
	if len(c.Levels) == 0 {
		c.Levels = []CascaderLevel{{Items: c.Items}}
	}
	return nil
}
func (c *Cascader) Unmount(_ context.Context) error { return nil }

func (c *Cascader) Validate() bool {
	return c.FieldValidation.Run(c.RawValue(), c.T)
}

func (c *Cascader) eventName() string {
	if c.EventName != "" {
		return c.EventName
	}
	return c.CommonAttrs.Name
}

func (c *Cascader) ev(action string) string {
	base := c.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (c *Cascader) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := eventAction(event, c.eventName(), payload)
	switch action {
	case "toggle":
		c.Open = !c.Open
		c.MarkDirty()
	case "close":
		c.Open = false
		c.MarkDirty()
	case "pick":
		level := payloadInt(payload, "level")
		val := payloadString(payload, "value")
		if level < 0 {
			level = 0
		}
		if level >= len(c.Levels) {
			return nil
		}
		c.Levels[level].Selected = val
		c.Levels = c.Levels[:level+1]
		if c.LoadChildren != nil {
			children := c.LoadChildren(level, val)
			if len(children) > 0 {
				c.Levels = append(c.Levels, CascaderLevel{Items: children})
			} else {
				c.Value = c.RawValue()
				c.Open = false
				if c.OnChange != nil {
					c.OnChange(c.Value)
				}
			}
		} else {
			c.Value = c.RawValue()
			c.Open = false
			if c.OnChange != nil {
				c.OnChange(c.Value)
			}
		}
		c.MarkDirty()
	}
	return nil
}

func (c *Cascader) Render() (string, error) {
	if len(c.Levels) == 0 {
		c.Levels = []CascaderLevel{{Items: c.Items}}
	}
	attrs := Attrs{}
	attrs = c.CommonAttrs.Apply(attrs)
	attrs = c.FieldValidation.ApplyErrorState(attrs, "goui-searchable goui-cascader")

	label := c.RawValue()
	if label == "" {
		label = c.Placeholder
		if label == "" {
			label = "Seçin..."
		}
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + ` data-goui-select="cascader">`)
	b.WriteString(`<button type="button" class="goui-searchable-trigger border border-goui-border rounded-goui px-goui-field py-goui-field w-full" g-click="` + html.EscapeString(c.ev("toggle")) + `">`)
	b.WriteString(html.EscapeString(label))
	b.WriteString(`</button>`)
	if c.Open {
		b.WriteString(`<div class="goui-searchable-backdrop" g-click="` + html.EscapeString(c.ev("close")) + `"></div>`)
		b.WriteString(`<div class="goui-cascader-panel border border-goui-border rounded-goui">`)
		for li, lvl := range c.Levels {
			b.WriteString(`<ul class="goui-cascader-col">`)
			for _, it := range lvl.Items {
				cls := "goui-searchable-option"
				if it.Value == lvl.Selected {
					cls += " is-selected"
				}
				b.WriteString(`<li class="` + cls + `" g-click="` + html.EscapeString(c.ev("pick")) + `"`)
				b.WriteString(` data-goui-value="` + html.EscapeString(it.Value) + `"`)
				b.WriteString(` data-goui-level="` + strconv.Itoa(li) + `">`)
				b.WriteString(html.EscapeString(displayLabel(it)))
				b.WriteString(`</li>`)
			}
			b.WriteString(`</ul>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(c.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
