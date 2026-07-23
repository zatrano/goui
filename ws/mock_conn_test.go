package ws

import (
	"io"
	"sync"
	"time"
)

type mockConn struct {
	mu      sync.Mutex
	readCh  chan []byte
	writeCh chan []byte
	closed  bool
}

func newMockConn() *mockConn {
	return &mockConn{
		readCh:  make(chan []byte, 8),
		writeCh: make(chan []byte, 8),
	}
}

func (m *mockConn) ReadMessage() (int, []byte, error) {
	msg, ok := <-m.readCh
	if !ok {
		return 0, nil, io.EOF
	}
	return 1, msg, nil
}

func (m *mockConn) WriteMessage(_ int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return io.EOF
	}

	cp := append([]byte(nil), data...)
	select {
	case m.writeCh <- cp:
		return nil
	default:
		return io.ErrShortBuffer
	}
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.readCh)
	}
	return nil
}

func (m *mockConn) send(data []byte) {
	m.readCh <- data
}

func (m *mockConn) readWrite(timeout time.Duration) ([]byte, bool) {
	select {
	case msg := <-m.writeCh:
		return msg, true
	case <-time.After(timeout):
		return nil, false
	}
}
