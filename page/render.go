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

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/diff"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/ws"
)

// SSRComponentID is the placeholder data-goui-component value in SSR HTML.
// The client remaps it to the live session id on first WebSocket render.
const SSRComponentID = "ssr"

// Route binds an HTTP path to a registered component name.
type Route struct {
	Path      string
	Component string
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

// Request describes one page render.
type Request struct {
	Component   string
	Locale      string
	HTTPRequest *http.Request
}

// Result is a rendered document.
type Result struct {
	HTML string
	Mode core.PageMode
	Head core.Head
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

	head := core.Head{Title: req.Component, Lang: locale}
	var body string

	if mode == core.ModeSEO || mode == core.ModeStatic {
		comp, err := r.opts.Registry.Create(req.Component)
		if err != nil {
			return Result{}, err
		}

		ws.PrepareComponent(comp, SSRComponentID, locale, r.opts.Translator)
		if err := comp.Mount(ctx); err != nil {
			return Result{}, err
		}
		defer func() { _ = comp.Unmount(ctx) }()

		htmlFrag, err := comp.Render()
		if err != nil {
			return Result{}, err
		}
		htmlFrag, err = ws.DecorateHTML(htmlFrag, SSRComponentID)
		if err != nil {
			return Result{}, err
		}
		if mode == core.ModeSEO {
			htmlFrag, err = markSSR(htmlFrag)
			if err != nil {
				return Result{}, err
			}
		}
		body = htmlFrag

		if hp, ok := comp.(core.HeadProvider); ok {
			head = mergeHead(head, hp.Head(), locale)
		}
	}

	doc, err := r.buildDocument(req.Component, locale, mode, head, body)
	if err != nil {
		return Result{}, err
	}
	return Result{HTML: doc, Mode: mode, Head: head}, nil
}

// Handler returns a net/http handler for one component page.
func (r *Renderer) Handler(component string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		locale := req.URL.Query().Get("locale")
		if locale == "" {
			locale = r.opts.DefaultLocale
		}
		res, err := r.Render(req.Context(), Request{
			Component:   component,
			Locale:      locale,
			HTTPRequest: req,
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
}

func (r *Renderer) buildDocument(component, locale string, mode core.PageMode, head core.Head, body string) (string, error) {
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
});
client.connect();
</script>
{{- end}}
</body>
</html>
`))
