package forms

import (
	"context"

	"github.com/zatrano/goui/core"
)

// ColorInput covers type=color.
type ColorInput struct {
	core.BaseComponent
	CommonAttrs
	Value     string
	EventName string
	OnChange  func(newValue string)
}

func (c *ColorInput) Name() string         { return c.CommonAttrs.Name }
func (c *ColorInput) RawValue() string     { return c.Value }
func (c *ColorInput) SetRawValue(v string) { c.Value = v }

func (c *ColorInput) Mount(_ context.Context) error   { return nil }
func (c *ColorInput) Unmount(_ context.Context) error { return nil }

func (c *ColorInput) eventName() string {
	if c.EventName != "" {
		return c.EventName
	}
	return c.CommonAttrs.Name
}

func (c *ColorInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = c.CommonAttrs.apply(attrs)
	if c.Class == "" {
		attrs = attrs.Set("class", classInput)
	}
	attrs = attrs.Set("type", "color")
	if c.Value == "" {
		c.Value = "#000000"
	}
	attrs = attrs.Set("value", c.Value)
	if ev := c.eventName(); ev != "" {
		attrs = attrs.Set("g-change", ev)
	}
	return "<input" + attrs.String() + ">", nil
}

func (c *ColorInput) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	if event != "change" && event != "input" && event != c.eventName() {
		return nil
	}
	val := payloadString(payload, "value")
	c.Value = val
	c.MarkDirty()
	if c.OnChange != nil {
		c.OnChange(val)
	}
	return nil
}
