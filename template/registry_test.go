package template

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func registryRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "registry")
}

func TestRegistry_RenderHome(t *testing.T) {
	reg, err := NewRegistry(Config{Root: registryRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	got, err := reg.Render("pages.home", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Normalize whitespace for comparison.
	flat := collapseWS(got)
	for _, want := range []string{
		"<title>Home</title>",
		"<nav>Nav</nav>",
		"<h1>Welcome</h1>",
		`<button type="button">Click</button>`,
	} {
		if !strings.Contains(flat, collapseWS(want)) {
			t.Fatalf("missing %q in output:\n%s", want, got)
		}
	}
}

func TestRegistry_IncludeIf(t *testing.T) {
	reg, err := NewRegistry(Config{Root: registryRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	got, err := reg.Render("pages.includeif", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "partials.missing") || strings.Contains(got, "__includeif") {
		t.Fatalf("marker leaked: %s", got)
	}
	if !strings.Contains(got, "<nav>Nav</nav>") {
		t.Fatalf("existing includeIf should render nav, got:\n%s", got)
	}
	if !strings.Contains(got, "<span>done</span>") {
		t.Fatalf("missing trailing content:\n%s", got)
	}
}

func TestRegistry_Exists(t *testing.T) {
	reg, err := NewRegistry(Config{Root: registryRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	if !reg.Exists("pages.home") {
		t.Fatal("pages.home should exist")
	}
	if !reg.Exists("layouts.app") {
		t.Fatal("layouts.app should exist")
	}
	if reg.Exists("pages.nope") {
		t.Fatal("pages.nope should not exist")
	}
}

func TestRegistry_CircularExtends(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/a.goui.html", `@extends("pages.b")
@section("c", "x")
`)
	writeGoui(t, dir, "pages/b.goui.html", `@extends("pages.a")
@section("c", "y")
`)
	_, err := NewRegistry(Config{Root: dir})
	if err == nil {
		t.Fatal("expected circular extends error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "circular @extends chain") {
		t.Fatalf("message = %q", msg)
	}
	if !strings.Contains(msg, "pages.a") || !strings.Contains(msg, "pages.b") {
		t.Fatalf("cycle path missing: %q", msg)
	}
}

func TestRegistry_MissingInclude(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/x.goui.html", `@include("partials.missing")
`)
	_, err := NewRegistry(Config{Root: dir})
	if err == nil {
		t.Fatal("expected missing include error")
	}
	if !strings.Contains(err.Error(), `@include target "partials.missing" not found`) {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestRegistry_MultipleErrors(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/one.goui.html", "@if(.A)\nno end")
	writeGoui(t, dir, "pages/two.goui.html", "@foreach(.Items as .bad)\nx\n@endforeach")
	_, err := NewRegistry(Config{Root: dir})
	if err == nil {
		t.Fatal("expected errors")
	}
	msg := err.Error()
	// Both files should appear in the joined error.
	if !strings.Contains(msg, "unclosed @if") && !strings.Contains(msg, "one.goui.html") {
		t.Fatalf("missing first error: %q", msg)
	}
	if !strings.Contains(msg, "foreach variable must start with $") && !strings.Contains(msg, "two.goui.html") {
		t.Fatalf("missing second error: %q", msg)
	}
	// errors.Join unwrap support
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		if len(joined.Unwrap()) < 2 {
			t.Fatalf("expected >=2 joined errors, got %d", len(joined.Unwrap()))
		}
	}
}

func TestRegistry_RenderNotFound(t *testing.T) {
	reg, err := NewRegistry(Config{Root: registryRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = reg.Render("no.such", nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestRegistry_ExtraFuncs(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/hi.goui.html", `{{ shout "go" }}`)
	reg, err := NewRegistry(Config{
		Root: dir,
		ExtraFuncs: map[string]any{
			"shout": func(s string) string { return strings.ToUpper(s) },
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := reg.Render("pages.hi", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "GO" {
		t.Fatalf("got %q", got)
	}
}

func BenchmarkNewRegistry(b *testing.B) {
	dir := b.TempDir()
	// 100 synthetic templates.
	for i := 0; i < 100; i++ {
		name := filepath.Join("pages", "p"+itoa(i)+".goui.html")
		writeGoui(b, dir, name, "<div>{{ .N }}</div>\n")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewRegistry(Config{Root: dir})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func writeGoui(t testing.TB, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func collapseWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [12]byte
	n := len(buf)
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[n:])
}
