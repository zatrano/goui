package forms

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// NumberFormat formats value for display using locale conventions.
// tr: 1.234,56 — en: 1,234.56
func NumberFormat(value float64, locale string, decimals int) string {
	if decimals < 0 {
		decimals = 2
	}
	neg := value < 0
	if neg {
		value = -value
	}
	// Avoid -0
	if value == 0 {
		neg = false
	}

	pow := math.Pow(10, float64(decimals))
	rounded := math.Round(value*pow) / pow
	intPart := int64(math.Floor(rounded + 1e-9))
	frac := rounded - float64(intPart)
	fracDigits := int64(math.Round(frac * pow))
	if fracDigits >= int64(pow) {
		intPart++
		fracDigits = 0
	}

	group, decimal := separators(locale)
	intStr := groupDigits(strconv.FormatInt(intPart, 10), group)
	out := intStr
	if decimals > 0 {
		fracStr := strconv.FormatInt(fracDigits, 10)
		if len(fracStr) < decimals {
			fracStr = strings.Repeat("0", decimals-len(fracStr)) + fracStr
		}
		out = intStr + decimal + fracStr
	}
	if neg {
		return "-" + out
	}
	return out
}

func separators(locale string) (group, decimal string) {
	switch strings.ToLower(locale) {
	case "en", "en-us", "en_us":
		return ",", "."
	default:
		// tr and unknown → Turkish-style (RenewOS / TR default)
		return ".", ","
	}
}

func groupDigits(digits string, sep string) string {
	if sep == "" || len(digits) <= 3 {
		return digits
	}
	var b strings.Builder
	n := len(digits)
	for i, ch := range digits {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteString(sep)
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// ParseLocalizedNumber parses a user-typed number for the given locale.
// Accepts optional currency/percent symbols and spaces.
func ParseLocalizedNumber(raw, locale string) (float64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	s = stripNumberNoise(s)
	if s == "" || s == "-" || s == "+" {
		return 0, fmt.Errorf("empty")
	}

	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}

	group, decimal := separators(locale)
	// Remove group separators, then map decimal to '.'
	s = strings.ReplaceAll(s, group, "")
	if decimal != "." {
		s = strings.ReplaceAll(s, decimal, ".")
	}
	// If both . and , somehow remain, fail soft by keeping last sep as decimal
	s = cleanupAmbiguous(s)

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if neg {
		v = -v
	}
	return v, nil
}

func stripNumberNoise(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsDigit(r), r == '.', r == ',', r == '-', r == '+':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			continue
		default:
			// skip currency letters/symbols (₺ $ € % etc.)
			continue
		}
	}
	return b.String()
}

func cleanupAmbiguous(s string) string {
	// After locale mapping we should only have '.' as decimal; drop extra dots
	// except the last one if multiple remain.
	if strings.Count(s, ".") <= 1 {
		return s
	}
	parts := strings.Split(s, ".")
	return strings.Join(parts[:len(parts)-1], "") + "." + parts[len(parts)-1]
}

// CurrencySymbol returns a display symbol for common ISO codes.
func CurrencySymbol(code string) string {
	switch strings.ToUpper(code) {
	case "TRY", "TL":
		return "₺"
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	default:
		if code == "" {
			return "₺"
		}
		return strings.ToUpper(code) + " "
	}
}
