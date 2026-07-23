package template

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// PropDecl is one entry from an @props(...) declaration.
type PropDecl struct {
	Name    string
	Type    string // documentation only (e.g. "string", "[]string")
	Default string // raw default text; empty if none
}

var propsFieldRe = regexp.MustCompile(`\.Props\.([A-Za-z_][A-Za-z0-9_]*)`)

// ParseProps parses a PropsNode.Raw string into declarations.
// Example: "Name string, Count int = 0" → two PropDecls.
func ParseProps(raw string) ([]PropDecl, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := splitPropsList(raw)
	out := make([]PropDecl, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		decl, err := parseOneProp(part)
		if err != nil {
			return nil, err
		}
		out = append(out, decl)
	}
	return out, nil
}

// ExtractFieldRefs returns unique .Props.X identifiers referenced in the AST.
func ExtractFieldRefs(f *File) []string {
	seen := map[string]bool{}
	var names []string
	WalkExprs(f, func(expr string) {
		for _, m := range propsFieldRe.FindAllStringSubmatch(expr, -1) {
			if len(m) < 2 {
				continue
			}
			name := m[1]
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	})
	sort.Strings(names)
	return names
}

func parseOneProp(s string) (PropDecl, error) {
	name, rest, ok := splitIdent(s)
	if !ok || name == "" {
		return PropDecl{}, fmt.Errorf("invalid @props entry %q", s)
	}
	rest = strings.TrimSpace(rest)
	eq := indexTopLevel(rest, '=')
	var typ, def string
	if eq >= 0 {
		typ = strings.TrimSpace(rest[:eq])
		def = strings.TrimSpace(rest[eq+1:])
	} else {
		typ = rest
	}
	return PropDecl{Name: name, Type: typ, Default: def}, nil
}

func splitIdent(s string) (ident, rest string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", false
	}
	i := 0
	r := rune(s[0])
	if r != '_' && !unicode.IsLetter(r) {
		return "", s, false
	}
	i = 1
	for i < len(s) {
		r = rune(s[i])
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			i++
			continue
		}
		break
	}
	return s[:i], s[i:], true
}

// splitPropsList splits on commas not inside quotes or brackets.
func splitPropsList(s string) []string {
	var parts []string
	var cur strings.Builder
	depth := 0
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
		case '[', '(', '{', '<':
			depth++
			cur.WriteByte(c)
		case ']', ')', '}', '>':
			if depth > 0 {
				depth--
			}
			cur.WriteByte(c)
		case ',':
			if depth == 0 {
				parts = append(parts, cur.String())
				cur.Reset()
				continue
			}
			cur.WriteByte(c)
		default:
			cur.WriteByte(c)
		}
	}
	parts = append(parts, cur.String())
	return parts
}

func indexTopLevel(s string, sep byte) int {
	depth := 0
	inSingle, inDouble := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inDouble {
			if c == '\\' && i+1 < len(s) {
				i++
				continue
			}
			if c == '"' {
				inDouble = false
			}
			continue
		}
		if inSingle {
			if c == '\\' && i+1 < len(s) {
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
		case '\'':
			inSingle = true
		case '[', '(', '{':
			depth++
		case ']', ')', '}':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && c == sep {
				return i
			}
		}
	}
	return -1
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = min(del, ins, sub)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func didYouMean(got string, candidates []string) string {
	best := ""
	bestDist := -1
	for _, c := range candidates {
		d := levenshtein(strings.ToLower(got), strings.ToLower(c))
		if bestDist < 0 || d < bestDist {
			bestDist = d
			best = c
		}
	}
	if best == "" || bestDist > 2 {
		return ""
	}
	return best
}

// checkStrictProps compares @props declarations with .Props.X usages.
// Returns hard errors (undeclared use) and soft warnings (unused decls).
func checkStrictProps(files map[string]*File) (errs []error, warnings []string) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		f := files[name]
		propsRaw := ""
		for _, n := range f.Nodes {
			if p, ok := n.(*PropsNode); ok {
				propsRaw = p.Raw
				break
			}
		}
		if propsRaw == "" {
			continue
		}
		decls, err := ParseProps(propsRaw)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: @props: %w", displayPath(f), err))
			continue
		}
		declared := make(map[string]bool, len(decls))
		declNames := make([]string, 0, len(decls))
		for _, d := range decls {
			declared[d.Name] = true
			declNames = append(declNames, d.Name)
		}
		used := ExtractFieldRefs(f)
		usedSet := make(map[string]bool, len(used))
		for _, u := range used {
			usedSet[u] = true
			if !declared[u] {
				msg := fmt.Sprintf("%s: .Props.%s used but not declared in @props", displayPath(f), u)
				if sug := didYouMean(u, declNames); sug != "" {
					msg += fmt.Sprintf(" (did you mean %s?)", sug)
				}
				errs = append(errs, fmt.Errorf("%s", msg))
			}
		}
		for _, d := range decls {
			if !usedSet[d.Name] {
				warnings = append(warnings, fmt.Sprintf(
					"%s: @props field %q is declared but unused", displayPath(f), d.Name))
			}
		}
	}
	return errs, warnings
}

func displayPath(f *File) string {
	if f == nil || f.Path == "" {
		return "template"
	}
	return filepath.ToSlash(f.Path)
}
