package template

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/goui/core"
)

type plainComp struct {
	core.BaseComponent
	html string
}

func (c *plainComp) Mount(context.Context) error { return nil }
func (c *plainComp) Render() (string, error)     { return c.html, nil }
func (c *plainComp) HandleEvent(context.Context, string, map[string]any) error {
	return nil
}
func (c *plainComp) Unmount(context.Context) error { return nil }

type viewComp struct {
	core.BaseComponent
	Name string
}

func (c *viewComp) Mount(context.Context) error { return nil }
func (c *viewComp) View() string                { return "pages.hello" }
func (c *viewComp) Render() (string, error)     { return "", ErrViewRenderDirect }
func (c *viewComp) HandleEvent(context.Context, string, map[string]any) error {
	return nil
}
func (c *viewComp) Unmount(context.Context) error { return nil }

func TestWrap_PlainComponentUsesOwnRender(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewRegistry(Config{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	factory := Wrap(reg, func() core.Component {
		return &plainComp{html: "<b>plain</b>"}
	})
	c := factory()
	got, err := c.Render()
	if err != nil {
		t.Fatal(err)
	}
	if got != "<b>plain</b>" {
		t.Fatalf("got %q", got)
	}
}

func TestWrap_ViewComponentUsesRegistry(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/hello.goui.html", `<p>Hello {{ .Name }}</p>`)
	reg, err := NewRegistry(Config{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	factory := Wrap(reg, func() core.Component {
		return &viewComp{Name: "Ada"}
	})
	c := factory()
	got, err := c.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Hello Ada") {
		t.Fatalf("got %q", got)
	}
	inner := c.(*componentWrapper).Component.(*viewComp)
	if inner.IsDirty() {
		t.Fatal("expected dirty cleared after successful view render")
	}
	inner.MarkDirty()
	if _, err := c.Render(); err != nil {
		t.Fatal(err)
	}
	if inner.IsDirty() {
		t.Fatal("expected ResetDirty after Wrap Render")
	}
}

func TestRenderComponent_NotView(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewRegistry(Config{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	html, ok, err := reg.RenderComponent(&plainComp{html: "x"})
	if err != nil || ok || html != "" {
		t.Fatalf("html=%q ok=%v err=%v", html, ok, err)
	}
}

func TestRenderComponent_View(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/hello.goui.html", `<span>{{ .Name }}</span>`)
	reg, err := NewRegistry(Config{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	html, ok, err := reg.RenderComponent(&viewComp{Name: "Bo"})
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if html != "<span>Bo</span>" {
		t.Fatalf("got %q", html)
	}
}

func TestWrap_NilPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = Wrap(nil, func() core.Component { return &plainComp{} })
}

func TestExampleCounterViewBuilds(t *testing.T) {
	// Ensure the example module package is present and views exist.
	root := filepath.Join("..", "examples", "counter-view")
	mainGo := filepath.Join(root, "main.go")
	view := filepath.Join(root, "views", "counter.goui.html")
	if _, err := os.Stat(mainGo); err != nil {
		t.Skip("examples/counter-view not in this module tree layout")
	}
	if _, err := os.Stat(view); err != nil {
		t.Fatalf("missing view: %v", err)
	}
}
