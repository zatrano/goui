package ws

import (
	"context"
	"encoding/json"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/goui/core"
)

type pendingProbe struct {
	core.BaseComponent
	mounts   *atomic.Int32
	unmounts *atomic.Int32
	delay    time.Duration
}

func (p *pendingProbe) Mount(ctx context.Context) error {
	p.mounts.Add(1)
	if p.delay > 0 {
		select {
		case <-time.After(p.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (p *pendingProbe) Render() (string, error) {
	return `<div class="pending">ok</div>`, nil
}

func (p *pendingProbe) HandleEvent(context.Context, string, map[string]any) error { return nil }

func (p *pendingProbe) Unmount(context.Context) error {
	p.unmounts.Add(1)
	return nil
}

func TestPendingMounts_TakeAdoptsWithoutRemount(t *testing.T) {
	store := NewPendingMounts(time.Second)
	defer store.Stop()

	var mounts, unmounts atomic.Int32
	probe := &pendingProbe{mounts: &mounts, unmounts: &unmounts, delay: 20 * time.Millisecond}

	done := make(chan error, 1)
	go func() { done <- probe.Mount(context.Background()) }()

	token := store.Park("dash", "tr", probe, done, nil)
	if token == "" {
		t.Fatal("expected pending token")
	}

	c, name, locale, err := store.Take(token)
	if err != nil {
		t.Fatalf("Take: %v", err)
	}
	if c != probe {
		t.Fatal("Take returned different component")
	}
	if name != "dash" || locale != "tr" {
		t.Fatalf("name/locale = %q/%q", name, locale)
	}
	if mounts.Load() != 1 {
		t.Fatalf("mounts = %d, want 1", mounts.Load())
	}
	if unmounts.Load() != 0 {
		t.Fatalf("unmounts = %d, want 0 after adopt", unmounts.Load())
	}
	if store.Len() != 0 {
		t.Fatalf("store len = %d after Take", store.Len())
	}
}

func TestPendingMounts_TTLExpiresAndUnmounts(t *testing.T) {
	store := NewPendingMounts(40 * time.Millisecond)
	store.cleanupInterval = 15 * time.Millisecond
	defer store.Stop()

	var mounts, unmounts atomic.Int32
	probe := &pendingProbe{mounts: &mounts, unmounts: &unmounts}

	done := make(chan error, 1)
	go func() { done <- probe.Mount(context.Background()) }()
	token := store.Park("dash", "tr", probe, done, nil)
	if token == "" {
		t.Fatal("expected token")
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if store.Len() == 0 && unmounts.Load() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if store.Len() != 0 {
		t.Fatalf("pending entry still present after TTL")
	}
	if unmounts.Load() != 1 {
		t.Fatalf("unmounts = %d, want 1 after TTL", unmounts.Load())
	}
	if _, _, _, err := store.Take(token); err != ErrPendingNotFound {
		t.Fatalf("Take after TTL err = %v, want ErrPendingNotFound", err)
	}
}

func TestServer_ServeConn_AdoptsPendingMount(t *testing.T) {
	tr := loadTestTranslator(t)
	hub := NewHub()
	defer hub.Stop()

	var mounts atomic.Int32
	reg := core.NewRegistry()
	_ = reg.Register("probe", func() core.Component {
		return &pendingProbe{mounts: &mounts, unmounts: &atomic.Int32{}, delay: 30 * time.Millisecond}
	})

	server := NewServer(hub, reg, tr)
	defer server.Pending.Stop()

	// Simulate page FastRenderTimeout: create+Mount in background, park.
	comp, err := reg.Create("probe")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		PrepareComponent(comp, "ssr", "tr", tr)
		done <- comp.Mount(context.Background())
	}()
	token := server.Pending.Park("probe", "tr", comp, done, nil)

	conn := newMockConn()
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.ServeConn(context.Background(), conn, ConnectParams{
			ComponentName: "probe",
			Locale:        "tr",
			PendingID:     token,
		})
	}()

	deadline := time.Now().Add(2 * time.Second)
	var sawRender bool
	for time.Now().Before(deadline) && !sawRender {
		raw, ok := conn.readWrite(50 * time.Millisecond)
		if !ok {
			continue
		}
		var frame Frame
		if err := json.Unmarshal(raw, &frame); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if frame.Type == FrameTypeRender {
			sawRender = true
		}
	}
	if !sawRender {
		t.Fatal("expected render after adopt")
	}
	if mounts.Load() != 1 {
		t.Fatalf("Mount called %d times, want 1 (no remount on adopt)", mounts.Load())
	}

	_ = conn.Close()
	select {
	case <-serveDone:
	case <-time.After(2 * time.Second):
		t.Fatal("ServeConn did not return")
	}
}

func TestPendingID_CryptoRandom128Bit(t *testing.T) {
	seen := make(map[string]struct{}, 64)
	for i := 0; i < 64; i++ {
		id := newSessionID()
		if len(id) != 32 { // 16 bytes hex-encoded
			t.Fatalf("id len = %d, want 32 hex chars (128 bit)", len(id))
		}
		for _, c := range id {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				t.Fatalf("non-hex char in id %q", id)
			}
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %q (not unpredictable enough)", id)
		}
		seen[id] = struct{}{}
	}
}

func TestPendingMounts_TakeIsSingleUse(t *testing.T) {
	store := NewPendingMounts(time.Second)
	defer store.Stop()

	probe := &pendingProbe{mounts: &atomic.Int32{}, unmounts: &atomic.Int32{}}
	done := make(chan error, 1)
	done <- nil
	token := store.Park("dash", "tr", probe, done, nil)

	if _, _, _, err := store.Take(token); err != nil {
		t.Fatalf("first Take: %v", err)
	}
	if _, _, _, err := store.Take(token); err != ErrPendingNotFound {
		t.Fatalf("second Take err = %v, want ErrPendingNotFound", err)
	}
}

func TestPendingMounts_ConcurrentTakeOnlyOneWins(t *testing.T) {
	store := NewPendingMounts(time.Second)
	defer store.Stop()

	probe := &pendingProbe{mounts: &atomic.Int32{}, unmounts: &atomic.Int32{}, delay: 30 * time.Millisecond}
	done := make(chan error, 1)
	go func() { done <- probe.Mount(context.Background()) }()
	token := store.Park("dash", "tr", probe, done, nil)

	const n = 32
	var success atomic.Int32
	var fail atomic.Int32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _, _, err := store.Take(token)
			if err == nil {
				success.Add(1)
			} else {
				fail.Add(1)
			}
		}()
	}
	wg.Wait()

	if success.Load() != 1 {
		t.Fatalf("successful Takes = %d, want 1", success.Load())
	}
	if fail.Load() != n-1 {
		t.Fatalf("failed Takes = %d, want %d", fail.Load(), n-1)
	}
}

