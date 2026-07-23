package template

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReload_Successful(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/home.goui.html", "<p>v1</p>\n")

	reloaded := make(chan struct{}, 1)
	reg, err := NewRegistry(Config{
		Root:            dir,
		WatchForChanges: true,
		OnReload: func() {
			select {
			case reloaded <- struct{}{}:
			default:
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()

	got, err := reg.Render("pages.home", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "v1") {
		t.Fatalf("got %q", got)
	}

	writeGoui(t, dir, "pages/home.goui.html", "<p>v2</p>\n")
	waitCh(t, reloaded, 3*time.Second, "OnReload")

	got, err = reg.Render("pages.home", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "v2") {
		t.Fatalf("expected v2 after reload, got %q", got)
	}
}

func TestReload_KeepsOldRootOnError(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/home.goui.html", "<p>good</p>\n")

	reloadErr := make(chan error, 1)
	reg, err := NewRegistry(Config{
		Root:            dir,
		WatchForChanges: true,
		OnReloadError: func(e error) {
			select {
			case reloadErr <- e:
			default:
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()

	before, err := reg.Render("pages.home", nil)
	if err != nil {
		t.Fatal(err)
	}

	writeGoui(t, dir, "pages/home.goui.html", "@if(.A)\nbroken\n")
	waitCh(t, reloadErr, 3*time.Second, "OnReloadError")

	after, err := reg.Render("pages.home", nil)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("root should stay on last good compile:\nbefore=%q\nafter=%q", before, after)
	}
	if !strings.Contains(after, "good") {
		t.Fatalf("expected old content, got %q", after)
	}
}

func TestReload_CloseStopsWatching(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/home.goui.html", "<p>a</p>\n")

	reloaded := make(chan struct{}, 4)
	reg, err := NewRegistry(Config{
		Root:            dir,
		WatchForChanges: true,
		OnReload: func() {
			select {
			case reloaded <- struct{}{}:
			default:
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Close(); err != nil {
		t.Fatal(err)
	}
	// Second Close is safe.
	if err := reg.Close(); err != nil {
		t.Fatal(err)
	}

	writeGoui(t, dir, "pages/home.goui.html", "<p>b</p>\n")
	select {
	case <-reloaded:
		t.Fatal("OnReload must not fire after Close")
	case <-time.After(500 * time.Millisecond):
	}
}

func TestReload_WatchDisabledNoWatcher(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "pages/home.goui.html", "<p>x</p>\n")
	reg, err := NewRegistry(Config{Root: dir, WatchForChanges: false})
	if err != nil {
		t.Fatal(err)
	}
	if reg.watcher != nil {
		t.Fatal("watcher must not start when WatchForChanges=false")
	}
	if err := reg.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestTemplateError_Snippet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.goui.html")
	content := "line1\nline2-error-here\nline3\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	te := &TemplateError{File: path, Line: 2, Column: 1, Message: "boom"}
	got := te.Snippet()
	want := "  1 | line1\n> 2 | line2-error-here\n  3 | line3\n"
	if got != want {
		t.Fatalf("snippet:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestTemplateError_SnippetMissingFile(t *testing.T) {
	te := &TemplateError{File: "no-such-file.goui.html", Line: 1, Message: "x"}
	if te.Snippet() != "" {
		t.Fatal("expected empty snippet")
	}
}

func waitCh[T any](t *testing.T, ch <-chan T, timeout time.Duration, name string) T {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case v := <-ch:
		return v
	case <-ctx.Done():
		t.Fatalf("timed out waiting for %s", name)
		var zero T
		return zero
	}
}
