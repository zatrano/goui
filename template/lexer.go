package template

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	pfxComment = []byte("{{--")
	pfxRaw     = []byte("{!!")
	pfxOut     = []byte("{{")
	pfxAtAt    = []byte("@@")
	sfxComment = []byte("--}}")
	sfxRaw     = []byte("!!}")
	sfxOut     = []byte("}}")
)

// Lexer tokenizes .goui.html source into a []Token stream.
// It does not parse {{ }} / {!! !!} expressions — only finds their boundaries.
type Lexer struct {
	filename string
	src      []byte
	offset   int // byte offset
	line     int // 1-based
	column   int // 1-based, rune columns
}

// NewLexer creates a lexer for the given source.
func NewLexer(filename string, src []byte) *Lexer {
	return &Lexer{
		filename: filename,
		src:      src,
		line:     1,
		column:   1,
	}
}

// Tokenize scans the entire source and returns tokens (including a trailing TokenEOF).
func (l *Lexer) Tokenize() ([]Token, error) {
	tokens := make([]Token, 0, 32)
	var textBuf strings.Builder
	textBuf.Grow(64)
	var textStart Position
	textActive := false

	flushText := func() {
		if !textActive {
			return
		}
		tokens = append(tokens, Token{
			Type:  TokenText,
			Value: textBuf.String(),
			Pos:   textStart,
		})
		textBuf.Reset()
		textActive = false
	}

	appendTextBytes := func(s []byte) {
		if len(s) == 0 {
			return
		}
		if !textActive {
			textStart = l.pos()
			textActive = true
		}
		textBuf.Write(s)
	}

	appendTextStr := func(s string) {
		if !textActive {
			textStart = l.pos()
			textActive = true
		}
		textBuf.WriteString(s)
	}

	for !l.atEnd() {
		rest := l.src[l.offset:]

		switch {
		case bytes.HasPrefix(rest, pfxComment):
			flushText()
			tok, err := l.scanComment()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)

		case bytes.HasPrefix(rest, pfxRaw):
			flushText()
			tok, err := l.scanRawOutput()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)

		case bytes.HasPrefix(rest, pfxOut):
			flushText()
			tok, err := l.scanEscapedOutput()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)

		case bytes.HasPrefix(rest, pfxAtAt):
			appendTextStr("@")
			l.advanceASCII(2)

		case rest[0] == '@' && l.isDirectiveCandidate():
			flushText()
			tok, err := l.scanDirective()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)

		default:
			// Fast path: copy contiguous plain text until '{' or '@'.
			if !textActive {
				textStart = l.pos()
				textActive = true
			}
			start := l.offset
			// If current byte is a non-special '{'/'@', consume it first.
			if rest[0] == '{' || rest[0] == '@' {
				l.advanceASCII(1)
			}
			for l.offset < len(l.src) {
				b := l.src[l.offset]
				if b == '{' || b == '@' {
					break
				}
				if b == '\n' {
					l.offset++
					l.line++
					l.column = 1
					continue
				}
				if b < utf8.RuneSelf {
					l.offset++
					l.column++
					continue
				}
				// Emit ASCII run, then one multi-byte rune.
				appendTextBytes(l.src[start:l.offset])
				start = l.offset
				_ = l.nextRune()
				appendTextBytes(l.src[start:l.offset])
				start = l.offset
			}
			appendTextBytes(l.src[start:l.offset])
		}
	}

	flushText()
	tokens = append(tokens, Token{Type: TokenEOF, Pos: l.pos()})
	return tokens, nil
}

// isDirectiveCandidate reports whether '@' at the current position starts a directive.
// Requires: previous rune is not [A-Za-z0-9_], and next rune is [A-Za-z_].
// This keeps emails like user@example.com as plain text.
func (l *Lexer) isDirectiveCandidate() bool {
	if l.offset >= len(l.src) || l.src[l.offset] != '@' {
		return false
	}
	if l.offset > 0 {
		prev := l.src[l.offset-1]
		// Fast ASCII check for previous byte (HTML sources are overwhelmingly ASCII).
		if prev < utf8.RuneSelf {
			if isASCIIIdentCont(prev) {
				return false
			}
		} else {
			r, _ := utf8.DecodeLastRune(l.src[:l.offset])
			if isIdentContinue(r) {
				return false
			}
		}
	}
	if l.offset+1 >= len(l.src) {
		return false
	}
	next := l.src[l.offset+1]
	if next < utf8.RuneSelf {
		return isASCIIIdentStart(next)
	}
	r, _ := utf8.DecodeRune(l.src[l.offset+1:])
	return isIdentStart(r)
}

func (l *Lexer) scanComment() (Token, error) {
	start := l.pos()
	l.advanceASCII(4) // {{--
	closeIdx := bytes.Index(l.src[l.offset:], sfxComment)
	if closeIdx < 0 {
		return Token{}, l.errAt(start, "unclosed comment")
	}
	l.advanceContent(closeIdx)
	l.advanceASCII(4) // --}}
	return Token{Type: TokenComment, Value: "", Pos: start}, nil
}

func (l *Lexer) scanRawOutput() (Token, error) {
	start := l.pos()
	l.advanceASCII(3) // {!!
	closeIdx := bytes.Index(l.src[l.offset:], sfxRaw)
	if closeIdx < 0 {
		return Token{}, l.errAt(start, "unclosed raw output")
	}
	expr := strings.TrimSpace(string(l.src[l.offset : l.offset+closeIdx]))
	l.advanceContent(closeIdx)
	l.advanceASCII(3) // !!}
	return Token{Type: TokenOutputRaw, Value: expr, Pos: start}, nil
}

