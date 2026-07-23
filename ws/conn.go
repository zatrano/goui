package ws

// TextMessage is the WebSocket text opcode (RFC 6455).
const TextMessage = 1

// Conn abstracts a WebSocket connection for adapters and in-memory tests.
type Conn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}
