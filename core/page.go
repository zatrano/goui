package core

// PageMode controls how a registered component is delivered over HTTP.
type PageMode int

const (
	// ModeLive serves an empty mount shell and fills the UI over WebSocket
	// (default). Prefer for admin panels and highly interactive tools.
	ModeLive PageMode = iota
	// ModeSEO renders full HTML on the first GET for crawlers and first paint,
	// then hydrates over WebSocket for interactivity.
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
