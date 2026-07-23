package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/zatrano/goui/core"
)

type serveProbe struct {
	core.BaseComponent
	label string
}

func (p *serveProbe) Mount(_ context.Context) error { return nil }

func (p *serveProbe) Render() (string, error) {
	return "<div>" + p.label + "</div>", nil
}

func (p *serveProbe) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

func (p *serveProbe) Unmount(_ context.Context) error { return nil }

func TestServer_ServeConn_FreshSession(t *testing.T) {
	tr := loadTestTranslator(t)
	hub := NewHub()
	defer hub.Stop()

	reg := core.NewRegistry()
	if err := reg.Register("probe", func() core.Component {
		return &serveProbe{label: "fresh"}
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	server := NewServer(hub, reg, tr)
	conn := newMockConn()

	done := make(chan error, 1)
	go func() {
		done <- server.ServeConn(context.Background(), conn, ConnectParams{
			ComponentName: "probe",
			Locale:        "tr",
		})
	}()

	deadline := time.Now().Add(2 * time.Second)
	var sawSession, sawRender bool
	for time.Now().Before(deadline) && !(sawSession && sawRender) {
		raw, ok := conn.readWrite(50 * time.Millisecond)
		if !ok {
			continue
		}
		var frame Frame
		if err := json.Unmarshal(raw, &frame); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		switch frame.Type {
		case FrameTypeSession:
			sawSession = true
		case FrameTypeRender:
			sawRender = true
		}
	}
	if !sawSession || !sawRender {
		t.Fatalf("expected session+render frames, got session=%v render=%v", sawSession, sawRender)
	}

	_ = conn.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ServeConn did not return after close")
	}
}

func TestServer_ServeConn_MissingComponent(t *testing.T) {
	tr := loadTestTranslator(t)
	hub := NewHub()
	defer hub.Stop()
	server := NewServer(hub, core.NewRegistry(), tr)
	conn := newMockConn()

	err := server.ServeConn(context.Background(), conn, ConnectParams{})
	if err != ErrComponentRequired {
		t.Fatalf("err = %v, want ErrComponentRequired", err)
	}

	raw, ok := conn.readWrite(time.Second)
	if !ok {
		t.Fatal("expected error frame")
	}
	var frame Frame
	if err := json.Unmarshal(raw, &frame); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if frame.Type != FrameTypeError {
		t.Fatalf("frame type = %q, want error", frame.Type)
	}
}

func TestServer_ServeConn_UnknownSession(t *testing.T) {
	tr := loadTestTranslator(t)
	hub := NewHub()
	defer hub.Stop()
	server := NewServer(hub, core.NewRegistry(), tr)
	conn := newMockConn()

	err := server.ServeConn(context.Background(), conn, ConnectParams{SessionID: "missing"})
	if err != ErrSessionNotFound {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
}

func TestServer_ServeConn_Reconnect(t *testing.T) {
	tr := loadTestTranslator(t)
	hub := NewHub()
	defer hub.Stop()

	reg := core.NewRegistry()
	if err := reg.Register("probe", func() core.Component {
		return &serveProbe{label: "reconnect"}
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	server := NewServer(hub, reg, tr)
	conn1 := newMockConn()
	done1 := make(chan struct{})
	go func() {
		_ = server.ServeConn(context.Background(), conn1, ConnectParams{ComponentName: "probe"})
		close(done1)
	}()

	var sessionID string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && sessionID == "" {
		raw, ok := conn1.readWrite(50 * time.Millisecond)
		if !ok {
			continue
		}
		var frame Frame
		if err := json.Unmarshal(raw, &frame); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if frame.Type == FrameTypeSession {
			var payload SessionPayload
			if err := json.Unmarshal(frame.Payload, &payload); err != nil {
				t.Fatalf("session payload: %v", err)
			}
			sessionID = payload.ID
		}
	}
	if sessionID == "" {
		t.Fatal("expected session id")
	}

	_ = conn1.Close()
	select {
	case <-done1:
	case <-time.After(2 * time.Second):
		t.Fatal("first ServeConn did not return")
	}

	conn2 := newMockConn()
	done2 := make(chan error, 1)
	go func() {
		done2 <- server.ServeConn(context.Background(), conn2, ConnectParams{SessionID: sessionID})
	}()

	deadline = time.Now().Add(2 * time.Second)
	sawSession := false
	sawRender := false
	for time.Now().Before(deadline) && !sawRender {
		raw, ok := conn2.readWrite(50 * time.Millisecond)
		if !ok {
			continue
		}
		var frame Frame
		if err := json.Unmarshal(raw, &frame); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if frame.Type == FrameTypeSession {
			sawSession = true
		}
		if frame.Type == FrameTypeRender {
			sawRender = true
		}
	}
	if sawSession {
		t.Fatal("reconnect must not send session frame")
	}
	if !sawRender {
		t.Fatal("expected render after reconnect")
	}

	_ = conn2.Close()
	select {
	case <-done2:
	case <-time.After(2 * time.Second):
		t.Fatal("second ServeConn did not return")
	}
}
