package forms

import (
	"context"
	"github.com/zatrano/goui/validation"
	"strings"
	"testing"
)

func TestFilterItems_ServerSide(t *testing.T) {
	items := []SelectItem{
		{Value: "tr", Label: "Türkiye"},
		{Value: "de", Label: "Almanya"},
		{Value: "us", Label: "United States"},
	}
	got := FilterItems(items, "alm", 10)
	if len(got) != 1 || got[0].Value != "de" {
		t.Fatalf("FilterItems = %#v", got)
	}
	got = FilterItems(items, "", 2)
	if len(got) != 2 {
		t.Fatalf("limit: got %d", len(got))
	}
}

func TestSearchableSelect_QueryAndSelect(t *testing.T) {
	s := &SearchableSelect{
		BaseSelectField: BaseSelectField{
			CommonAttrs: CommonAttrs{Name: "city"},
			Items: []SelectItem{
				{Value: "ist", Label: "İstanbul"},
				{Value: "ank", Label: "Ankara"},
				{Value: "izm", Label: "İzmir"},
			},
			Placeholder: "Şehir seçin",
		},
		EventName: "city",
	}

	ctx := context.Background()
	if err := s.HandleEvent(ctx, "city.toggle", nil); err != nil {
		t.Fatal(err)
	}
	if !s.Open {
		t.Fatal("expected open")
	}

	if err := s.HandleEvent(ctx, "city.query", map[string]any{"value": "ank"}); err != nil {
		t.Fatal(err)
	}
	if len(s.Filtered) != 1 || s.Filtered[0].Value != "ank" {
		t.Fatalf("server filter failed: %#v", s.Filtered)
	}

	html, err := s.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Ankara") || strings.Contains(html, "İstanbul") {
		t.Fatalf("render should only list filtered items: %s", html)
	}
	if !strings.Contains(html, `g-input="city.query"`) {
		t.Fatalf("missing query binding: %s", html)
	}

	if err := s.HandleEvent(ctx, "city.select", map[string]any{"value": "ank"}); err != nil {
		t.Fatal(err)
	}
	if s.Value != "ank" || s.Open {
		t.Fatalf("after select value=%q open=%v", s.Value, s.Open)
	}

	html, _ = s.Render()
	if !strings.Contains(html, "Ankara") || strings.Contains(html, "goui-searchable-panel") {
		t.Fatalf("closed render should show label, no panel: %s", html)
	}
}

func TestSearchableSelect_Validate(t *testing.T) {
	s := &SearchableSelect{
		BaseSelectField: BaseSelectField{
			CommonAttrs:     CommonAttrs{Name: "city"},
			FieldValidation: FieldValidation{Rules: []validation.Rule{validation.Required()}},
		},
	}
	if s.Validate() {
		t.Fatal("empty should fail Required")
	}
	s.Value = "ank"
	if !s.Validate() {
		t.Fatal("expected pass")
	}
}
