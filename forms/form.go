package forms

import (
	"context"
	"html"
	"strings"

	"github.com/zatrano/goui/core"
)

// Form is a container for form fields.
type Form struct {
	core.BaseComponent
	CommonAttrs
	Action    string
	Method    string
	EncType   string
	InnerHTML string
	OnSubmit  string // g-submit event name
}

func (f *Form) Mount(_ context.Context) error   { return nil }
func (f *Form) Unmount(_ context.Context) error { return nil }

func (f *Form) Render() (string, error) {
	attrs := Attrs{}
	attrs = f.CommonAttrs.apply(attrs)
	if f.Class == "" {
		attrs = attrs.Set("class", classForm)
	}
	attrs = attrs.Set("action", f.Action)
	attrs = attrs.Set("method", f.Method)
	attrs = attrs.Set("enctype", f.EncType)
	if f.OnSubmit != "" {
		attrs = attrs.Set("g-submit", f.OnSubmit)
	}
	return "<form" + attrs.String() + ">" + f.InnerHTML + "</form>", nil
}

func (f *Form) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

// Fieldset groups related controls.
type Fieldset struct {
	core.BaseComponent
	CommonAttrs
	InnerHTML string
}

func (f *Fieldset) Mount(_ context.Context) error   { return nil }
func (f *Fieldset) Unmount(_ context.Context) error { return nil }

func (f *Fieldset) Render() (string, error) {
	attrs := Attrs{}
	attrs = f.CommonAttrs.apply(attrs)
	if f.Class == "" {
		attrs = attrs.Set("class", classFieldset)
	}
	return "<fieldset" + attrs.String() + ">" + f.InnerHTML + "</fieldset>", nil
}

func (f *Fieldset) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

// Legend is a fieldset caption.
type Legend struct {
	core.BaseComponent
	CommonAttrs
	Text string
}

func (l *Legend) Mount(_ context.Context) error   { return nil }
func (l *Legend) Unmount(_ context.Context) error { return nil }

func (l *Legend) Render() (string, error) {
	attrs := Attrs{}
	attrs = l.CommonAttrs.apply(attrs)
	return "<legend" + attrs.String() + ">" + html.EscapeString(l.Text) + "</legend>", nil
}

func (l *Legend) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

// Label associates text with a control.
type Label struct {
	core.BaseComponent
	CommonAttrs
	For  string
	Text string
}

func (l *Label) Mount(_ context.Context) error   { return nil }
func (l *Label) Unmount(_ context.Context) error { return nil }

func (l *Label) Render() (string, error) {
	attrs := Attrs{}
	attrs = l.CommonAttrs.apply(attrs)
	if l.Class == "" {
		attrs = attrs.Set("class", classLabel)
	}
	attrs = attrs.Set("for", l.For)
	return "<label" + attrs.String() + ">" + html.EscapeString(l.Text) + "</label>", nil
}

func (l *Label) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

// JoinHTML concatenates HTML fragments.
func JoinHTML(parts ...string) string {
	return strings.Join(parts, "")
}
