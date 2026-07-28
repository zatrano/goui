package forms

import (
	"bytes"
	"context"
	"html"
	"strconv"

	"github.com/yuin/goldmark"
	gmhtml "github.com/yuin/goldmark/renderer/html"

	"github.com/zatrano/goui/core"
)

// MarkdownEditor is a textarea + server-rendered live preview (goldmark).
type MarkdownEditor struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Value       string
	PreviewHTML string
	Rows        int
	Placeholder string
	EventName   string
	DebounceMS  int
	OnChange    func(value string)
}

func (m *MarkdownEditor) Name() string         { return m.CommonAttrs.Name }
func (m *MarkdownEditor) RawValue() string     { return m.Value }
func (m *MarkdownEditor) SetRawValue(v string) { m.Value = v; m.refreshPreview() }

func (m *MarkdownEditor) Mount(_ context.Context) error {
	m.refreshPreview()
	return nil
}
func (m *MarkdownEditor) Unmount(_ context.Context) error { return nil }

func (m *MarkdownEditor) Validate() bool {
	return m.FieldValidation.Run(m.Value, m.T)
}

func (m *MarkdownEditor) eventName() string {
	if m.EventName != "" {
		return m.EventName
	}
	return m.CommonAttrs.Name
}

func (m *MarkdownEditor) rows() int {
	if m.Rows <= 0 {
		return 10
	}
	return m.Rows
}

func (m *MarkdownEditor) debounce() int {
	if m.DebounceMS > 0 {
		return m.DebounceMS
	}
	return 250
}

func (m *MarkdownEditor) refreshPreview() {
	m.PreviewHTML = RenderMarkdown(m.Value)
}

// RenderMarkdown converts markdown source to HTML.
func RenderMarkdown(source string) string {
	md := goldmark.New(
		goldmark.WithRendererOptions(
			gmhtml.WithHardWraps(),
			gmhtml.WithXHTML(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		return `<p class="goui-field-error">Markdown render hatası</p>`
	}
	return buf.String()
}

func (m *MarkdownEditor) HandleEvent(_ context.Context, event string, payload map[string]any) error {
	action := dottedAction(event, m.eventName())
	switch action {
	case "sync", "change", "input", m.eventName():
		m.Value = payloadString(payload, "value")
		m.refreshPreview()
		m.MarkDirty()
		if m.OnChange != nil {
			m.OnChange(m.Value)
		}
	}
	return nil
}

func (m *MarkdownEditor) Render() (string, error) {
	if m.PreviewHTML == "" && m.Value != "" {
		m.refreshPreview()
	}
	attrs := Attrs{}
	attrs = m.CommonAttrs.Apply(attrs)
	attrs = m.FieldValidation.ApplyErrorState(attrs, "goui-markdown")

	ta := Attrs{}
	ta = ta.Set("class", classTextarea+" goui-markdown-source")
	ta = ta.Set("placeholder", m.Placeholder)
	ta = ta.SetInt("rows", m.rows())
	if ev := m.eventName(); ev != "" {
		ta = ta.Set("g-input", ev+".sync")
		ta = ta.Set("g-debounce", strconv.Itoa(m.debounce()))
	}

	return `<div` + attrs.String() + `>` +
		`<div class="goui-markdown-pane"><textarea` + ta.String() + `>` + html.EscapeString(m.Value) + `</textarea></div>` +
		`<div class="goui-markdown-preview border border-goui-border rounded-goui">` + m.PreviewHTML + `</div>` +
		`</div>` + m.FieldValidation.ErrorsHTML(), nil
}
