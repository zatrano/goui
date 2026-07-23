package ws

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/diff"
	"github.com/zatrano/goui/i18n"
)

const outboundBufferSize = 32

// Session owns component instances for a single WebSocket connection lifecycle.
type Session struct {
	ID            string
	Locale        string
	conn          Conn
	translator    *i18n.Translator
	registry      *core.Registry
	components    map[string]core.Component
	prefetched    map[string]core.Component // registry name -> mounted but not visible
	prefetchOrder []string                  // oldest first (simple LRU insertion order)
	renderTrees   map[string]*diff.Node
	outbound      chan Frame
	mu            sync.RWMutex

	disconnectedAt time.Time
	outboundClosed bool
	runWG          sync.WaitGroup
}

// NewSession creates a session bound to a WebSocket connection.
func NewSession(conn Conn, translator *i18n.Translator, locale string) *Session {
	if locale == "" {
		locale = i18n.BaseLocale
	}

	return &Session{
		ID:          newSessionID(),
		Locale:      locale,
		conn:        conn,
		translator:  translator,
		components:  make(map[string]core.Component),
		prefetched:  make(map[string]core.Component),
		renderTrees: make(map[string]*diff.Node),
		outbound:    make(chan Frame, outboundBufferSize),
	}
}

// newSessionWithConn is used by tests with an in-memory connection.
func newSessionWithConn(conn Conn, translator *i18n.Translator, locale string) *Session {
	return NewSession(conn, translator, locale)
}

// SetRegistry attaches the component registry used for prefetch/activate.
func (s *Session) SetRegistry(registry *core.Registry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registry = registry
}

// MountComponent configures, mounts, and stores a component instance.
func (s *Session) MountComponent(id string, c core.Component) error {
	if err := s.prepareComponent(c, id); err != nil {
		return err
	}

	s.mu.Lock()
	s.components[id] = c
	s.mu.Unlock()

	return nil
}

// Prefetch creates and mounts a component by registry name without rendering.
// Duplicate names are a no-op. Evicts the oldest prefetch when MaxPrefetch is exceeded.
func (s *Session) Prefetch(name string) error {
	if name == "" {
		return nil
	}

	s.mu.RLock()
	_, exists := s.prefetched[name]
	registry := s.registry
	s.mu.RUnlock()
	if exists {
		return nil
	}
	if registry == nil {
		return core.ErrComponentNotRegistered
	}

	c, err := registry.Create(name)
	if err != nil {
		return err
	}

	if err := s.prepareComponent(c, "prefetch:"+name); err != nil {
		return err
	}

	var evicted []core.Component
	s.mu.Lock()
	if _, exists := s.prefetched[name]; exists {
		s.mu.Unlock()
		_ = c.Unmount(context.Background())
		return nil
	}
	for len(s.prefetchOrder) >= MaxPrefetch {
		oldest := s.prefetchOrder[0]
		s.prefetchOrder = s.prefetchOrder[1:]
		if old, ok := s.prefetched[oldest]; ok {
			delete(s.prefetched, oldest)
			evicted = append(evicted, old)
		}
	}
	s.prefetched[name] = c
	s.prefetchOrder = append(s.prefetchOrder, name)
	s.mu.Unlock()

	for _, old := range evicted {
		_ = old.Unmount(context.Background())
	}

	log.Printf("[goui] prefetch mounted %q (session %s)", name, s.ID)
	return nil
}

// Activate promotes a prefetched component into the active set and sends its first render.
// If the name was not prefetched, a fresh instance is created and mounted.
func (s *Session) Activate(name string) (string, error) {
	if name == "" {
		return "", core.ErrComponentNotRegistered
	}

	s.mu.Lock()
	c, fromPrefetch := s.prefetched[name]
	if fromPrefetch {
		delete(s.prefetched, name)
		s.prefetchOrder = removePrefetchOrder(s.prefetchOrder, name)
	}
	registry := s.registry
	s.mu.Unlock()

	id := newSessionID()

	if fromPrefetch {
		applySessionContext(c, id, s.Locale, s.translator)
		s.injectPusher(c)
	} else {
		if registry == nil {
			return "", core.ErrComponentNotRegistered
		}
		var err error
		c, err = registry.Create(name)
		if err != nil {
			return "", err
		}
		if err := s.prepareComponent(c, id); err != nil {
			return "", err
		}
	}

	s.mu.Lock()
	s.components[id] = c
	s.mu.Unlock()

	s.sendFullRender(id, c)
	return id, nil
}

func (s *Session) prepareComponent(c core.Component, id string) error {
	applySessionContext(c, id, s.Locale, s.translator)
	s.injectPusher(c)
	return c.Mount(context.Background())
}

func (s *Session) injectPusher(c core.Component) {
	if setter, ok := c.(interface{ SetPusher(func(kind, text string)) }); ok {
		setter.SetPusher(func(kind, text string) {
			s.EnqueuePush(PushMessage{Kind: kind, Text: text})
		})
	}
}

