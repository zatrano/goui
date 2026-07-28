package ws

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zatrano/goui/v2/core"
)

// DefaultPendingTTL is how long a FastRenderTimeout Mount may wait for a
// WebSocket adopt before it is cancelled, Unmounted, and dropped.
const DefaultPendingTTL = 30 * time.Second

// DefaultMaxPending caps concurrent parked Mounts to bound memory under load
// or intentional Park flooding without WebSocket adopt.
const DefaultMaxPending = 10_000

const pendingCleanupInterval = 5 * time.Second

// PendingStats is a point-in-time snapshot of PendingMounts counters.
type PendingStats struct {
	Parked   int64
	Adopted  int64
	Expired  int64
	Rejected int64 // Park refused at MaxPending
}

// PendingMounts holds ModeLive/SEO Mounts waiting to be adopted by the first
// WebSocket connect (avoiding a second Mount).
type PendingMounts struct {
	mu              sync.Mutex
	items           map[string]*pendingEntry
	ttl             time.Duration
	maxPending      int
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	cleanupDone     chan struct{}

	parked   atomic.Int64
	adopted  atomic.Int64
	expired  atomic.Int64
	rejected atomic.Int64
}

type pendingEntry struct {
	component core.Component
	name      string
	locale    string
	ready     chan struct{} // closed when Mount finishes
	mountErr  error
	expiresAt time.Time
	taken     bool
	cancel    context.CancelFunc // cancels Mount ctx on expire/discard; may be nil
}

// NewPendingMounts creates a store with the given TTL (DefaultPendingTTL when <= 0),
// DefaultMaxPending capacity, and a single background sweep goroutine.
func NewPendingMounts(ttl time.Duration) *PendingMounts {
	return NewPendingMountsLimited(ttl, DefaultMaxPending)
}

// NewPendingMountsLimited is like NewPendingMounts with a custom capacity.
func NewPendingMountsLimited(ttl time.Duration, maxPending int) *PendingMounts {
	if ttl <= 0 {
		ttl = DefaultPendingTTL
	}
	if maxPending <= 0 {
		maxPending = DefaultMaxPending
	}
	p := &PendingMounts{
		items:           make(map[string]*pendingEntry),
		ttl:             ttl,
		maxPending:      maxPending,
		cleanupInterval: pendingCleanupInterval,
		stopCleanup:     make(chan struct{}),
		cleanupDone:     make(chan struct{}),
	}
	go p.cleanupLoop()
	return p
}

// Park registers a component whose Mount is already running or finished.
// mountDone must receive the Mount error exactly once when Mount returns.
// cancel (optional) is invoked when the entry is expired or discarded so Mount
// can observe ctx.Done(); FastRenderTimeout itself must not cancel Mount.
// The returned token is sent to the browser as ?pending= for WS adopt.
// If the store is at MaxPending after expiring stale entries, Park returns ""
// and discards the Mount (caller falls back to empty shell + fresh WS Mount).
func (p *PendingMounts) Park(componentName, locale string, c core.Component, mountDone <-chan error, cancel context.CancelFunc) string {
	if p == nil {
		go drainMountDiscard(c, mountDone, cancel)
		return ""
	}

	token := newSessionID()
	e := &pendingEntry{
		component: c,
		name:      componentName,
		locale:    locale,
		ready:     make(chan struct{}),
		expiresAt: time.Now().Add(p.ttl),
		cancel:    cancel,
	}

	p.mu.Lock()
	expired := p.collectExpiredLocked(time.Now())
	atCap := len(p.items) >= p.maxPending
	if !atCap {
		p.items[token] = e
	}
	p.mu.Unlock()

	for _, ex := range expired {
		p.expired.Add(1)
		discardPendingEntry(ex)
	}

	if atCap {
		p.rejected.Add(1)
		go drainMountDiscard(c, mountDone, cancel)
		return ""
	}

	p.parked.Add(1)
	go func() {
		err := <-mountDone
		e.mountErr = err
		close(e.ready)
		if err != nil {
			p.mu.Lock()
			if cur, ok := p.items[token]; ok && cur == e && !cur.taken {
				delete(p.items, token)
			}
			p.mu.Unlock()
		}
	}()

	return token
}

