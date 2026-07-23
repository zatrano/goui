package forms

import (
	"context"
	"html"
	"strings"
)

// DualListbox moves items between available and selected lists (server-owned).
// Filtering on both sides is server-side by default (ApplyQuery / ApplySelectedQuery).
type DualListbox struct {
	BaseSelectField
	EventName      string
	SelectedQuery  string
	SelectedFilter []SelectItem // filtered view of selected side
}

func (d *DualListbox) Name() string { return d.CommonAttrs.Name }

func (d *DualListbox) RawValue() string {
	return strings.Join(d.Values, ",")
}

func (d *DualListbox) SetRawValue(v string) {
	if v == "" {
		d.Values = nil
		return
	}
	d.Values = strings.Split(v, ",")
}

func (d *DualListbox) Mount(_ context.Context) error   { return nil }
func (d *DualListbox) Unmount(_ context.Context) error { return nil }

func (d *DualListbox) Validate() bool {
	return d.FieldValidation.Run(d.RawValue(), d.T)
}

func (d *DualListbox) eventName() string {
	if d.EventName != "" {
		return d.EventName
	}
	return d.CommonAttrs.Name
}

func (d *DualListbox) ev(action string) string {
	base := d.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (d *DualListbox) isSelected(value string) bool {
	for _, v := range d.Values {
		if v == value {
			return true
		}
	}
	return false
}

func (d *DualListbox) availableItems() []SelectItem {
	out := make([]SelectItem, 0, len(d.Items))
	for _, it := range d.Items {
		if !d.isSelected(it.Value) {
			out = append(out, it)
		}
	}
	return out
}

func (d *DualListbox) selectedItems() []SelectItem {
	byVal := make(map[string]SelectItem, len(d.Items))
	for _, it := range d.Items {
		byVal[it.Value] = it
	}
	out := make([]SelectItem, 0, len(d.Values))
	for _, v := range d.Values {
		if it, ok := byVal[v]; ok {
			out = append(out, it)
		} else {
			out = append(out, SelectItem{Value: v, Label: v})
		}
	}
	return out
}

// ApplyAvailableQuery filters the left (available) list server-side.
func (d *DualListbox) ApplyAvailableQuery(query string) {
	d.Query = query
	limit := d.MaxResults
	if limit <= 0 {
		limit = defaultMaxResults
	}
	d.Filtered = FilterItems(d.availableItems(), query, limit)
	d.MarkDirty()
}

// ApplySelectedQuery filters the right (selected) list server-side.
func (d *DualListbox) ApplySelectedQuery(query string) {
	d.SelectedQuery = query
	limit := d.MaxResults
	if limit <= 0 {
		limit = defaultMaxResults
	}
	d.SelectedFilter = FilterItems(d.selectedItems(), query, limit)
	d.MarkDirty()
}

func (d *DualListbox) ensureLists() {
	if d.Filtered == nil {
		d.ApplyAvailableQuery(d.Query)
	}
	if d.SelectedFilter == nil {
		d.ApplySelectedQuery(d.SelectedQuery)
	}
}

func (d *DualListbox) refreshLists() {
	d.ApplyAvailableQuery(d.Query)
	d.ApplySelectedQuery(d.SelectedQuery)
}

func (d *DualListbox) add(value string) {
	if value == "" || d.isSelected(value) {
		return
	}
	d.Values = append(d.Values, value)
}

func (d *DualListbox) remove(value string) {
	out := make([]string, 0, len(d.Values))
	for _, v := range d.Values {
		if v != value {
			out = append(out, v)
		}
	}
	d.Values = out
}

func (d *DualListbox) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := eventAction(event, d.eventName(), payload)
	switch action {
	case "query_left", "query":
		d.ApplyAvailableQuery(payloadString(payload, "value"))
	case "query_right":
		d.ApplySelectedQuery(payloadString(payload, "value"))
	case "add":
		d.add(payloadString(payload, "value"))
		d.refreshLists()
		if d.OnChange != nil {
			d.OnChange(d.RawValue())
		}
	case "remove":
		d.remove(payloadString(payload, "value"))
		d.refreshLists()
		if d.OnChange != nil {
			d.OnChange(d.RawValue())
		}
	case "add_all":
		for _, it := range d.availableItems() {
			d.add(it.Value)
		}
		d.refreshLists()
		if d.OnChange != nil {
			d.OnChange(d.RawValue())
		}
	case "remove_all":
		d.Values = nil
		d.refreshLists()
		if d.OnChange != nil {
			d.OnChange(d.RawValue())
		}
	}
	return nil
}

