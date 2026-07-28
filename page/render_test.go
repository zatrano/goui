package page_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/goui/v2/core"
	"github.com/zatrano/goui/v2/i18n"
	"github.com/zatrano/goui/v2/page"
	"github.com/zatrano/goui/v2/ws"
)

type article struct {
	core.BaseComponent
	Slug string
}

func (a *article) Mount(ctx context.Context) error {
	if req := core.RequestFromContext(ctx); req != nil {
		a.Slug = req.URL.Query().Get("slug")
	}
	if a.Slug == "" {
		a.Slug = "welcome"
	}
	return nil
}

func (a *article) Head() core.Head {
	return core.Head{
		Title:       "Article: " + a.Slug,
		Description: "SEO article about " + a.Slug,
		OGTitle:     "Article: " + a.Slug,
	}
}

func (a *article) Render() (string, error) {
	return core.RenderTemplate(`<article class="post">
<h1>{{.Slug}}</h1>
<p>Hello from SSR.</p>
<button type="button" g-click="ping">Ping</button>
</article>`, a)
}

func (a *article) HandleEvent(_ context.Context, event string, _ map[string]any) error {
	if event == "ping" {
		a.MarkDirty()
	}
	return nil
}

func (a *article) Unmount(_ context.Context) error { return nil }

type privacy struct {
	core.BaseComponent
}

func (p *privacy) Mount(_ context.Context) error { return nil }
func (p *privacy) Head() core.Head {
	return core.Head{Title: "Privacy", Description: "Privacy policy", Robots: "noindex"}
}
func (p *privacy) Render() (string, error) {
	return `<main><h1>Privacy</h1><p>Static only.</p></main>`, nil
}
func (p *privacy) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}
func (p *privacy) Unmount(_ context.Context) error { return nil }