func (l *Lexer) scanEscapedOutput() (Token, error) {
	start := l.pos()
	l.advanceASCII(2) // {{
	closeIdx := bytes.Index(l.src[l.offset:], sfxOut)
	if closeIdx < 0 {
		return Token{}, l.errAt(start, "unclosed output")
	}
	expr := strings.TrimSpace(string(l.src[l.offset : l.offset+closeIdx]))
	l.advanceContent(closeIdx)
	l.advanceASCII(2) // }}
	return Token{Type: TokenOutputEscaped, Value: expr, Pos: start}, nil
}

func (l *Lexer) scanDirective() (Token, error) {
	start := l.pos()
	l.advanceASCII(1) // @

	nameStart := l.offset
	for l.offset < len(l.src) {
		b := l.src[l.offset]
		if b < utf8.RuneSelf {
			if !isASCIIIdentCont(b) {
				break
			}
			l.offset++
			l.column++
			continue
		}
		r, size := utf8.DecodeRune(l.src[l.offset:])
		if !isIdentContinue(r) {
			break
		}
		l.offset += size
		l.column++
	}
	name := string(l.src[nameStart:l.offset])

	args := ""
	if !l.atEnd() && l.src[l.offset] == '(' {
		var err error
		args, err = l.scanBalancedArgs(start)
		if err != nil {
			return Token{}, err
		}
	}

	return Token{
		Type:  TokenDirective,
		Value: name,
		Args:  args,
		Pos:   start,
	}, nil
}

// scanBalancedArgs consumes a parenthesized argument list starting at '('.
// Depth starts at 1; parentheses inside quotes are ignored.
func (l *Lexer) scanBalancedArgs(dirPos Position) (string, error) {
	l.advanceASCII(1) // '('
	depth := 1
	contentStart := l.offset
	inSingle, inDouble := false, false

	for !l.atEnd() {
		b := l.src[l.offset]

		if inDouble {
			if b == '\\' && l.offset+1 < len(l.src) {
				next := l.src[l.offset+1]
				if next == '"' || next == '\'' || next == '\\' {
					l.advanceASCII(2)
					continue
				}
			}
			if b == '"' {
				inDouble = false
			}
			l.advanceOne()
			continue
		}

		if inSingle {
			if b == '\\' && l.offset+1 < len(l.src) {
				next := l.src[l.offset+1]
				if next == '"' || next == '\'' || next == '\\' {
					l.advanceASCII(2)
					continue
				}
			}
			if b == '\'' {
				inSingle = false
			}
			l.advanceOne()
			continue
		}

		switch b {
		case '"':
			inDouble = true
			l.advanceASCII(1)
		case '\'':
			inSingle = true
			l.advanceASCII(1)
		case '(':
			depth++
			l.advanceASCII(1)
		case ')':
			depth--
			if depth == 0 {
				args := strings.TrimSpace(string(l.src[contentStart:l.offset]))
				l.advanceASCII(1) // ')'
				return args, nil
			}
			l.advanceASCII(1)
		default:
			l.advanceOne()
		}
	}

	return "", l.errAt(dirPos, "unclosed directive arguments")
}

func (l *Lexer) pos() Position {
	return Position{
		File:   l.filename,
		Line:   l.line,
		Column: l.column,
		Offset: l.offset,
	}
}

func (l *Lexer) errAt(p Position, msg string) *TemplateError {
	return &TemplateError{
		File:    p.File,
		Line:    p.Line,
		Column:  p.Column,
		Message: msg,
	}
}

func (l *Lexer) atEnd() bool {
	return l.offset >= len(l.src)
}

// advanceASCII advances n bytes known to be single-byte ASCII.
func (l *Lexer) advanceASCII(n int) {
	end := l.offset + n
	if end > len(l.src) {
		end = len(l.src)
	}
	for l.offset < end {
		if l.src[l.offset] == '\n' {
			l.offset++
			l.line++
			l.column = 1
		} else {
			l.offset++
			l.column++
		}
	}
}

// advanceContent advances n content bytes, tracking newlines in ASCII-fast path.
func (l *Lexer) advanceContent(n int) {
	end := l.offset + n
	if end > len(l.src) {
		end = len(l.src)
	}
	for l.offset < end {
		b := l.src[l.offset]
		if b == '\n' {
			l.offset++
			l.line++
			l.column = 1
			continue
		}
		if b < utf8.RuneSelf {
			l.offset++
			l.column++
			continue
		}
		l.nextRune()
	}
}

func (l *Lexer) advanceOne() {
	if l.atEnd() {
		return
	}
	if l.src[l.offset] < utf8.RuneSelf {
		l.advanceASCII(1)
		return
	}
	l.nextRune()
}

func (l *Lexer) nextRune() rune {
	r, size := utf8.DecodeRune(l.src[l.offset:])
	if r == '\n' {
		l.offset += size
		l.line++
		l.column = 1
		return r
	}
	l.offset += size
	l.column++
	return r
}

func isASCIIIdentStart(b byte) bool {
	return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isASCIIIdentCont(b byte) bool {
	return isASCIIIdentStart(b) || (b >= '0' && b <= '9')
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentContinue(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
