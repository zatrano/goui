package ws

import (
	"sync"
	"time"
)

// DefaultGracePeriod is how long disconnected sessions are kept before cleanup.
const DefaultGracePeriod = 60 * time.Second

const defaultCleanupInterval = 10 * time.Second

// Hub tracks active sessions and routes push messages.
type Hub struct {
	sessions        map[string]*Session
	mu              sync.RWMutex
	gracePeriod     time.Duration
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	cleanupDone     chan struct{}
}

// NewHub creates a hub with the default grace period and starts cleanup.
func NewHub() *Hub {
	return NewHubWithGracePeriod(DefaultGracePeriod)
}

// NewHubWithGracePeriod creates a hub with a custom grace period (mainly for tests).
func NewHubWithGracePeriod(grace time.Duration) *Hub {
	h := &Hub{
		sessions:        make(map[string]*Session),
		gracePeriod:     grace,
		cleanupInterval: defaultCleanupInterval,
		stopCleanup:     make(chan struct{}),
		cleanupDone:     make(chan struct{}),
	}
	go h.cleanupLoop()
	return h
}

// Register adds a session to the hub.
func (h *Hub) Register(s *Session) {
	h.mu.Lock()
	h.sessions[s.ID] = s
	h.mu.Unlock()
}

// Unregister removes a session from the hub.
func (h *Hub) Unregister(sessionID string) {
	h.mu.Lock()
	delete(h.sessions, sessionID)
	h.mu.Unlock()
}

// Get returns a session by ID.
func (h *Hub) Get(sessionID string) (*Session, bool) {
	h.mu.RLock()
	s, ok := h.sessions[sessionID]
	h.mu.RUnlock()
	return s, ok
}

// Push sends a push message to a specific session.
func (h *Hub) Push(sessionID string, msg PushMessage) error {
	h.mu.RLock()
	s, ok := h.sessions[sessionID]
	h.mu.RUnlock()

	if !ok {
		return ErrSessionNotFound
	}

	s.EnqueuePush(msg)
	return nil
}

// Broadcast sends a push message to all registered sessions.
func (h *Hub) Broadcast(msg PushMessage) {
	h.mu.RLock()
	sessions := make([]*Session, 0, len(h.sessions))
	for _, s := range h.sessions {
		sessions = append(sessions, s)
	}
	h.mu.RUnlock()

	for _, s := range sessions {
		s.EnqueuePush(msg)
	}
}

// Stop shuts down the background cleanup goroutine.
func (h *Hub) Stop() {
	close(h.stopCleanup)
	<-h.cleanupDone
}

func (h *Hub) cleanupLoop() {
	defer close(h.cleanupDone)

	ticker := time.NewTicker(h.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCleanup:
			return
		case <-ticker.C:
			h.cleanupExpired()
		}
	}
}

func (h *Hub) cleanupExpired() {
	h.mu.Lock()
	expired := make([]*Session, 0)
	for id, s := range h.sessions {
		if s.IsExpired(h.gracePeriod) {
			expired = append(expired, s)
			delete(h.sessions, id)
		}
	}
	h.mu.Unlock()

	for _, s := range expired {
		_ = s.Close()
	}
}
