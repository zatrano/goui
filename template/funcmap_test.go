package template

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"testing"
)

func TestDictFn_EvenPairs(t *testing.T) {
	m, err := dictFn("a", 1, "b", "two")
	if err != nil {
		t.Fatal(err)
	}
	if m["a"] != 1 || m["b"] != "two" {
		t.Fatalf("map = %#v", m)
	}
}

func TestDictFn_OddArgs(t *testing.T) {
	_, err := dictFn("a", 1, "b")
	if err == nil || err.Error() != "dict: odd number of arguments" {
		t.Fatalf("err = %v", err)
	}
}

func TestDictFn_NonStringKey(t *testing.T) {
	_, err := dictFn(1, "x")
	if err == nil || err.Error() != "dict: key 0 is not a string" {
		t.Fatalf("err = %v", err)
	}
}

func TestDictFn_Nested(t *testing.T) {
	inner, err := dictFn("inner", 1)
	if err != nil {
		t.Fatal(err)
	}
	outer, err := dictFn("outer", inner)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := outer["outer"].(map[string]any)
	if !ok || got["inner"] != 1 {
		t.Fatalf("nested = %#v", outer)
	}
}

func TestListFn(t *testing.T) {
	if got := listFn(); len(got) != 0 {
		t.Fatalf("empty list = %#v", got)
	}
	got := listFn(1, "a", true)
	if len(got) != 3 || got[0] != 1 || got[1] != "a" || got[2] != true {
		t.Fatalf("list = %#v", got)
	}
}

func TestDefaultFn(t *testing.T) {
	cases := []struct {
		name     string
		fallback any
		value    any
		want     any
	}{
		{"nil", "Guest", nil, "Guest"},
		{"empty_string", "Guest", "", "Guest"},
		{"zero_int", 42, 0, 42},
		{"false", true, false, true},
		{"empty_slice", "x", []string{}, "x"},
		{"empty_map", "x", map[string]int{}, "x"},
		{"filled_string", "Guest", "Ada", "Ada"},
		{"filled_slice", "x", []int{1}, []int{1}},
		{"nil_pointer", "fb", (*int)(nil), "fb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := defaultFn(tc.fallback, tc.value)
			switch want := tc.want.(type) {
			case []int:
				g, ok := got.([]int)
				if !ok || len(g) != len(want) || (len(want) > 0 && g[0] != want[0]) {
					t.Fatalf("got %#v, want %#v", got, tc.want)
				}
			default:
				if fmt.Sprint(got) != fmt.Sprint(tc.want) && got != tc.want {
					t.Fatalf("got %#v (%T), want %#v", got, got, tc.want)
				}
			}
		})
	}
}

func TestRawFn(t *testing.T) {
	if got := rawFn("<b>x</b>"); got != htmltemplate.HTML("<b>x</b>") {
		t.Fatalf("string: %q", got)
	}
	if got := rawFn(42); got != htmltemplate.HTML("42") {
		t.Fatalf("int: %q", got)
	}
	in := htmltemplate.HTML("<i>y</i>")
	if got := rawFn(in); got != in {
		t.Fatalf("HTML: %q", got)
	}
	if got := rawFn(nil); got != "" {
		t.Fatalf("nil: %q", got)
	}
}

func TestBaseFuncMap_Integration(t *testing.T) {
	tmpl, err := htmltemplate.New("x").Funcs(BaseFuncMap()).Parse(
		`{{ $d := dict "a" 1 "b" 2 }}{{ index $d "a" }}-{{ default "Guest" "" }}-{{ index (list "x" "y") 1 }}-{{ raw "<ok>" }}`,
	)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatal(err)
	}
	want := `1-Guest-y-<ok>`
	if got := buf.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDot_TypeShape(t *testing.T) {
	d := &Dot{
		Props:       map[string]any{"Type": "submit"},
		Slots:       map[string]htmltemplate.HTML{"icon": "<svg/>"},
		DefaultSlot: htmltemplate.HTML("label"),
	}
	if d.Props.(map[string]any)["Type"] != "submit" {
		t.Fatal("Props")
	}
	if d.Slots["icon"] != "<svg/>" || d.DefaultSlot != "label" {
		t.Fatal("Slots")
	}
}
