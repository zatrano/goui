package core

// ViewComponent is implemented in addition to Component by types that render
// from a .goui.html view instead of a hand-written Render() body.
//
// Pair with template.Wrap so Render() is dispatched through template.Registry.
// The core package intentionally does not import the template package.
type ViewComponent interface {
	Component
	// View returns the template Registry dot-path, e.g. "pages.counter".
	View() string
}
