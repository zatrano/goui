package fiber

import (
	"github.com/gofiber/contrib/v3/websocket"

	"github.com/zatrano/goui/ws"
)

type fiberConn struct {
	*websocket.Conn
}

func wrapConn(c *websocket.Conn) ws.Conn {
	if c == nil {
		return nil
	}
	return fiberConn{Conn: c}
}

func (c fiberConn) ReadMessage() (int, []byte, error) {
	return c.Conn.ReadMessage()
}

func (c fiberConn) WriteMessage(messageType int, data []byte) error {
	return c.Conn.WriteMessage(messageType, data)
}

func (c fiberConn) Close() error {
	return c.Conn.Close()
}
