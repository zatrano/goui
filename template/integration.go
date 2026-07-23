package template

import (
	"fmt"

	"github.com/zatrano/goui/core"
)

// RenderComponent renders c through the template registry when c implements
// core.ViewComponent. If not, ok is false and the caller should use c.Render().
func (r *Registry) RenderComponent(c core.Component) (html string, ok bool, err error) {
	vc, isView := c.(core.ViewComponent)
	if !isView {
		return "", false, nil
	}
	html, err = r.Render(vc.View(), c)
	return html, true, err
}

// componentWrapper routes Render() to the template Registry for ViewComponents.
type componentWrapper struct {
	core.Component
	registry *Registry
}

func (w *componentWrapper) Render() (string, error) {
	if html, ok, err := w.registry.RenderComponent(w.Component); ok {
		if err == nil {
			resetDirty(w.Component)
		}
		return html, err
	}
	return w.Component.Render()
}

// Wrap wraps a component factory so ViewComponent instances render via reg.
// Use with core.Registry.Register:
//
//	coreReg.Register("counter", template.Wrap(tmplReg, func() core.Component {
//	    return &Counter{}
//	}))
func Wrap(reg *Registry, factory func() core.Component) func() core.Component {
	if reg == nil {
		panic("template.Wrap: registry is nil")
	}
	if factory == nil {
		panic("template.Wrap: factory is nil")
	}
	return func() core.Component {
		return &componentWrapper{Component: factory(), registry: reg}
	}
}

func resetDirty(c core.Component) {
	type dirtyResetter interface {
		ResetDirty()
	}
	if d, ok := c.(dirtyResetter); ok {
		d.ResetDirty()
	}
}

// ErrViewRenderDirect is returned by stub Render methods when Wrap was not used.
var ErrViewRenderDirect = fmt.Errorf("ViewComponent.Render called directly; register the factory with template.Wrap")