func (d *DualListbox) Render() (string, error) {
	d.ensureLists()
	attrs := Attrs{}
	attrs = d.CommonAttrs.Apply(attrs)
	attrs = d.FieldValidation.ApplyErrorState(attrs, "goui-dual-listbox")

	phLeft := d.Placeholder
	if phLeft == "" {
		phLeft = "Ara..."
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + ` data-goui-select="dual">`)
	b.WriteString(`<div class="goui-dual-col">`)
	b.WriteString(`<div class="goui-dual-title">Mevcut</div>`)
	b.WriteString(`<input class="goui-searchable-input border border-goui-border rounded-goui px-goui-field py-goui-field w-full" type="search"`)
	b.WriteString(` value="` + html.EscapeString(d.Query) + `" placeholder="` + html.EscapeString(phLeft) + `"`)
	b.WriteString(` g-input="` + html.EscapeString(d.ev("query_left")) + `" g-debounce="150">`)
	b.WriteString(`<ul class="goui-dual-list border border-goui-border rounded-goui">`)
	if len(d.Filtered) == 0 {
		b.WriteString(`<li class="goui-searchable-empty">Sonuç yok</li>`)
	} else {
		for _, it := range d.Filtered {
			b.WriteString(`<li class="goui-searchable-option" g-click="` + html.EscapeString(d.ev("add")) + `" data-goui-value="` + html.EscapeString(it.Value) + `">`)
			b.WriteString(html.EscapeString(displayLabel(it)))
			b.WriteString(`</li>`)
		}
	}
	b.WriteString(`</ul></div>`)

	b.WriteString(`<div class="goui-dual-actions">`)
	b.WriteString(`<button type="button" class="goui-dual-btn" g-click="` + html.EscapeString(d.ev("add_all")) + `">≫</button>`)
	b.WriteString(`<button type="button" class="goui-dual-btn" g-click="` + html.EscapeString(d.ev("remove_all")) + `">≪</button>`)
	b.WriteString(`</div>`)

	b.WriteString(`<div class="goui-dual-col">`)
	b.WriteString(`<div class="goui-dual-title">Seçili</div>`)
	b.WriteString(`<input class="goui-searchable-input border border-goui-border rounded-goui px-goui-field py-goui-field w-full" type="search"`)
	b.WriteString(` value="` + html.EscapeString(d.SelectedQuery) + `" placeholder="` + html.EscapeString(phLeft) + `"`)
	b.WriteString(` g-input="` + html.EscapeString(d.ev("query_right")) + `" g-debounce="150">`)
	b.WriteString(`<ul class="goui-dual-list border border-goui-border rounded-goui">`)
	if len(d.SelectedFilter) == 0 {
		b.WriteString(`<li class="goui-searchable-empty">Seçim yok</li>`)
	} else {
		for _, it := range d.SelectedFilter {
			b.WriteString(`<li class="goui-searchable-option" g-click="` + html.EscapeString(d.ev("remove")) + `" data-goui-value="` + html.EscapeString(it.Value) + `">`)
			b.WriteString(html.EscapeString(displayLabel(it)))
			b.WriteString(`</li>`)
		}
	}
	b.WriteString(`</ul></div>`)
	b.WriteString(`</div>`)
	b.WriteString(d.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