// ParkReady parks an already-mounted component (Mount returned nil).
func (p *PendingMounts) ParkReady(componentName, locale string, c core.Component, cancel context.CancelFunc) string {
	done := make(chan error, 1)
	done <- nil
	return p.Park(componentName, locale, c, done, cancel)
}

// Take removes a pending entry by token (single-use), waits for Mount to finish,
// and returns the component for session adopt. Callers must not Mount again.
// Concurrent Takes on the same token: exactly one succeeds; others get ErrPendingNotFound.
func (p *PendingMounts) Take(token string) (c core.Component, name, locale string, err error) {
	if p == nil || token == "" {
		return nil, "", "", ErrPendingNotFound
	}

	p.mu.Lock()
	e, ok := p.items[token]
	if !ok || e.taken {
		p.mu.Unlock()
		return nil, "", "", ErrPendingNotFound
	}
	e.taken = true
	delete(p.items, token)
	p.mu.Unlock()

	<-e.ready
	if e.mountErr != nil {
		if e.cancel != nil {
			e.cancel()
		}
		return nil, "", "", e.mountErr
	}
	// Adopted: drop cancel so session owns the lifetime (do not cancel Mount).
	e.cancel = nil
	p.adopted.Add(1)
	return e.component, e.name, e.locale, nil
}

// Stats returns cumulative counters for monitoring.
func (p *PendingMounts) Stats() PendingStats {
	if p == nil {
		return PendingStats{}
	}
	return PendingStats{
		Parked:   p.parked.Load(),
		Adopted:  p.adopted.Load(),
		Expired:  p.expired.Load(),
		Rejected: p.rejected.Load(),
	}
}

// Len reports how many entries are waiting (tests / diagnostics).
func (p *PendingMounts) Len() int {
	if p == nil {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.items)
}

// MaxPending returns the configured capacity.
func (p *PendingMounts) MaxPending() int {
	if p == nil {
		return 0
	}
	return p.maxPending
}

// Stop ends the cleanup loop. Safe to call once.
func (p *PendingMounts) Stop() {
	if p == nil {
		return
	}
	select {
	case <-p.stopCleanup:
		return
	default:
		close(p.stopCleanup)
	}
	<-p.cleanupDone

	p.mu.Lock()
	leftover := make([]*pendingEntry, 0, len(p.items))
	for id, e := range p.items {
		if !e.taken {
			e.taken = true
			leftover = append(leftover, e)
		}
		delete(p.items, id)
	}
	p.mu.Unlock()

	for _, e := range leftover {
		p.expired.Add(1)
		discardPendingEntry(e)
	}
}

func (p *PendingMounts) cleanupLoop() {
	defer close(p.cleanupDone)
	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCleanup:
			return
		case <-ticker.C:
			p.expire()
		}
	}
}

func (p *PendingMounts) expire() {
	p.mu.Lock()
	expired := p.collectExpiredLocked(time.Now())
	p.mu.Unlock()
	for _, e := range expired {
		p.expired.Add(1)
		discardPendingEntry(e)
	}
}

// collectExpiredLocked removes expired entries from the map. Caller must hold p.mu.
func (p *PendingMounts) collectExpiredLocked(now time.Time) []*pendingEntry {
	var expired []*pendingEntry
	for id, e := range p.items {
		if e.taken || now.Before(e.expiresAt) {
			continue
		}
		e.taken = true
		delete(p.items, id)
		expired = append(expired, e)
	}
	return expired
}

func discardPendingEntry(e *pendingEntry) {
	if e.cancel != nil {
		e.cancel()
	}
	<-e.ready
	if e.mountErr == nil && e.component != nil {
		_ = e.component.Unmount(context.Background())
	}
}

func drainMountDiscard(c core.Component, mountDone <-chan error, cancel context.CancelFunc) {
	if cancel != nil {
		cancel()
	}
	if err := <-mountDone; err == nil && c != nil {
		_ = c.Unmount(context.Background())
	}
}
