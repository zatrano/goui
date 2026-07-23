package core

import (
	"path/filepath"
	"testing"

	"github.com/zatrano/goui/i18n"
)

func TestBaseComponent_T(t *testing.T) {
	tr := i18n.NewTranslator()
	if err := tr.LoadLocale("tr", filepath.Join("..", "i18n", "locales", "tr.json")); err != nil {
		t.Fatalf("LoadLocale: %v", err)
	}
	if err := tr.LoadLocale("en", filepath.Join("..", "i18n", "locales", "en.json")); err != nil {
		t.Fatalf("LoadLocale en: %v", err)
	}

	bc := &BaseComponent{Locale: "en"}
	bc.SetTranslator(tr)

	got := bc.T("form.submit")
	if got != "Submit" {
		t.Fatalf("T(form.submit) = %q, want %q", got, "Submit")
	}

	got = bc.T("welcome_message", map[string]any{"Name": "Serhan"})
	if got != "Welcome, Serhan" {
		t.Fatalf("T(welcome_message) = %q, want %q", got, "Welcome, Serhan")
	}

	// Default locale falls back to base when Locale is empty.
	bc.Locale = ""
	got = bc.T("nav.home")
	if got != "Ana Sayfa" {
		t.Fatalf("default locale T(nav.home) = %q, want %q", got, "Ana Sayfa")
	}
}

func TestRenderTemplate_WithTInData(t *testing.T) {
	tr := i18n.NewTranslator()
	if err := tr.LoadLocale("tr", filepath.Join("..", "i18n", "locales", "tr.json")); err != nil {
		t.Fatalf("LoadLocale: %v", err)
	}

	bc := &BaseComponent{Locale: "tr"}
	bc.SetTranslator(tr)

	data := map[string]any{
		"Name": "GoUI",
		"T":    bc.T,
	}

	html, err := RenderTemplate(`<p>{{call .T "welcome_message" .}}</p>`, data)
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}

	want := "<p>Hoş geldin, GoUI</p>"
	if html != want {
		t.Fatalf("got %q, want %q", html, want)
	}

	html, err = RenderTemplate(`<button>{{call .T "form.submit"}}</button>`, map[string]any{"T": bc.T})
	if err != nil {
		t.Fatalf("RenderTemplate without placeholders: %v", err)
	}

	want = "<button>Gönder</button>"
	if html != want {
		t.Fatalf("got %q, want %q", html, want)
	}
}
