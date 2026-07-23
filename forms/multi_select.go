package forms

import (
	"context"
	"html"
	"strings"
)

// MultiSelect allows selecting multiple values with server-side search.
type MultiSelect struct {
	BaseSelectField
	EventName string
}

func (m *MultiSelect) Name() string { return m.CommonAttrs.Name }

func (m *MultiSelect) RawValue() string {
	return strings.Join(m.Values, ",")
}

func (m *MultiSelect) SetRawValue(v string) {
	if v == "" {
		m.Values = nil
		return
	}
	m.Values = strings.Split(v, ",")
}

func (m *MultiSelect) Mount(_ context.Context) error   { return nil }
func (m *MultiSelect) Unmount(_ context.Context) error { return nil }

func (m *MultiSelect) Validate() bool {
	return m.FieldValidation.Run(m.RawValue(), m.T)
}

func (m *MultiSelect) eventName() string {
	if m.EventName != "" {
		return m.EventName
	}
	return m.CommonAttrs.Name
}

func (m *MultiSelect) ev(action string) string {
	base := m.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (m *MultiSelect) isSelected(value string) bool {
	for _, v := range m.Values {
		if v == value {
			return true
		}
	}
	return false
}

func (m *MultiSelect) toggleValue(value string) {
	for i, v := range m.Values {
		if v == value {
			m.Values = append(m.Values[:i], m.Values[i+1:]...)
			return
		}
	}
	m.Values = append(m.Values, value)
}

func (m *MultiSelect) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := eventAction(event, m.eventName(), payload)
	switch action {
	case "toggle":
		m.Open = !m.Open
		if m.Open {
			m.EnsureFiltered()
		}
		m.MarkDirty()
	case "open":
		m.Open = true
		m.EnsureFiltered()
		m.MarkDirty()
	case "close":
		m.Open = false
		m.MarkDirty()
	case "query":
		m.ApplyQuery(payloadString(payload, "value"))
		m.Open = true
	case "select":
		val := payloadString(payload, "value")
		m.toggleValue(val)
		m.MarkDirty()
		if m.OnChange != nil {
			m.OnChange(m.RawValue())
		}
	case "remove":
		val := payloadString(payload, "value")
		out := make([]string, 0, len(m.Values))
		for _, v := range m.Values {
			if v != val {
				out = append(out, v)
			}
		}
		m.Values = out
		m.MarkDirty()
		if m.OnChange != nil {
			m.OnChange(m.RawValue())
		}
	}
	return nil
}

func (m *MultiSelect) Render() (string, error) {
	m.EnsureFiltered()

	attrs := Attrs{}
	attrs = m.CommonAttrs.Apply(attrs)
	attrs = m.FieldValidation.ApplyErrorState(attrs, "goui-searchable goui-multiselect")

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + ` data-goui-select="multi">`)

	b.WriteString(`<div class="goui-multi-trigger border border-goui-border rounded-goui px-goui-field py-goui-field" g-click="` + html.EscapeString(m.ev("toggle")) + `">`)
	if len(m.Values) == 0 {
		ph := m.Placeholder
		if ph == "" {
			ph = "Seçin..."
		}
		b.WriteString(`<span class="goui-multi-placeholder">` + html.EscapeString(ph) + `</span>`)
	} else {
		for _, v := range m.Values {
			label := v
			for _, it := range m.Items {
				if it.Value == v {
					label = displayLabel(it)
					break
				}
			}
			b.WriteString(`<span class="goui-chip" g-click="` + html.EscapeString(m.ev("remove")) + `" data-goui-value="` + html.EscapeString(v) + `">`)
			b.WriteString(html.EscapeString(label))
			b.WriteString(` ×</span>`)
		}
	}
	b.WriteString(`</div>`)

	if m.Open {
		b.WriteString(`<div class="goui-searchable-backdrop" g-click="` + html.EscapeString(m.ev("close")) + `"></div>`)
		b.WriteString(`<div class="goui-searchable-panel border border-goui-border rounded-goui" role="listbox">`)
		b.WriteString(`<input class="goui-input border border-goui-border rounded-goui px-goui-field py-goui-field w-full" type="search"`)
		b.WriteString(` placeholder="Ara..." value="` + html.EscapeString(m.Query) + `"`)
		b.WriteString(` g-input="` + html.EscapeString(m.ev("query")) + `" g-debounce="200" autocomplete="off">`)
		b.WriteString(`<ul class="goui-searchable-list">`)
		if len(m.Filtered) == 0 {
			b.WriteString(`<li class="goui-searchable-empty">Sonuç yok</li>`)
		}
		for _, it := range m.Filtered {
			itemClass := "goui-searchable-option"
			if m.isSelected(it.Value) {
				itemClass += " is-selected"
			}
			if it.Disabled {
				b.WriteString(`<li class="` + itemClass + ` is-disabled">` + html.EscapeString(displayLabel(it)) + `</li>`)
				continue
			}
			b.WriteString(`<li class="` + itemClass + `" g-click="` + html.EscapeString(m.ev("select")) + `" data-goui-value="` + html.EscapeString(it.Value) + `">`)
			b.WriteString(html.EscapeString(displayLabel(it)))
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul></div>`)
	}

	b.WriteString(`</div>`)
	b.WriteString(m.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
