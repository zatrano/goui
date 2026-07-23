package echoadapter

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/zatrano/goui/page"
	"github.com/zatrano/goui/upload"
	"github.com/zatrano/goui/ws"
)

// Options configures Echo registration.
type Options struct {
	Server      *ws.Server
	Store       upload.Storage
	CheckOrigin func(*http.Request) bool

	Page   *page.Renderer
	Routes []page.Route
}

// Register mounts GoUI WebSocket, optional upload, and optional page routes.
func Register(e *echo.Echo, opts Options) {
	if opts.Server != nil {
		RegisterWS(e, opts.Server, opts.CheckOrigin)
	}
	if opts.Store != nil {
		RegisterUpload(e, opts.Store)
	}
	RegisterPages(e, opts.Page, opts.Routes)
}

// RegisterWS wires GET /goui/ws.
func RegisterWS(e *echo.Echo, server *ws.Server, checkOrigin func(*http.Request) bool) {
	upgrader := websocket.Upgrader{CheckOrigin: checkOrigin}
	if upgrader.CheckOrigin == nil {
		upgrader.CheckOrigin = func(*http.Request) bool { return true }
	}

	e.GET(ws.Path, func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()

		q := c.QueryParams()
		_ = server.ServeConn(c.Request().Context(), wrapConn(conn), ws.ConnectParams{
			SessionID:     q.Get("session"),
			ComponentName: q.Get("component"),
			Locale:        q.Get("locale"),
		})
		return nil
	})
}

// RegisterUpload wires upload and file download endpoints.
func RegisterUpload(e *echo.Echo, store upload.Storage) {
	h := upload.NewHandler(store)
	e.Any(upload.UploadPath, echo.WrapHandler(h))
	e.Any(upload.FilesPrefix+"/*", echo.WrapHandler(h))
}

// RegisterPages mounts HTML page handlers for each route.
func RegisterPages(e *echo.Echo, renderer *page.Renderer, routes []page.Route) {
	if renderer == nil || len(routes) == 0 {
		return
	}
	for _, route := range routes {
		if route.Path == "" || route.Component == "" {
			continue
		}
		e.GET(route.Path, echo.WrapHandler(renderer.Handler(route.Component)))
	}
}

// Page returns an Echo handler that renders a registered component document.
func Page(renderer *page.Renderer, component string) echo.HandlerFunc {
	return echo.WrapHandler(renderer.Handler(component))
}

type gorillaConn struct {
	*websocket.Conn
}

func wrapConn(c *websocket.Conn) ws.Conn {
	return gorillaConn{Conn: c}
}

func (c gorillaConn) ReadMessage() (int, []byte, error) {
	return c.Conn.ReadMessage()
}

func (c gorillaConn) WriteMessage(messageType int, data []byte) error {
	return c.Conn.WriteMessage(messageType, data)
}

func (c gorillaConn) Close() error {
	return c.Conn.Close()
}
