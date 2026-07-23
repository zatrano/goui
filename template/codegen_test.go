package template

import (
	htmltemplate "html/template"
	"strings"
	"testing"
)

func genSrc(t *testing.T, src string) *CompileUnit {
	t.Helper()
	f, err := ParseSource("t.goui.html", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	unit, err := Generate(f)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return unit
}

func assertSmokeParse(t *testing.T, src string) {
	t.Helper()
	funcs := BaseFuncMap()
	funcs["component"] = func(target string, props any, callerDot any, slotArgs ...string) htmltemplate.HTML {
		return ""
	}
	tmpl := htmltemplate.New("smoke").Funcs(funcs)
	if _, err := tmpl.Parse(src); err != nil {
		t.Fatalf("html/template.Parse failed: %v\nsource:\n%s", err, src)
	}
}

func TestCodegen_EscapedOutput(t *testing.T) {
	unit := genSrc(t, `Hello {{ .Name }}!`)
	got := strings.TrimSpace(unit.Body)
	want := `Hello {{ .Name }}!`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_RawOutput(t *testing.T) {
	unit := genSrc(t, `{!! .RawHTML !!}`)
	got := strings.TrimSpace(unit.Body)
	want := `{{ raw (.RawHTML) }}`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_IfElseIfElse(t *testing.T) {
	src := `@if(.A)
A
@elseif(.B)
B
@else
C
@endif`
	unit := genSrc(t, src)
	got := strings.TrimSpace(unit.Body)
	want := "{{if .A}}\nA\n{{else if .B}}\nB\n{{else}}\nC\n{{end}}"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_Unless(t *testing.T) {
	unit := genSrc(t, "@unless(.Hidden)\nx\n@endunless")
	got := strings.TrimSpace(unit.Body)
	want := "{{if not (.Hidden)}}\nx\n{{end}}"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_SwitchCasesDefault(t *testing.T) {
	src := `@switch(.Status)
@case("a")
A
@break
@case("b")
B
@break
@default
D
@endswitch`
	unit := genSrc(t, src)
	got := strings.TrimSpace(unit.Body)
	want := `{{if eq .Status "a"}}
A
{{else if eq .Status "b"}}
B
{{else}}
D
{{end}}`
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_ForeachEmpty(t *testing.T) {
	src := `@foreach(.Items as $item)
{{ $item }}
@empty
none
@endforeach`
	unit := genSrc(t, src)
	got := strings.TrimSpace(unit.Body)
	if !strings.Contains(got, "{{else}}") {
		t.Fatalf("expected {{else}} for @empty, got %q", got)
	}
	want := "{{range $item := .Items}}\n{{ $item }}\n{{else}}\nnone\n{{end}}"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_ForeachWithKey(t *testing.T) {
	unit := genSrc(t, "@foreach(.Items as $k, $v)\nx\n@endforeach")
	got := strings.TrimSpace(unit.Body)
	want := "{{range $k, $v := .Items}}\nx\n{{end}}"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_Yield(t *testing.T) {
	unit := genSrc(t, `@yield("title", "App")`)
	got := strings.TrimSpace(unit.Body)
	want := `{{block "title" .}}App{{end}}`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_Include(t *testing.T) {
	unit := genSrc(t, `@include("partials.nav")
@include("partials.user", .User)`)
	if !strings.Contains(unit.Body, `{{template "partials.nav" .}}`) {
		t.Fatalf("body = %q", unit.Body)
	}
	if !strings.Contains(unit.Body, `{{template "partials.user" .User}}`) {
		t.Fatalf("body = %q", unit.Body)
	}
	if len(unit.Includes) != 2 {
		t.Fatalf("Includes = %#v", unit.Includes)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_IncludeIfMarker(t *testing.T) {
	unit := genSrc(t, `@includeIf("partials.x")`)
	marker := FormatIncludeIfMarker("partials.x")
	if strings.TrimSpace(unit.Body) != marker {
		t.Fatalf("body = %q, want %q", unit.Body, marker)
	}
	if len(unit.ConditionalIncludes) != 1 || unit.ConditionalIncludes[0] != "partials.x" {
		t.Fatalf("ConditionalIncludes = %#v", unit.ConditionalIncludes)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_ExtendsSections(t *testing.T) {
	src := `@extends("layouts.app")
@section("title", "Home")
@section("content")
<p>hi</p>
@endsection
`
	unit := genSrc(t, src)
	if unit.Body != "" {
		t.Fatalf("Body should be empty with @extends, got %q", unit.Body)
	}
	if unit.Extends != "layouts.app" {
		t.Fatalf("Extends = %q", unit.Extends)
	}
	if unit.Sections["title"] != `{{define "title"}}Home{{end}}` {
		t.Fatalf("title section = %q", unit.Sections["title"])
	}
	content := unit.Sections["content"]
	if !strings.HasPrefix(content, `{{define "content"}}`) || !strings.HasSuffix(content, `{{end}}`) {
		t.Fatalf("content section = %q", content)
	}
	if !strings.Contains(content, "<p>hi</p>") {
		t.Fatalf("content missing body: %q", content)
	}
	for name, src := range unit.Sections {
		assertSmokeParse(t, src)
		_ = name
	}
}

func TestCodegen_ComponentPlaceholder(t *testing.T) {
	src := `@component("components.button", dict "Type" "submit")
ok
@endcomponent`
	unit := genSrc(t, src)
	got := strings.TrimSpace(unit.Body)
	want := `{{ component "components.button" (dict "Type" "submit") . "" "__slot__components.button__1____default__" }}`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if len(unit.Components) != 1 || unit.Components[0] != "components.button" {
		t.Fatalf("Components = %#v", unit.Components)
	}
	if _, ok := unit.SlotDefines["__slot__components.button__1____default__"]; !ok {
		t.Fatalf("SlotDefines = %#v", unit.SlotDefines)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_PropsCollected(t *testing.T) {
	unit := genSrc(t, `@props(Name string)
<div>{{ .Name }}</div>`)
	if unit.Props != "Name string" {
		t.Fatalf("Props = %q", unit.Props)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_CommentOmitted(t *testing.T) {
	unit := genSrc(t, `a{{-- secret --}}b`)
	if strings.TrimSpace(unit.Body) != "ab" {
		t.Fatalf("body = %q", unit.Body)
	}
	assertSmokeParse(t, unit.Body)
}

func TestCodegen_ArchitectureTableGolden(t *testing.T) {
	// One case per architecture §5 mapping row (where applicable as a fragment).
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"if", "@if(.X)\nY\n@endif", "{{if .X}}\nY\n{{end}}"},
		{"else", "@if(.X)\nA\n@else\nB\n@endif", "{{if .X}}\nA\n{{else}}\nB\n{{end}}"},
		{"elseif", "@if(.X)\nA\n@elseif(.Y)\nB\n@endif", "{{if .X}}\nA\n{{else if .Y}}\nB\n{{end}}"},
		{"unless", "@unless(.X)\nY\n@endunless", "{{if not (.X)}}\nY\n{{end}}"},
		{"foreach", "@foreach(.Items as $item)\nX\n@endforeach", "{{range $item := .Items}}\nX\n{{end}}"},
		{"yield", `@yield("n")`, `{{block "n" .}}{{end}}`},
		{"include", `@include("p.nav")`, `{{template "p.nav" .}}`},
		{"escaped", `{{ .A }}`, `{{ .A }}`},
		{"raw", `{!! .A !!}`, `{{ raw (.A) }}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unit := genSrc(t, tc.src)
			got := strings.TrimSpace(unit.Body)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			assertSmokeParse(t, unit.Body)
		})
	}
}

func TestParser_SectionRequiresExtends(t *testing.T) {
	te := parseErr(t, "t.goui.html", `@section("title")
Hi
@endsection
`)
	if !strings.Contains(te.Message, "@section is only allowed in files that use @extends") {
		t.Fatalf("message = %q", te.Message)
	}
}

func TestGenerate_NilFile(t *testing.T) {
	_, err := Generate(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeriveName(t *testing.T) {
	if got := deriveName("pages/home.goui.html"); got != "pages.home" {
		t.Fatalf("got %q", got)
	}
}
