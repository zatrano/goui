package forms

import (
	"html"
	"strconv"
	"strings"
	"unicode"
)

// PasswordStrengthLevel is a 0–4 score (empty → weak → fair → good → strong).
type PasswordStrengthLevel int

const (
	StrengthEmpty PasswordStrengthLevel = iota
	StrengthWeak
	StrengthFair
	StrengthGood
	StrengthStrong
)

// PasswordStrength scores a password with simple server-side heuristics.
func PasswordStrength(password string) PasswordStrengthLevel {
	if password == "" {
		return StrengthEmpty
	}
	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	score := 0
	if len(password) >= 8 {
		score++
	}
	if len(password) >= 12 {
		score++
	}
	classes := 0
	if hasLower {
		classes++
	}
	if hasUpper {
		classes++
	}
	if hasDigit {
		classes++
	}
	if hasSpecial {
		classes++
	}
	if classes >= 2 {
		score++
	}
	if classes >= 3 {
		score++
	}
	if score > 4 {
		score = 4
	}
	if score == 0 {
		return StrengthWeak
	}
	return PasswordStrengthLevel(score)
}

func (l PasswordStrengthLevel) LabelKey() string {
	switch l {
	case StrengthEmpty:
		return "forms.password_strength.empty"
	case StrengthWeak:
		return "forms.password_strength.weak"
	case StrengthFair:
		return "forms.password_strength.fair"
	case StrengthGood:
		return "forms.password_strength.good"
	default:
		return "forms.password_strength.strong"
	}
}

func (l PasswordStrengthLevel) CSSClass() string {
	switch l {
	case StrengthEmpty:
		return "is-empty"
	case StrengthWeak:
		return "is-weak"
	case StrengthFair:
		return "is-fair"
	case StrengthGood:
		return "is-good"
	default:
		return "is-strong"
	}
}

func strengthLabel(level PasswordStrengthLevel, translate func(key string, args ...any) string) string {
	key := level.LabelKey()
	if translate != nil {
		msg := translate(key)
		if msg != "" && msg != key {
			return msg
		}
	}
	switch level {
	case StrengthEmpty:
		return ""
	case StrengthWeak:
		return "Zayıf"
	case StrengthFair:
		return "Orta"
	case StrengthGood:
		return "İyi"
	default:
		return "Güçlü"
	}
}

func charCountHTML(value string, maxLength int) string {
	n := len([]rune(value))
	label := strconv.Itoa(n)
	if maxLength > 0 {
		label = strconv.Itoa(n) + " / " + strconv.Itoa(maxLength)
	}
	cls := "goui-char-count text-sm"
	if maxLength > 0 && n > maxLength {
		cls += " text-goui-error"
	}
	return `<p class="` + cls + `">` + html.EscapeString(label) + `</p>`
}

func helperTextHTML(text string) string {
	if text == "" {
		return ""
	}
	return `<p class="goui-helper-text text-sm">` + html.EscapeString(text) + `</p>`
}

func passwordStrengthHTML(password string, translate func(key string, args ...any) string) string {
	level := PasswordStrength(password)
	label := strengthLabel(level, translate)
	var b strings.Builder
	b.WriteString(`<div class="goui-password-strength ` + level.CSSClass() + `" role="status">`)
	b.WriteString(`<div class="goui-password-strength-bar"><span style="width:` + strconv.Itoa(int(level)*25) + `%"></span></div>`)
	if label != "" {
		b.WriteString(`<span class="goui-password-strength-label text-sm">` + html.EscapeString(label) + `</span>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func fieldMetaHTML(helper string, showCount bool, value string, maxLen int, showStrength bool, password string, translate func(key string, args ...any) string) string {
	var b strings.Builder
	b.WriteString(helperTextHTML(helper))
	if showCount {
		b.WriteString(charCountHTML(value, maxLen))
	}
	if showStrength {
		b.WriteString(passwordStrengthHTML(password, translate))
	}
	return b.String()
}
