package core

import (
	"sync"
	"testing"
)

func TestRenderTemplate_Basic(t *testing.T) {
	html, err := RenderTemplate(`<p>Hello, {{.Name}}!</p>`, struct{ Name string }{Name: "GoUI"})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}

	want := "<p>Hello, GoUI!</p>"
	if html != want {
		t.Fatalf("got %q, want %q", html, want)
	}
}

func TestRenderTemplate_Cache(t *testing.T) {
	templateCache = sync.Map{}
	parseCount.Store(0)

	tmpl := `<div>{{.Value}}</div>`

	_, err := RenderTemplate(tmpl, struct{ Value int }{Value: 1})
	if err != nil {
		t.Fatalf("first RenderTemplate: %v", err)
	}

	firstParseCount := parseCount.Load()
	if firstParseCount != 1 {
		t.Fatalf("expected 1 parse, got %d", firstParseCount)
	}

	_, err = RenderTemplate(tmpl, struct{ Value int }{Value: 2})
	if err != nil {
		t.Fatalf("second RenderTemplate: %v", err)
	}

	secondParseCount := parseCount.Load()
	if secondParseCount != 1 {
		t.Fatalf("expected cache hit (still 1 parse), got %d parses", secondParseCount)
	}
}
