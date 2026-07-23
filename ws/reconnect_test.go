package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/zatrano/goui/diff"
)

func TestReconnect_WithinGracePeriod(t *testing.T) {
	hub := newTestHub(DefaultGracePeriod, time.Hour)
	defer hub.Stop()

	tr := loadTestTranslator(t)
	conn1 := newMockConn()
	session := newSessionWithConn(conn1, tr, "tr")
	hub.Register(session)

	counter := &eventCounter{Count: 3}
	if err := session.MountComponent("counter-1", counter); err != nil {
		t.Fatalf("MountComponent: %v", err)
	}

	ctx1, cancel1 := context.WithCancel(context.Background())
	runDone := make(chan struct{})
	go func() {
		session.Run(ctx1)
		close(runDone)
	}()

	cancel1()

	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("first Run did not exit")
	}

	if counter.Count != 3 {
		t.Fatalf("count = %d, want 3 after disconnect", counter.Count)
	}

	conn2 := newMockConn()
	if err := session.Reattach(conn2); err != nil {
		t.Fatalf("Reattach: %v", err)
	}

	session.SendInitialRenders()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	go session.Run(ctx2)

	msg, ok := conn2.readWrite(2 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for reattach render frame")
	}

	var frame Frame
	if err := json.Unmarshal(msg, &frame); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if frame.Type != FrameTypeRender {
		t.Fatalf("frame type = %q, want %q", frame.Type, FrameTypeRender)
	}

	var patches []diff.Patch
	if err := json.Unmarshal(frame.Payload, &patches); err != nil {
		t.Fatalf("Unmarshal render payload: %v", err)
	}
	if len(patches) != 1 {
		t.Fatalf("patch count = %d, want 1", len(patches))
	}
	if patches[0].Op != diff.OpReplace {
		t.Fatalf("patch op = %q, want %q", patches[0].Op, diff.OpReplace)
	}
	if patches[0].HTML != `<span data-goui-component="counter-1">3</span>` {
		t.Fatalf("render html = %q, want %q", patches[0].HTML, `<span data-goui-component="counter-1">3</span>`)
	}

	if _, ok := hub.Get(session.ID); !ok {
		t.Fatal("session should remain registered within grace period")
	}

	cancel2()
}

func TestReconnect_AfterGracePeriod(t *testing.T) {
	hub := newTestHub(50*time.Millisecond, 20*time.Millisecond)
	defer hub.Stop()

	tr := loadTestTranslator(t)
	conn := newMockConn()
	session := newSessionWithConn(conn, tr, "tr")
	hub.Register(session)

	comp := &testComponent{}
	if err := session.MountComponent("comp-1", comp); err != nil {
		t.Fatalf("MountComponent: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan struct{})
	go func() {
		session.Run(ctx)
		close(runDone)
	}()

	cancel()

	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit")
	}

	time.Sleep(150 * time.Millisecond)

	if _, ok := hub.Get(session.ID); ok {
		t.Fatal("session should be removed after grace period")
	}
	if !comp.unmounted {
		t.Fatal("expected Unmount to be called after grace period")
	}
}
