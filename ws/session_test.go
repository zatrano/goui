package ws

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/diff"
	"github.com/zatrano/goui/i18n"
)

type testComponent struct {
	core.BaseComponent
	mounted   bool
	unmounted bool
	renderVal string
}

func (t *testComponent) Mount(_ context.Context) error {
	t.mounted = true
	return nil
}

func (t *testComponent) Render() (string, error) {
	if t.renderVal != "" {
		return t.renderVal, nil
	}
	return "<div>ok</div>", nil
}

func (t *testComponent) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

func (t *testComponent) Unmount(_ context.Context) error {
	t.unmounted = true
	return nil
}

type eventCounter struct {
	core.BaseComponent
	Count int
}

func (c *eventCounter) Mount(_ context.Context) error { return nil }

func (c *eventCounter) Render() (string, error) {
	return "<span>" + strconv.Itoa(c.Count) + "</span>", nil
}

func (c *eventCounter) HandleEvent(_ context.Context, event string, _ map[string]any) error {
	switch event {
	case "increment":
		c.Count++
	case "decrement":
		c.Count--
	}
	return nil
}

func (c *eventCounter) Unmount(_ context.Context) error { return nil }

func loadTestTranslator(t *testing.T) *i18n.Translator {
	t.Helper()

	tr := i18n.NewTranslator()
	if err := tr.LoadLocale("tr", filepath.Join("..", "i18n", "locales", "tr.json")); err != nil {
		t.Fatalf("LoadLocale: %v", err)
	}
	if err := tr.LoadLocale("en", filepath.Join("..", "i18n", "locales", "en.json")); err != nil {
		t.Fatalf("LoadLocale en: %v", err)
	}
	return tr
}

func TestSession_MountComponent(t *testing.T) {
	tr := loadTestTranslator(t)
	session := newSessionWithConn(newMockConn(), tr, "en")

	comp := &testComponent{}
	if err := session.MountComponent("comp-1", comp); err != nil {
		t.Fatalf("MountComponent: %v", err)
	}

	if comp.ID != "comp-1" {
		t.Fatalf("component ID = %q, want %q", comp.ID, "comp-1")
	}
	if comp.Locale != "en" {
		t.Fatalf("component Locale = %q, want %q", comp.Locale, "en")
	}
	if !comp.mounted {
		t.Fatal("expected component Mount to be called")
	}
	if comp.T("form.submit") != "Submit" {
		t.Fatalf("translator not injected, got %q", comp.T("form.submit"))
	}
}

