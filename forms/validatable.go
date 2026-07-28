package forms

import (
	"html"
	"strings"

	"github.com/zatrano/goui/validation"
)

// Validatable is implemented by form fields that support server-side validation.
type Validatable interface {
	Validate() bool
}

// FieldValidation holds rules and translated error messages for a field.
type FieldValidation struct {
	Rules  []validation.Rule
	Errors []string
}

// ValidateAll runs Validate on every field (no early return) and reports overall success.
func ValidateAll(fields ...Validatable) bool {
	ok := true
	for _, f := range fields {
		if f == nil {
			continue
		}
		if !f.Validate() {
			ok = false
		}
	}
	return ok
}

func (f *FieldValidation) run(value string, translate func(key string, args ...any) string) bool {
	keys := validation.Validate(value, f.Rules...)
	f.Errors = make([]string, len(keys))
	for i, k := range keys {
		if translate != nil {
			f.Errors[i] = translate(k)
		} else {
			f.Errors[i] = "[[" + k + "]]"
		}
	}
	return len(keys) == 0
}

// Run validates value and fills Errors with translated messages (for subpackages).
func (f *FieldValidation) Run(value string, translate func(key string, args ...any) string) bool {
	return f.run(value, translate)
}

func (f *FieldValidation) applyErrorState(attrs Attrs, baseClass string) Attrs {
	if len(f.Errors) == 0 {
		if baseClass != "" && attrs["class"] == "" {
			attrs = attrs.Set("class", baseClass)
		}
		return attrs
	}
	attrs = attrs.Set("aria-invalid", "true")
	class := attrs["class"]
	if class == "" {
		class = baseClass
	}
	if !strings.Contains(class, "border-goui-error") {
		class = strings.TrimSpace(class + " border-goui-error")
	}
	attrs["class"] = class
	return attrs
}

// ApplyErrorState marks invalid visual state on attrs (for subpackages).
func (f *FieldValidation) ApplyErrorState(attrs Attrs, baseClass string) Attrs {
	return f.applyErrorState(attrs, baseClass)
}

func (f *FieldValidation) errorsHTML() string {
	if len(f.Errors) == 0 {
		return ""
	}
	var b strings.Builder
	for _, msg := range f.Errors {
		b.WriteString(`<p class="goui-field-error text-goui-error text-sm">`)
		b.WriteString(html.EscapeString(msg))
		b.WriteString(`</p>`)
	}
	return b.String()
}

// ErrorsHTML renders translated field errors (for subpackages).
func (f *FieldValidation) ErrorsHTML() string {
	return f.errorsHTML()
}
