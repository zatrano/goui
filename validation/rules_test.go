package validation

import "testing"

func TestRequired(t *testing.T) {
	ok, key := Required()("  ")
	if ok || key != "validation.required" {
		t.Fatalf("Required blank: ok=%v key=%q", ok, key)
	}
	ok, _ = Required()("x")
	if !ok {
		t.Fatal("Required should pass for non-empty")
	}
}

func TestMinLength(t *testing.T) {
	ok, key := MinLength(3)("ab")
	if ok || key != "validation.min_length" {
		t.Fatalf("MinLength: ok=%v key=%q", ok, key)
	}
	ok, _ = MinLength(3)("abc")
	if !ok {
		t.Fatal("MinLength should pass")
	}
}

func TestMaxLength(t *testing.T) {
	ok, key := MaxLength(2)("abc")
	if ok || key != "validation.max_length" {
		t.Fatalf("MaxLength: ok=%v key=%q", ok, key)
	}
	ok, _ = MaxLength(2)("ab")
	if !ok {
		t.Fatal("MaxLength should pass")
	}
}

func TestPattern(t *testing.T) {
	ok, key := Pattern(`^\d+$`)("12a")
	if ok || key != "validation.pattern" {
		t.Fatalf("Pattern: ok=%v key=%q", ok, key)
	}
	ok, _ = Pattern(`^\d+$`)("12")
	if !ok {
		t.Fatal("Pattern should pass")
	}
}

func TestEmail(t *testing.T) {
	ok, key := Email()("not-an-email")
	if ok || key != "validation.email" {
		t.Fatalf("Email: ok=%v key=%q", ok, key)
	}
	ok, _ = Email()("a@b.co")
	if !ok {
		t.Fatal("Email should pass")
	}
}

func TestNumericRange(t *testing.T) {
	ok, key := NumericRange(1, 10)("0")
	if ok || key != "validation.numeric_range" {
		t.Fatalf("NumericRange: ok=%v key=%q", ok, key)
	}
	ok, _ = NumericRange(1, 10)("5")
	if !ok {
		t.Fatal("NumericRange should pass")
	}
}

func TestCustom(t *testing.T) {
	rule := Custom(func(v string) bool { return v == "ok" }, "validation.custom")
	ok, key := rule("no")
	if ok || key != "validation.custom" {
		t.Fatalf("Custom: ok=%v key=%q", ok, key)
	}
	ok, _ = rule("ok")
	if !ok {
		t.Fatal("Custom should pass")
	}
}
