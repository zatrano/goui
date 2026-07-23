package forms

import (
	"context"
	"html"
	"strings"
)

// Autocomplete shows suggestions for an input; selecting a suggestion sets Value.
// Typing filters server-side; free text remains in Query until select/commit.
type Autocomplete struct {
	BaseSelectField
	EventName string
}

func (a *Autocomplete) Name() string         { return a.CommonAttrs.Name }
func (a *Autocomplete) RawValue() string     { return a.Value }
func (a *Autocomplete) SetRawValue(v string) { a.Value = v }

func (a *Autocomplete) Mount(_ context.Context) error   { return nil }
func (a *Autocomplete) Unmount(_ context.Context) error { return nil }

func (a *Autocomplete) Validate() bool {
	return a.FieldValidation.Run(a.Value, a.T)
}

func (a *Autocomplete) eventName() string {
	if a.EventName != "" {
		return a.EventName
	}
	return a.CommonAttrs.Name
}

func (a *Autocomplete) ev(action string) string {
	base := a.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (a *Autocomplete) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := eventAction(event, a.eventName(), payload)
	switch action {
	case "query":
		q := payloadString(payload, "value")
		a.ApplyQuery(q)
		a.Open = q != "" && len(a.Filtered) > 0
		a.MarkDirty()
	case "select":
		val := payloadString(payload, "value")
		label := val
		for _, it := range a.Items {
			if it.Value == val {
				label = displayLabel(it)
				break
			}
		}
		a.Value = val
		a.Query = label
		a.Open = false
		a.MarkDirty()
		if a.OnChange != nil {
			a.OnChange(val)
		}
	case "commit":
		// Keep typed text as Value if user didn't pick a suggestion.
		if a.Query != "" && a.Value == "" {
			a.Value = a.Query
			if a.OnChange != nil {
				a.OnChange(a.Value)
			}
		}
		a.Open = false
		a.MarkDirty()
	case "close":
		a.Open = false
		a.MarkDirty()
	}
	return nil
}

func (a *Autocomplete) Render() (string, error) {
	a.EnsureFiltered()

	attrs := Attrs{}
	attrs = a.CommonAttrs.Apply(attrs)
	attrs = a.FieldValidation.ApplyErrorState(attrs, "goui-searchable goui-autocomplete")

	display := a.Query
	if display == "" {
		display = a.Value
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + ` data-goui-select="autocomplete">`)
	b.WriteString(`<input class="goui-input border border-goui-border rounded-goui px-goui-field py-goui-field w-full" type="search"`)
	ph := a.Placeholder
	if ph == "" {
		ph = "Yazmaya başlayın..."
	}
	b.WriteString(` placeholder="` + html.EscapeString(ph) + `" value="` + html.EscapeString(display) + `"`)
	b.WriteString(` g-input="` + html.EscapeString(a.ev("query")) + `" g-debounce="200"`)
	b.WriteString(` g-change="` + html.EscapeString(a.ev("commit")) + `" autocomplete="off">`)

	if a.Open {
		b.WriteString(`<div class="goui-searchable-panel border border-goui-border rounded-goui" role="listbox">`)
		b.WriteString(`<ul class="goui-searchable-list">`)
		for _, it := range a.Filtered {
			b.WriteString(`<li class="goui-searchable-option" g-click="` + html.EscapeString(a.ev("select")) + `" data-goui-value="` + html.EscapeString(it.Value) + `">`)
			b.WriteString(html.EscapeString(displayLabel(it)))
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul></div>`)
	}

	b.WriteString(`</div>`)
	b.WriteString(a.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