func TestPendingMounts_MaxPendingRejectsOverflow(t *testing.T) {
	store := NewPendingMountsLimited(time.Second, 2)
	defer store.Stop()

	parkOK := func() string {
		probe := &pendingProbe{mounts: &atomic.Int32{}, unmounts: &atomic.Int32{}}
		done := make(chan error, 1)
		done <- nil
		return store.Park("dash", "tr", probe, done, nil)
	}

	if tok := parkOK(); tok == "" {
		t.Fatal("first Park should succeed")
	}
	if tok := parkOK(); tok == "" {
		t.Fatal("second Park should succeed")
	}
	if tok := parkOK(); tok != "" {
		t.Fatalf("third Park at capacity should return empty token, got %q", tok)
	}
	if store.Len() != 2 {
		t.Fatalf("len = %d, want 2", store.Len())
	}
}

func TestPendingMounts_ParkLoadDoesNotAccumulateGoroutines(t *testing.T) {
	store := NewPendingMountsLimited(30*time.Millisecond, 500)
	store.cleanupInterval = 10 * time.Millisecond
	defer store.Stop()

	runtime.GC()
	base := runtime.NumGoroutine()

	const waves = 5
	const perWave = 200
	for w := 0; w < waves; w++ {
		for i := 0; i < perWave; i++ {
			probe := &pendingProbe{mounts: &atomic.Int32{}, unmounts: &atomic.Int32{}}
			done := make(chan error, 1)
			done <- nil
			_ = store.Park("dash", "tr", probe, done, nil)
		}
		// Allow Mount waiters to exit and TTL sweep to reclaim.
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) && store.Len() > 0 {
			time.Sleep(5 * time.Millisecond)
		}
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && store.Len() > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if store.Len() != 0 {
		t.Fatalf("store still has %d entries after TTL", store.Len())
	}

	runtime.GC()
	time.Sleep(20 * time.Millisecond)
	after := runtime.NumGoroutine()
	// Allow slack for the single cleanup goroutine and test runtime noise.
	if after > base+20 {
		t.Fatalf("goroutine leak: before=%d after=%d (grew by %d)", base, after, after-base)
	}
}
