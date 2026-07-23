package forms

import (
	"context"
	"strings"
	"testing"
)

func TestSwatchAndGradient(t *testing.T) {
	c := &SwatchColorPicker{CommonAttrs: CommonAttrs{Name: "c"}, EventName: "c", Value: "#111827"}
	_ = c.HandleEvent(context.Background(), "c.pick", map[string]any{"value": "#dc2626"})
	if c.Value != "#dc2626" {
		t.Fatal(c.Value)
	}
	g := &GradientPicker{CommonAttrs: CommonAttrs{Name: "g"}, EventName: "g"}
	_ = g.HandleEvent(context.Background(), "g.from", map[string]any{"value": "#000000"})
	_ = g.HandleEvent(context.Background(), "g.to", map[string]any{"value": "#ffffff"})
	if !strings.Contains(g.CSS(), "#000000") || !strings.Contains(g.CSS(), "#ffffff") {
		t.Fatal(g.CSS())
	}
}

func TestMentionTextarea(t *testing.T) {
	m := &MentionTextarea{
		CommonAttrs: CommonAttrs{Name: "msg"},
		EventName:   "msg",
		Users:       []MentionUser{{ID: "ayse", Label: "Ayşe"}, {ID: "ali", Label: "Ali"}},
	}
	_ = m.HandleEvent(context.Background(), "msg.sync", map[string]any{"value": "Merhaba @a"})
	if !m.Open || len(m.Filtered) == 0 {
		t.Fatalf("open=%v filtered=%#v", m.Open, m.Filtered)
	}
	_ = m.HandleEvent(context.Background(), "msg.pick", map[string]any{"value": "ayse"})
	if !strings.Contains(m.Value, "@ayse") || m.Open {
		t.Fatalf("value=%q open=%v", m.Value, m.Open)
	}
}

func TestSignaturePad_Uploaded(t *testing.T) {
	s := &SignaturePad{CommonAttrs: CommonAttrs{Name: "sig"}, EventName: "sig"}
	_ = s.HandleEvent(context.Background(), "sig.uploaded", map[string]any{
		"id": "1", "name": "signature.png", "url": "/goui/files/1", "size": "10", "contentType": "image/png",
	})
	html, _ := s.Render()
	if !strings.Contains(html, "data-goui-signature") || !strings.Contains(html, "signature.png") {
		t.Fatal(html)
	}
}
