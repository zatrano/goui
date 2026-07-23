package template

import (
	"fmt"
	"os"
	"strings"
)

// TemplateError is the central compile-time error type for the template package.
type TemplateError struct {
	File    string
	Line    int
	Column  int
	Message string
	// Wrapped is an optional underlying error (e.g. a lexer failure).
	Wrapped error
}

func (e *TemplateError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
}

func (e *TemplateError) Unwrap() error { return e.Wrapped }

// Snippet returns a Cargo-style source excerpt around the error line
// (previous, current, next). Returns "" if the file cannot be read.
func (e *TemplateError) Snippet() string {
	if e == nil || e.File == "" || e.Line < 1 {
		return ""
	}
	data, err := os.ReadFile(e.File)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if e.Line > len(lines) {
		return ""
	}
	var b strings.Builder
	start := e.Line - 1
	if start < 1 {
		start = 1
	}
	end := e.Line + 1
	if end > len(lines) {
		end = len(lines)
	}
	width := len(fmt.Sprintf("%d", end))
	for i := start; i <= end; i++ {
		mark := " "
		if i == e.Line {
			mark = ">"
		}
		fmt.Fprintf(&b, "%s %*d | %s\n", mark, width, i, lines[i-1])
	}
	return b.String()
}

// errorAt builds a TemplateError at the given position.
func errorAt(pos Position, msg string) *TemplateError {
	return &TemplateError{
		File:    pos.File,
		Line:    pos.Line,
		Column:  pos.Column,
		Message: msg,
	}
}

// errorAtf builds a TemplateError with a formatted message.
func errorAtf(pos Position, format string, args ...any) *TemplateError {
	return errorAt(pos, fmt.Sprintf(format, args...))
}
