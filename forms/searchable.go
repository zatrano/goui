package forms

import (
	"context"
	"html"
	"strings"
)

// SearchableSelect is a single-select control with server-side search filtering.
type SearchableSelect struct {
	BaseSelectField
	EventName string // prefix for events, e.g. "city" → city.query / city.select
}

func (s *SearchableSelect) Name() string         { return s.CommonAttrs.Name }
func (s *SearchableSelect) RawValue() string     { return s.Value }
func (s *SearchableSelect) SetRawValue(v string) { s.Value = v }

func (s *SearchableSelect) Mount(_ context.Context) error   { return nil }
func (s *SearchableSelect) Unmount(_ context.Context) error { return nil }

func (s *SearchableSelect) Validate() bool {
	return s.FieldValidation.Run(s.Value, s.T)
}

func (s *SearchableSelect) eventName() string {
	if s.EventName != "" {
		return s.EventName
	}
	return s.CommonAttrs.Name
}

func (s *SearchableSelect) ev(action string) string {
	base := s.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (s *SearchableSelect) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := eventAction(event, s.eventName(), payload)
	switch action {
	case "toggle":
		s.Open = !s.Open
		if s.Open {
			s.EnsureFiltered()
		}
		s.MarkDirty()
	case "open":
		s.Open = true
		s.EnsureFiltered()
		s.MarkDirty()
	case "close":
		s.Open = false
		s.MarkDirty()
	case "query":
		// Server-side filter (default). FilterClient still recomputes here for small lists;
		// client never filters the option DOM itself.
		s.ApplyQuery(payloadString(payload, "value"))
		s.Open = true
	case "select":
		val := payloadString(payload, "value")
		s.Value = val
		s.Query = ""
		s.Open = false
		s.ApplyQuery("")
		s.MarkDirty()
		if s.OnChange != nil {
			s.OnChange(val)
		}
	}
	return nil
}

func (s *SearchableSelect) Render() (string, error) {
	s.EnsureFiltered()

	triggerClass := "goui-searchable-trigger border border-goui-border rounded-goui px-goui-field py-goui-field text-goui-text bg-white w-full"
	attrs := Attrs{}
	attrs = s.CommonAttrs.Apply(attrs)
	attrs = s.FieldValidation.ApplyErrorState(attrs, "goui-searchable")

	label := s.SelectedLabel()
	if label == "" {
		if s.Placeholder != "" {
			label = s.Placeholder
		} else {
			label = "Seçin..."
		}
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + ` data-goui-select="searchable">`)

	b.WriteString(`<button type="button" class="` + triggerClass + `" g-click="` + html.EscapeString(s.ev("toggle")) + `"`)
	b.WriteString(` aria-haspopup="listbox" aria-expanded="`)
	if s.Open {
		b.WriteString(`true"`)
	} else {
		b.WriteString(`false"`)
	}
	b.WriteString(`>`)
	b.WriteString(html.EscapeString(label))
	b.WriteString(`</button>`)

	if s.Open {
		b.WriteString(`<div class="goui-searchable-backdrop" g-click="` + html.EscapeString(s.ev("close")) + `"></div>`)
		b.WriteString(`<div class="goui-searchable-panel border border-goui-border rounded-goui" role="listbox">`)
		b.WriteString(`<input class="goui-input border border-goui-border rounded-goui px-goui-field py-goui-field w-full" type="search"`)
		b.WriteString(` placeholder="Ara..." value="` + html.EscapeString(s.Query) + `"`)
		b.WriteString(` g-input="` + html.EscapeString(s.ev("query")) + `" g-debounce="200"`)
		b.WriteString(` autocomplete="off">`)
		b.WriteString(`<ul class="goui-searchable-list">`)
		if len(s.Filtered) == 0 {
			b.WriteString(`<li class="goui-searchable-empty">Sonuç yok</li>`)
		}
		for _, it := range s.Filtered {
			itemClass := "goui-searchable-option"
			if it.Value == s.Value {
				itemClass += " is-selected"
			}
			if it.Disabled {
				b.WriteString(`<li class="` + itemClass + ` is-disabled" aria-disabled="true">`)
				b.WriteString(html.EscapeString(displayLabel(it)))
				b.WriteString(`</li>`)
				continue
			}
			b.WriteString(`<li class="` + itemClass + `" role="option" g-click="` + html.EscapeString(s.ev("select")) + `"`)
			// value travels via a hidden convention: g-click payload from button uses empty {};
			// use data-value + client module OR encode in event. For Tier1 client, payload is {}.
			// Workaround: use separate event names per item is bad. Use g-click with data-goui-value
			// and extend client minimally — OR put value in event name. Spec says minimal client.
			// Prefer: button with name attribute? collectPayload doesn't read data-value.
			// We'll add minimal data-goui-value support in selectable module / goui click handler.
			b.WriteString(` data-goui-value="` + html.EscapeString(it.Value) + `">`)
			b.WriteString(html.EscapeString(displayLabel(it)))
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul></div>`)
	}

	b.WriteString(`</div>`)
	b.WriteString(s.FieldValidation.ErrorsHTML())
	return b.String(), nil
}

func displayLabel(it SelectItem) string {
	if it.Label != "" {
		return it.Label
	}
	return it.Value
}
