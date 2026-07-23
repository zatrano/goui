package ginadapter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/zatrano/goui/page"
	"github.com/zatrano/goui/upload"
	"github.com/zatrano/goui/ws"
)

// Options configures Gin registration.
type Options struct {
	Server      *ws.Server
	Store       upload.Storage
	CheckOrigin func(*http.Request) bool

	Page   *page.Renderer
	Routes []page.Route
}

// Register mounts GoUI WebSocket, optional upload, and optional page routes.
func Register(r gin.IRoutes, opts Options) {
	if opts.Server != nil {
		RegisterWS(r, opts.Server, opts.CheckOrigin)
	}
	if opts.Store != nil {
		RegisterUpload(r, opts.Store)
	}
	RegisterPages(r, opts.Page, opts.Routes)
}

// RegisterWS wires GET /goui/ws.
func RegisterWS(r gin.IRoutes, server *ws.Server, checkOrigin func(*http.Request) bool) {
	upgrader := websocket.Upgrader{CheckOrigin: checkOrigin}
	if upgrader.CheckOrigin == nil {
		upgrader.CheckOrigin = func(*http.Request) bool { return true }
	}

	r.GET(ws.Path, func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		q := c.Request.URL.Query()
		_ = server.ServeConn(c.Request.Context(), wrapConn(conn), ws.ConnectParams{
			SessionID:     q.Get("session"),
			ComponentName: q.Get("component"),
			Locale:        q.Get("locale"),
		})
	})
}

// RegisterUpload wires upload and file download endpoints.
func RegisterUpload(r gin.IRoutes, store upload.Storage) {
	h := upload.NewHandler(store)
	r.POST(upload.UploadPath, gin.WrapH(h))
	r.GET(upload.FilesPrefix+"/*filepath", gin.WrapH(h))
}

// RegisterPages mounts HTML page handlers for each route.
func RegisterPages(r gin.IRoutes, renderer *page.Renderer, routes []page.Route) {
	if renderer == nil || len(routes) == 0 {
		return
	}
	for _, route := range routes {
		if route.Path == "" || route.Component == "" {
			continue
		}
		r.GET(route.Path, gin.WrapH(renderer.Handler(route.Component)))
	}
}

// Page returns a Gin handler that renders a registered component document.
func Page(renderer *page.Renderer, component string) gin.HandlerFunc {
	return gin.WrapH(renderer.Handler(component))
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
