package forms

import (
	"strings"
	"unicode"

	"github.com/zatrano/goui/core"
)

// SelectItem is one option in an advanced select list.
type SelectItem struct {
	Value    string
	Label    string
	Disabled bool
}

// FilterMode controls where list filtering runs.
type FilterMode int

const (
	// FilterServer filters in Go on query events (default).
	FilterServer FilterMode = iota
	// FilterClient is allowed only for small fixed lists; server still owns selection state.
	FilterClient
)

const defaultMaxResults = 50

// BaseSelectField is shared state for searchable / multi / combobox-style controls.
type BaseSelectField struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	// Items is the full source list (or last loaded page from a provider).
	Items []SelectItem
	// Filtered is the list rendered after the last server-side filter pass.
	Filtered []SelectItem

	Query       string
	Open        bool
	Value       string   // single selection
	Values      []string // multi selection (used by later controls)
	FilterMode  FilterMode
	MaxResults  int
	Placeholder string

	OnChange func(value string)
	OnQuery  func(query string)
}

// EnsureFiltered initializes Filtered from Items when empty (first render).
func (b *BaseSelectField) EnsureFiltered() {
	if b.Filtered == nil {
		b.ApplyQuery(b.Query)
	}
}

// ApplyQuery updates Query and recomputes Filtered in Go (server-side default).
func (b *BaseSelectField) ApplyQuery(query string) {
	b.Query = query
	limit := b.MaxResults
	if limit <= 0 {
		limit = defaultMaxResults
	}
	b.Filtered = FilterItems(b.Items, query, limit)
	if b.OnQuery != nil {
		b.OnQuery(query)
	}
	b.MarkDirty()
}

// SelectedLabel returns the label for the current single Value.
func (b *BaseSelectField) SelectedLabel() string {
	for _, it := range b.Items {
		if it.Value == b.Value {
			if it.Label != "" {
				return it.Label
			}
			return it.Value
		}
	}
	return ""
}

// FilterItems performs case-insensitive substring match on label/value (server-side).
func FilterItems(items []SelectItem, query string, limit int) []SelectItem {
	q := normalize(query)
	out := make([]SelectItem, 0, len(items))
	for _, it := range items {
		if q != "" {
			label := normalize(it.Label)
			val := normalize(it.Value)
			if !strings.Contains(label, q) && !strings.Contains(val, q) {
				continue
			}
		}
		out = append(out, it)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func eventAction(event, eventName string, payload map[string]any) string {
	if eventName != "" {
		prefix := eventName + "."
		if strings.HasPrefix(event, prefix) {
			return strings.TrimPrefix(event, prefix)
		}
		if event == eventName {
			if a, ok := payload["action"].(string); ok && a != "" {
				return a
			}
		}
	}
	if a, ok := payload["action"].(string); ok && a != "" {
		return a
	}
	return event
}