func TestRender_ModeSEO(t *testing.T) {
	reg := core.NewRegistry()
	if err := reg.RegisterPage("article", func() core.Component { return &article{} }, core.ModeSEO); err != nil {
		t.Fatal(err)
	}

	r := page.NewRenderer(page.Options{Registry: reg, Translator: i18n.NewTranslator()})
	req := httptest.NewRequest(http.MethodGet, "/article?slug=go-ui&locale=tr", nil)
	res, err := r.Render(req.Context(), page.Request{
		Component:   "article",
		Locale:      "tr",
		HTTPRequest: req,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != core.ModeSEO {
		t.Fatalf("mode = %v", res.Mode)
	}
	if !strings.Contains(res.HTML, "<title>Article: go-ui</title>") {
		t.Fatalf("missing title in:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, `data-goui-ssr="1"`) {
		t.Fatalf("missing data-goui-ssr in:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "Hello from SSR") {
		t.Fatalf("missing body in:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "GoUIClient") {
		t.Fatalf("SEO mode must embed WS client")
	}
	if !strings.Contains(res.HTML, `name="description" content="SEO article about go-ui"`) {
		t.Fatalf("missing description meta")
	}
}

func TestRender_ModeStatic(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.RegisterPage("privacy", func() core.Component { return &privacy{} }, core.ModeStatic)

	r := page.NewRenderer(page.Options{Registry: reg})
	res, err := r.Render(context.Background(), page.Request{Component: "privacy"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != core.ModeStatic {
		t.Fatalf("mode = %v", res.Mode)
	}
	if strings.Contains(res.HTML, "GoUIClient") || strings.Contains(res.HTML, "WebSocket") {
		t.Fatalf("static mode must not embed WS client:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "<h1>Privacy</h1>") {
		t.Fatalf("missing body")
	}
	if !strings.Contains(res.HTML, `name="robots" content="noindex"`) {
		t.Fatalf("missing robots")
	}
}

func TestRender_ModeLive(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("admin", func() core.Component { return &privacy{} })

	r := page.NewRenderer(page.Options{Registry: reg})
	res, err := r.Render(context.Background(), page.Request{Component: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != core.ModeLive {
		t.Fatalf("mode = %v", res.Mode)
	}
	// Default ModeLive = sync first paint (Vue SSR / LiveView dead render).
	if !strings.Contains(res.HTML, "<h1>Privacy</h1>") {
		t.Fatalf("live mode should SSR body by default:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, `data-goui-ssr="1"`) {
		t.Fatalf("live SSR should mark hydrate:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "GoUIClient") {
		t.Fatalf("live mode must embed WS client")
	}
}

func TestRender_ModeLive_DeferFirstRender(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("admin", func() core.Component { return &privacy{} })

	r := page.NewRenderer(page.Options{Registry: reg})
	res, err := r.Render(context.Background(), page.Request{
		Component:        "admin",
		DeferFirstRender: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.HTML, "Privacy") && strings.Contains(res.HTML, "<h1>") {
		t.Fatalf("DeferFirstRender must not SSR body:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, `<div id="app"></div>`) {
		t.Fatalf("DeferFirstRender should have empty #app:\n%s", res.HTML)
	}
}

type timedLive struct {
	core.BaseComponent
	delay      time.Duration
	mountBegan chan struct{}
	mountDone  chan struct{}
	sawCancel  atomic.Bool
}

func (c *timedLive) Mount(ctx context.Context) error {
	if c.mountBegan != nil {
		close(c.mountBegan)
	}
	if c.delay > 0 {
		timer := time.NewTimer(c.delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			c.sawCancel.Store(true)
			return ctx.Err()
		}
	}
	if c.mountDone != nil {
		close(c.mountDone)
	}
	return nil
}

func (c *timedLive) Render() (string, error) {
	return `<section class="dash"><h1>Dashboard</h1><p>Ready</p></section>`, nil
}
func (c *timedLive) HandleEvent(context.Context, string, map[string]any) error { return nil }
func (c *timedLive) Unmount(context.Context) error                             { return nil }

func TestRender_ModeLive_FastPathUnderTimeout(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("dash", func() core.Component {
		return &timedLive{delay: 5 * time.Millisecond}
	})

	pending := ws.NewPendingMounts(time.Second)
	defer pending.Stop()

	r := page.NewRenderer(page.Options{Registry: reg, PendingMounts: pending})
	res, err := r.Render(context.Background(), page.Request{
		Component:         "dash",
		FastRenderTimeout: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.HTML, "Dashboard") {
		t.Fatalf("fast-path should SSR body:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, `data-goui-ssr="1"`) {
		t.Fatalf("fast-path should mark SSR for hydrate:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "GoUIClient") {
		t.Fatal("fast-path ModeLive must still embed WS client")
	}
	if res.PendingID == "" {
		t.Fatal("fast-path success must park instance for WS adopt (no remount)")
	}
	c, _, _, err := pending.Take(res.PendingID)
	if err != nil {
		t.Fatalf("Take: %v", err)
	}
	if c == nil {
		t.Fatal("expected parked component")
	}
}

func TestRender_ModeLive_FastPathRequiresPending(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("dash", func() core.Component { return &timedLive{} })
	r := page.NewRenderer(page.Options{Registry: reg})
	_, err := r.Render(context.Background(), page.Request{
		Component:         "dash",
		FastRenderTimeout: 50 * time.Millisecond,
	})
	if !errors.Is(err, page.ErrPendingRequired) {
		t.Fatalf("err = %v, want ErrPendingRequired", err)
	}
}

func TestRender_ModeLive_FastPathTimeoutFallsBackToShell(t *testing.T) {
	reg := core.NewRegistry()
	began := make(chan struct{})
	done := make(chan struct{})
	var probe *timedLive
	_ = reg.Register("slow", func() core.Component {
		probe = &timedLive{
			delay:      150 * time.Millisecond,
			mountBegan: began,
			mountDone:  done,
		}
		return probe
	})

	pending := ws.NewPendingMounts(time.Second)
	defer pending.Stop()

	r := page.NewRenderer(page.Options{Registry: reg, PendingMounts: pending})
	start := time.Now()
	res, err := r.Render(context.Background(), page.Request{
		Component:         "slow",
		FastRenderTimeout: 30 * time.Millisecond,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("HTTP render blocked too long on timeout path: %v", elapsed)
	}
	if !strings.Contains(res.HTML, `<div id="app"></div>`) {
		t.Fatalf("timeout should return empty shell:\n%s", res.HTML)
	}
	if strings.Contains(res.HTML, "Dashboard") {
		t.Fatalf("timeout must not SSR body:\n%s", res.HTML)
	}
	if res.PendingID == "" {
		t.Fatal("timeout should park Mount and return PendingID")
	}
	if !strings.Contains(res.HTML, "pending: '"+res.PendingID+"'") {
		t.Fatalf("HTML should embed pending id for WS adopt:\n%s", res.HTML)
	}

	select {
	case <-began:
	case <-time.After(time.Second):
		t.Fatal("Mount never started")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Mount should finish in background after timeout (not cancelled)")
	}
	if probe.sawCancel.Load() {
		t.Fatal("Mount context must not be cancelled on FastRenderTimeout")
	}

	// Adopt must not Mount again — Take returns the same parked instance.
	c, _, _, err := pending.Take(res.PendingID)
	if err != nil {
		t.Fatalf("Take parked mount: %v", err)
	}
	if c != probe {
		t.Fatal("adopted component should be the parked instance")
	}
}

func TestRender_ModeLive_FastPathUnsetUnchanged(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("admin", func() core.Component {
		return &timedLive{delay: 0}
	})

	r := page.NewRenderer(page.Options{Registry: reg})
	res, err := r.Render(context.Background(), page.Request{Component: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	// Without DeferFirstRender, Live SSR's by default.
	if !strings.Contains(res.HTML, "Dashboard") {
		t.Fatalf("default Live should SSR:\n%s", res.HTML)
	}
}

func TestHandlerRoute_FastPath(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("dash", func() core.Component {
		return &timedLive{delay: 0}
	})
	pending := ws.NewPendingMounts(time.Second)
	defer pending.Stop()
	r := page.NewRenderer(page.Options{Registry: reg, PendingMounts: pending})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.HandlerRoute(page.Route{
		Component:         "dash",
		FastRenderTimeout: 50 * time.Millisecond,
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Dashboard") || !strings.Contains(body, `data-goui-ssr="1"`) {
		t.Fatalf("HandlerRoute fast-path missing SSR body:\n%s", body)
	}
	if !strings.Contains(body, "pending:") {
		t.Fatalf("HandlerRoute fast-path should embed pending for adopt:\n%s", body)
	}
}

const dashSkeleton = `<div class="skeleton" style="min-height:240px"><div class="skel-row"></div></div>`

func TestRender_ModeLive_SkeletonHTML(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("admin", func() core.Component { return &timedLive{} })

	r := page.NewRenderer(page.Options{Registry: reg})
	res, err := r.Render(context.Background(), page.Request{
		Component:        "admin",
		DeferFirstRender: true,
		SkeletonHTML:     dashSkeleton,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.HTML, `class="skeleton"`) {
		t.Fatalf("expected skeleton in #app:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, `<div id="app">`+dashSkeleton+`</div>`) {
		t.Fatalf("skeleton should fill #app:\n%s", res.HTML)
	}
	if strings.Contains(res.HTML, "Dashboard") {
		t.Fatalf("skeleton path must not SSR real body:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "GoUIClient") {
		t.Fatal("skeleton ModeLive must still embed WS client")
	}
}

func TestRender_ModeLive_SkeletonUnsetKeepsEmptyShell(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("admin", func() core.Component { return &timedLive{} })

	r := page.NewRenderer(page.Options{Registry: reg})
	res, err := r.Render(context.Background(), page.Request{
		Component:        "admin",
		DeferFirstRender: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.HTML, `<div id="app"></div>`) {
		t.Fatalf("DeferFirstRender without skeleton must keep empty #app:\n%s", res.HTML)
	}
}

func TestRender_ModeLive_SkeletonUsedOnFastPathTimeout(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("slow", func() core.Component {
		return &timedLive{delay: 100 * time.Millisecond}
	})
	pending := ws.NewPendingMounts(time.Second)
	defer pending.Stop()

	r := page.NewRenderer(page.Options{Registry: reg, PendingMounts: pending})
	res, err := r.Render(context.Background(), page.Request{
		Component:         "slow",
		FastRenderTimeout: 5 * time.Millisecond,
		SkeletonHTML:      dashSkeleton,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.HTML, `class="skeleton"`) {
		t.Fatalf("timeout fallback should show skeleton:\n%s", res.HTML)
	}
	if strings.Contains(res.HTML, "Dashboard") {
		t.Fatalf("timeout must not SSR real body:\n%s", res.HTML)
	}
	if res.PendingID == "" {
		t.Fatal("timeout should still park Mount")
	}
}

func TestRender_ModeLive_FastPathIgnoresSkeleton(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("dash", func() core.Component {
		return &timedLive{delay: 0}
	})

	pending := ws.NewPendingMounts(time.Second)
	defer pending.Stop()
	r := page.NewRenderer(page.Options{Registry: reg, PendingMounts: pending})
	res, err := r.Render(context.Background(), page.Request{
		Component:         "dash",
		FastRenderTimeout: 50 * time.Millisecond,
		SkeletonHTML:      dashSkeleton,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.HTML, "Dashboard") {
		t.Fatalf("fast-path should SSR real body:\n%s", res.HTML)
	}
	if strings.Contains(res.HTML, `class="skeleton"`) {
		t.Fatalf("fast-path must not include skeleton:\n%s", res.HTML)
	}
}

func TestHandlerRoute_SkeletonHTML(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.Register("admin", func() core.Component { return &timedLive{} })
	r := page.NewRenderer(page.Options{Registry: reg})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.HandlerRoute(page.Route{
		Component:        "admin",
		DeferFirstRender: true,
		SkeletonHTML:     dashSkeleton,
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `class="skeleton"`) {
		t.Fatalf("HandlerRoute missing skeleton:\n%s", rr.Body.String())
	}
}

func TestRender_ModeSEO_ParksForAdopt(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.RegisterPage("article", func() core.Component { return &article{} }, core.ModeSEO)
	pending := ws.NewPendingMounts(time.Second)
	defer pending.Stop()

	r := page.NewRenderer(page.Options{Registry: reg, PendingMounts: pending})
	req := httptest.NewRequest(http.MethodGet, "/article?slug=x", nil)
	res, err := r.Render(req.Context(), page.Request{
		Component:   "article",
		Locale:      "tr",
		HTTPRequest: req,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.PendingID == "" {
		t.Fatal("ModeSEO with PendingMounts should park for adopt")
	}
	if !strings.Contains(res.HTML, "pending: '"+res.PendingID+"'") {
		t.Fatalf("missing pending in HTML:\n%s", res.HTML)
	}
	if pending.Stats().Parked < 1 {
		t.Fatal("expected Parked stat")
	}
}

func TestHandler_SEO(t *testing.T) {
	reg := core.NewRegistry()
	_ = reg.RegisterPage("article", func() core.Component { return &article{} }, core.ModeSEO)
	r := page.NewRenderer(page.Options{Registry: reg})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?slug=x", nil)
	r.Handler("article").ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "Hello from SSR") {
		t.Fatalf("body missing SSR content")
	}
}
