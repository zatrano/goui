package core

import (
	"context"
	"testing"

	"github.com/zatrano/goui/i18n"
)

func TestBaseComponent_DirtyTracking(t *testing.T) {
	bc := &BaseComponent{}

	if bc.IsDirty() {
		t.Fatal("expected new BaseComponent to be clean")
	}

	bc.MarkDirty()
	if !bc.IsDirty() {
		t.Fatal("expected MarkDirty to set dirty flag")
	}

	bc.ResetDirty()
	if bc.IsDirty() {
		t.Fatal("expected ResetDirty to clear dirty flag")
	}
}

type Counter struct {
	BaseComponent
	Count int
}

func (c *Counter) Mount(_ context.Context) error {
	return nil
}

func (c *Counter) Render() (string, error) {
	html, err := RenderTemplate(`<span>{{.Count}}</span>`, c)
	if err != nil {
		return "", err
	}
	c.ResetDirty()
	return html, nil
}

func (c *Counter) HandleEvent(_ context.Context, event string, _ map[string]any) error {
	switch event {
	case "increment":
		c.Count++
		c.MarkDirty()
	case "decrement":
		c.Count--
		c.MarkDirty()
	}
	return nil
}

func (c *Counter) Unmount(_ context.Context) error {
	return nil
}

func TestCounter_Lifecycle(t *testing.T) {
	ctx := context.Background()
	counter := &Counter{}

	if err := counter.Mount(ctx); err != nil {
		t.Fatalf("Mount: %v", err)
	}

	html, err := counter.Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if html != "<span>0</span>" {
		t.Fatalf("initial render = %q, want %q", html, "<span>0</span>")
	}

	if err := counter.HandleEvent(ctx, "increment", nil); err != nil {
		t.Fatalf("HandleEvent increment: %v", err)
	}

	html, err = counter.Render()
	if err != nil {
		t.Fatalf("Render after increment: %v", err)
	}
	if html != "<span>1</span>" {
		t.Fatalf("after increment = %q, want %q", html, "<span>1</span>")
	}

	if err := counter.HandleEvent(ctx, "decrement", nil); err != nil {
		t.Fatalf("HandleEvent decrement: %v", err)
	}

	html, err = counter.Render()
	if err != nil {
		t.Fatalf("Render after decrement: %v", err)
	}
	if html != "<span>0</span>" {
		t.Fatalf("after decrement = %q, want %q", html, "<span>0</span>")
	}
}

func TestBaseComponent_Toast_CallsPusher(t *testing.T) {
	bc := &BaseComponent{}
	var gotKind, gotText string
	bc.SetPusher(func(kind, text string) {
		gotKind, gotText = kind, text
	})
	bc.Toast("success", "Kayıt tamam")
	if gotKind != "success" || gotText != "Kayıt tamam" {
		t.Fatalf("got kind=%q text=%q", gotKind, gotText)
	}
}

func TestBaseComponent_Toast_NoPusher_NoPanic(t *testing.T) {
	bc := &BaseComponent{}
	bc.Toast("error", "should not panic")
}

func TestBaseComponent_ToastT(t *testing.T) {
	bc := &BaseComponent{Locale: "tr"}
	bc.SetTranslator(i18n.NewTranslator())
	var got string
	bc.SetPusher(func(_, text string) { got = text })
	bc.ToastT("info", "missing.key")
	if got != "[[missing.key]]" {
		t.Fatalf("got %q", got)
	}
}
