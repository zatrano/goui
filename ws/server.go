package ws

import (
	"context"
	"encoding/json"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/i18n"
)

// Path is the default WebSocket endpoint path.
const Path = "/goui/ws"

// ConnectParams carries query values from the WebSocket upgrade request.
type ConnectParams struct {
	SessionID     string
	ComponentName string
	Locale        string
}

// Server owns hub/registry wiring for framework-agnostic WebSocket accepts.
type Server struct {
	Hub        *Hub
	Registry   *core.Registry
	Translator *i18n.Translator
}

// NewServer builds a Server with the given dependencies.
func NewServer(hub *Hub, registry *core.Registry, translator *i18n.Translator) *Server {
	return &Server{
		Hub:        hub,
		Registry:   registry,
		Translator: translator,
	}
}

// ServeConn runs one GoUI session on an already-upgraded WebSocket connection.
// It blocks until the connection ends.
func (s *Server) ServeConn(ctx context.Context, conn Conn, p ConnectParams) error {
	if s == nil || s.Hub == nil || s.Registry == nil {
		_ = writeErrorFrame(conn, "server not configured")
		_ = conn.Close()
		return ErrServerNotConfigured
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var session *Session
	var reconnect bool

	if p.SessionID != "" {
		existing, ok := s.Hub.Get(p.SessionID)
		if !ok {
			_ = writeErrorFrame(conn, "session not found")
			_ = conn.Close()
			return ErrSessionNotFound
		}
		if err := existing.Reattach(conn); err != nil {
			_ = writeErrorFrame(conn, err.Error())
			_ = conn.Close()
			return err
		}
		session = existing
		reconnect = true
	} else {
		if p.ComponentName == "" {
			_ = writeErrorFrame(conn, ErrComponentRequired.Error())
			_ = conn.Close()
			return ErrComponentRequired
		}

		component, err := s.Registry.Create(p.ComponentName)
		if err != nil {
			_ = writeErrorFrame(conn, err.Error())
			_ = conn.Close()
			return err
		}

		session = NewSession(conn, s.Translator, p.Locale)
		session.SetRegistry(s.Registry)
		componentID := newSessionID()

		if err := session.MountComponent(componentID, component); err != nil {
			_ = writeErrorFrame(conn, err.Error())
			_ = conn.Close()
			return err
		}

		s.Hub.Register(session)
	}

	session.SetRegistry(s.Registry)

	if !reconnect {
		session.SendSessionFrame()
	}

	session.SendInitialRenders()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	session.Run(runCtx)
	return nil
}

func writeErrorFrame(conn Conn, message string) error {
	if conn == nil {
		return nil
	}
	data, err := json.Marshal(newErrorFrame(message))
	if err != nil {
		return err
	}
	return conn.WriteMessage(TextMessage, data)
}
