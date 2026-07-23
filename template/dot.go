package template

import htmltemplate "html/template"

// Dot wraps the data passed into a @component target template.
//
// The component's .goui.html receives Dot as "." (see componentFn).
// Authors read props via .Props.* and rendered slot HTML via .Slots / .DefaultSlot.
type Dot struct {
	// Props is the data passed as the component's second argument (often a
	// dict(...) map, but any type is allowed).
	Props any
	// Slots holds named slot contents that were pre-rendered to HTML.
	Slots map[string]htmltemplate.HTML
	// DefaultSlot is the pre-rendered HTML of content outside @slot blocks.
	DefaultSlot htmltemplate.HTML
}
