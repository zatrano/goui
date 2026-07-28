package forms

import (
	"context"
	"strings"

	"github.com/zatrano/goui/core"
)

// CurrencyInput stores a float64 amount; display formatting is server-side.
type CurrencyInput struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Value     float64
	Currency  string // ISO code, default TRY
	Locale    string // default "tr"
	Decimals  int    // default 2 (zero-value → 2)
	Draft     string // raw text while typing; empty → show formatted Value
	EventName string
	OnChange  func(value float64)
}

func (c *CurrencyInput) Name() string { return c.CommonAttrs.Name }

func (c *CurrencyInput) RawValue() string {
	return NumberFormat(c.Value, c.locale(), c.decimals())
}

func (c *CurrencyInput) SetRawValue(v string) {
	if n, err := ParseLocalizedNumber(v, c.locale()); err == nil {
		c.Value = n
		c.Draft = ""
	} else {
		c.Draft = v
	}
}

func (c *CurrencyInput) Mount(_ context.Context) error   { return nil }
func (c *CurrencyInput) Unmount(_ context.Context) error { return nil }

func (c *CurrencyInput) Validate() bool {
	return c.FieldValidation.Run(c.RawValue(), c.T)
}

func (c *CurrencyInput) locale() string {
	if c.Locale != "" {
		return c.Locale
	}
	return "tr"
}

func (c *CurrencyInput) decimals() int {
	if c.Decimals <= 0 {
		return 2
	}
	return c.Decimals
}

func (c *CurrencyInput) currencyCode() string {
	if c.Currency == "" {
		return "TRY"
	}
	return strings.ToUpper(c.Currency)
}

func (c *CurrencyInput) eventName() string {
	if c.EventName != "" {
		return c.EventName
	}
	return c.CommonAttrs.Name
}

func (c *CurrencyInput) displayText() string {
	if c.Draft != "" {
		return c.Draft
	}
	sym := CurrencySymbol(c.currencyCode())
	formatted := NumberFormat(c.Value, c.locale(), c.decimals())
	switch c.currencyCode() {
	case "USD", "GBP":
		return sym + formatted
	default:
		return formatted + " " + sym
	}
}

func (c *CurrencyInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = c.CommonAttrs.Apply(attrs)
	attrs = c.FieldValidation.ApplyErrorState(attrs, classInput+" goui-currency")
	attrs = attrs.Set("type", "text")
	attrs = attrs.Set("inputmode", "decimal")
	attrs = attrs.Set("value", c.displayText())
	attrs = attrs.Set("autocomplete", "off")
	if ev := c.eventName(); ev != "" {
		attrs = attrs.Set("g-input", ev+".draft")
		attrs = attrs.Set("g-change", ev+".commit")
		attrs = attrs.Set("g-debounce", "100")
	}
	return "<input" + attrs.String() + ">" + c.FieldValidation.ErrorsHTML(), nil
}

func (c *CurrencyInput) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, c.eventName())
	raw := payloadString(payload, "value")
	switch action {
	case "draft":
		c.Draft = raw
		c.MarkDirty()
		return nil
	case "commit", "change":
		// ok
	default:
		if event != c.eventName() {
			return nil
		}
	}
	if n, err := ParseLocalizedNumber(raw, c.locale()); err == nil {
		c.Value = n
		c.Draft = ""
		c.MarkDirty()
		if c.OnChange != nil {
			c.OnChange(n)
		}
	} else {
		c.Draft = raw
		c.MarkDirty()
	}
	return nil
}

func dottedAction(event, base string) string {
	if base != "" {
		prefix := base + "."
		if strings.HasPrefix(event, prefix) {
			return strings.TrimPrefix(event, prefix)
		}
	}
	return event
}
