package fiber

import (
	"context"
	"net/http"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/zatrano/goui/page"
	"github.com/zatrano/goui/upload"
	"github.com/zatrano/goui/ws"
)

// Options configures Fiber registration.
type Options struct {
	Server *ws.Server
	Store  upload.Storage

	// Page renders ModeLive / ModeSEO / ModeStatic documents when Routes is set.
	Page *page.Renderer
	// Routes maps HTTP paths to registered component names.
	Routes []page.Route
}

// Register mounts GoUI WebSocket, optional upload, and optional page routes.
func Register(app *fiber.App, opts Options) {
	if opts.Server != nil {
		RegisterWS(app, opts.Server)
	}
	if opts.Store != nil {
		RegisterUpload(app, opts.Store)
	}
	RegisterPages(app, opts.Page, opts.Routes)
}

// RegisterWS wires GET /goui/ws.
func RegisterWS(app *fiber.App, server *ws.Server) {
	app.Use(ws.Path, func(c fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get(ws.Path, websocket.New(func(conn *websocket.Conn) {
		_ = server.ServeConn(context.Background(), wrapConn(conn), ws.ConnectParams{
			SessionID:     conn.Query("session"),
			ComponentName: conn.Query("component"),
			Locale:        conn.Query("locale"),
		})
	}))
}

// RegisterUpload wires POST /goui/upload and GET /goui/files/:id via net/http handler.
func RegisterUpload(app *fiber.App, store upload.Storage) {
	h := upload.NewHandler(store)
	app.All(upload.UploadPath, adaptHTTP(h))
	app.All(upload.FilesPrefix+"/*", adaptHTTP(h))
}

// RegisterPages mounts HTML page handlers for each route.
func RegisterPages(app *fiber.App, renderer *page.Renderer, routes []page.Route) {
	if renderer == nil || len(routes) == 0 {
		return
	}
	for _, route := range routes {
		if route.Path == "" || route.Component == "" {
			continue
		}
		comp := route.Component
		app.Get(route.Path, Page(renderer, comp))
	}
}

// Page returns a Fiber handler that renders a registered component document.
func Page(renderer *page.Renderer, component string) fiber.Handler {
	h := renderer.Handler(component)
	return adaptHTTP(h)
}

func adaptHTTP(h http.Handler) fiber.Handler {
	return func(c fiber.Ctx) error {
		fasthttpadaptor.NewFastHTTPHandler(h)(c.RequestCtx())
		return nil
	}
}
