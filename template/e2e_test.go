package template

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func e2eRoot(t testing.TB) string {
	t.Helper()
	return filepath.Join("testdata", "e2e")
}

func TestE2E_FullPageRender(t *testing.T) {
	reg, err := NewRegistry(Config{Root: e2eRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	got, err := reg.Render("pages.home", map[string]any{"Heading": "Welcome"})
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := os.ReadFile(filepath.Join("testdata", "e2e", "golden", "home.html"))
	if err != nil {
		t.Fatal(err)
	}
	want := string(wantBytes)
	if got != want {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestE2E_EmptyForeach(t *testing.T) {
	reg, err := NewRegistry(Config{Root: e2eRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	got, err := reg.Render("pages.list", map[string]any{
		"Items": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `class="empty"`) || !strings.Contains(got, "none") {
		t.Fatalf("expected @empty branch, got %q", got)
	}
	if strings.Contains(got, "active:") || strings.Contains(got, "idle:") {
		t.Fatalf("should not render items: %q", got)
	}
}

func TestE2E_SwitchAllBranches(t *testing.T) {
	reg, err := NewRegistry(Config{Root: e2eRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		code string
		want string
	}{
		{"not_found", "404"},
		{"forbidden", "403"},
		{"other", "500"},
	}
	for _, tc := range cases {
		got, err := reg.Render("pages.error", map[string]any{"Code": tc.code})
		if err != nil {
			t.Fatalf("%s: %v", tc.code, err)
		}
		if !strings.Contains(got, tc.want) {
			t.Fatalf("code=%q got %q, want substring %q", tc.code, got, tc.want)
		}
	}
}

func TestE2E_ConcurrentRender(t *testing.T) {
	reg, err := NewRegistry(Config{Root: e2eRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errCh := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := reg.Render("pages.home", map[string]any{"Heading": "Welcome"})
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}

func TestE2E_HotReloadFullCycle(t *testing.T) {
	dir := t.TempDir()
	writeGoui(t, dir, "components/box.goui.html", `<div class="box">v1</div>`)
	writeGoui(t, dir, "pages/p.goui.html", `@component("components.box")
@endcomponent
`)

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

	before, err := reg.Render("pages.p", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(before, "v1") {
		t.Fatalf("before=%q", before)
	}

	writeGoui(t, dir, "components/box.goui.html", `<div class="box">v2</div>`)
	select {
	case <-reloaded:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for reload")
	}

	after, err := reg.Render("pages.p", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(after, "v2") {
		t.Fatalf("after reload expected v2, got %q", after)
	}
	if strings.Contains(after, "v1") {
		t.Fatalf("old content still present: %q", after)
	}
}
