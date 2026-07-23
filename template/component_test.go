package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func componentRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "component")
}

func TestComponent_SlotsAndCallerDot(t *testing.T) {
	reg, err := NewRegistry(Config{Root: componentRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	got, err := reg.Render("pages.demo", map[string]any{
		"PageTitle": "Demo Page",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Caller context available inside @slot.
	if !strings.Contains(got, "Header: Demo Page") {
		t.Fatalf("slot did not see caller Dot (.PageTitle):\n%s", got)
	}
	if !strings.Contains(got, "First card body") {
		t.Fatalf("missing default slot:\n%s", got)
	}

	// Second card: optional header absent, no error, default slot present.
	if !strings.Contains(got, "Second card — no header slot") {
		t.Fatalf("missing second card default:\n%s", got)
	}
	if strings.Count(got, "<header>") != 1 {
		t.Fatalf("expected exactly one header, got:\n%s", got)
	}

	// Nested component rendered badge with prop.
	if !strings.Contains(got, `<span class="badge">NESTED</span>`) {
		t.Fatalf("nested badge missing:\n%s", got)
	}
	if !strings.Contains(got, "nested-default") {
		t.Fatalf("nested default slot missing:\n%s", got)
	}
}

func TestComponent_DuplicateCallsUniqueDefines(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "component", "pages", "demo.goui.html"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := ParseSource("pages/demo.goui.html", src)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := Generate(f)
	if err != nil {
		t.Fatal(err)
	}
	var cardSlots []string
	for name := range unit.SlotDefines {
		if strings.Contains(name, "components.card") {
			cardSlots = append(cardSlots, name)
		}
	}
	if len(cardSlots) < 2 {
		t.Fatalf("expected multiple card slot defines, got %#v", unit.SlotDefines)
	}
	seen := map[string]bool{}
	for _, n := range cardSlots {
		if seen[n] {
			t.Fatalf("duplicate define name %q", n)
		}
		seen[n] = true
	}
	has1, has2 := false, false
	for _, n := range cardSlots {
		if strings.Contains(n, "__1__") {
			has1 = true
		}
		if strings.Contains(n, "__2__") {
			has2 = true
		}
	}
	if !has1 || !has2 {
		t.Fatalf("expected counters 1 and 2 in %#v", cardSlots)
	}
}

func TestComponent_OptionalSlotEmpty(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "components/box.goui.html", `<div>{{ if .Slots.opt }}{{ .Slots.opt }}{{ end }}{{ .DefaultSlot }}</div>`)
	writeGoui(t, dir, "pages/p.goui.html", `@component("components.box")
only-default
@endcomponent
`)
	reg, err := NewRegistry(Config{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	got, err := reg.Render("pages.p", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "only-default") {
		t.Fatalf("got %q", got)
	}
}
