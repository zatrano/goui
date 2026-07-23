package stdlib

import (
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/zatrano/goui/ws"
)

// NewWSHandler upgrades HTTP requests to WebSocket and runs ws.Server.ServeConn.
func NewWSHandler(server *ws.Server, checkOrigin func(*http.Request) bool) http.Handler {
	upgrader := websocket.Upgrader{
		CheckOrigin: checkOrigin,
	}
	if upgrader.CheckOrigin == nil {
		upgrader.CheckOrigin = func(*http.Request) bool { return true }
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		q := r.URL.Query()
		_ = server.ServeConn(r.Context(), wrapConn(conn), ws.ConnectParams{
			SessionID:     q.Get("session"),
			ComponentName: q.Get("component"),
			Locale:        q.Get("locale"),
		})
	})
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
