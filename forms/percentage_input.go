package forms

import (
	"context"
	"strings"

	"github.com/zatrano/goui/core"
)

// PercentageInput stores percentage points (e.g. 45.5 means 45,5%).
// Display formatting is server-side (locale-aware).
type PercentageInput struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Value     float64
	Locale    string
	Decimals  int // default 1
	Min       *float64
	Max       *float64
	Draft     string
	EventName string
	OnChange  func(value float64)
}

func (p *PercentageInput) Name() string { return p.CommonAttrs.Name }

func (p *PercentageInput) RawValue() string {
	return NumberFormat(p.Value, p.locale(), p.decimals())
}

func (p *PercentageInput) SetRawValue(v string) {
	v = strings.TrimSuffix(strings.TrimSpace(v), "%")
	if n, err := ParseLocalizedNumber(v, p.locale()); err == nil {
		p.Value = p.clamp(n)
		p.Draft = ""
	} else {
		p.Draft = v
	}
}

func (p *PercentageInput) Mount(_ context.Context) error   { return nil }
func (p *PercentageInput) Unmount(_ context.Context) error { return nil }

func (p *PercentageInput) Validate() bool {
	return p.FieldValidation.Run(p.RawValue(), p.T)
}

func (p *PercentageInput) locale() string {
	if p.Locale != "" {
		return p.Locale
	}
	return "tr"
}

func (p *PercentageInput) decimals() int {
	if p.Decimals < 0 {
		return 0
	}
	if p.Decimals == 0 {
		return 1
	}
	return p.Decimals
}

func (p *PercentageInput) clamp(n float64) float64 {
	if p.Min != nil && n < *p.Min {
		n = *p.Min
	}
	if p.Max != nil && n > *p.Max {
		n = *p.Max
	}
	return n
}

func (p *PercentageInput) eventName() string {
	if p.EventName != "" {
		return p.EventName
	}
	return p.CommonAttrs.Name
}

func (p *PercentageInput) displayText() string {
	if p.Draft != "" {
		return p.Draft
	}
	return NumberFormat(p.Value, p.locale(), p.decimals()) + " %"
}

func (p *PercentageInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = p.CommonAttrs.Apply(attrs)
	attrs = p.FieldValidation.ApplyErrorState(attrs, classInput+" goui-percentage")
	attrs = attrs.Set("type", "text")
	attrs = attrs.Set("inputmode", "decimal")
	attrs = attrs.Set("value", p.displayText())
	attrs = attrs.Set("autocomplete", "off")
	if ev := p.eventName(); ev != "" {
		attrs = attrs.Set("g-input", ev+".draft")
		attrs = attrs.Set("g-change", ev+".commit")
		attrs = attrs.Set("g-debounce", "100")
	}
	return "<input" + attrs.String() + ">" + p.FieldValidation.ErrorsHTML(), nil
}

func (p *PercentageInput) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, p.eventName())
	raw := payloadString(payload, "value")
	switch action {
	case "draft":
		p.Draft = raw
		p.MarkDirty()
	case "commit", "change", p.eventName():
		cleaned := strings.TrimSuffix(strings.TrimSpace(raw), "%")
		if n, err := ParseLocalizedNumber(cleaned, p.locale()); err == nil {
			p.Value = p.clamp(n)
			p.Draft = ""
			p.MarkDirty()
			if p.OnChange != nil {
				p.OnChange(p.Value)
			}
		} else {
			p.Draft = raw
			p.MarkDirty()
		}
	}
	return nil
}
