package forms

import (
	"context"
	"strings"
	"testing"
)

func TestNumberFormat_TR(t *testing.T) {
	got := NumberFormat(1234.5, "tr", 2)
	if got != "1.234,50" {
		t.Fatalf("got %q", got)
	}
	got = NumberFormat(1234.56, "en", 2)
	if got != "1,234.56" {
		t.Fatalf("en got %q", got)
	}
}

func TestParseLocalizedNumber_TR(t *testing.T) {
	n, err := ParseLocalizedNumber("1.234,56 ₺", "tr")
	if err != nil || n != 1234.56 {
		t.Fatalf("n=%v err=%v", n, err)
	}
	n, err = ParseLocalizedNumber("$1,234.50", "en")
	if err != nil || n != 1234.5 {
		t.Fatalf("en n=%v err=%v", n, err)
	}
}

func TestCurrencyInput_CommitFormats(t *testing.T) {
	c := &CurrencyInput{
		CommonAttrs: CommonAttrs{Name: "price"},
		Currency:    "TRY",
		Locale:      "tr",
		EventName:   "price",
	}
	ctx := context.Background()
	_ = c.HandleEvent(ctx, "price.commit", map[string]any{"value": "1.250,75"})
	if c.Value != 1250.75 {
		t.Fatalf("value=%v", c.Value)
	}
	if c.Draft != "" {
		t.Fatalf("draft should clear")
	}
	html, err := c.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "1.250,75") || !strings.Contains(html, "₺") {
		t.Fatalf("render: %s", html)
	}
}

func TestPercentageInput_Commit(t *testing.T) {
	max := 100.0
	p := &PercentageInput{
		CommonAttrs: CommonAttrs{Name: "vat"},
		EventName:   "vat",
		Max:         &max,
	}
	ctx := context.Background()
	_ = p.HandleEvent(ctx, "vat.commit", map[string]any{"value": "18,5 %"})
	if p.Value != 18.5 {
		t.Fatalf("value=%v", p.Value)
	}
	html, _ := p.Render()
	if !strings.Contains(html, "%") {
		t.Fatalf("render: %s", html)
	}
}

func TestRating_SetAndToggle(t *testing.T) {
	r := &Rating{
		CommonAttrs: CommonAttrs{Name: "score"},
		EventName:   "score",
		Max:         5,
	}
	ctx := context.Background()
	_ = r.HandleEvent(ctx, "score.set", map[string]any{"value": "4"})
	if r.Value != 4 {
		t.Fatalf("value=%d", r.Value)
	}
	_ = r.HandleEvent(ctx, "score.set", map[string]any{"value": "4"})
	if r.Value != 0 {
		t.Fatalf("toggle off expected 0, got %d", r.Value)
	}
	html, _ := r.Render()
	if !strings.Contains(html, "goui-rating") || strings.Count(html, "goui-rating-star") != 5 {
		t.Fatalf("render: %s", html)
	}
}
