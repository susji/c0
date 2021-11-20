package analyze

import (
	"fmt"

	"github.com/susji/c0/node"
)

type SyntaxError struct {
	Node    node.Node
	Fn      string
	Wrapped error
}

func (e *SyntaxError) Error() string {
	lineno, col := e.Node.Tok().Lineno(), e.Node.Tok().Col()
	return fmt.Sprintf("%s:%d:%d: %s", e.Fn, lineno, col, e.Wrapped)
}

func (e *SyntaxError) Unwrap() error {
	return e.Wrapped
}
