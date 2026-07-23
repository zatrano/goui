package forms

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/validation"
)

func loadTranslator(t *testing.T) *i18n.Translator {
	t.Helper()
	tr := i18n.NewTranslator()
	if err := tr.LoadLocale("tr", filepath.Join("..", "i18n", "locales", "tr.json")); err != nil {
		t.Fatalf("LoadLocale: %v", err)
	}
	return tr
}

func TestTextInput_Validate_SetsErrors(t *testing.T) {
	tr := loadTranslator(t)
	in := &TextInput{
		CommonAttrs:     CommonAttrs{Name: "email"},
		Type:            "email",
		Value:           "bad",
		FieldValidation: FieldValidation{Rules: []validation.Rule{validation.Required(), validation.Email()}},
	}
	in.SetTranslator(tr)

	if in.Validate() {
		t.Fatal("expected validation failure")
	}
	if len(in.Errors) != 1 || in.Errors[0] != "Geçerli bir e-posta adresi girin" {
		t.Fatalf("Errors = %#v", in.Errors)
	}

	html, err := in.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `aria-invalid="true"`) || !strings.Contains(html, "border-goui-error") {
		t.Fatalf("missing error attrs: %s", html)
	}
	if !strings.Contains(html, `class="goui-field-error text-goui-error text-sm"`) {
		t.Fatalf("missing error message block: %s", html)
	}
}

type stubField struct {
	ok    bool
	calls *int
}

func (s *stubField) Validate() bool {
	*s.calls++
	return s.ok
}

func TestValidateAll_StopsOnAnyFailure(t *testing.T) {
	calls := 0
	a := &stubField{ok: true, calls: &calls}
	b := &stubField{ok: false, calls: &calls}
	c := &stubField{ok: true, calls: &calls}

	if ValidateAll(a, b, c) {
		t.Fatal("expected overall failure")
	}
	if calls != 3 {
		t.Fatalf("expected all 3 fields validated, got %d calls", calls)
	}
}
