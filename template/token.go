package template

// TokenType identifies the kind of a lexical token.
type TokenType int

const (
	TokenEOF           TokenType = iota
	TokenText                    // raw HTML/text outside directives and {{ }}
	TokenOutputEscaped           // {{ EXPR }} — Value is trimmed EXPR
	TokenOutputRaw               // {!! EXPR !!} — Value is trimmed EXPR
	TokenComment                 // {{-- ... --}} — Value may be empty; never emitted to output
	TokenDirective               // @name(...) or @name (no parentheses)
)

// String returns a human-readable name for the token type.
func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenText:
		return "Text"
	case TokenOutputEscaped:
		return "OutputEscaped"
	case TokenOutputRaw:
		return "OutputRaw"
	case TokenComment:
		return "Comment"
	case TokenDirective:
		return "Directive"
	default:
		return "Unknown"
	}
}

// Position holds a location inside a source file for error reporting.
type Position struct {
	File   string
	Line   int
	Column int
	Offset int
}

// Token is a single lexical unit produced by the lexer.
type Token struct {
	Type TokenType
	// Value: TokenText → raw text; TokenOutputEscaped/Raw → EXPR;
	// TokenDirective → directive name (e.g. "if", "foreach", "endif")
	Value string
	// Args: TokenDirective only — raw argument string inside parentheses
	// (parentheses excluded). Empty when there are no parentheses.
	// Example: @if(.User.IsAdmin) → Args = ".User.IsAdmin"
	Args string
	Pos  Position
}
