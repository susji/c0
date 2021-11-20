package span

import "fmt"

// Span defines a range formed by two pairs of (lineno, col).
type Span struct {
	Lineno0, Col0, Lineno, Col int
}

func (span Span) String() string {
	return fmt.Sprintf(
		"(%d, %d) -> (%d, %d)",
		span.Lineno0, span.Col0,
		span.Lineno, span.Col)
}
