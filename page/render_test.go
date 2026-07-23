package page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/page"
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
	if strings.Contains(res.HTML, "Privacy") && strings.Contains(res.HTML, "<h1>") {
		t.Fatalf("live mode must not SSR body:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "GoUIClient") {
		t.Fatalf("live mode must embed WS client")
	}
	if !strings.Contains(res.HTML, `<div id="app"></div>`) {
		t.Fatalf("live mode should have empty #app")
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
