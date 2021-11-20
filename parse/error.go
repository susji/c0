package parse

import (
	"fmt"

	"github.com/susji/c0/token"
)

type ParseError struct {
	Wrapped error
	Fn      string
	Tok     *token.Token
}

func (e *ParseError) Error() string {
	lineno, col := e.Tok.Lineno(), e.Tok.Col()
	return fmt.Sprintf("%s:%d:%d: %s", e.Fn, lineno, col, e.Wrapped)
}

func (e *ParseError) Unwrap() error {
	return e.Wrapped
}
