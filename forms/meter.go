package forms

import (
	"context"
	"strconv"

	"github.com/zatrano/goui/core"
)

// Meter represents a scalar measurement within a known range.
type Meter struct {
	core.BaseComponent
	CommonAttrs
	Value   float64
	Min     float64
	Max     float64
	Low     float64
	High    float64
	Optimum float64
}

func (m *Meter) Mount(_ context.Context) error   { return nil }
func (m *Meter) Unmount(_ context.Context) error { return nil }

func (m *Meter) Render() (string, error) {
	attrs := Attrs{}
	attrs = m.CommonAttrs.apply(attrs)
	attrs = attrs.Set("value", formatFloat(m.Value))
	attrs = attrs.Set("min", formatFloat(m.Min))
	attrs = attrs.Set("max", formatFloat(m.Max))
	if m.Low != 0 {
		attrs = attrs.Set("low", formatFloat(m.Low))
	}
	if m.High != 0 {
		attrs = attrs.Set("high", formatFloat(m.High))
	}
	if m.Optimum != 0 {
		attrs = attrs.Set("optimum", formatFloat(m.Optimum))
	}
	return "<meter" + attrs.String() + "></meter>", nil
}

func (m *Meter) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
