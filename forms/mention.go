package forms

import (
	"context"
	"html"
	"strings"

	"github.com/zatrano/goui/core"
)

// MentionTextarea is a textarea that opens an @mention suggestion list (server-filtered).
type MentionTextarea struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Value       string
	Placeholder string
	Rows        int
	Users       []MentionUser // full directory
	Filtered    []MentionUser
	Query       string // text after @
	Open        bool
	EventName   string
	OnChange    func(value string)
}

// MentionUser is one @mention candidate.
type MentionUser struct {
	ID    string
	Label string
}

func (m *MentionTextarea) Name() string         { return m.CommonAttrs.Name }
func (m *MentionTextarea) RawValue() string     { return m.Value }
func (m *MentionTextarea) SetRawValue(v string) { m.Value = v }

func (m *MentionTextarea) Mount(_ context.Context) error   { return nil }
func (m *MentionTextarea) Unmount(_ context.Context) error { return nil }

func (m *MentionTextarea) Validate() bool {
	return m.FieldValidation.Run(m.Value, m.T)
}

func (m *MentionTextarea) eventName() string {
	if m.EventName != "" {
		return m.EventName
	}
	return m.CommonAttrs.Name
}

func (m *MentionTextarea) ev(action string) string {
	base := m.eventName()
	if base == "" {
		return action
	}
	return base + "." + action
}

func (m *MentionTextarea) rows() int {
	if m.Rows <= 0 {
		return 4
	}
	return m.Rows
}

func (m *MentionTextarea) filterUsers(q string) {
	q = strings.ToLower(strings.TrimSpace(q))
	out := make([]MentionUser, 0, len(m.Users))
	for _, u := range m.Users {
		if q == "" || strings.Contains(strings.ToLower(u.ID), q) || strings.Contains(strings.ToLower(u.Label), q) {
			out = append(out, u)
		}
		if len(out) >= 8 {
			break
		}
	}
	m.Filtered = out
}

func mentionQuery(value string) (q string, ok bool) {
	i := strings.LastIndex(value, "@")
	if i < 0 {
		return "", false
	}
	frag := value[i+1:]
	if strings.ContainsAny(frag, " \n\t") {
		return "", false
	}
	return frag, true
}

func (m *MentionTextarea) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, m.eventName())
	switch action {
	case "sync", "input", "change":
		m.Value = payloadString(payload, "value")
		if q, ok := mentionQuery(m.Value); ok {
			m.Query = q
			m.Open = true
			m.filterUsers(q)
		} else {
			m.Open = false
			m.Query = ""
			m.Filtered = nil
		}
		m.MarkDirty()
		if m.OnChange != nil {
			m.OnChange(m.Value)
		}
	case "pick":
		id := payloadString(payload, "value")
		label := id
		for _, u := range m.Users {
			if u.ID == id {
				label = u.ID
				break
			}
		}
		if q, ok := mentionQuery(m.Value); ok {
			at := strings.LastIndex(m.Value, "@")
			m.Value = m.Value[:at] + "@" + label + " "
			_ = q
		} else {
			m.Value += "@" + label + " "
		}
		m.Open = false
		m.Query = ""
		m.Filtered = nil
		m.MarkDirty()
		if m.OnChange != nil {
			m.OnChange(m.Value)
		}
	case "close":
		m.Open = false
		m.MarkDirty()
	}
	return nil
}

func (m *MentionTextarea) Render() (string, error) {
	attrs := Attrs{}
	attrs = m.CommonAttrs.Apply(attrs)
	attrs = m.FieldValidation.ApplyErrorState(attrs, "goui-mention")

	ta := Attrs{}
	ta = ta.Set("class", classTextarea)
	ta = ta.Set("placeholder", m.Placeholder)
	ta = ta.SetInt("rows", m.rows())
	if ev := m.eventName(); ev != "" {
		ta = ta.Set("g-input", m.ev("sync"))
		ta = ta.Set("g-debounce", "120")
	}

	var b strings.Builder
	b.WriteString(`<div` + attrs.String() + `>`)
	b.WriteString(`<textarea` + ta.String() + `>` + html.EscapeString(m.Value) + `</textarea>`)
	if m.Open && len(m.Filtered) > 0 {
		b.WriteString(`<ul class="goui-mention-list border border-goui-border rounded-goui">`)
		for _, u := range m.Filtered {
			b.WriteString(`<li class="goui-mention-item" g-click="` + html.EscapeString(m.ev("pick")) + `" data-goui-value="` + html.EscapeString(u.ID) + `">`)
			b.WriteString(`<strong>@` + html.EscapeString(u.ID) + `</strong> ` + html.EscapeString(u.Label))
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(m.FieldValidation.ErrorsHTML())
	return b.String(), nil
}