func removePrefetchOrder(order []string, name string) []string {
	out := order[:0]
	for _, n := range order {
		if n != name {
			out = append(out, n)
		}
	}
	return out
}

// Run starts read and write loops until the context is cancelled or the connection closes.
func (s *Session) Run(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	readDone := make(chan struct{})
	writeDone := make(chan struct{})

	s.runWG.Add(2)

	go func() {
		defer s.runWG.Done()
		defer close(readDone)
		s.readLoop(runCtx)
	}()

	go func() {
		defer s.runWG.Done()
		defer close(writeDone)
		s.writeLoop(runCtx)
	}()

	select {
	case <-ctx.Done():
	case <-readDone:
	case <-writeDone:
	}

	cancel()
	s.markDisconnected()
	s.closeConn()
	s.runWG.Wait()
}

// Reattach binds a new WebSocket connection after a disconnect within the grace period.
func (s *Session) Reattach(conn Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		return ErrSessionAlreadyActive
	}

	s.conn = conn
	s.disconnectedAt = time.Time{}
	return nil
}

// SendSessionFrame notifies the client of the session identifier.
func (s *Session) SendSessionFrame() {
	s.enqueue(newSessionFrame(s.ID))
}

// SendInitialRenders enqueues render frames for all mounted components.
func (s *Session) SendInitialRenders() {
	s.mu.RLock()
	ids := make([]string, 0, len(s.components))
	components := make(map[string]core.Component, len(s.components))
	for id, c := range s.components {
		ids = append(ids, id)
		components[id] = c
	}
	s.mu.RUnlock()

	for _, id := range ids {
		s.sendFullRender(id, components[id])
	}
}

// Close unmounts all components (active and prefetched) and closes the outbound channel.
func (s *Session) Close() error {
	s.mu.Lock()
	components := make([]core.Component, 0, len(s.components)+len(s.prefetched))
	for _, c := range s.components {
		components = append(components, c)
	}
	for _, c := range s.prefetched {
		components = append(components, c)
	}
	s.components = make(map[string]core.Component)
	s.prefetched = make(map[string]core.Component)
	s.prefetchOrder = nil
	s.renderTrees = make(map[string]*diff.Node)
	s.mu.Unlock()

	ctx := context.Background()
	for _, c := range components {
		_ = c.Unmount(ctx)
	}

	s.closeConn()

	s.mu.Lock()
	if !s.outboundClosed {
		close(s.outbound)
		s.outboundClosed = true
	}
	s.mu.Unlock()

	return nil
}

// IsDisconnected reports whether the session has no active connection.
func (s *Session) IsDisconnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conn == nil
}

// DisconnectedAt returns the time the session was marked disconnected.
func (s *Session) DisconnectedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.disconnectedAt
}

// IsExpired reports whether the grace period has elapsed since disconnect.
func (s *Session) IsExpired(grace time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.disconnectedAt.IsZero() {
		return false
	}
	return time.Since(s.disconnectedAt) > grace
}

// EnqueuePush adds a push frame to the outbound queue.
func (s *Session) EnqueuePush(msg PushMessage) {
	s.enqueue(newPushFrame(msg))
}

func (s *Session) readLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		conn := s.activeConn()
		if conn == nil {
			return
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			if !errors.Is(err, io.EOF) && ctx.Err() == nil {
				s.enqueue(newErrorFrame(err.Error()))
			}
			return
		}

		var frame Frame
		if err := json.Unmarshal(data, &frame); err != nil {
			s.enqueue(newErrorFrame("invalid frame"))
			continue
		}

		switch frame.Type {
		case FrameTypeEvent:
			s.handleEventFrame(ctx, frame)
		case FrameTypePrefetch:
			if err := s.Prefetch(frame.Component); err != nil {
				log.Printf("[goui] prefetch %q failed: %v", frame.Component, err)
			}
		case FrameTypeActivate:
			if _, err := s.Activate(frame.Component); err != nil {
				s.enqueue(newErrorFrame(err.Error()))
			}
		}
	}
}

func (s *Session) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-s.outbound:
			if !ok {
				return
			}

			conn := s.activeConn()
			if conn == nil {
				continue
			}

			data, err := json.Marshal(frame)
			if err != nil {
				continue
			}

			if err := conn.WriteMessage(TextMessage, data); err != nil {
				return
			}
		}
	}
}

func (s *Session) handleEventFrame(ctx context.Context, frame Frame) {
	s.mu.RLock()
	component, ok := s.components[frame.Component]
	s.mu.RUnlock()

	if !ok {
		s.enqueue(newErrorFrame("unknown component"))
		return
	}

	payload, err := decodeEventPayload(frame.Payload)
	if err != nil {
		s.enqueue(newErrorFrame("invalid event payload"))
		return
	}

	if err := component.HandleEvent(ctx, frame.Event, payload); err != nil {
		if errors.Is(err, core.ErrSkipRender) {
			return
		}
		s.enqueue(newErrorFrame(err.Error()))
		return
	}

	s.sendRender(frame.Component, component)
}

