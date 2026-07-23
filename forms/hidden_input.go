package forms

import (
	"context"

	"github.com/zatrano/goui/core"
)

// HiddenInput covers type=hidden.
type HiddenInput struct {
	core.BaseComponent
	CommonAttrs
	Value string
}

func (h *HiddenInput) Name() string         { return h.CommonAttrs.Name }
func (h *HiddenInput) RawValue() string     { return h.Value }
func (h *HiddenInput) SetRawValue(v string) { h.Value = v }

func (h *HiddenInput) Mount(_ context.Context) error   { return nil }
func (h *HiddenInput) Unmount(_ context.Context) error { return nil }

func (h *HiddenInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = h.CommonAttrs.apply(attrs)
	attrs = attrs.Set("type", "hidden")
	attrs = attrs.Set("value", h.Value)
	return "<input" + attrs.String() + ">", nil
}

func (h *HiddenInput) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}
