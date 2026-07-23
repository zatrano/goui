package core

import (
	"context"

	"github.com/zatrano/goui/i18n"
)

// Component is the core contract for all GoUI view components.
type Component interface {
	Mount(ctx context.Context) error
	Render() (string, error)
	HandleEvent(ctx context.Context, event string, payload map[string]any) error
	Unmount(ctx context.Context) error
}

// BaseComponent provides shared fields and helpers for component implementations.
// It does not implement Component on its own; concrete types embed it and
// implement Render, HandleEvent, and lifecycle methods themselves.
type BaseComponent struct {
	ID       string
	Children map[string]Component
	dirty    bool

	Locale     string
	translator *i18n.Translator
	pusher     func(kind, text string)
}

// MarkDirty marks the component as needing a re-render.
func (b *BaseComponent) MarkDirty() {
	b.dirty = true
}

// IsDirty reports whether the component has unrendered changes.
func (b *BaseComponent) IsDirty() bool {
	return b.dirty
}

// ResetDirty clears the dirty flag after a successful render.
func (b *BaseComponent) ResetDirty() {
	b.dirty = false
}

// SetTranslator injects the shared application translator instance.
func (b *BaseComponent) SetTranslator(t *i18n.Translator) {
	b.translator = t
}

// SetPusher injects a toast/push callback (usually bound to the WS session).
func (b *BaseComponent) SetPusher(fn func(kind, text string)) {
	b.pusher = fn
}

// Toast sends a push notification to the current session (no-op if no pusher).
func (b *BaseComponent) Toast(kind, text string) {
	if b.pusher != nil {
		b.pusher(kind, text)
	}
}

// ToastT translates key then sends a toast for the current session.
func (b *BaseComponent) ToastT(kind, key string, args ...any) {
	b.Toast(kind, b.T(key, args...))
}

// T translates a key for the component locale using the injected translator.
func (b *BaseComponent) T(key string, args ...any) string {
	if b.translator == nil {
		return "[[" + key + "]]"
	}

	locale := b.Locale
	if locale == "" {
		locale = i18n.BaseLocale
	}

	return b.translator.Translate(locale, key, args...)
}
