package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parseSrc(t *testing.T, filename, src string) *File {
	t.Helper()
	f, err := ParseSource(filename, []byte(src))
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	return f
}

func parseErr(t *testing.T, filename, src string) *TemplateError {
	t.Helper()
	_, err := ParseSource(filename, []byte(src))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	te, ok := err.(*TemplateError)
	if !ok {
		t.Fatalf("expected *TemplateError, got %T: %v", err, err)
	}
	return te
}

func TestParser_RawTextAndOutput(t *testing.T) {
	f := parseSrc(t, "t.goui.html", `Hi {{ .Name }} {!! .HTML !!}`)
	if len(f.Nodes) != 4 {
		t.Fatalf("nodes = %d: %#v", len(f.Nodes), f.Nodes)
	}
	if _, ok := f.Nodes[0].(*RawTextNode); !ok {
		t.Fatalf("node0: %T", f.Nodes[0])
	}
	out := f.Nodes[1].(*OutputNode)
	if out.Raw || out.Expr != ".Name" {
		t.Fatalf("escaped: %+v", out)
	}
	raw := f.Nodes[3].(*OutputNode)
	if !raw.Raw || raw.Expr != ".HTML" {
		t.Fatalf("raw: %+v", raw)
	}
}

func TestParser_CommentSkipped(t *testing.T) {
	f := parseSrc(t, "t.goui.html", `a{{-- secret --}}b`)
	if len(f.Nodes) != 2 {
		t.Fatalf("nodes = %d (comment must be skipped)", len(f.Nodes))
	}
}

func TestParser_IfElseIfElse(t *testing.T) {
	src := `@if(.A)
A
@elseif(.B)
B
@else
C
@endif`
	f := parseSrc(t, "t.goui.html", src)
	n := f.Nodes[0].(*IfNode)
	if len(n.Branches) != 2 {
		t.Fatalf("branches = %d", len(n.Branches))
	}
	if n.Branches[0].Expr != ".A" || n.Branches[1].Expr != ".B" {
		t.Fatalf("exprs: %+v", n.Branches)
	}
	if n.Else == nil {
		t.Fatal("expected else body")
	}
}

func TestParser_Unless(t *testing.T) {
	f := parseSrc(t, "t.goui.html", "@unless(.Hidden)\nx\n@endunless")
	n := f.Nodes[0].(*UnlessNode)
	if n.Expr != ".Hidden" {
		t.Fatalf("expr = %q", n.Expr)
	}
}

func TestParser_Switch(t *testing.T) {
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
	f := parseSrc(t, "t.goui.html", src)
	n := f.Nodes[0].(*SwitchNode)
	if n.Expr != ".Status" {
		t.Fatalf("expr = %q", n.Expr)
	}
	if len(n.Cases) != 2 || n.Cases[0].Value != `"a"` || n.Cases[1].Value != `"b"` {
		t.Fatalf("cases: %+v", n.Cases)
	}
	if n.Default == nil {
		t.Fatal("expected default")
	}
}

func TestParser_ForeachWithKey(t *testing.T) {
	src := `@foreach(.Items as $key, $item)
{{ $item }}
@empty
none
@endforeach`
	f := parseSrc(t, "t.goui.html", src)
	n := f.Nodes[0].(*ForeachNode)
	if n.Expr != ".Items" || n.KeyVar != "$key" || n.ValueVar != "$item" {
		t.Fatalf("foreach: %+v", n)
	}
	if n.Empty == nil {
		t.Fatal("expected empty body")
	}
}

func TestParser_ForeachValueOnly(t *testing.T) {
	f := parseSrc(t, "t.goui.html", "@foreach(.Items as $item)\nx\n@endforeach")
	n := f.Nodes[0].(*ForeachNode)
	if n.KeyVar != "" || n.ValueVar != "$item" {
		t.Fatalf("foreach: %+v", n)
	}
}

func TestParser_ForeachRequiresDollar(t *testing.T) {
	te := parseErr(t, "t.goui.html", `@foreach(.Items as .item)x@endforeach`)
	if !strings.Contains(te.Message, "foreach variable must start with $") {
		t.Fatalf("message = %q", te.Message)
	}
}

