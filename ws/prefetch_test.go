package ws

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zatrano/goui/core"
)

type prefetchProbe struct {
	core.BaseComponent
	MountCount int
	Token      string
	unmounted  bool
}

func (p *prefetchProbe) Mount(_ context.Context) error {
	p.MountCount++
	if p.Token == "" {
		p.Token = "mounted-once"
	}
	return nil
}

func (p *prefetchProbe) Render() (string, error) {
	return "<div>" + p.Token + "</div>", nil
}

func (p *prefetchProbe) HandleEvent(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

func (p *prefetchProbe) Unmount(_ context.Context) error {
	p.unmounted = true
	return nil
}

func newPrefetchRegistry(t *testing.T, names ...string) (*core.Registry, map[string]*prefetchProbe, map[string]int) {
	t.Helper()
	reg := core.NewRegistry()
	probes := make(map[string]*prefetchProbe)
	creates := make(map[string]int)
	for _, name := range names {
		n := name
		if err := reg.Register(n, func() core.Component {
			creates[n]++
			p := &prefetchProbe{Token: "token-" + n}
			probes[n] = p
			return p
		}); err != nil {
			t.Fatalf("Register %s: %v", n, err)
		}
	}
	return reg, probes, creates
}

func assertNoOutbound(t *testing.T, session *Session) {
	t.Helper()
	select {
	case frame := <-session.outbound:
		t.Fatalf("unexpected outbound frame: type=%q component=%q", frame.Type, frame.Component)
	default:
	}
}

func TestSession_Prefetch_MountsWithoutRender(t *testing.T) {
	tr := loadTestTranslator(t)
	session := newSessionWithConn(newMockConn(), tr, "tr")
	reg, probes, creates := newPrefetchRegistry(t, "warm")
	session.SetRegistry(reg)

	if err := session.Prefetch("warm"); err != nil {
		t.Fatalf("Prefetch: %v", err)
	}

	assertNoOutbound(t, session)

	session.mu.RLock()
	_, inPrefetch := session.prefetched["warm"]
	_, inActive := session.components["warm"]
	session.mu.RUnlock()
	if !inPrefetch {
		t.Fatal("expected component in prefetched map")
	}
	if inActive {
		t.Fatal("prefetch must not place component in active components")
	}

	p := probes["warm"]
	if p == nil || p.MountCount != 1 {
		t.Fatalf("MountCount = %v, want 1", p)
	}
	if creates["warm"] != 1 {
		t.Fatalf("Create count = %d, want 1", creates["warm"])
	}
}

func TestSession_Prefetch_DuplicateIsNoOp(t *testing.T) {
	tr := loadTestTranslator(t)
	session := newSessionWithConn(newMockConn(), tr, "tr")
	reg, probes, creates := newPrefetchRegistry(t, "warm")
	session.SetRegistry(reg)

	if err := session.Prefetch("warm"); err != nil {
		t.Fatalf("Prefetch: %v", err)
	}
	if err := session.Prefetch("warm"); err != nil {
		t.Fatalf("Prefetch duplicate: %v", err)
	}

	if creates["warm"] != 1 {
		t.Fatalf("Create count = %d, want 1 (second prefetch must not Create)", creates["warm"])
	}
	if probes["warm"].MountCount != 1 {
		t.Fatalf("MountCount = %d, want 1 (second prefetch must be no-op)", probes["warm"].MountCount)
	}
}

func TestSession_Prefetch_ActivateUsesExisting(t *testing.T) {
	tr := loadTestTranslator(t)
	session := newSessionWithConn(newMockConn(), tr, "tr")
	reg, probes, creates := newPrefetchRegistry(t, "warm")
	session.SetRegistry(reg)

	if err := session.Prefetch("warm"); err != nil {
		t.Fatalf("Prefetch: %v", err)
	}
	original := probes["warm"]
	originalToken := original.Token

	id, err := session.Activate("warm")
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty component id")
	}

	if creates["warm"] != 1 {
		t.Fatalf("Create count = %d, want 1 (activate must not Create again)", creates["warm"])
	}
	if original.MountCount != 1 {
		t.Fatalf("MountCount = %d, want 1 (activate must not remount)", original.MountCount)
	}
	if original.Token != originalToken {
		t.Fatalf("Token = %q, want %q (state must be preserved)", original.Token, originalToken)
	}

	session.mu.RLock()
	active, ok := session.components[id]
	_, stillPrefetched := session.prefetched["warm"]
	session.mu.RUnlock()
	if !ok || active != original {
		t.Fatal("activate must reuse the prefetched instance")
	}
	if stillPrefetched {
		t.Fatal("activated component must leave prefetched map")
	}

	select {
	case frame := <-session.outbound:
		if frame.Type != FrameTypeRender {
			t.Fatalf("frame type = %q, want render", frame.Type)
		}
		if frame.Component != id {
			t.Fatalf("frame component = %q, want %q", frame.Component, id)
		}
	case <-time.After(time.Second):
		t.Fatal("expected render frame after activate")
	}
}

func TestSession_Prefetch_LRUEviction(t *testing.T) {
	tr := loadTestTranslator(t)
	session := newSessionWithConn(newMockConn(), tr, "tr")

	names := make([]string, MaxPrefetch+1)
	for i := range names {
		names[i] = fmt.Sprintf("c%d", i)
	}
	reg, probes, _ := newPrefetchRegistry(t, names...)
	session.SetRegistry(reg)

	for _, name := range names {
		if err := session.Prefetch(name); err != nil {
			t.Fatalf("Prefetch %s: %v", name, err)
		}
	}

	oldest := names[0]
	if !probes[oldest].unmounted {
		t.Fatalf("expected oldest prefetch %q to be Unmounted", oldest)
	}

	session.mu.RLock()
	_, stillThere := session.prefetched[oldest]
	count := len(session.prefetched)
	session.mu.RUnlock()
	if stillThere {
		t.Fatalf("oldest %q should be evicted from prefetched", oldest)
	}
	if count != MaxPrefetch {
		t.Fatalf("prefetched count = %d, want %d", count, MaxPrefetch)
	}
}

func TestSession_Prefetch_CleanedOnGracePeriodExpiry(t *testing.T) {
	hub := newTestHub(50*time.Millisecond, 20*time.Millisecond)
	defer hub.Stop()

	tr := loadTestTranslator(t)
	conn := newMockConn()
	session := newSessionWithConn(conn, tr, "tr")
	reg, probes, _ := newPrefetchRegistry(t, "warm")
	session.SetRegistry(reg)
	hub.Register(session)

	if err := session.Prefetch("warm"); err != nil {
		t.Fatalf("Prefetch: %v", err)
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
	if !probes["warm"].unmounted {
		t.Fatal("expected prefetched component Unmount after grace period")
	}
}