func (s *Session) sendRender(componentID string, component core.Component) {
	html, err := component.Render()
	if err != nil {
		s.enqueue(newErrorFrame(err.Error()))
		return
	}

	html, err = decorateComponentHTML(html, componentID)
	if err != nil {
		s.enqueue(newErrorFrame(err.Error()))
		return
	}

	newTree, err := parseComponentTree(html)
	if err != nil {
		s.enqueue(newErrorFrame(err.Error()))
		return
	}

	s.mu.Lock()
	oldTree, hasOldTree := s.renderTrees[componentID]
	s.renderTrees[componentID] = newTree
	s.mu.Unlock()

	var patches []diff.Patch
	if !hasOldTree {
		patches = []diff.Patch{{
			Op:   diff.OpReplace,
			Path: []int{},
			HTML: diff.Serialize(newTree),
			Tag:  newTree.Tag,
		}}
	} else {
		patches = diff.Diff(oldTree, newTree)
	}

	payload, err := json.Marshal(patches)
	if err != nil {
		s.enqueue(newErrorFrame(err.Error()))
		return
	}

	s.enqueue(Frame{
		Type:      FrameTypeRender,
		Component: componentID,
		Payload:   payload,
	})
}

func (s *Session) sendFullRender(componentID string, component core.Component) {
	html, err := component.Render()
	if err != nil {
		s.enqueue(newErrorFrame(err.Error()))
		return
	}

	html, err = decorateComponentHTML(html, componentID)
	if err != nil {
		s.enqueue(newErrorFrame(err.Error()))
		return
	}

	tree, err := parseComponentTree(html)
	if err != nil {
		s.enqueue(newErrorFrame(err.Error()))
		return
	}

	s.mu.Lock()
	s.renderTrees[componentID] = tree
	s.mu.Unlock()

	payload, err := json.Marshal([]diff.Patch{{
		Op:   diff.OpReplace,
		Path: []int{},
		HTML: diff.Serialize(tree),
		Tag:  tree.Tag,
	}})
	if err != nil {
		s.enqueue(newErrorFrame(err.Error()))
		return
	}

	s.enqueue(Frame{
		Type:      FrameTypeRender,
		Component: componentID,
		Payload:   payload,
	})
}

func (s *Session) enqueue(frame Frame) {
	s.mu.RLock()
	if s.outboundClosed {
		s.mu.RUnlock()
		return
	}
	ch := s.outbound
	s.mu.RUnlock()

	select {
	case ch <- frame:
	default:
	}
}

func (s *Session) activeConn() Conn {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conn
}

func (s *Session) markDisconnected() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.disconnectedAt.IsZero() {
		s.disconnectedAt = time.Now()
	}
}

func (s *Session) closeConn() {
	s.mu.Lock()
	conn := s.conn
	s.conn = nil
	s.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
}

func applySessionContext(c core.Component, id, locale string, translator *i18n.Translator) {
	if setter, ok := c.(interface{ SetTranslator(*i18n.Translator) }); ok {
		setter.SetTranslator(translator)
	}

	rv := reflect.ValueOf(c)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	rv = rv.Elem()

	base := rv.FieldByName("BaseComponent")
	if !base.IsValid() {
		return
	}

	idField := base.FieldByName("ID")
	if idField.IsValid() && idField.CanSet() {
		idField.SetString(id)
	}

	localeField := base.FieldByName("Locale")
	if localeField.IsValid() && localeField.CanSet() {
		localeField.SetString(locale)
	}
}

func newSessionID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func decorateComponentHTML(html, componentID string) (string, error) {
	tree, err := diff.ParseHTML(html)
	if err != nil {
		return "", err
	}

	if len(tree.Children) == 0 {
		return `<div data-goui-component="` + componentID + `"></div>`, nil
	}

	if len(tree.Children) == 1 {
		child := tree.Children[0]
		if child.Attrs == nil {
			child.Attrs = make(map[string]string)
		}
		child.Attrs["data-goui-component"] = componentID
		return diff.Serialize(tree), nil
	}

	wrapper := &diff.Node{
		Tag:      "div",
		Attrs:    map[string]string{"data-goui-component": componentID},
		Children: tree.Children,
	}
	tree.Children = []*diff.Node{wrapper}
	return diff.Serialize(tree), nil
}

// parseComponentTree returns the component root element as the tree root so
// patch paths are relative to the DOM element that carries data-goui-component.
func parseComponentTree(html string) (*diff.Node, error) {
	tree, err := diff.ParseHTML(html)
	if err != nil {
		return nil, err
	}
	if len(tree.Children) == 1 {
		return tree.Children[0], nil
	}
	return tree, nil
}
