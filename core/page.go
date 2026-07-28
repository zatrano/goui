package core

// PageMode controls how a registered component is delivered over HTTP.
type PageMode int

const (
	// ModeLive is the default interactive mode: with the page renderer the
	// first GET includes SSR HTML (LiveView-style dead render), then the
	// WebSocket hydrates/adopts for events. Opt into DeferFirstRender only
	// when you intentionally want an empty shell.
	ModeLive PageMode = iota
	// ModeSEO is interactive like ModeLive, plus document Head metadata for
	// crawlers and social previews. First paint is also SSR over HTTP.
	ModeSEO
	// ModeStatic renders full HTML only — no WebSocket client is embedded.
	ModeStatic
)

// String returns a stable name for logs and docs.
func (m PageMode) String() string {
	switch m {
	case ModeLive:
		return "live"
	case ModeSEO:
		return "seo"
	case ModeStatic:
		return "static"
	default:
		return "live"
	}
}

// Head holds document metadata for ModeSEO / ModeStatic pages.
// Components may implement HeadProvider; otherwise Title falls back to the
// component registry name.
type Head struct {
	Title         string
	Description   string
	Canonical     string
	Lang          string // html lang; empty → locale or "tr"
	Robots        string
	OGTitle       string
	OGDescription string
	OGImage       string
	OGType        string // default "website" when any OG field is set
	Extra         []Meta
}

// Meta is an extra <meta> tag. Set Name and/or Property with Content.
type Meta struct {
	Name     string
	Property string
	Content  string
}

// HeadProvider is an optional Component capability for SEO document metadata.
type HeadProvider interface {
	Head() Head
}
