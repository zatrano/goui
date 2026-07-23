package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func newTestHub(grace, cleanupEvery time.Duration) *Hub {
	h := &Hub{
		sessions:        make(map[string]*Session),
		gracePeriod:     grace,
		cleanupInterval: cleanupEvery,
		stopCleanup:     make(chan struct{}),
		cleanupDone:     make(chan struct{}),
	}
	go h.cleanupLoop()
	return h
}

func TestHub_RegisterUnregisterPush(t *testing.T) {
	hub := newTestHub(DefaultGracePeriod, time.Hour)
	defer hub.Stop()

	tr := loadTestTranslator(t)
	conn := newMockConn()
	session := newSessionWithConn(conn, tr, "tr")
	hub.Register(session)

	ctx, cancel := contextWithCancel(t)
	defer cancel()

	go session.Run(ctx)

	if err := hub.Push(session.ID, PushMessage{Kind: "success", Text: "Kayıt başarılı"}); err != nil {
		t.Fatalf("Push: %v", err)
	}

	msg, ok := conn.readWrite(2 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for push frame")
	}

	var frame Frame
	if err := json.Unmarshal(msg, &frame); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if frame.Type != FrameTypePush {
		t.Fatalf("frame type = %q, want %q", frame.Type, FrameTypePush)
	}

	var push PushMessage
	if err := json.Unmarshal(frame.Payload, &push); err != nil {
		t.Fatalf("Unmarshal push payload: %v", err)
	}
	if push.Kind != "success" || push.Text != "Kayıt başarılı" {
		t.Fatalf("push = %+v, want success/Kayıt başarılı", push)
	}

	hub.Unregister(session.ID)
	if _, ok := hub.Get(session.ID); ok {
		t.Fatal("session should be unregistered")
	}
}

func TestHub_Broadcast(t *testing.T) {
	hub := newTestHub(DefaultGracePeriod, time.Hour)
	defer hub.Stop()

	tr := loadTestTranslator(t)

	conn1 := newMockConn()
	session1 := newSessionWithConn(conn1, tr, "tr")
	hub.Register(session1)

	conn2 := newMockConn()
	session2 := newSessionWithConn(conn2, tr, "tr")
	hub.Register(session2)

	ctx1, cancel1 := contextWithCancel(t)
	defer cancel1()
	ctx2, cancel2 := contextWithCancel(t)
	defer cancel2()

	go session1.Run(ctx1)
	go session2.Run(ctx2)

	hub.Broadcast(PushMessage{Kind: "info", Text: "Duyuru"})

	msg1, ok := conn1.readWrite(2 * time.Second)
	if !ok {
		t.Fatal("session1 did not receive broadcast")
	}
	msg2, ok := conn2.readWrite(2 * time.Second)
	if !ok {
		t.Fatal("session2 did not receive broadcast")
	}

	for i, msg := range [][]byte{msg1, msg2} {
		var frame Frame
		if err := json.Unmarshal(msg, &frame); err != nil {
			t.Fatalf("session%d unmarshal: %v", i+1, err)
		}
		if frame.Type != FrameTypePush {
			t.Fatalf("session%d frame type = %q", i+1, frame.Type)
		}
	}
}

func TestHub_PushToUnknownSession(t *testing.T) {
	hub := newTestHub(DefaultGracePeriod, time.Hour)
	defer hub.Stop()

	err := hub.Push("missing-session", PushMessage{Kind: "error", Text: "x"})
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func contextWithCancel(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithCancel(context.Background())
}
