package forms

import (
	"context"
	"html"

	"github.com/zatrano/goui/core"
)

// Button covers type=submit|button|reset|image.
type Button struct {
	core.BaseComponent
	CommonAttrs
	Type      string
	Value     string
	Text      string
	Src       string // type=image
	Alt       string
	EventName string // g-click event
}

func (b *Button) Mount(_ context.Context) error   { return nil }
func (b *Button) Unmount(_ context.Context) error { return nil }

func (b *Button) buttonType() string {
	if b.Type == "" {
		return "submit"
	}
	return b.Type
}

func (b *Button) Render() (string, error) {
	attrs := Attrs{}
	attrs = b.CommonAttrs.apply(attrs)
	if b.Class == "" {
		attrs = attrs.Set("class", classButton)
	}
	attrs = attrs.Set("type", b.buttonType())
	attrs = attrs.Set("value", b.Value)
	if b.buttonType() == "image" {
		attrs = attrs.Set("src", b.Src)
		attrs = attrs.Set("alt", b.Alt)
		if b.EventName != "" {
			attrs = attrs.Set("g-click", b.EventName)
		}
		return "<input" + attrs.String() + ">", nil
	}
	if b.EventName != "" {
		attrs = attrs.Set("g-click", b.EventName)
	}
	return "<button" + attrs.String() + ">" + html.EscapeString(b.Text) + "</button>", nil
}

func (b *Button) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}
