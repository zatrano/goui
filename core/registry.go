package core

import "sync"

type entry struct {
	factory func() Component
	mode    PageMode
}

// Registry maps component type names to factory functions and page modes.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]entry
}

// NewRegistry creates an empty component registry.
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]entry),
	}
}

// Register associates a component type name with its factory in ModeLive.
// Registering the same name twice returns ErrComponentAlreadyRegistered.
func (r *Registry) Register(name string, factory func() Component) error {
	return r.RegisterPage(name, factory, ModeLive)
}

// RegisterPage associates a component with a delivery mode (Live / SEO / Static).
func (r *Registry) RegisterPage(name string, factory func() Component, mode PageMode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[name]; exists {
		return ErrComponentAlreadyRegistered
	}

	r.entries[name] = entry{factory: factory, mode: mode}
	return nil
}

// Create instantiates a component by its registered type name.
func (r *Registry) Create(name string) (Component, error) {
	r.mu.RLock()
	e, ok := r.entries[name]
	r.mu.RUnlock()

	if !ok {
		return nil, ErrComponentNotRegistered
	}

	return e.factory(), nil
}

// Mode returns the page delivery mode for a registered component.
func (r *Registry) Mode(name string) (PageMode, bool) {
	r.mu.RLock()
	e, ok := r.entries[name]
	r.mu.RUnlock()
	if !ok {
		return ModeLive, false
	}
	return e.mode, true
}
