package template

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func BenchmarkRender_SimplePage(b *testing.B) {
	dir := b.TempDir()
	writeGoui(b, dir, "pages/simple.goui.html", `<h1>{{ .Title }}</h1><p>{{ .Body }}</p>`)
	reg, err := NewRegistry(Config{Root: dir})
	if err != nil {
		b.Fatal(err)
	}
	data := map[string]any{"Title": "T", "Body": "B"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := reg.Render("pages.simple", data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRender_ExtendsChain(b *testing.B) {
	dir := b.TempDir()
	writeGoui(b, dir, "layouts/base.goui.html", `<html>@yield("mid")</html>`)
	writeGoui(b, dir, "layouts/mid.goui.html", `@extends("layouts.base")
@section("mid")
<main>@yield("content")</main>
@endsection
`)
	writeGoui(b, dir, "pages/leaf.goui.html", `@extends("layouts.mid")
@section("content")
{{ .X }}
@endsection
`)
	reg, err := NewRegistry(Config{Root: dir})
	if err != nil {
		b.Fatal(err)
	}
	data := map[string]any{"X": "ok"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := reg.Render("pages.leaf", data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRender_ComponentHeavy(b *testing.B) {
	dir := b.TempDir()
	writeGoui(b, dir, "components/chip.goui.html", `<span>{{ .Props.N }}</span>`)
	var page strings.Builder
	for i := 0; i < 20; i++ {
		page.WriteString(`@component("components.chip", dict "N" "`)
		page.WriteString(strconv.Itoa(i))
		page.WriteString(`")
@endcomponent
`)
	}
	writeGoui(b, dir, "pages/heavy.goui.html", page.String())
	reg, err := NewRegistry(Config{Root: dir})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := reg.Render("pages.heavy", nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewRegistry_100Files(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < 100; i++ {
		name := filepath.Join("pages", "p"+strconv.Itoa(i)+".goui.html")
		writeGoui(b, dir, name, "<div>{{ .N }}</div>\n")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewRegistry(Config{Root: dir}); err != nil {
			b.Fatal(err)
		}
	}
}
