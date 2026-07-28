package page

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/zatrano/goui/v2/core"
	"github.com/zatrano/goui/v2/diff"
	"github.com/zatrano/goui/v2/i18n"
	"github.com/zatrano/goui/v2/ws"
)

// SSRComponentID is the placeholder data-goui-component value in SSR HTML.
// The client remaps it to the live session id on first WebSocket render.
const SSRComponentID = "ssr"

// ErrPendingRequired is returned when FastRenderTimeout is set but PendingMounts
// is not wired (share the same store as ws.Server.Pending).
var ErrPendingRequired = errors.New("page: FastRenderTimeout requires Options.PendingMounts (share ws.Server.Pending)")

// Route binds an HTTP path to a registered component name.
type Route struct {
	Path      string
	Component string

	// FastRenderTimeout, when > 0, races Mount against the duration without
	// cancelling Mount on miss. Success → SSR + park for WS adopt.
	// Miss → SkeletonHTML/empty + park. Zero means wait for Mount (sync SSR).
	// Requires PendingMounts.
	FastRenderTimeout time.Duration

	// SkeletonHTML fills #app when SSR body is missing (timeout miss, or
	// DeferFirstRender). Ignored when SSR succeeds.
	SkeletonHTML string

	// DeferFirstRender (ModeLive only) skips HTTP SSR and returns an empty
	// #app (or SkeletonHTML). Default false: first paint is synchronous —
	// same idea as Vue SSR / LiveView dead render. Use only when Mount must
	// not run on the GET (rare).
	DeferFirstRender bool
}

// Options configures the page renderer.
type Options struct {
	Registry   *core.Registry
	Translator *i18n.Translator

	// ClientScript is the ES module URL for goui.js (default /client/goui.js).
	ClientScript string
	// WSPath is the WebSocket endpoint (default ws.Path).
	WSPath string
	// MountSelector is the CSS selector for GoUIClient mount (default #app).
	MountSelector string
	// DefaultLocale used when the request has no ?locale= (default "tr").
	DefaultLocale string
	// Styles optional stylesheet hrefs injected into <head>.
	Styles []string

	// PendingMounts parks SSR/fast-path Mounts for WebSocket adopt.
	// Required when any route uses FastRenderTimeout. Share ws.Server.Pending.
	PendingMounts *ws.PendingMounts
}

// Renderer turns registered components into full HTML documents.
type Renderer struct {
	opts Options
}

// NewRenderer builds a Renderer. Registry is required.
func NewRenderer(opts Options) *Renderer {
	if opts.ClientScript == "" {
		opts.ClientScript = "/client/goui.js"
	}
	if opts.WSPath == "" {
		opts.WSPath = ws.Path
	}
	if opts.MountSelector == "" {
		opts.MountSelector = "#app"
	}
	if opts.DefaultLocale == "" {
		opts.DefaultLocale = i18n.BaseLocale
	}
	if opts.Translator == nil {
		opts.Translator = i18n.NewTranslator()
	}
	return &Renderer{opts: opts}
}

// UsePendingMounts shares a PendingMounts store with ws.Server (SSR/fast-path adopt).
func (r *Renderer) UsePendingMounts(p *ws.PendingMounts) {
	if r == nil {
		return
	}
	r.opts.PendingMounts = p
}

// Request describes one page render.
type Request struct {
	Component   string
	Locale      string
	HTTPRequest *http.Request

	FastRenderTimeout time.Duration
	SkeletonHTML      string
	DeferFirstRender  bool
}

// Result is a rendered document.
type Result struct {
	HTML string
	Mode core.PageMode
	Head core.Head
	// PendingID is set when a Mount was parked for WS adopt (no remount).
	PendingID string
}

