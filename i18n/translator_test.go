package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func localePath(name string) string {
	return filepath.Join("locales", name)
}

func TestTranslator_LoadAndTranslate(t *testing.T) {
	tr := NewTranslator()

	if err := tr.LoadLocale("tr", localePath("tr.json")); err != nil {
		t.Fatalf("LoadLocale tr: %v", err)
	}
	if err := tr.LoadLocale("en", localePath("en.json")); err != nil {
		t.Fatalf("LoadLocale en: %v", err)
	}

	got := tr.Translate("tr", "form.required_field")
	want := "Bu alan zorunludur"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	got = tr.Translate("en", "nav.home")
	want = "Home"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTranslator_WithArgs(t *testing.T) {
	tr := NewTranslator()
	if err := tr.LoadLocale("tr", localePath("tr.json")); err != nil {
		t.Fatalf("LoadLocale: %v", err)
	}

	got := tr.Translate("tr", "welcome_message", map[string]any{"Name": "Serhan"})
	want := "Hoş geldin, Serhan"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTranslator_MissingKey(t *testing.T) {
	tr := NewTranslator()
	if err := tr.LoadLocale("tr", localePath("tr.json")); err != nil {
		t.Fatalf("LoadLocale: %v", err)
	}

	got := tr.Translate("tr", "does.not.exist")
	want := "[[does.not.exist]]"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTranslator_FallbackToBase(t *testing.T) {
	tr := NewTranslator()
	if err := tr.LoadLocale("tr", localePath("tr.json")); err != nil {
		t.Fatalf("LoadLocale tr: %v", err)
	}

	// Unsupported locale should fall back to base locale.
	got := tr.Translate("de", "form.submit")
	want := "Gönder"
	if got != want {
		t.Fatalf("unsupported locale fallback: got %q, want %q", got, want)
	}

	// Key missing in requested locale but present in base locale.
	tr.mu.Lock()
	tr.messages["en"] = map[string]string{
		"nav.home": "Home",
	}
	tr.mu.Unlock()

	got = tr.Translate("en", "form.cancel")
	want = "İptal"
	if got != want {
		t.Fatalf("key fallback to base: got %q, want %q", got, want)
	}
}

func TestTranslator_LoadLocale_InvalidJSON(t *testing.T) {
	tr := NewTranslator()

	tmp, err := os.CreateTemp("", "goui-locale-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(`{"broken": `); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	tmp.Close()

	if err := tr.LoadLocale("tr", tmp.Name()); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
