package forms

import (
	"context"
	"html"
	"strconv"
	"strings"

	"github.com/zatrano/goui/core"
)

// Rating is an N-star (or custom icon) control; selection lives on the server.
type Rating struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Value     int    // 0..Max
	Max       int    // default 5
	Icon      string // default ★
	EmptyIcon string // default ☆
	EventName string
	OnChange  func(value int)
}

func (r *Rating) Name() string { return r.CommonAttrs.Name }

func (r *Rating) RawValue() string {
	return strconv.Itoa(r.Value)
}

func (r *Rating) SetRawValue(v string) {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return
	}
	r.Value = r.clamp(n)
}

func (r *Rating) Mount(_ context.Context) error   { return nil }
func (r *Rating) Unmount(_ context.Context) error { return nil }

func (r *Rating) Validate() bool {
	return r.FieldValidation.Run(r.RawValue(), r.T)
}

func (r *Rating) max() int {
	if r.Max <= 0 {
		return 5
	}
	return r.Max
}

func (r *Rating) clamp(n int) int {
	if n < 0 {
		return 0
	}
	if n > r.max() {
		return r.max()
	}
	return n
}

func (r *Rating) eventName() string {
	if r.EventName != "" {
		return r.EventName
	}
	return r.CommonAttrs.Name
}

func (r *Rating) filledIcon() string {
	if r.Icon != "" {
		return r.Icon
	}
	return "★"
}

func (r *Rating) emptyIcon() string {
	if r.EmptyIcon != "" {
		return r.EmptyIcon
	}
	return "☆"
}

func (r *Rating) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, r.eventName())
	switch action {
	case "set", "select", r.eventName(), "change":
		n := 0
		if raw := payloadString(payload, "value"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				n = parsed
			}
		}
		// Toggle off when clicking the same star again
		if n == r.Value {
			r.Value = 0
		} else {
			r.Value = r.clamp(n)
		}
		r.MarkDirty()
		if r.OnChange != nil {
			r.OnChange(r.Value)
		}
	}
	return nil
}

func (r *Rating) Render() (string, error) {
	attrs := Attrs{}
	attrs = r.CommonAttrs.Apply(attrs)
	attrs = r.FieldValidation.ApplyErrorState(attrs, "goui-rating")
	attrs = attrs.Set("role", "radiogroup")
	attrs = attrs.Set("aria-label", r.Name())

	ev := r.eventName()
	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	for i := 1; i <= r.max(); i++ {
		icon := r.emptyIcon()
		cls := "goui-rating-star"
		if i <= r.Value {
			icon = r.filledIcon()
			cls += " is-active"
		}
		b.WriteString(`<button type="button" class="` + cls + `"`)
		b.WriteString(` aria-label="` + html.EscapeString(strconv.Itoa(i)) + `"`)
		if ev != "" {
			b.WriteString(` g-click="` + html.EscapeString(ev+".set") + `"`)
		}
		b.WriteString(` data-goui-value="` + strconv.Itoa(i) + `">`)
		b.WriteString(html.EscapeString(icon))
		b.WriteString(`</button>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(r.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
