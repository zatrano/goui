package forms

import (
	"context"
	"strings"
	"testing"
)

func TestPasswordStrength_Levels(t *testing.T) {
	if PasswordStrength("") != StrengthEmpty {
		t.Fatal("empty")
	}
	if PasswordStrength("abc") != StrengthWeak {
		t.Fatalf("short weak: %v", PasswordStrength("abc"))
	}
	if PasswordStrength("Abcdef12!") < StrengthGood {
		t.Fatalf("expected good+, got %v", PasswordStrength("Abcdef12!"))
	}
}

func TestTextInput_CharCountAndStrength(t *testing.T) {
	in := &TextInput{
		CommonAttrs:   CommonAttrs{Name: "bio"},
		Value:         "merhaba",
		MaxLength:     20,
		ShowCharCount: true,
		HelperText:    "Kısa bio",
		EventName:     "bio",
	}
	html, err := in.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "7 / 20") || !strings.Contains(html, "Kısa bio") {
		t.Fatalf("meta missing: %s", html)
	}

	pw := &TextInput{
		CommonAttrs:  CommonAttrs{Name: "pw"},
		Type:         "password",
		Value:        "Abcdef12!",
		ShowStrength: true,
		EventName:    "pw",
	}
	_ = pw.HandleEvent(context.Background(), "pw", map[string]any{"value": "Abcdef12!"})
	html, _ = pw.Render()
	if !strings.Contains(html, "goui-password-strength") || !strings.Contains(html, "is-") {
		t.Fatalf("strength missing: %s", html)
	}
}

func TestTextarea_CharCount(t *testing.T) {
	ta := &Textarea{
		CommonAttrs:   CommonAttrs{Name: "msg"},
		Value:         "hi",
		MaxLength:     10,
		ShowCharCount: true,
	}
	html, _ := ta.Render()
	if !strings.Contains(html, "2 / 10") {
		t.Fatalf("count: %s", html)
	}
}
