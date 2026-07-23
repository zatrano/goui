package forms

import (
	"context"

	"github.com/zatrano/goui/core"
)

// NumericInput covers type=number|range.
type NumericInput struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation
	Type      string
	Value     string
	Min       string
	Max       string
	Step      string
	EventName string
	OnChange  func(newValue string)
}

func (n *NumericInput) Name() string         { return n.CommonAttrs.Name }
func (n *NumericInput) RawValue() string     { return n.Value }
func (n *NumericInput) SetRawValue(v string) { n.Value = v }

func (n *NumericInput) Mount(_ context.Context) error   { return nil }
func (n *NumericInput) Unmount(_ context.Context) error { return nil }

func (n *NumericInput) Validate() bool {
	return n.FieldValidation.run(n.Value, n.T)
}

func (n *NumericInput) inputType() string {
	if n.Type == "" {
		return "number"
	}
	return n.Type
}

func (n *NumericInput) eventName() string {
	if n.EventName != "" {
		return n.EventName
	}
	return n.CommonAttrs.Name
}

func (n *NumericInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = n.CommonAttrs.apply(attrs)
	attrs = n.FieldValidation.applyErrorState(attrs, classInput)
	attrs = attrs.Set("type", n.inputType())
	attrs = attrs.Set("value", n.Value)
	attrs = attrs.Set("min", n.Min)
	attrs = attrs.Set("max", n.Max)
	attrs = attrs.Set("step", n.Step)
	if ev := n.eventName(); ev != "" {
		attrs = attrs.Set("g-change", ev)
		attrs = attrs.Set("g-input", ev)
	}
	return "<input" + attrs.String() + ">" + n.FieldValidation.errorsHTML(), nil
}

func (n *NumericInput) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	if event != "change" && event != "input" && event != n.eventName() {
		return nil
	}
	val := payloadString(payload, "value")
	n.Value = val
	n.MarkDirty()
	if n.OnChange != nil {
		n.OnChange(val)
	}
	return nil
}
