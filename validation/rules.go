package validation

import (
	"regexp"
	"strconv"
	"strings"
)

// Rule validates a string value and returns an i18n message key on failure.
type Rule func(value string) (ok bool, messageKey string)

// Required fails when the value is empty or only whitespace.
func Required() Rule {
	return func(value string) (bool, string) {
		if strings.TrimSpace(value) == "" {
			return false, "validation.required"
		}
		return true, ""
	}
}

// MinLength fails when rune length is below n.
func MinLength(n int) Rule {
	return func(value string) (bool, string) {
		if len([]rune(value)) < n {
			return false, "validation.min_length"
		}
		return true, ""
	}
}

// MaxLength fails when rune length is above n.
func MaxLength(n int) Rule {
	return func(value string) (bool, string) {
		if len([]rune(value)) > n {
			return false, "validation.max_length"
		}
		return true, ""
	}
}

// Pattern fails when the value does not match the regex.
func Pattern(regex string) Rule {
	re, err := regexp.Compile(regex)
	if err != nil {
		return func(string) (bool, string) {
			return false, "validation.pattern"
		}
	}
	return func(value string) (bool, string) {
		if !re.MatchString(value) {
			return false, "validation.pattern"
		}
		return true, ""
	}
}

// Email fails when the value is not a simple email shape.
func Email() Rule {
	re := regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	return func(value string) (bool, string) {
		if !re.MatchString(value) {
			return false, "validation.email"
		}
		return true, ""
	}
}

// NumericRange fails when the value is not a number within [min, max].
func NumericRange(min, max float64) Rule {
	return func(value string) (bool, string) {
		n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || n < min || n > max {
			return false, "validation.numeric_range"
		}
		return true, ""
	}
}

// Custom wraps an arbitrary predicate with a message key.
func Custom(fn func(value string) bool, messageKey string) Rule {
	return func(value string) (bool, string) {
		if !fn(value) {
			return false, messageKey
		}
		return true, ""
	}
}
