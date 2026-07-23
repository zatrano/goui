package template

// Node is a node in the .goui.html abstract syntax tree.
type Node interface {
	Pos() Position
}

// File is the root AST for a single .goui.html source file.
type File struct {
	Path  string
	Nodes []Node
}

// RawTextNode is literal HTML/text.
type RawTextNode struct {
	Position Position
	Text     string
}

func (n *RawTextNode) Pos() Position { return n.Position }

// OutputNode is {{ EXPR }} (Raw=false) or {!! EXPR !!} (Raw=true).
type OutputNode struct {
	Position Position
	Expr     string
	Raw      bool
}

func (n *OutputNode) Pos() Position { return n.Position }

// IfBranch is one condition arm of an IfNode (@if / @elseif).
type IfBranch struct {
	Cond Position
	Expr string
	Body []Node
}

// IfNode is @if / @elseif / @else / @endif.
type IfNode struct {
	Position Position
	Branches []IfBranch // at least one (@if); further entries are @elseif
	Else     []Node     // @else body, or nil
}

func (n *IfNode) Pos() Position { return n.Position }

// UnlessNode is @unless / @endunless.
type UnlessNode struct {
	Position Position
	Expr     string
	Body     []Node
}

func (n *UnlessNode) Pos() Position { return n.Position }

// SwitchCase is one @case arm.
type SwitchCase struct {
	Value string
	Body  []Node
}

// SwitchNode is @switch / @case / @break / @default / @endswitch.
type SwitchNode struct {
	Position Position
	Expr     string
	Cases    []SwitchCase
	Default  []Node // nil if no @default
}

func (n *SwitchNode) Pos() Position { return n.Position }

// ForeachNode is @foreach / @empty / @endforeach.
type ForeachNode struct {
	Position Position
	Expr     string // collection expression, e.g. ".Items"
	KeyVar   string // optional "$key"; empty if omitted
	ValueVar string // required "$item" (leading $ kept)
	Body     []Node
	Empty    []Node // @empty body, or nil
}

func (n *ForeachNode) Pos() Position { return n.Position }

// ExtendsNode is @extends("layout.name").
type ExtendsNode struct {
	Position Position
	Layout   string
}

func (n *ExtendsNode) Pos() Position { return n.Position }

// SectionNode is @section ... @endsection or the short @section("name", "value").
type SectionNode struct {
	Position Position
	Name     string
	Inline   string // short form value; when non-empty, Body must be empty
	Body     []Node
}

func (n *SectionNode) Pos() Position { return n.Position }

// YieldNode is @yield("name") or @yield("name", "default").
type YieldNode struct {
	Position Position
	Name     string
	Default  []Node
}

func (n *YieldNode) Pos() Position { return n.Position }

// IncludeNode is @include / @includeIf.
type IncludeNode struct {
	Position Position
	Target   string
	DataExpr string // optional; empty means "."
	If       bool   // true for @includeIf
}

func (n *IncludeNode) Pos() Position { return n.Position }

// SlotNode is @slot("name") ... @endslot inside a component.
type SlotNode struct {
	Position Position
	Name     string
	Body     []Node
}

func (n *SlotNode) Pos() Position { return n.Position }

// ComponentNode is @component ... @endcomponent.
type ComponentNode struct {
	Position Position
	Target   string
	DataExpr string // optional props expression; empty means "nil"
	Slots    []SlotNode
	Default  []Node // nodes outside @slot blocks (default slot)
}

func (n *ComponentNode) Pos() Position { return n.Position }

// PropsNode is @props(...) metadata.
type PropsNode struct {
	Position Position
	Raw      string
}

func (n *PropsNode) Pos() Position { return n.Position }

// WalkExprs invokes fn for every expression string embedded in the file AST
// (outputs, conditions, ranges, includes, components, etc.).
func WalkExprs(f *File, fn func(expr string)) {
	if f == nil || fn == nil {
		return
	}
	walkNodeExprs(f.Nodes, fn)
}

func walkNodeExprs(nodes []Node, fn func(expr string)) {
	for _, n := range nodes {
		switch x := n.(type) {
		case *OutputNode:
			fn(x.Expr)
		case *IfNode:
			for _, br := range x.Branches {
				fn(br.Expr)
				walkNodeExprs(br.Body, fn)
			}
			walkNodeExprs(x.Else, fn)
		case *UnlessNode:
			fn(x.Expr)
			walkNodeExprs(x.Body, fn)
		case *SwitchNode:
			fn(x.Expr)
			for _, c := range x.Cases {
				fn(c.Value)
				walkNodeExprs(c.Body, fn)
			}
			walkNodeExprs(x.Default, fn)
		case *ForeachNode:
			fn(x.Expr)
			walkNodeExprs(x.Body, fn)
			walkNodeExprs(x.Empty, fn)
		case *SectionNode:
			walkNodeExprs(x.Body, fn)
		case *YieldNode:
			walkNodeExprs(x.Default, fn)
		case *IncludeNode:
			if x.DataExpr != "" {
				fn(x.DataExpr)
			}
		case *ComponentNode:
			if x.DataExpr != "" {
				fn(x.DataExpr)
			}
			for i := range x.Slots {
				walkNodeExprs(x.Slots[i].Body, fn)
			}
			walkNodeExprs(x.Default, fn)
		case *SlotNode:
			walkNodeExprs(x.Body, fn)
		}
	}
}
