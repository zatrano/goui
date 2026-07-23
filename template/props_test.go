package template

import (
	"strings"
	"testing"
)

func TestParseProps(t *testing.T) {
	cases := []struct {
		raw  string
		want []PropDecl
	}{
		{"", nil},
		{"Name string", []PropDecl{{Name: "Name", Type: "string"}}},
		{"Count int = 0", []PropDecl{{Name: "Count", Type: "int", Default: "0"}}},
		{
			"Name string, Count int = 0, Items []string",
			[]PropDecl{
				{Name: "Name", Type: "string"},
				{Name: "Count", Type: "int", Default: "0"},
				{Name: "Items", Type: "[]string"},
			},
		},
		{
			"Data map[string]string",
			[]PropDecl{{Name: "Data", Type: "map[string]string"}},
		},
		{"OnlyName", []PropDecl{{Name: "OnlyName"}}},
	}
	for _, tc := range cases {
		got, err := ParseProps(tc.raw)
		if err != nil {
			t.Fatalf("ParseProps(%q): %v", tc.raw, err)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("ParseProps(%q) len=%d want %d (%#v)", tc.raw, len(got), len(tc.want), got)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("ParseProps(%q)[%d]=%#v want %#v", tc.raw, i, got[i], tc.want[i])
			}
		}
	}
}

func TestExtractFieldRefs(t *testing.T) {
	src := `@props(Name string, Count int)
@if(.Props.Name)
  {{ .Props.Count }}
  {!! .Props.Name !!}
@endif
@foreach(.Props.Items as $item)
  x
@endforeach
`
	f, err := ParseSource("c.goui.html", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractFieldRefs(f)
	joined := strings.Join(got, ",")
	for _, want := range []string{"Count", "Items", "Name"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, got)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		dist int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"kitten", "sitting", 3},
		{"Name", "Naem", 2},
		{"Name", "Name", 0},
	}
	for _, tc := range cases {
		if got := levenshtein(tc.a, tc.b); got != tc.dist {
			t.Fatalf("levenshtein(%q,%q)=%d want %d", tc.a, tc.b, got, tc.dist)
		}
	}
	if sug := didYouMean("Naem", []string{"Name", "Count"}); sug != "Name" {
		t.Fatalf("didYouMean = %q", sug)
	}
}

func TestStrictProps_TypoError(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/home.goui.html", `@props(Name string)
<p>{{ .Props.Naem }}</p>
`)
	_, err := NewRegistry(Config{Root: dir, StrictProps: true})
	if err == nil {
		t.Fatal("expected StrictProps error")
	}
	msg := err.Error()
	if !strings.Contains(msg, ".Props.Naem used but not declared") {
		t.Fatalf("message = %q", msg)
	}
	if !strings.Contains(msg, "did you mean Name?") {
		t.Fatalf("expected suggestion, got %q", msg)
	}
}

func TestStrictProps_DisabledAllowsTypo(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/home.goui.html", `@props(Name string)
<p>{{ .Props.Naem }}</p>
`)
	reg, err := NewRegistry(Config{Root: dir, StrictProps: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Warnings()) != 0 {
		// unused Name may still warn only when StrictProps is on
		t.Fatalf("warnings without StrictProps: %v", reg.Warnings())
	}
}

func TestStrictProps_UnusedWarning(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/home.goui.html", `@props(Name string, Unused int)
<p>{{ .Props.Name }}</p>
`)
	reg, err := NewRegistry(Config{Root: dir, StrictProps: true})
	if err != nil {
		t.Fatal(err)
	}
	warns := reg.Warnings()
	if len(warns) != 1 {
		t.Fatalf("warnings = %v", warns)
	}
	if !strings.Contains(warns[0], `field "Unused" is declared but unused`) {
		t.Fatalf("warning = %q", warns[0])
	}
}

func TestStrictProps_OK(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "components/card.goui.html", `@props(Title string)
<h1>{{ .Props.Title }}</h1>
`)
	reg, err := NewRegistry(Config{Root: dir, StrictProps: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Warnings()) != 0 {
		t.Fatalf("unexpected warnings: %v", reg.Warnings())
	}
}

func TestParseProps_Invalid(t *testing.T) {
	_, err := ParseProps("123bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWalkExprs_CoversNodes(t *testing.T) {
	src := `@unless(.Props.A)
@switch(.Props.B)
@case("x")
{{ .Props.C }}
@break
@default
y
@endswitch
@endunless
@include("partials.nav", .Props.D)
@component("components.card", dict "T" .Props.E)
  @slot("h")
  {{ .Props.F }}
  @endslot
@endcomponent
`
	f, err := ParseSource("t.goui.html", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractFieldRefs(f)
	for _, want := range []string{"A", "B", "C", "D", "E", "F"} {
		found := false
		for _, g := range got {
			if g == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %q in %v", want, got)
		}
	}
}
