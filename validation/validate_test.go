package validation

import (
	"reflect"
	"testing"
)

func TestValidate_MultipleFailures(t *testing.T) {
	keys := Validate("a", Required(), MinLength(3), Email())
	want := []string{"validation.min_length", "validation.email"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
}

func TestValidate_AllPass(t *testing.T) {
	keys := Validate("user@example.com", Required(), Email())
	if len(keys) != 0 {
		t.Fatalf("expected no failures, got %v", keys)
	}
}