// Render produces a full HTML document for the component's page mode.
func (r *Renderer) Render(ctx context.Context, req Request) (Result, error) {
	if r.opts.Registry == nil {
		return Result{}, fmt.Errorf("page: registry is nil")
	}
	if req.Component == "" {
		return Result{}, fmt.Errorf("page: component name is required")
	}

	mode, ok := r.opts.Registry.Mode(req.Component)
	if !ok {
		return Result{}, core.ErrComponentNotRegistered
	}

	locale := req.Locale
	if locale == "" {
		locale = r.opts.DefaultLocale
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if req.HTTPRequest != nil {
		ctx = core.ContextWithRequest(ctx, req.HTTPRequest)
	}

	if req.FastRenderTimeout > 0 && r.opts.PendingMounts == nil {
		return Result{}, ErrPendingRequired
	}

	head := core.Head{Title: req.Component, Lang: locale}
	var body string
	var pendingID string

	switch mode {
	case core.ModeStatic:
		frag, extra, err := r.renderSyncSSR(ctx, req.Component, locale, false)
		if err != nil {
			return Result{}, err
		}
		body = frag
		if extra != nil {
			head = mergeHead(head, *extra, locale)
		}

	case core.ModeSEO:
		var err error
		body, pendingID, head, err = r.renderInteractive(ctx, req, locale, head, true)
		if err != nil {
			return Result{}, err
		}

	case core.ModeLive:
		if req.DeferFirstRender {
			if req.SkeletonHTML != "" {
				body = req.SkeletonHTML
			}
			break
		}
		var err error
		body, pendingID, head, err = r.renderInteractive(ctx, req, locale, head, true)
		if err != nil {
			return Result{}, err
		}
	}

	doc, err := r.buildDocument(req.Component, locale, mode, head, body, pendingID)
	if err != nil {
		return Result{}, err
	}
	return Result{HTML: doc, Mode: mode, Head: head, PendingID: pendingID}, nil
}

// renderInteractive is shared by ModeLive and ModeSEO: sync SSR by default,
// optional FastRenderTimeout race, park for WS adopt when PendingMounts is set.
func (r *Renderer) renderInteractive(ctx context.Context, req Request, locale string, head core.Head, mark bool) (body string, pendingID string, outHead core.Head, err error) {
	outHead = head
	if req.FastRenderTimeout > 0 {
		frag, extra, parkID, ok, terr := r.tryTimedSSR(ctx, req.Component, locale, req.FastRenderTimeout, mark)
		if terr != nil {
			return "", "", outHead, terr
		}
		if ok {
			body = frag
			pendingID = parkID
			if extra != nil {
				outHead = mergeHead(outHead, *extra, locale)
			}
		} else {
			pendingID = parkID
		}
	} else {
		frag, extra, parkID, perr := r.renderParkedSSR(ctx, req.Component, locale, mark)
		if perr != nil {
			return "", "", outHead, perr
		}
		body = frag
		pendingID = parkID
		if extra != nil {
			outHead = mergeHead(outHead, *extra, locale)
		}
	}
	if body == "" && req.SkeletonHTML != "" {
		body = req.SkeletonHTML
	}
	return body, pendingID, outHead, nil
}

// renderSyncSSR Mount+Render then Unmount (ModeStatic — no WebSocket adopt).
func (r *Renderer) renderSyncSSR(ctx context.Context, component, locale string, mark bool) (string, *core.Head, error) {
	comp, err := r.opts.Registry.Create(component)
	if err != nil {
		return "", nil, err
	}
	ws.PrepareComponent(comp, SSRComponentID, locale, r.opts.Translator)
	if err := comp.Mount(ctx); err != nil {
		return "", nil, err
	}
	defer func() { _ = comp.Unmount(ctx) }()

	htmlFrag, err := renderComponentFragment(comp, mark)
	if err != nil {
		return "", nil, err
	}
	var h *core.Head
	if hp, ok := comp.(core.HeadProvider); ok {
		hh := hp.Head()
		h = &hh
	}
	return htmlFrag, h, nil
}

// renderParkedSSR Mount+Render and parks for WS adopt when PendingMounts is set.
// Without PendingMounts, falls back to Unmount (legacy double-Mount on WS).
func (r *Renderer) renderParkedSSR(ctx context.Context, component, locale string, mark bool) (body string, head *core.Head, pendingID string, err error) {
	comp, err := r.opts.Registry.Create(component)
	if err != nil {
		return "", nil, "", err
	}

	mountCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	ws.PrepareComponent(comp, SSRComponentID, locale, r.opts.Translator)
	if err := comp.Mount(mountCtx); err != nil {
		cancel()
		return "", nil, "", err
	}

	htmlFrag, err := renderComponentFragment(comp, mark)
	if err != nil {
		cancel()
		_ = comp.Unmount(mountCtx)
		return "", nil, "", err
	}
	var h *core.Head
	if hp, ok := comp.(core.HeadProvider); ok {
		hh := hp.Head()
		h = &hh
	}

	if r.opts.PendingMounts != nil {
		parkID := r.opts.PendingMounts.ParkReady(component, locale, comp, cancel)
		if parkID == "" {
			// At capacity — degrade to Unmount; WS will Mount fresh.
			cancel()
			_ = comp.Unmount(mountCtx)
			return htmlFrag, h, "", nil
		}
		return htmlFrag, h, parkID, nil
	}

	cancel()
	_ = comp.Unmount(mountCtx)
	return htmlFrag, h, "", nil
}

// tryTimedSSR races Mount against timeout without cancelling Mount on miss.
// On success parks the instance for WS adopt. On miss parks the in-flight Mount.
func (r *Renderer) tryTimedSSR(ctx context.Context, component, locale string, timeout time.Duration, mark bool) (body string, head *core.Head, pendingID string, ok bool, err error) {
	comp, err := r.opts.Registry.Create(component)
	if err != nil {
		return "", nil, "", false, err
	}

	mountCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	done := make(chan error, 1)
	go func() {
		ws.PrepareComponent(comp, SSRComponentID, locale, r.opts.Translator)
		done <- comp.Mount(mountCtx)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case mountErr := <-done:
		if mountErr != nil {
			cancel()
			return "", nil, "", false, mountErr
		}
		htmlFrag, err := renderComponentFragment(comp, mark)
		if err != nil {
			cancel()
			_ = comp.Unmount(mountCtx)
			return "", nil, "", false, err
		}
		var h *core.Head
		if hp, ok := comp.(core.HeadProvider); ok {
			hh := hp.Head()
			h = &hh
		}
		parkID := r.opts.PendingMounts.ParkReady(component, locale, comp, cancel)
		if parkID == "" {
			cancel()
			_ = comp.Unmount(mountCtx)
			return htmlFrag, h, "", true, nil
		}
		return htmlFrag, h, parkID, true, nil

	case <-timer.C:
		// Do not cancel — Mount keeps running until adopt or Pending TTL.
		parkID := r.opts.PendingMounts.Park(component, locale, comp, done, cancel)
		return "", nil, parkID, false, nil

	case <-ctx.Done():
		go func() {
			cancel()
			if err := <-done; err == nil {
				_ = comp.Unmount(mountCtx)
			}
		}()
		return "", nil, "", false, ctx.Err()
	}
}

func renderComponentFragment(comp core.Component, mark bool) (string, error) {
	htmlFrag, err := comp.Render()
	if err != nil {
		return "", err
	}
	htmlFrag, err = ws.DecorateHTML(htmlFrag, SSRComponentID)
	if err != nil {
		return "", err
	}
	if mark {
		htmlFrag, err = markSSR(htmlFrag)
		if err != nil {
			return "", err
		}
	}
	return htmlFrag, nil
}

// Handler returns a net/http handler for one component page (no FastRenderTimeout).
func (r *Renderer) Handler(component string) http.Handler {
	return r.HandlerRoute(Route{Component: component})
}

// HandlerRoute returns a net/http handler honouring Route.FastRenderTimeout
// and Route.SkeletonHTML.
func (r *Renderer) HandlerRoute(route Route) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		locale := req.URL.Query().Get("locale")
		if locale == "" {
			locale = r.opts.DefaultLocale
		}
		res, err := r.Render(req.Context(), Request{
			Component:         route.Component,
			Locale:            locale,
			HTTPRequest:       req,
			FastRenderTimeout: route.FastRenderTimeout,
			SkeletonHTML:      route.SkeletonHTML,
			DeferFirstRender:  route.DeferFirstRender,
		})
		if err != nil {
			if errors.Is(err, core.ErrComponentNotRegistered) {
				http.NotFound(w, req)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(res.HTML))
	})
}

