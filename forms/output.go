package forms

import (
	"context"
	"html"

	"github.com/zatrano/goui/core"
)

// Output displays a calculation result.
type Output struct {
	core.BaseComponent
	CommonAttrs
	For   string
	Form  string
	Value string
	Text  string
}

func (o *Output) Name() string         { return o.CommonAttrs.Name }
func (o *Output) RawValue() string     { return o.Value }
func (o *Output) SetRawValue(v string) { o.Value = v; o.Text = v }

func (o *Output) Mount(_ context.Context) error   { return nil }
func (o *Output) Unmount(_ context.Context) error { return nil }

func (o *Output) Render() (string, error) {
	attrs := Attrs{}
	attrs = o.CommonAttrs.apply(attrs)
	attrs = attrs.Set("for", o.For)
	attrs = attrs.Set("form", o.Form)
	text := o.Text
	if text == "" {
		text = o.Value
	}
	return "<output" + attrs.String() + ">" + html.EscapeString(text) + "</output>", nil
}

func (o *Output) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}
