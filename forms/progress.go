package forms

import (
	"context"

	"github.com/zatrano/goui/core"
)

// Progress represents task completion.
type Progress struct {
	core.BaseComponent
	CommonAttrs
	Value float64
	Max   float64
}

func (p *Progress) Mount(_ context.Context) error   { return nil }
func (p *Progress) Unmount(_ context.Context) error { return nil }

func (p *Progress) Render() (string, error) {
	attrs := Attrs{}
	attrs = p.CommonAttrs.apply(attrs)
	attrs = attrs.Set("value", formatFloat(p.Value))
	if p.Max == 0 {
		p.Max = 1
	}
	attrs = attrs.Set("max", formatFloat(p.Max))
	return "<progress" + attrs.String() + "></progress>", nil
}

func (p *Progress) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}
