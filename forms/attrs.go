package forms

import (
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
)

// FieldValue is the shared value contract for form fields.
type FieldValue interface {
	Name() string
	RawValue() string
	SetRawValue(string)
}

// Attrs is a helper for building HTML attribute strings.
type Attrs map[string]string

func (a Attrs) Set(key, value string) Attrs {
	if a == nil {
		a = Attrs{}
	}
	if value != "" {
		a[key] = value
	}
	return a
}

func (a Attrs) SetBool(key string, on bool) Attrs {
	if a == nil {
		a = Attrs{}
	}
	if on {
		a[key] = key
	}
	return a
}

func (a Attrs) SetInt(key string, n int) Attrs {
	if n != 0 {
		return a.Set(key, strconv.Itoa(n))
	}
	return a
}

func (a Attrs) String() string {
	if len(a) == 0 {
		return ""
	}
	var b strings.Builder
	keys := sortedKeys(a)
	for _, k := range keys {
		v := a[k]
		b.WriteByte(' ')
		b.WriteString(k)
		if v == k {
			// boolean attribute
			continue
		}
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(v))
		b.WriteString(`"`)
	}
	return b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func payloadString(payload map[string]any, keys ...string) string {
	if payload == nil {
		return ""
	}
	for _, key := range keys {
		if v, ok := payload[key]; ok && v != nil {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func payloadInt(payload map[string]any, keys ...string) int {
	s := payloadString(payload, keys...)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func payloadBool(payload map[string]any, key string) bool {
	if payload == nil {
		return false
	}
	v, ok := payload[key]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "on" || t == "1"
	default:
		return false
	}
}

// CommonAttrs holds HTML attributes shared across most form controls.
type CommonAttrs struct {
	ID              string
	Class           string
	Title           string
	TabIndex        *int
	Spellcheck      *bool
	Draggable       *bool
	AriaLabel       string
	AriaDescribedBy string
	Autocomplete    string
	Disabled        bool
	ReadOnly        bool
	Required        bool
	Autofocus       bool
	Name            string
}

func (c CommonAttrs) apply(a Attrs) Attrs {
	a = a.Set("id", c.ID)
	a = a.Set("class", c.Class)
	a = a.Set("title", c.Title)
	a = a.Set("name", c.Name)
	a = a.Set("autocomplete", c.Autocomplete)
	a = a.Set("aria-label", c.AriaLabel)
	a = a.Set("aria-describedby", c.AriaDescribedBy)
	a = a.SetBool("disabled", c.Disabled)
	a = a.SetBool("readonly", c.ReadOnly)
	a = a.SetBool("required", c.Required)
	a = a.SetBool("autofocus", c.Autofocus)
	if c.TabIndex != nil {
		a = a.Set("tabindex", strconv.Itoa(*c.TabIndex))
	}
	if c.Spellcheck != nil {
		if *c.Spellcheck {
			a = a.Set("spellcheck", "true")
		} else {
			a = a.Set("spellcheck", "false")
		}
	}
	if c.Draggable != nil {
		if *c.Draggable {
			a = a.Set("draggable", "true")
		} else {
			a = a.Set("draggable", "false")
		}
	}
	return a
}

// Apply exports common HTML attributes onto attrs (for subpackages).
func (c CommonAttrs) Apply(a Attrs) Attrs {
	return c.apply(a)
}

const (
	classInput    = "goui-input border border-goui-border rounded-goui px-goui-field py-goui-field text-goui-text bg-white w-full"
	classLabel    = "goui-label block text-sm font-medium text-goui-text mb-1"
	classButton   = "goui-button inline-flex items-center justify-center rounded-goui px-goui-field py-goui-field bg-goui-primary text-white border border-transparent"
	classSelect   = classInput
	classTextarea = classInput
	classChoice   = "goui-choice accent-goui-primary"
	classFieldset = "goui-fieldset border border-goui-border rounded-goui p-goui-field"
	classForm     = "goui-form flex flex-col gap-goui-field"
)