func TestSession_PusherInjection(t *testing.T) {
	tr := loadTestTranslator(t)
	session := newSessionWithConn(newMockConn(), tr, "tr")
	comp := &testComponent{}
	if err := session.MountComponent("comp-1", comp); err != nil {
		t.Fatalf("MountComponent: %v", err)
	}

	comp.Toast("warning", "Dikkat")

	select {
	case frame := <-session.outbound:
		if frame.Type != FrameTypePush {
			t.Fatalf("type=%q", frame.Type)
		}
		var msg PushMessage
		if err := json.Unmarshal(frame.Payload, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Kind != "warning" || msg.Text != "Dikkat" {
			t.Fatalf("%#v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("expected push frame on outbound")
	}
}

func TestSession_HandleEvent(t *testing.T) {
	tr := loadTestTranslator(t)
	conn := newMockConn()
	session := newSessionWithConn(conn, tr, "tr")

	counter := &eventCounter{}
	if err := session.MountComponent("counter-1", counter); err != nil {
		t.Fatalf("MountComponent: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		session.Run(ctx)
		close(done)
	}()

	session.SendInitialRenders()
	if _, ok := conn.readWrite(2 * time.Second); !ok {
		t.Fatal("timed out waiting for initial render frame")
	}

	eventFrame, err := json.Marshal(Frame{
		Type:      FrameTypeEvent,
		Component: "counter-1",
		Event:     "increment",
	})
	if err != nil {
		t.Fatalf("Marshal event frame: %v", err)
	}

	conn.send(eventFrame)

	msg, ok := conn.readWrite(2 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for render frame")
	}

	var frame Frame
	if err := json.Unmarshal(msg, &frame); err != nil {
		t.Fatalf("Unmarshal response: %v", err)
	}
	if frame.Type != FrameTypeRender {
		t.Fatalf("frame type = %q, want %q", frame.Type, FrameTypeRender)
	}
	if frame.Component != "counter-1" {
		t.Fatalf("frame component = %q, want %q", frame.Component, "counter-1")
	}

	var patches []diff.Patch
	if err := json.Unmarshal(frame.Payload, &patches); err != nil {
		t.Fatalf("Unmarshal render payload: %v", err)
	}
	if len(patches) != 1 {
		t.Fatalf("patch count = %d, want 1", len(patches))
	}
	if patches[0].Op != diff.OpUpdateText {
		t.Fatalf("patch op = %q, want %q", patches[0].Op, diff.OpUpdateText)
	}
	if patches[0].Text != "1" {
		t.Fatalf("patch text = %q, want %q", patches[0].Text, "1")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("session Run did not exit after cancel")
	}
}

func TestSession_RunGoroutineCleanup(t *testing.T) {
	tr := loadTestTranslator(t)
	conn := newMockConn()
	session := newSessionWithConn(conn, tr, "tr")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		session.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit after context cancellation")
	}

	if !session.IsDisconnected() {
		t.Fatal("expected session to be marked disconnected")
	}
}

func TestSession_FirstRenderIsFullReplace(t *testing.T) {
	tr := loadTestTranslator(t)
	conn := newMockConn()
	session := newSessionWithConn(conn, tr, "tr")

	component := &testComponent{renderVal: "<div><span>ok</span></div>"}
	if err := session.MountComponent("comp-1", component); err != nil {
		t.Fatalf("MountComponent: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		session.Run(ctx)
		close(done)
	}()

	session.SendInitialRenders()

	msg, ok := conn.readWrite(2 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for initial render frame")
	}

	var frame Frame
	if err := json.Unmarshal(msg, &frame); err != nil {
		t.Fatalf("Unmarshal frame: %v", err)
	}

	var patches []diff.Patch
	if err := json.Unmarshal(frame.Payload, &patches); err != nil {
		t.Fatalf("Unmarshal patches: %v", err)
	}
	if len(patches) != 1 {
		t.Fatalf("patch count = %d, want 1", len(patches))
	}
	if patches[0].Op != diff.OpReplace {
		t.Fatalf("patch op = %q, want %q", patches[0].Op, diff.OpReplace)
	}
	if patches[0].HTML != `<div data-goui-component="comp-1"><span>ok</span></div>` {
		t.Fatalf("patch html = %q, want %q", patches[0].HTML, `<div data-goui-component="comp-1"><span>ok</span></div>`)
	}

	cancel()
	<-done
}

func TestSession_SecondRenderIsMinimalPatch(t *testing.T) {
	tr := loadTestTranslator(t)
	conn := newMockConn()
	session := newSessionWithConn(conn, tr, "tr")

	counter := &eventCounter{}
	if err := session.MountComponent("counter-1", counter); err != nil {
		t.Fatalf("MountComponent: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		session.Run(ctx)
		close(done)
	}()

	session.SendInitialRenders()
	if _, ok := conn.readWrite(2 * time.Second); !ok {
		t.Fatal("timed out waiting for first render")
	}

	eventFrame, err := json.Marshal(Frame{
		Type:      FrameTypeEvent,
		Component: "counter-1",
		Event:     "increment",
	})
	if err != nil {
		t.Fatalf("Marshal event frame: %v", err)
	}
	conn.send(eventFrame)

	msg, ok := conn.readWrite(2 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for second render")
	}

	var frame Frame
	if err := json.Unmarshal(msg, &frame); err != nil {
		t.Fatalf("Unmarshal frame: %v", err)
	}

	var patches []diff.Patch
	if err := json.Unmarshal(frame.Payload, &patches); err != nil {
		t.Fatalf("Unmarshal patches: %v", err)
	}
	if len(patches) != 1 {
		t.Fatalf("patch count = %d, want 1", len(patches))
	}
	if patches[0].Op != diff.OpUpdateText {
		t.Fatalf("patch op = %q, want %q", patches[0].Op, diff.OpUpdateText)
	}
	if len(patches[0].Path) != 1 || patches[0].Path[0] != 0 {
		t.Fatalf("patch path = %v, want [0] relative to component root", patches[0].Path)
	}
	if patches[0].HTML != "" {
		t.Fatalf("expected minimal patch without full html, got %q", patches[0].HTML)
	}
	if patches[0].Text != "1" {
		t.Fatalf("patch text = %q, want 1", patches[0].Text)
	}

	cancel()
	<-done
}
