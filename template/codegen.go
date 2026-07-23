package template

import (
	"fmt"
	"path/filepath"
	"strings"
)

// IncludeIfMarkerPrefix is the start of a compile-time @includeIf placeholder.
// Registry replaces `/*__includeif:target__*/` with a real
// {{template}} call or an empty string.
const (
	IncludeIfMarkerPrefix = "/*__includeif:"
	IncludeIfMarkerSuffix = "*/"
)

// CompileUnit is the codegen output for one .goui.html file.
type CompileUnit struct {
	// Name is the file's dot-path name, e.g. "pages.home".
	Name string
	// Extends is the @extends target dot-path, or "".
	Extends string
	// Sections maps section name → full {{define}}...{{end}} source.
	// Populated when the file uses @extends; Body is then empty.
	Sections map[string]string
	// Body is the standalone Go template source when the file does not @extends.
	Body string
	// Includes lists @include targets (unconditional).
	Includes []string
	// ConditionalIncludes lists @includeIf targets (resolved by the registry).
	ConditionalIncludes []string
	// Components lists @component targets.
	Components []string
	// SlotDefines maps unique slot template names → body source (no {{define}} wrapper).
	// Parsed into the shared template set by the registry.
	SlotDefines map[string]string
	// Props is the raw @props declaration text, if any.
	Props string
}

// Generate converts a File AST into a CompileUnit of native html/template source.
func Generate(f *File) (*CompileUnit, error) {
	if f == nil {
		return nil, fmt.Errorf("Generate: nil File")
	}
	g := &generator{
		unit: &CompileUnit{
			Name:        deriveName(f.Path),
			Sections:    make(map[string]string),
			SlotDefines: make(map[string]string),
		},
	}

	for _, n := range f.Nodes {
		switch x := n.(type) {
		case *ExtendsNode:
			g.unit.Extends = x.Layout
		case *PropsNode:
			g.unit.Props = x.Raw
		}
	}

	if g.unit.Extends != "" {
		for _, n := range f.Nodes {
			switch x := n.(type) {
			case *SectionNode:
				g.unit.Sections[x.Name] = g.genSectionDefine(x)
			case *ExtendsNode, *PropsNode:
				// metadata only
			case *RawTextNode:
				if !isWhitespaceOnly(x.Text) {
					return nil, errorAt(x.Pos(), "internal error: non-whitespace text with @extends")
				}
			default:
				return nil, errorAt(n.Pos(), "internal error: unexpected node with @extends")
			}
		}
		// Body stays empty for @extends files.
		return g.unit, nil
	}

	var b strings.Builder
	for _, n := range f.Nodes {
		b.WriteString(g.genNode(n))
	}
	g.unit.Body = b.String()
	return g.unit, nil
}

type generator struct {
	unit        *CompileUnit
	slotCounter int
}

func (g *generator) genNodes(nodes []Node) string {
	var b strings.Builder
	for _, n := range nodes {
		b.WriteString(g.genNode(n))
	}
	return b.String()
}

func (g *generator) genNode(n Node) string {
	switch x := n.(type) {
	case *RawTextNode:
		return g.genRawText(x)
	case *OutputNode:
		return g.genOutput(x)
	case *IfNode:
		return g.genIf(x)
	case *UnlessNode:
		return g.genUnless(x)
	case *SwitchNode:
		return g.genSwitch(x)
	case *ForeachNode:
		return g.genForeach(x)
	case *YieldNode:
		return g.genYield(x)
	case *IncludeNode:
		return g.genInclude(x)
	case *ComponentNode:
		return g.genComponent(x)
	case *ExtendsNode, *PropsNode, *SectionNode:
		// Handled at file level / not emitted into Body.
		return ""
	case *SlotNode:
		// Slots only appear inside ComponentNode; body already extracted by parser.
		return g.genNodes(x.Body)
	default:
		return ""
	}
}

func (g *generator) genRawText(n *RawTextNode) string {
	return n.Text
}

func (g *generator) genOutput(n *OutputNode) string {
	if n.Raw {
		return "{{ raw (" + n.Expr + ") }}"
	}
	return "{{ " + n.Expr + " }}"
}

func (g *generator) genIf(n *IfNode) string {
	var b strings.Builder
	for i, br := range n.Branches {
		if i == 0 {
			b.WriteString("{{if " + br.Expr + "}}")
		} else {
			b.WriteString("{{else if " + br.Expr + "}}")
		}
		b.WriteString(g.genNodes(br.Body))
	}
	if n.Else != nil {
		b.WriteString("{{else}}")
		b.WriteString(g.genNodes(n.Else))
	}
	b.WriteString("{{end}}")
	return b.String()
}

func (g *generator) genUnless(n *UnlessNode) string {
	return "{{if not (" + n.Expr + ")}}" + g.genNodes(n.Body) + "{{end}}"
}

