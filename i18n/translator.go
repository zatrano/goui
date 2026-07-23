package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"sync"
	"text/template"
)

// BaseLocale is the default fallback locale when a key or locale is missing.
const BaseLocale = "tr"

// Translator stores flat key-value translations per locale.
type Translator struct {
	mu       sync.RWMutex
	messages map[string]map[string]string
}

// NewTranslator creates an empty translation store.
func NewTranslator() *Translator {
	return &Translator{
		messages: make(map[string]map[string]string),
	}
}

// LoadLocale reads a flat JSON key-value file into the given locale.
// Nested JSON objects are not supported.
func (t *Translator) LoadLocale(locale string, filePath string) error {
	data, err := os.ReadFile(filePath) //nolint:gosec // G304: path is chosen by the app, not request input
	if err != nil {
		return err
	}

	var msgs map[string]string
	if err := json.Unmarshal(data, &msgs); err != nil {
		return err
	}

	t.mu.Lock()
	t.messages[locale] = msgs
	t.mu.Unlock()

	return nil
}

// Translate returns the translation for locale and key.
// Missing keys fall back to BaseLocale, then to [[key]] if still not found.
// Optional args provide template data for placeholders such as {{.Name}}.
func (t *Translator) Translate(locale, key string, args ...any) string {
	text := t.lookup(locale, key)
	if text == "" {
		return "[[" + key + "]]"
	}

	if len(args) == 0 {
		return text
	}

	return interpolate(text, args[0])
}

func (t *Translator) lookup(locale, key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if val, ok := t.messageFor(locale, key); ok {
		return val
	}

	if locale != BaseLocale {
		if val, ok := t.messageFor(BaseLocale, key); ok {
			return val
		}
	}

	return ""
}

func (t *Translator) messageFor(locale, key string) (string, bool) {
	msgs, ok := t.messages[locale]
	if !ok {
		return "", false
	}

	val, ok := msgs[key]
	return val, ok
}

func interpolate(text string, data any) string {
	tmpl, err := template.New("").Parse(text)
	if err != nil {
		return text
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return text
	}

	return buf.String()
}