func TestParser_ExtendsAndSections(t *testing.T) {
	src := `@extends("layouts.app")
@section("title", "Home")
@section("content")
<p>hi</p>
@endsection
`
	f := parseSrc(t, "t.goui.html", src)
	var ext *ExtendsNode
	var short *SectionNode
	var long *SectionNode
	for _, n := range f.Nodes {
		switch v := n.(type) {
		case *ExtendsNode:
			ext = v
		case *SectionNode:
			if v.Inline != "" {
				short = v
			} else {
				long = v
			}
		}
	}
	if ext == nil || ext.Layout != "layouts.app" {
		t.Fatalf("extends: %+v", ext)
	}
	if short == nil || short.Name != "title" || short.Inline != "Home" {
		t.Fatalf("short section: %+v", short)
	}
	if long == nil || long.Name != "content" {
		t.Fatalf("long section: %+v", long)
	}
}

func TestParser_SectionCommaInQuotes(t *testing.T) {
	f := parseSrc(t, "t.goui.html", `@extends("layouts.app")
@section("name", "a, b, c")
`)
	var sec *SectionNode
	for _, n := range f.Nodes {
		if s, ok := n.(*SectionNode); ok {
			sec = s
			break
		}
	}
	if sec == nil || sec.Inline != "a, b, c" {
		t.Fatalf("section: %+v", sec)
	}
}

func TestParser_YieldInclude(t *testing.T) {
	src := `@yield("title", "App")
@include("partials.nav")
@include("partials.user", .User)
@includeIf("partials.opt")
`
	f := parseSrc(t, "t.goui.html", src)
	var y *YieldNode
	var incs []*IncludeNode
	for _, n := range f.Nodes {
		switch v := n.(type) {
		case *YieldNode:
			y = v
		case *IncludeNode:
			incs = append(incs, v)
		}
	}
	if y == nil || y.Name != "title" || len(y.Default) != 1 {
		t.Fatalf("yield: %+v", y)
	}
	if len(incs) != 3 {
		t.Fatalf("includes = %d", len(incs))
	}
	if incs[0].Target != "partials.nav" || incs[0].If {
		t.Fatalf("include: %+v", incs[0])
	}
	if incs[1].DataExpr != ".User" {
		t.Fatalf("include data: %+v", incs[1])
	}
	if !incs[2].If || incs[2].Target != "partials.opt" {
		t.Fatalf("includeIf: %+v", incs[2])
	}
}

func TestParser_ComponentSlots(t *testing.T) {
	src := `@component("components.card", dict "Title" "Hi")
  @slot("header")
    <h1>{{ .Title }}</h1>
  @endslot
  <p>default body</p>
  @slot("footer")
    bye
  @endslot
@endcomponent`
	f := parseSrc(t, "t.goui.html", src)
	c := f.Nodes[0].(*ComponentNode)
	if c.Target != "components.card" {
		t.Fatalf("target = %q", c.Target)
	}
	if !strings.Contains(c.DataExpr, "dict") {
		t.Fatalf("DataExpr = %q", c.DataExpr)
	}
	if len(c.Slots) != 2 {
		t.Fatalf("slots = %d", len(c.Slots))
	}
	if c.Slots[0].Name != "header" || c.Slots[1].Name != "footer" {
		t.Fatalf("slot names: %+v", c.Slots)
	}
	if len(c.Default) == 0 {
		t.Fatal("expected default slot content")
	}
}

func TestParser_Props(t *testing.T) {
	f := parseSrc(t, "t.goui.html", `@props(Name string, Count int = 0)
<div>{{ .Name }}</div>`)
	p := f.Nodes[0].(*PropsNode)
	if p.Raw != "Name string, Count int = 0" {
		t.Fatalf("Raw = %q", p.Raw)
	}
}

func TestParser_PropsMustBeFirst(t *testing.T) {
	te := parseErr(t, "t.goui.html", `<div></div>
@props(Name string)
`)
	if !strings.Contains(te.Message, "@props must be the first statement") {
		t.Fatalf("message = %q", te.Message)
	}
}

func TestParser_UnclosedIf(t *testing.T) {
	te := parseErr(t, "t.goui.html", "@if(.A)\nhello")
	if !strings.Contains(te.Message, "unclosed @if") {
		t.Fatalf("message = %q", te.Message)
	}
	if te.Line != 1 {
		t.Fatalf("line = %d", te.Line)
	}
}

