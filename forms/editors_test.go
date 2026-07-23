package forms

import (
	"context"
	"strings"
	"testing"
)

func TestRenderMarkdown_Basic(t *testing.T) {
	html := RenderMarkdown("# Merhaba\n\n**kalın**")
	if !strings.Contains(html, "<h1") || !strings.Contains(html, "<strong>") {
		t.Fatalf("html=%s", html)
	}
}

func TestMarkdownEditor_SyncPreview(t *testing.T) {
	m := &MarkdownEditor{
		CommonAttrs: CommonAttrs{Name: "md"},
		EventName:   "md",
	}
	_ = m.HandleEvent(context.Background(), "md.sync", map[string]any{"value": "## Başlık"})
	if !strings.Contains(m.PreviewHTML, "<h2") {
		t.Fatalf("preview=%s", m.PreviewHTML)
	}
	out, _ := m.Render()
	if !strings.Contains(out, "goui-markdown-preview") {
		t.Fatalf("render=%s", out)
	}
}

func TestRichTextAndCode_Sync(t *testing.T) {
	r := &RichTextEditor{CommonAttrs: CommonAttrs{Name: "rt"}, EventName: "rt"}
	_ = r.HandleEvent(context.Background(), "rt.sync", map[string]any{"value": "<p>Hi</p>"})
	if r.Value != "<p>Hi</p>" {
		t.Fatal(r.Value)
	}
	html, _ := r.Render()
	if !strings.Contains(html, "data-goui-quill-mount") || !strings.Contains(html, "data-goui-ignore") {
		t.Fatal(html)
	}
	if strings.Contains(html, "<p>Hi</p>") {
		t.Fatal("live value must not appear in editor body HTML")
	}

	c := &CodeEditor{CommonAttrs: CommonAttrs{Name: "code"}, EventName: "code", Language: "go"}
	_ = c.HandleEvent(context.Background(), "code.sync", map[string]any{"value": "package main"})
	if c.Value != "package main" {
		t.Fatal(c.Value)
	}
	html, _ = c.Render()
	if !strings.Contains(html, "data-goui-cm-mount") || !strings.Contains(html, `data-mode="go"`) {
		t.Fatal(html)
	}
}
