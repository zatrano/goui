package stdlib

import (
	"net/http"

	"github.com/zatrano/goui/page"
	"github.com/zatrano/goui/upload"
	"github.com/zatrano/goui/ws"
)

// Options configures stdlib/Chi registration.
type Options struct {
	Server *ws.Server
	Store  upload.Storage
	// CheckOrigin validates WebSocket Origin headers. nil allows all.
	CheckOrigin func(*http.Request) bool

	// Page renders ModeLive / ModeSEO / ModeStatic documents when Routes is set.
	Page *page.Renderer
	// Routes maps HTTP paths to registered component names (mode comes from Registry).
	Routes []page.Route
}

// Register mounts GoUI WebSocket, optional upload, and optional page routes on mux.
func Register(mux *http.ServeMux, opts Options) {
	if opts.Server != nil {
		mux.Handle(ws.Path, NewWSHandler(opts.Server, opts.CheckOrigin))
	}
	if opts.Store != nil {
		upload.Mount(mux, opts.Store)
	}
	registerPages(mux, opts.Page, opts.Routes)
}

// Router is any mux that implements Handle (Chi, net/http ServeMux, etc.).
type Router interface {
	Handle(pattern string, handler http.Handler)
}

// Mount registers WS/upload/pages on a generic router (e.g. chi.Mux).
func Mount(r Router, opts Options) {
	if opts.Server != nil {
		r.Handle(ws.Path, NewWSHandler(opts.Server, opts.CheckOrigin))
	}
	if opts.Store != nil {
		h := upload.NewHandler(opts.Store)
		r.Handle(upload.UploadPath, h)
		r.Handle(upload.FilesPrefix+"/", h)
	}
	registerPages(r, opts.Page, opts.Routes)
}

// PageHandler is a convenience wrapper around page.Renderer.Handler.
func PageHandler(r *page.Renderer, component string) http.Handler {
	return r.Handler(component)
}

func registerPages(r Router, renderer *page.Renderer, routes []page.Route) {
	if renderer == nil || len(routes) == 0 {
		return
	}
	for _, route := range routes {
		if route.Path == "" || route.Component == "" {
			continue
		}
		r.Handle(route.Path, renderer.Handler(route.Component))
	}
}
