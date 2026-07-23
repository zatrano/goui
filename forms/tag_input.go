package forms

import (
	"context"
	"html"
	"strings"

	"github.com/zatrano/goui/core"
)

// TagInput collects free-text tags; Values live on the server.
type TagInput struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Values      []string
	Draft       string
	Placeholder string
	EventName   string
	OnChange    func(tags []string)
}

func (t *TagInput) Name() string { return t.CommonAttrs.Name }
func (t *TagInput) RawValue() string {
	return strings.Join(t.Values, ",")
}
func (t *TagInput) SetRawValue(v string) {
	if v == "" {
		t.Values = nil
		return
	}
	t.Values = strings.Split(v, ",")
}

func (t *TagInput) Mount(_ context.Context) error   { return nil }
func (t *TagInput) Unmount(_ context.Context) error { return nil }

func (t *TagInput) Validate() bool {
	return t.FieldValidation.Run(t.RawValue(), t.T)
}

func (t *TagInput) eventName() string {
	if t.EventName != "" {
		return t.EventName
	}
	return t.CommonAttrs.Name
}

func (t *TagInput) ev(action string) string {
	base := t.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (t *TagInput) addTag(raw string) {
	for _, part := range strings.Split(raw, ",") {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		dup := false
		for _, v := range t.Values {
			if strings.EqualFold(v, tag) {
				dup = true
				break
			}
		}
		if !dup {
			t.Values = append(t.Values, tag)
		}
	}
	t.Draft = ""
}

func (t *TagInput) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := eventAction(event, t.eventName(), payload)
	switch action {
	case "draft":
		t.Draft = payloadString(payload, "value")
		t.MarkDirty()
	case "add", "commit":
		val := payloadString(payload, "value")
		if val == "" {
			val = t.Draft
		}
		t.addTag(val)
		t.MarkDirty()
		if t.OnChange != nil {
			t.OnChange(t.Values)
		}
	case "remove":
		val := payloadString(payload, "value")
		out := make([]string, 0, len(t.Values))
		for _, v := range t.Values {
			if v != val {
				out = append(out, v)
			}
		}
		t.Values = out
		t.MarkDirty()
		if t.OnChange != nil {
			t.OnChange(t.Values)
		}
	}
	return nil
}

func (t *TagInput) Render() (string, error) {
	attrs := Attrs{}
	attrs = t.CommonAttrs.Apply(attrs)
	attrs = t.FieldValidation.ApplyErrorState(attrs, "goui-tag-input")

	ph := t.Placeholder
	if ph == "" {
		ph = "Etiket ekle..."
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + ` data-goui-select="tags">`)
	b.WriteString(`<div class="goui-tag-box border border-goui-border rounded-goui px-goui-field py-goui-field">`)
	for _, v := range t.Values {
		b.WriteString(`<span class="goui-chip" g-click="` + html.EscapeString(t.ev("remove")) + `" data-goui-value="` + html.EscapeString(v) + `">`)
		b.WriteString(html.EscapeString(v))
		b.WriteString(` ×</span>`)
	}
	b.WriteString(`<input class="goui-tag-draft" type="text" value="` + html.EscapeString(t.Draft) + `"`)
	b.WriteString(` placeholder="` + html.EscapeString(ph) + `"`)
	b.WriteString(` g-input="` + html.EscapeString(t.ev("draft")) + `" g-debounce="100"`)
	b.WriteString(` g-change="` + html.EscapeString(t.ev("add")) + `">`)
	b.WriteString(`</div></div>`)
	b.WriteString(t.FieldValidation.ErrorsHTML())
	return b.String(), nil
}

// ChipsInput is an alias highlighting chip presentation of TagInput / MultiSelect values.
type ChipsInput = TagInput
