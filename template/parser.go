package template

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var foreachVarRe = regexp.MustCompile(`^\$[A-Za-z_][A-Za-z0-9_]*$`)

// stop set helpers for block parsing
type stopSet map[string]bool

// Parser turns a token stream into an AST File.
type Parser struct {
	tokens   []Token
	filename string
	pos      int
}

// NewParser creates a parser over tokens produced by the lexer.
func NewParser(tokens []Token, filename string) *Parser {
	return &Parser{tokens: tokens, filename: filename}
}

// Parse builds the AST for the token stream.
func (p *Parser) Parse() (*File, error) {
	nodes, err := p.parseBlock(nil)
	if err != nil {
		return nil, err
	}
	if err := validateTopLevel(nodes); err != nil {
		return nil, err
	}
	return &File{Path: p.filename, Nodes: nodes}, nil
}

// ParseSource lexes then parses a .goui.html source in one step.
func ParseSource(filename string, src []byte) (*File, error) {
	toks, err := NewLexer(filename, src).Tokenize()
	if err != nil {
		return nil, err
	}
	return NewParser(toks, filename).Parse()
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF, Pos: Position{File: p.filename}}
	}
	return p.tokens[p.pos]
}

func (p *Parser) next() Token {
	tok := p.peek()
	if tok.Type != TokenEOF {
		p.pos++
	}
	return tok
}

func (p *Parser) atEOF() bool {
	return p.peek().Type == TokenEOF
}

