package forms

import (
	"context"
	"html"
	"strings"

	"github.com/zatrano/goui/core"
)

// Option is a select option.
type Option struct {
	Value    string
	Label    string
	Selected bool
	Disabled bool
}

// Optgroup groups options.
type Optgroup struct {
	Label    string
	Disabled bool
	Options  []Option
}

// Select is a dropdown control.
type Select struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation
	Value     string
	Multiple  bool
	Size      int
	Options   []Option
	Groups    []Optgroup
	EventName string
	OnChange  func(newValue string)
}

func (s *Select) Name() string         { return s.CommonAttrs.Name }
func (s *Select) RawValue() string     { return s.Value }
func (s *Select) SetRawValue(v string) { s.Value = v }

func (s *Select) Mount(_ context.Context) error   { return nil }
func (s *Select) Unmount(_ context.Context) error { return nil }

func (s *Select) Validate() bool {
	return s.FieldValidation.run(s.Value, s.T)
}

func (s *Select) eventName() string {
	if s.EventName != "" {
		return s.EventName
	}
	return s.CommonAttrs.Name
}

func (s *Select) Render() (string, error) {
	attrs := Attrs{}
	attrs = s.CommonAttrs.apply(attrs)
	attrs = s.FieldValidation.applyErrorState(attrs, classSelect)
	attrs = attrs.SetBool("multiple", s.Multiple)
	attrs = attrs.SetInt("size", s.Size)
	if ev := s.eventName(); ev != "" {
		attrs = attrs.Set("g-change", ev)
	}

	var body strings.Builder
	for _, opt := range s.Options {
		body.WriteString(renderOption(opt, s.Value, s.Multiple))
	}
	for _, g := range s.Groups {
		ga := Attrs{}.Set("label", g.Label).SetBool("disabled", g.Disabled)
		body.WriteString("<optgroup" + ga.String() + ">")
		for _, opt := range g.Options {
			body.WriteString(renderOption(opt, s.Value, s.Multiple))
		}
		body.WriteString("</optgroup>")
	}

	return "<select" + attrs.String() + ">" + body.String() + "</select>" + s.FieldValidation.errorsHTML(), nil
}

func renderOption(opt Option, selectedValue string, multiple bool) string {
	attrs := Attrs{}
	attrs = attrs.Set("value", opt.Value)
	attrs = attrs.SetBool("disabled", opt.Disabled)
	selected := opt.Selected
	if !multiple && selectedValue != "" {
		selected = opt.Value == selectedValue
	}
	attrs = attrs.SetBool("selected", selected)
	label := opt.Label
	if label == "" {
		label = opt.Value
	}
	return "<option" + attrs.String() + ">" + html.EscapeString(label) + "</option>"
}

func (s *Select) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	if event != "change" && event != s.eventName() {
		return nil
	}
	val := payloadString(payload, "value")
	s.Value = val
	for i := range s.Options {
		s.Options[i].Selected = s.Options[i].Value == val
	}
	for gi := range s.Groups {
		for oi := range s.Groups[gi].Options {
			s.Groups[gi].Options[oi].Selected = s.Groups[gi].Options[oi].Value == val
		}
	}
	s.MarkDirty()
	if s.OnChange != nil {
		s.OnChange(val)
	}
	return nil
}