func mergeHead(base, override core.Head, locale string) core.Head {
	if override.Title != "" {
		base.Title = override.Title
	}
	if override.Description != "" {
		base.Description = override.Description
	}
	if override.Canonical != "" {
		base.Canonical = override.Canonical
	}
	if override.Lang != "" {
		base.Lang = override.Lang
	} else if base.Lang == "" {
		base.Lang = locale
	}
	if override.Robots != "" {
		base.Robots = override.Robots
	}
	if override.OGTitle != "" {
		base.OGTitle = override.OGTitle
	}
	if override.OGDescription != "" {
		base.OGDescription = override.OGDescription
	}
	if override.OGImage != "" {
		base.OGImage = override.OGImage
	}
	if override.OGType != "" {
		base.OGType = override.OGType
	}
	if len(override.Extra) > 0 {
		base.Extra = override.Extra
	}
	return base
}

func markSSR(htmlFrag string) (string, error) {
	tree, err := diff.ParseHTML(htmlFrag)
	if err != nil {
		return "", err
	}
	if len(tree.Children) == 0 {
		return `<div data-goui-component="` + SSRComponentID + `" data-goui-ssr="1"></div>`, nil
	}
	root := tree.Children[0]
	if root.Attrs == nil {
		root.Attrs = make(map[string]string)
	}
	root.Attrs["data-goui-ssr"] = "1"
	return diff.Serialize(tree), nil
}