func (g *generator) genSwitch(n *SwitchNode) string {
	if len(n.Cases) == 0 {
		return g.genNodes(n.Default)
	}
	var b strings.Builder
	for i, c := range n.Cases {
		if i == 0 {
			b.WriteString("{{if eq " + n.Expr + " " + c.Value + "}}")
		} else {
			b.WriteString("{{else if eq " + n.Expr + " " + c.Value + "}}")
		}
		b.WriteString(g.genNodes(c.Body))
	}
	if n.Default != nil {
		b.WriteString("{{else}}")
		b.WriteString(g.genNodes(n.Default))
	}
	b.WriteString("{{end}}")
	return b.String()
}

func (g *generator) genForeach(n *ForeachNode) string {
	var b strings.Builder
	if n.KeyVar != "" {
		b.WriteString("{{range " + n.KeyVar + ", " + n.ValueVar + " := " + n.Expr + "}}")
	} else {
		b.WriteString("{{range " + n.ValueVar + " := " + n.Expr + "}}")
	}
	b.WriteString(g.genNodes(n.Body))
	if n.Empty != nil {
		b.WriteString("{{else}}")
		b.WriteString(g.genNodes(n.Empty))
	}
	b.WriteString("{{end}}")
	return b.String()
}

func (g *generator) genSectionDefine(n *SectionNode) string {
	var body string
	if n.Inline != "" {
		body = n.Inline
	} else {
		body = g.genNodes(n.Body)
	}
	return "{{define \"" + n.Name + "\"}}" + body + "{{end}}"
}

func (g *generator) genYield(n *YieldNode) string {
	return "{{block \"" + n.Name + "\" .}}" + g.genNodes(n.Default) + "{{end}}"
}

func (g *generator) genInclude(n *IncludeNode) string {
	data := n.DataExpr
	if data == "" {
		data = "."
	} else {
		data = wrapExpr(data)
	}
	if n.If {
		g.unit.ConditionalIncludes = appendUnique(g.unit.ConditionalIncludes, n.Target)
		// Placeholder resolved by the registry after existence check.
		return IncludeIfMarkerPrefix + n.Target + IncludeIfMarkerSuffix
	}
	g.unit.Includes = appendUnique(g.unit.Includes, n.Target)
	return "{{template \"" + n.Target + "\" " + data + "}}"
}

func (g *generator) genComponent(n *ComponentNode) string {
	g.unit.Components = appendUnique(g.unit.Components, n.Target)
	g.slotCounter++
	counter := g.slotCounter

	data := n.DataExpr
	if data == "" {
		data = "(dict)"
	} else {
		data = wrapExpr(data)
	}

	var b strings.Builder
	b.WriteString(`{{ component "`)
	b.WriteString(n.Target)
	b.WriteString(`" `)
	b.WriteString(data)
	b.WriteString(` .`) // callerDot: slots render in the caller's context

	for _, slot := range n.Slots {
		defName := fmt.Sprintf("__slot__%s__%d__%s", n.Target, counter, slot.Name)
		g.unit.SlotDefines[defName] = g.genNodes(slot.Body)
		b.WriteString(` "`)
		b.WriteString(slot.Name)
		b.WriteString(`" "`)
		b.WriteString(defName)
		b.WriteString(`"`)
	}

	if hasNonWhitespaceNodes(n.Default) {
		defName := fmt.Sprintf("__slot__%s__%d____default__", n.Target, counter)
		g.unit.SlotDefines[defName] = g.genNodes(n.Default)
		b.WriteString(` "" "`)
		b.WriteString(defName)
		b.WriteString(`"`)
	}

	b.WriteString(` }}`)
	return b.String()
}

func hasNonWhitespaceNodes(nodes []Node) bool {
	for _, n := range nodes {
		if rt, ok := n.(*RawTextNode); ok && isWhitespaceOnly(rt.Text) {
			continue
		}
		return true
	}
	return false
}

func deriveName(path string) string {
	if path == "" {
		return ""
	}
	p := filepath.ToSlash(path)
	p = strings.TrimSuffix(p, ".goui.html")
	p = strings.TrimSuffix(p, ".html")
	p = strings.ReplaceAll(p, "/", ".")
	return p
}

// wrapExpr parenthesizes non-trivial expressions so they form a single
// template argument (e.g. dict "a" 1 → (dict "a" 1)).
func wrapExpr(expr string) string {
	expr = strings.TrimSpace(expr)
	if expr == "" || expr == "." {
		return expr
	}
	if strings.HasPrefix(expr, "(") {
		return expr
	}
	if expr[0] == '.' || expr[0] == '$' {
		return expr
	}
	return "(" + expr + ")"
}

func appendUnique(slice []string, s string) []string {
	for _, x := range slice {
		if x == s {
			return slice
		}
	}
	return append(slice, s)
}

// FormatIncludeIfMarker builds the @includeIf placeholder for a target.
func FormatIncludeIfMarker(target string) string {
	return IncludeIfMarkerPrefix + target + IncludeIfMarkerSuffix
}