func TestParser_UnexpectedEndif(t *testing.T) {
	te := parseErr(t, "t.goui.html", "@endif")
	if !strings.Contains(te.Message, "unexpected @endif") {
		t.Fatalf("message = %q", te.Message)
	}
}

func TestParser_UnclosedForeach(t *testing.T) {
	te := parseErr(t, "bad.goui.html", "@foreach(.Items as $item)\nx")
	if !strings.Contains(te.Message, "unclosed @foreach") {
		t.Fatalf("message = %q", te.Message)
	}
	if te.File != "bad.goui.html" || te.Line != 1 {
		t.Fatalf("pos = %s:%d", te.File, te.Line)
	}
}

func TestParser_ExtendsForbiddenContent(t *testing.T) {
	te := parseErr(t, "t.goui.html", `@extends("layouts.app")
<p>not allowed</p>
@section("content")
x
@endsection
`)
	if !strings.Contains(te.Message, "only @section blocks are allowed at top level") {
		t.Fatalf("message = %q", te.Message)
	}
}

func TestParser_NestedThreeLevels(t *testing.T) {
	src := `@if(.Outer)
  @foreach(.Items as $item)
    @if($item.Ok)
      yes
    @endif
  @endforeach
@endif`
	f := parseSrc(t, "t.goui.html", src)
	outer := f.Nodes[0].(*IfNode)
	var foreach *ForeachNode
	for _, n := range outer.Branches[0].Body {
		if fn, ok := n.(*ForeachNode); ok {
			foreach = fn
			break
		}
	}
	if foreach == nil {
		t.Fatal("missing nested foreach")
	}
	var inner *IfNode
	for _, n := range foreach.Body {
		if in, ok := n.(*IfNode); ok {
			inner = in
			break
		}
	}
	if inner == nil {
		t.Fatal("missing inner if")
	}
	if inner.Branches[0].Expr != "$item.Ok" {
		t.Fatalf("inner expr = %q", inner.Branches[0].Expr)
	}
}

func TestParser_UnknownDirective(t *testing.T) {
	te := parseErr(t, "t.goui.html", "@foo(1)")
	if !strings.Contains(te.Message, "unknown directive @foo") {
		t.Fatalf("message = %q", te.Message)
	}
}

func TestParser_SlotOutsideComponent(t *testing.T) {
	te := parseErr(t, "t.goui.html", `@slot("x")y@endslot`)
	if !strings.Contains(te.Message, "unexpected @slot") {
		t.Fatalf("message = %q", te.Message)
	}
}

func TestNode_PosMethods(t *testing.T) {
	pos := Position{File: "x", Line: 1, Column: 2}
	nodes := []Node{
		&RawTextNode{Position: pos},
		&OutputNode{Position: pos},
		&IfNode{Position: pos},
		&UnlessNode{Position: pos},
		&SwitchNode{Position: pos},
		&ForeachNode{Position: pos},
		&ExtendsNode{Position: pos},
		&SectionNode{Position: pos},
		&YieldNode{Position: pos},
		&IncludeNode{Position: pos},
		&SlotNode{Position: pos},
		&ComponentNode{Position: pos},
		&PropsNode{Position: pos},
	}
	for _, n := range nodes {
		if n.Pos() != pos {
			t.Fatalf("%T.Pos() = %+v", n, n.Pos())
		}
	}
}

func TestParser_ErrorPaths(t *testing.T) {
	cases := []struct {
		name string
		src  string
		msg  string
	}{
		{"if_empty_cond", "@if\n@endif", "@if requires a condition"},
		{"if_empty_parens", "@if()\n@endif", "@if requires a condition"},
		{"elseif_empty", "@if(.A)\n@elseif()\n@endif", "@elseif requires a condition"},
		{"unless_empty", "@unless\n@endunless", "@unless requires a condition"},
		{"switch_empty", "@switch\n@endswitch", "@switch requires an expression"},
		{"case_empty", "@switch(.X)\n@case()\n@endswitch", "@case requires a value"},
		{"foreach_empty", "@foreach\n@endforeach", "@foreach requires arguments"},
		{"foreach_no_as", "@foreach(.Items)\n@endforeach", `exactly one " as "`},
		{"foreach_bad_key", "@foreach(.Items as key, $item)\n@endforeach", "must start with $"},
		{"section_unclosed", "@section(\"c\")\nhello", "unclosed @section"},
		{"section_bad_name", "@section(name)\n@endsection", "@section name"},
		{"yield_bad", "@yield(name)", "@yield name"},
		{"extends_bad", "@extends(layouts.app)", "@extends"},
		{"include_too_many", `@include("a", .B, .C)`, "at most two arguments"},
		{"component_unclosed", `@component("c.button")
hello`, "unclosed @component"},
		{"slot_unclosed", `@component("c.x")
@slot("a")
hi`, "unclosed @slot"},
		{"multiple_extends", `@extends("a")
@extends("b")`, "multiple @extends"},
		{"switch_unexpected", "@switch(.X)\n@if(.A)\n@endif\n@endswitch", "unexpected"},
		{"lex_error_propagates", "hello {{ .Name", "unclosed output"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseSource("t.goui.html", []byte(tc.src))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.msg)
			}
		})
	}
}

