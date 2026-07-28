package forms

import (
	"context"
	"strconv"

	"github.com/zatrano/goui/core"
)

// CodeEditor stores source text on the server; CodeMirror UI is client-side (CDN).
// Render is stable (seed empty); initial value via data-initial. Use data-goui-ignore.
type CodeEditor struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Value      string
	Language   string // e.g. javascript, go, htmlmixed
	EventName  string
	DebounceMS int
	OnChange   func(value string)
}

func (c *CodeEditor) Name() string         { return c.CommonAttrs.Name }
func (c *CodeEditor) RawValue() string     { return c.Value }
func (c *CodeEditor) SetRawValue(v string) { c.Value = v }

func (c *CodeEditor) Mount(_ context.Context) error   { return nil }
func (c *CodeEditor) Unmount(_ context.Context) error { return nil }

func (c *CodeEditor) Validate() bool {
	return c.FieldValidation.Run(c.Value, c.T)
}

func (c *CodeEditor) eventName() string {
	if c.EventName != "" {
		return c.EventName
	}
	return c.CommonAttrs.Name
}

func (c *CodeEditor) debounce() int {
	if c.DebounceMS > 0 {
		return c.DebounceMS
	}
	return 350
}

func (c *CodeEditor) mode() string {
	if c.Language != "" {
		return c.Language
	}
	return "javascript"
}

func (c *CodeEditor) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, c.eventName())
	switch action {
	case "sync", "change", "input", c.eventName():
		c.Value = payloadString(payload, "value")
		if c.OnChange != nil {
			c.OnChange(c.Value)
		}
	}
	return nil
}

func (c *CodeEditor) Render() (string, error) {
	attrs := Attrs{}
	attrs = c.CommonAttrs.Apply(attrs)
	attrs = c.FieldValidation.ApplyErrorState(attrs, "goui-code-editor")
	attrs = attrs.Set("data-goui-code", "1")
	attrs = attrs.Set("data-goui-ignore", "1")
	attrs = attrs.Set("data-mode", c.mode())
	attrs = attrs.Set("data-initial", c.Value)

	sync := Attrs{}
	sync = sync.Set("class", "goui-editor-sync")
	sync = sync.Set("aria-hidden", "true")
	if ev := c.eventName(); ev != "" {
		sync = sync.Set("g-input", ev+".sync")
		sync = sync.Set("g-debounce", strconv.Itoa(c.debounce()))
	}

	return `<div` + attrs.String() + `>` +
		`<textarea class="goui-code-seed" data-goui-cm-mount></textarea>` +
		`<textarea` + sync.String() + `></textarea>` +
		`</div>` + c.FieldValidation.ErrorsHTML(), nil
}