type docData struct {
	Lang          string
	Title         string
	Description   string
	Canonical     string
	Robots        string
	OGTitle       string
	OGDescription string
	OGImage       string
	OGType        string
	Extra         []core.Meta
	Styles        []string
	Body          template.HTML
	ConnectWS     bool
	ClientScript  string
	WSPath        string
	Component     string
	Locale        string
	Mount         string
	PendingID     string
}

func (r *Renderer) buildDocument(component, locale string, mode core.PageMode, head core.Head, body, pendingID string) (string, error) {
	lang := head.Lang
	if lang == "" {
		lang = locale
	}
	ogType := head.OGType
	if ogType == "" && (head.OGTitle != "" || head.OGDescription != "" || head.OGImage != "") {
		ogType = "website"
	}

	data := docData{
		Lang:          lang,
		Title:         head.Title,
		Description:   head.Description,
		Canonical:     head.Canonical,
		Robots:        head.Robots,
		OGTitle:       head.OGTitle,
		OGDescription: head.OGDescription,
		OGImage:       head.OGImage,
		OGType:        ogType,
		Extra:         head.Extra,
		Styles:        r.opts.Styles,
		Body:          template.HTML(body), //nolint:gosec // component HTML is produced by trusted Render()
		ConnectWS:     mode != core.ModeStatic,
		ClientScript:  r.opts.ClientScript,
		WSPath:        r.opts.WSPath,
		Component:     component,
		Locale:        locale,
		Mount:         r.opts.MountSelector,
		PendingID:     pendingID,
	}

	var buf bytes.Buffer
	if err := documentTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var documentTmpl = template.Must(template.New("goui-page").Funcs(template.FuncMap{
	"attr": html.EscapeString,
	"jsstr": func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `'`, `\'`)
		s = strings.ReplaceAll(s, "\n", `\n`)
		s = strings.ReplaceAll(s, "\r", ``)
		return s
	},
}).Parse(`<!DOCTYPE html>
<html lang="{{attr .Lang}}">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{attr .Title}}</title>
{{- if .Description}}
<meta name="description" content="{{attr .Description}}">
{{- end}}
{{- if .Canonical}}
<link rel="canonical" href="{{attr .Canonical}}">
{{- end}}
{{- if .Robots}}
<meta name="robots" content="{{attr .Robots}}">
{{- end}}
{{- if .OGTitle}}
<meta property="og:title" content="{{attr .OGTitle}}">
{{- end}}
{{- if .OGDescription}}
<meta property="og:description" content="{{attr .OGDescription}}">
{{- end}}
{{- if .OGImage}}
<meta property="og:image" content="{{attr .OGImage}}">
{{- end}}
{{- if .OGType}}
<meta property="og:type" content="{{attr .OGType}}">
{{- end}}
{{- range .Extra}}
<meta{{if .Name}} name="{{attr .Name}}"{{end}}{{if .Property}} property="{{attr .Property}}"{{end}} content="{{attr .Content}}">
{{- end}}
{{- range .Styles}}
<link rel="stylesheet" href="{{attr .}}">
{{- end}}
</head>
<body>
<div id="app">{{.Body}}</div>
{{- if .ConnectWS}}
<script type="module">
import { GoUIClient } from '{{jsstr .ClientScript}}';
const client = new GoUIClient('{{jsstr .WSPath}}', '{{jsstr .Component}}', {
  mount: '{{jsstr .Mount}}',
  locale: '{{jsstr .Locale}}',
{{- if .PendingID}}
  pending: '{{jsstr .PendingID}}',
{{- end}}
});
client.connect();
</script>
{{- end}}
</body>
</html>
`))