// parseBlock collects nodes until EOF or a directive whose name is in stop
// (stop directives are NOT consumed). When stop is nil, only EOF ends the block.
func (p *Parser) parseBlock(stop stopSet) ([]Node, error) {
	var nodes []Node
	for !p.atEOF() {
		tok := p.peek()
		if tok.Type == TokenDirective && stop != nil && stop[tok.Value] {
			return nodes, nil
		}
		n, err := p.parseNode(stop)
		if err != nil {
			return nil, err
		}
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func (p *Parser) parseNode(stop stopSet) (Node, error) {
	tok := p.peek()
	switch tok.Type {
	case TokenEOF:
		return nil, nil
	case TokenComment:
		p.next()
		return nil, nil // comments never enter the AST
	case TokenText:
		p.next()
		return &RawTextNode{Position: tok.Pos, Text: tok.Value}, nil
	case TokenOutputEscaped:
		p.next()
		return &OutputNode{Position: tok.Pos, Expr: tok.Value, Raw: false}, nil
	case TokenOutputRaw:
		p.next()
		return &OutputNode{Position: tok.Pos, Expr: tok.Value, Raw: true}, nil
	case TokenDirective:
		if stop != nil && stop[tok.Value] {
			return nil, nil
		}
		return p.parseDirective(tok)
	default:
		return nil, errorAtf(tok.Pos, "unexpected token %s", tok.Type)
	}
}

func (p *Parser) parseDirective(tok Token) (Node, error) {
	switch tok.Value {
	case "if":
		return p.parseIf(tok)
	case "unless":
		return p.parseUnless(tok)
	case "switch":
		return p.parseSwitch(tok)
	case "foreach":
		return p.parseForeach(tok)
	case "extends":
		return p.parseExtends(tok)
	case "section":
		return p.parseSection(tok)
	case "yield":
		return p.parseYield(tok)
	case "include":
		return p.parseInclude(tok, false)
	case "includeIf":
		return p.parseInclude(tok, true)
	case "component":
		return p.parseComponent(tok)
	case "props":
		return p.parseProps(tok)
	case "slot":
		return nil, errorAt(tok.Pos, "unexpected @slot outside @component")
	case "elseif", "else", "endif",
		"endunless",
		"case", "break", "default", "endswitch",
		"empty", "endforeach",
		"endsection",
		"endcomponent", "endslot":
		return nil, errorAtf(tok.Pos, "unexpected @%s without matching opening directive", tok.Value)
	default:
		return nil, errorAtf(tok.Pos, "unknown directive @%s", tok.Value)
	}
}

func (p *Parser) parseIf(tok Token) (*IfNode, error) {
	p.next() // consume @if
	if strings.TrimSpace(tok.Args) == "" {
		return nil, errorAt(tok.Pos, "@if requires a condition")
	}
	node := &IfNode{Position: tok.Pos}

	body, err := p.parseBlock(stopSet{"elseif": true, "else": true, "endif": true})
	if err != nil {
		return nil, err
	}
	node.Branches = append(node.Branches, IfBranch{Cond: tok.Pos, Expr: tok.Args, Body: body})

	for {
		next := p.peek()
		if next.Type != TokenDirective {
			return nil, errorAt(tok.Pos, "unclosed @if starting here")
		}
		switch next.Value {
		case "elseif":
			p.next()
			if strings.TrimSpace(next.Args) == "" {
				return nil, errorAt(next.Pos, "@elseif requires a condition")
			}
			body, err := p.parseBlock(stopSet{"elseif": true, "else": true, "endif": true})
			if err != nil {
				return nil, err
			}
			node.Branches = append(node.Branches, IfBranch{Cond: next.Pos, Expr: next.Args, Body: body})
		case "else":
			p.next()
			body, err := p.parseBlock(stopSet{"endif": true})
			if err != nil {
				return nil, err
			}
			node.Else = body
			end := p.peek()
			if end.Type != TokenDirective || end.Value != "endif" {
				return nil, errorAt(tok.Pos, "unclosed @if starting here")
			}
			p.next()
			return node, nil
		case "endif":
			p.next()
			return node, nil
		default:
			return nil, errorAt(tok.Pos, "unclosed @if starting here")
		}
	}
}

func (p *Parser) parseUnless(tok Token) (*UnlessNode, error) {
	p.next()
	if strings.TrimSpace(tok.Args) == "" {
		return nil, errorAt(tok.Pos, "@unless requires a condition")
	}
	body, err := p.parseBlock(stopSet{"endunless": true})
	if err != nil {
		return nil, err
	}
	end := p.peek()
	if end.Type != TokenDirective || end.Value != "endunless" {
		return nil, errorAt(tok.Pos, "unclosed @unless starting here")
	}
	p.next()
	return &UnlessNode{Position: tok.Pos, Expr: tok.Args, Body: body}, nil
}

func (p *Parser) parseSwitch(tok Token) (*SwitchNode, error) {
	p.next()
	if strings.TrimSpace(tok.Args) == "" {
		return nil, errorAt(tok.Pos, "@switch requires an expression")
	}
	node := &SwitchNode{Position: tok.Pos, Expr: tok.Args}

	for {
		if err := p.skipIgnorable(); err != nil {
			return nil, err
		}
		next := p.peek()
		if next.Type == TokenEOF {
			return nil, errorAt(tok.Pos, "unclosed @switch starting here")
		}
		if next.Type != TokenDirective {
			return nil, errorAtf(next.Pos, "unexpected content inside @switch")
		}
		switch next.Value {
		case "case":
			p.next()
			if strings.TrimSpace(next.Args) == "" {
				return nil, errorAt(next.Pos, "@case requires a value")
			}
			body, err := p.parseBlock(stopSet{"case": true, "break": true, "default": true, "endswitch": true})
			if err != nil {
				return nil, err
			}
			// Optional @break after case body.
			if br := p.peek(); br.Type == TokenDirective && br.Value == "break" {
				p.next()
			}
			node.Cases = append(node.Cases, SwitchCase{Value: next.Args, Body: body})
		case "default":
			p.next()
			body, err := p.parseBlock(stopSet{"endswitch": true})
			if err != nil {
				return nil, err
			}
			node.Default = body
			end := p.peek()
			if end.Type != TokenDirective || end.Value != "endswitch" {
				return nil, errorAt(tok.Pos, "unclosed @switch starting here")
			}
			p.next()
			return node, nil
		case "endswitch":
			p.next()
			return node, nil
		default:
			return nil, errorAtf(next.Pos, "unexpected @%s inside @switch", next.Value)
		}
	}
}

func (p *Parser) parseForeach(tok Token) (*ForeachNode, error) {
	p.next()
	expr, key, val, err := parseForeachArgs(tok.Args, tok.Pos)
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock(stopSet{"empty": true, "endforeach": true})
	if err != nil {
		return nil, err
	}
	node := &ForeachNode{
		Position: tok.Pos,
		Expr:     expr,
		KeyVar:   key,
		ValueVar: val,
		Body:     body,
	}
	next := p.peek()
	if next.Type != TokenDirective {
		return nil, errorAt(tok.Pos, "unclosed @foreach starting here")
	}
	switch next.Value {
	case "empty":
		p.next()
		emptyBody, err := p.parseBlock(stopSet{"endforeach": true})
		if err != nil {
			return nil, err
		}
		node.Empty = emptyBody
		end := p.peek()
		if end.Type != TokenDirective || end.Value != "endforeach" {
			return nil, errorAt(tok.Pos, "unclosed @foreach starting here")
		}
		p.next()
	case "endforeach":
		p.next()
	default:
		return nil, errorAt(tok.Pos, "unclosed @foreach starting here")
	}
	return node, nil
}

func (p *Parser) parseExtends(tok Token) (*ExtendsNode, error) {
	p.next()
	layout, err := requireOneQuoted(tok.Args, tok.Pos, "@extends")
	if err != nil {
		return nil, err
	}
	return &ExtendsNode{Position: tok.Pos, Layout: layout}, nil
}

func (p *Parser) parseSection(tok Token) (*SectionNode, error) {
	p.next()
	parts, err := splitTopLevelArgs(tok.Args)
	if err != nil {
		return nil, errorAt(tok.Pos, err.Error())
	}
	if len(parts) == 0 || parts[0] == "" {
		return nil, errorAt(tok.Pos, "@section requires a name")
	}
	name, err := unquoteString(parts[0])
	if err != nil {
		return nil, errorAtf(tok.Pos, "@section name: %v", err)
	}

	// Short form: @section("name", "value")
	if len(parts) >= 2 {
		inline, err := unquoteString(parts[1])
		if err != nil {
			return nil, errorAtf(tok.Pos, "@section value: %v", err)
		}
		if len(parts) > 2 {
			return nil, errorAt(tok.Pos, "@section short form takes at most two arguments")
		}
		return &SectionNode{Position: tok.Pos, Name: name, Inline: inline}, nil
	}

	// Long form: @section("name") ... @endsection
	body, err := p.parseBlock(stopSet{"endsection": true})
	if err != nil {
		return nil, err
	}
	end := p.peek()
	if end.Type != TokenDirective || end.Value != "endsection" {
		return nil, errorAt(tok.Pos, "unclosed @section starting here")
	}
	p.next()
	return &SectionNode{Position: tok.Pos, Name: name, Body: body}, nil
}

func (p *Parser) parseYield(tok Token) (*YieldNode, error) {
	p.next()
	parts, err := splitTopLevelArgs(tok.Args)
	if err != nil {
		return nil, errorAt(tok.Pos, err.Error())
	}
	if len(parts) == 0 || parts[0] == "" {
		return nil, errorAt(tok.Pos, "@yield requires a name")
	}
	name, err := unquoteString(parts[0])
	if err != nil {
		return nil, errorAtf(tok.Pos, "@yield name: %v", err)
	}
	node := &YieldNode{Position: tok.Pos, Name: name}
	if len(parts) >= 2 {
		def, err := unquoteString(parts[1])
		if err != nil {
			return nil, errorAtf(tok.Pos, "@yield default: %v", err)
		}
		node.Default = []Node{&RawTextNode{Position: tok.Pos, Text: def}}
	}
	if len(parts) > 2 {
		return nil, errorAt(tok.Pos, "@yield takes at most two arguments")
	}
	return node, nil
}

func (p *Parser) parseInclude(tok Token, includeIf bool) (*IncludeNode, error) {
	p.next()
	parts, err := splitTopLevelArgs(tok.Args)
	if err != nil {
		return nil, errorAt(tok.Pos, err.Error())
	}
	dir := "@include"
	if includeIf {
		dir = "@includeIf"
	}
	if len(parts) == 0 || parts[0] == "" {
		return nil, errorAtf(tok.Pos, "%s requires a target", dir)
	}
	target, err := unquoteString(parts[0])
	if err != nil {
		return nil, errorAtf(tok.Pos, "%s target: %v", dir, err)
	}
	node := &IncludeNode{Position: tok.Pos, Target: target, If: includeIf}
	if len(parts) >= 2 {
		node.DataExpr = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		return nil, errorAtf(tok.Pos, "%s takes at most two arguments", dir)
	}
	return node, nil
}

func (p *Parser) parseComponent(tok Token) (*ComponentNode, error) {
	p.next()
	parts, err := splitTopLevelArgs(tok.Args)
	if err != nil {
		return nil, errorAt(tok.Pos, err.Error())
	}
	if len(parts) == 0 || parts[0] == "" {
		return nil, errorAt(tok.Pos, "@component requires a target")
	}
	target, err := unquoteString(parts[0])
	if err != nil {
		return nil, errorAtf(tok.Pos, "@component target: %v", err)
	}
	node := &ComponentNode{Position: tok.Pos, Target: target}
	if len(parts) >= 2 {
		node.DataExpr = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		return nil, errorAt(tok.Pos, "@component takes at most two arguments")
	}

	// Parse body until @endcomponent, routing @slot into Slots and the rest into Default.
	for {
		if err := p.skipIgnorable(); err != nil {
			return nil, err
		}
		next := p.peek()
		if next.Type == TokenEOF {
			return nil, errorAt(tok.Pos, "unclosed @component starting here")
		}
		if next.Type == TokenDirective && next.Value == "endcomponent" {
			p.next()
			return node, nil
		}
		if next.Type == TokenDirective && next.Value == "slot" {
			slot, err := p.parseSlot(next)
			if err != nil {
				return nil, err
			}
			node.Slots = append(node.Slots, *slot)
			continue
		}
		// Default-slot content: parse a single node (or a nested block opener).
		n, err := p.parseNode(stopSet{"slot": true, "endcomponent": true})
		if err != nil {
			return nil, err
		}
		if n != nil {
			node.Default = append(node.Default, n)
		}
	}
}

func (p *Parser) parseSlot(tok Token) (*SlotNode, error) {
	p.next()
	name, err := requireOneQuoted(tok.Args, tok.Pos, "@slot")
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock(stopSet{"endslot": true})
	if err != nil {
		return nil, err
	}
	end := p.peek()
	if end.Type != TokenDirective || end.Value != "endslot" {
		return nil, errorAt(tok.Pos, "unclosed @slot starting here")
	}
	p.next()
	return &SlotNode{Position: tok.Pos, Name: name, Body: body}, nil
}

func (p *Parser) parseProps(tok Token) (*PropsNode, error) {
	p.next()
	return &PropsNode{Position: tok.Pos, Raw: tok.Args}, nil
}

// skipIgnorable skips comments and whitespace-only text tokens.
func (p *Parser) skipIgnorable() error {
	for {
		tok := p.peek()
		switch tok.Type {
		case TokenComment:
			p.next()
		case TokenText:
			if isWhitespaceOnly(tok.Value) {
				p.next()
				continue
			}
			return nil
		default:
			return nil
		}
	}
}

func validateTopLevel(nodes []Node) error {
	meaningful := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		if rt, ok := n.(*RawTextNode); ok && isWhitespaceOnly(rt.Text) {
			continue
		}
		meaningful = append(meaningful, n)
	}
	if len(meaningful) == 0 {
		return nil
	}

	// @props must be the first meaningful statement when present.
	for i, n := range meaningful {
		if _, ok := n.(*PropsNode); ok && i != 0 {
			return errorAt(n.Pos(), "@props must be the first statement in the file")
		}
	}

	var extends *ExtendsNode
	for _, n := range meaningful {
		if e, ok := n.(*ExtendsNode); ok {
			if extends != nil {
				return errorAt(e.Pos(), "multiple @extends directives are not allowed")
			}
			extends = e
		}
	}
	if extends == nil {
		// Without @extends, top-level @section is not allowed.
		for _, n := range meaningful {
			if _, ok := n.(*SectionNode); ok {
				return errorAt(n.Pos(), "@section is only allowed in files that use @extends")
			}
		}
		return nil
	}

	// With @extends: only ExtendsNode + SectionNode (+ whitespace, already filtered).
	seenExtends := false
	for _, n := range meaningful {
		switch n.(type) {
		case *ExtendsNode:
			if seenExtends {
				return errorAt(n.Pos(), "multiple @extends directives are not allowed")
			}
			seenExtends = true
		case *SectionNode:
			if !seenExtends {
				return errorAt(n.Pos(), "@extends must appear before @section blocks")
			}
		default:
			return errorAt(n.Pos(), "when using @extends, only @section blocks are allowed at top level")
		}
	}
	return nil
}

func parseForeachArgs(args string, pos Position) (expr, key, value string, err error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", "", "", errorAt(pos, "@foreach requires arguments")
	}
	const sep = " as "
	count := strings.Count(args, sep)
	if count != 1 {
		return "", "", "", errorAt(pos, `@foreach requires exactly one " as " separator`)
	}
	idx := strings.Index(args, sep)
	expr = strings.TrimSpace(args[:idx])
	right := strings.TrimSpace(args[idx+len(sep):])
	if expr == "" || right == "" {
		return "", "", "", errorAt(pos, "@foreach has empty expression or variable list")
	}

	// "$item" or "$key, $item"
	if strings.Contains(right, ",") {
		parts := strings.SplitN(right, ",", 2)
		if len(parts) != 2 {
			return "", "", "", errorAt(pos, "invalid @foreach variable list")
		}
		key = strings.TrimSpace(parts[0])
		value = strings.TrimSpace(parts[1])
		if !foreachVarRe.MatchString(key) {
			return "", "", "", errorAtf(pos, "foreach variable must start with $, got: %s", key)
		}
		if !foreachVarRe.MatchString(value) {
			return "", "", "", errorAtf(pos, "foreach variable must start with $, got: %s", value)
		}
		return expr, key, value, nil
	}

	value = right
	if !foreachVarRe.MatchString(value) {
		return "", "", "", errorAtf(pos, "foreach variable must start with $, got: %s", value)
	}
	return expr, "", value, nil
}