func TestSplitAndUnquote(t *testing.T) {
	parts, err := splitTopLevelArgs(`"a, b", 'c\'d', .Expr`)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 3 {
		t.Fatalf("parts = %#v", parts)
	}
	s, err := unquoteString(`"hello\"world"`)
	if err != nil || s != `hello"world` {
		t.Fatalf("unquote = %q (%v)", s, err)
	}
	_, err = unquoteString("nope")
	if err == nil {
		t.Fatal("expected unquote error")
	}
}

func TestTemplateError_Unwrap(t *testing.T) {
	inner := errorAt(Position{File: "a", Line: 1, Column: 1}, "inner")
	outer := &TemplateError{File: "a", Line: 1, Column: 1, Message: "outer", Wrapped: inner}
	if outer.Unwrap() != inner {
		t.Fatal("Unwrap failed")
	}
}

func TestParser_Fixtures(t *testing.T) {
	cases := []struct {
		file      string
		wantTypes []string
	}{
		{
			file: "simple_page.goui.html",
			wantTypes: []string{
				"*template.RawTextNode",
				"*template.OutputNode",
				"*template.RawTextNode",
				"*template.OutputNode",
				"*template.RawTextNode",
				"*template.IfNode",
				"*template.RawTextNode",
			},
		},
		{
			file: "layout.goui.html",
			wantTypes: []string{
				"*template.RawTextNode",
				"*template.YieldNode",
				"*template.RawTextNode",
				"*template.IncludeNode",
				"*template.RawTextNode",
				"*template.YieldNode",
				"*template.RawTextNode",
			},
		},
		{
			file: "form.goui.html",
			wantTypes: []string{
				"*template.RawTextNode",
				"*template.RawTextNode", // comment skipped; text around it may merge? no - comment between texts = two RawText
				"*template.OutputNode",
				"*template.RawTextNode",
				"*template.IfNode",
				"*template.RawTextNode",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("testdata", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			f, err := ParseSource(tc.file, src)
			if err != nil {
				t.Fatal(err)
			}
			if len(f.Nodes) != len(tc.wantTypes) {
				var got []string
				for _, n := range f.Nodes {
					got = append(got, typeName(n))
				}
				t.Fatalf("node count = %d, want %d\ngot types: %v", len(f.Nodes), len(tc.wantTypes), got)
			}
			for i, want := range tc.wantTypes {
				if got := typeName(f.Nodes[i]); got != want {
					t.Fatalf("node[%d] = %s, want %s", i, got, want)
				}
			}
		})
	}
}

func typeName(n Node) string {
	switch n.(type) {
	case *RawTextNode:
		return "*template.RawTextNode"
	case *OutputNode:
		return "*template.OutputNode"
	case *IfNode:
		return "*template.IfNode"
	case *UnlessNode:
		return "*template.UnlessNode"
	case *SwitchNode:
		return "*template.SwitchNode"
	case *ForeachNode:
		return "*template.ForeachNode"
	case *ExtendsNode:
		return "*template.ExtendsNode"
	case *SectionNode:
		return "*template.SectionNode"
	case *YieldNode:
		return "*template.YieldNode"
	case *IncludeNode:
		return "*template.IncludeNode"
	case *ComponentNode:
		return "*template.ComponentNode"
	case *PropsNode:
		return "*template.PropsNode"
	default:
		return "unknown"
	}
}
