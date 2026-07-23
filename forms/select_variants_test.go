package forms

import (
	"context"
	"strings"
	"testing"
)

func sampleCities() []SelectItem {
	return []SelectItem{
		{Value: "ist", Label: "İstanbul"},
		{Value: "ank", Label: "Ankara"},
		{Value: "izm", Label: "İzmir"},
	}
}

func TestMultiSelect_ToggleValues(t *testing.T) {
	m := &MultiSelect{
		BaseSelectField: BaseSelectField{
			CommonAttrs: CommonAttrs{Name: "tags"},
			Items:       sampleCities(),
		},
		EventName: "tags",
	}
	ctx := context.Background()
	_ = m.HandleEvent(ctx, "tags.select", map[string]any{"value": "ank"})
	_ = m.HandleEvent(ctx, "tags.select", map[string]any{"value": "izm"})
	if m.RawValue() != "ank,izm" {
		t.Fatalf("values=%q", m.RawValue())
	}
	_ = m.HandleEvent(ctx, "tags.select", map[string]any{"value": "ank"})
	if m.RawValue() != "izm" {
		t.Fatalf("after toggle off: %q", m.RawValue())
	}
	_ = m.HandleEvent(ctx, "tags.query", map[string]any{"value": "izm"})
	if len(m.Filtered) != 1 || m.Filtered[0].Value != "izm" {
		t.Fatalf("filter: %#v", m.Filtered)
	}
	html, _ := m.Render()
	if !strings.Contains(html, "goui-chip") || !strings.Contains(html, "İzmir") {
		t.Fatalf("render chips: %s", html)
	}
}

func TestCombobox_FreeTextAndSelect(t *testing.T) {
	c := &Combobox{
		BaseSelectField: BaseSelectField{
			CommonAttrs: CommonAttrs{Name: "job"},
			Items:       sampleCities(),
		},
		EventName: "job",
	}
	ctx := context.Background()
	_ = c.HandleEvent(ctx, "job.query", map[string]any{"value": "custom-role"})
	if c.Value != "custom-role" {
		t.Fatalf("free text value=%q", c.Value)
	}
	_ = c.HandleEvent(ctx, "job.select", map[string]any{"value": "ank"})
	if c.Value != "ank" {
		t.Fatalf("select value=%q", c.Value)
	}
}

func TestAutocomplete_Suggestions(t *testing.T) {
	a := &Autocomplete{
		BaseSelectField: BaseSelectField{
			CommonAttrs: CommonAttrs{Name: "city"},
			Items:       sampleCities(),
		},
		EventName: "city",
	}
	ctx := context.Background()
	_ = a.HandleEvent(ctx, "city.query", map[string]any{"value": "ank"})
	if !a.Open || len(a.Filtered) != 1 {
		t.Fatalf("open=%v filtered=%#v", a.Open, a.Filtered)
	}
	html, _ := a.Render()
	if !strings.Contains(html, "Ankara") {
		t.Fatalf("html: %s", html)
	}
	_ = a.HandleEvent(ctx, "city.select", map[string]any{"value": "ank"})
	if a.Value != "ank" || a.Open {
		t.Fatalf("value=%q open=%v", a.Value, a.Open)
	}
}
