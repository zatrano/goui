package forms

import (
	"context"
	"html"
	"strings"

	"github.com/zatrano/goui/core"
)

// DatalistOption is an option inside a datalist.
type DatalistOption struct {
	Value string
	Label string
}

// Datalist provides autocomplete suggestions for an input via list=id.
type Datalist struct {
	core.BaseComponent
	CommonAttrs
	Options []DatalistOption
}

func (d *Datalist) Mount(_ context.Context) error   { return nil }
func (d *Datalist) Unmount(_ context.Context) error { return nil }

func (d *Datalist) Render() (string, error) {
	attrs := Attrs{}
	attrs = d.CommonAttrs.apply(attrs)
	var body strings.Builder
	for _, opt := range d.Options {
		oa := Attrs{}.Set("value", opt.Value).Set("label", opt.Label)
		body.WriteString("<option" + oa.String() + ">" + html.EscapeString(opt.Label) + "</option>")
	}
	return "<datalist" + attrs.String() + ">" + body.String() + "</datalist>", nil
}

func (d *Datalist) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}