func requireOneQuoted(args string, pos Position, dir string) (string, error) {
	parts, err := splitTopLevelArgs(args)
	if err != nil {
		return "", errorAt(pos, err.Error())
	}
	if len(parts) != 1 || parts[0] == "" {
		return "", errorAtf(pos, "%s requires exactly one quoted string argument", dir)
	}
	s, err := unquoteString(parts[0])
	if err != nil {
		return "", errorAtf(pos, "%s: %v", dir, err)
	}
	return s, nil
}

// splitTopLevelArgs splits on commas that are not inside quotes.
func splitTopLevelArgs(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var parts []string
	var cur strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inDouble {
			cur.WriteByte(c)
			if c == '\\' && i+1 < len(s) {
				cur.WriteByte(s[i+1])
				i++
				continue
			}
			if c == '"' {
				inDouble = false
			}
			continue
		}
		if inSingle {
			cur.WriteByte(c)
			if c == '\\' && i+1 < len(s) {
				cur.WriteByte(s[i+1])
				i++
				continue
			}
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		switch c {
		case '"':
			inDouble = true
			cur.WriteByte(c)
		case '\'':
			inSingle = true
			cur.WriteByte(c)
		case ',':
			parts = append(parts, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unclosed quote in arguments")
	}
	parts = append(parts, strings.TrimSpace(cur.String()))
	return parts, nil
}

func unquoteString(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return "", fmt.Errorf("expected quoted string, got %q", s)
	}
	quote := s[0]
	if (quote != '"' && quote != '\'') || s[len(s)-1] != quote {
		return "", fmt.Errorf("expected quoted string, got %q", s)
	}
	inner := s[1 : len(s)-1]
	var out strings.Builder
	out.Grow(len(inner))
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) {
			out.WriteByte(inner[i+1])
			i++
			continue
		}
		out.WriteByte(inner[i])
	}
	return out.String(), nil
}

func isWhitespaceOnly(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
