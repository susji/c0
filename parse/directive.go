package parse

import (
	"bytes"
	"io/ioutil"

	"github.com/susji/c0/lex"
	"github.com/susji/c0/node"
	"github.com/susji/c0/token"
)

// DirectiveUse handles and parses file includes. Noteworthily, the returned
// error merely indicates if parsing of the directive was correct. The returned
// struct will then contain potential lexing and parsing errors. Upper on the
// chain, someone needs to decide how to handle the nodes and typedefs receveid
// via this node -- see handleUse in parse.go.
func (p *Parser) DirectiveUse(toks *token.Tokens) (*node.DirectiveUse, error) {
	what := toks.Peek()
	if what == nil {
		panic("should not happen")
	}
	var ret *node.DirectiveUse
	var val node.Node
	switch what.Kind() {
	case token.UseStrLit:
		val = node.Store(what, &node.StrLit{Value: what.Value()})
	case token.UseLibLit:
		val = node.Store(what, &node.LibLit{Value: what.Value()})
	default:
		return nil, p.errorf(
			what,
			"expecting a string or library literal for #use, got %v",
			what)
	}
	toks.Pop()

	var lexerrs []error
	var parerr error
	var ntoks *token.Tokens

	pn := NewFile(what.Value())
	nsrc, readerr := ioutil.ReadFile(what.Value())
	if readerr != nil {
		goto end
	}
	ntoks, lexerrs = lex.Lex(bytes.Runes(nsrc))
	parerr = pn.Parse(ntoks)
end:
	ret = node.Store(what, &node.DirectiveUse{
		Success:     readerr == nil && len(lexerrs) == 0 && parerr == nil,
		How:         val,
		Nodes:       pn.Nodes(),
		LexErrors:   lexerrs,
		ParseErrors: pn.Errors(),
		Typedefs:    pn.typedefs,
	}).(*node.DirectiveUse)
	return ret, nil
}
