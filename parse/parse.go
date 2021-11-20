package parse

import (
	"errors"
	"fmt"

	"github.com/susji/c0/analyze"
	"github.com/susji/c0/node"
	"github.com/susji/c0/token"
)

var (
	ErrUse   = errors.New("use encountered errors")
	ErrParse = errors.New("parsing met with error(s)")
	EOT      = errors.New("end of tokens")
)

type Parser struct {
	fn       string
	nodes    []node.Node
	errs     []error
	typedefs map[string]struct{}
}

func (p *Parser) errorf(tok *token.Token, format string, a ...interface{}) error {
	err := &ParseError{
		Tok:     tok,
		Fn:      p.fn,
		Wrapped: fmt.Errorf(format, a...),
	}
	p.errs = append(p.errs, err)
	return err
}

func (p *Parser) Errors() []error {
	if len(p.errs) == 0 {
		return nil
	}
	return p.errs
}

func (p *Parser) Nodes() []node.Node {
	return p.nodes
}

func (p *Parser) Typedefs() map[string]struct{} {
	return p.typedefs
}

func (p *Parser) Fn() string {
	return p.fn
}

func (p *Parser) AddTypedef(tok *token.Token, name string) error {
	if analyze.IsReserved(name) {
		return p.errorf(tok, "typedef name %q is reserved", name)
	}
	if _, ok := p.typedefs[name]; ok {
		return fmt.Errorf("typedef %q already defined", name)
	}
	p.typedefs[name] = struct{}{}
	return nil
}

func (p *Parser) IsTypedef(name string) bool {
	_, ok := p.typedefs[name]
	return ok
}

func (p *Parser) handleUse(tok *token.Token, use *node.DirectiveUse) error {
	inerr := false
	if !use.Success {
		inerr = true
		for _, lexerr := range use.LexErrors {
			p.errs = append(p.errs, lexerr)
		}
		for _, parerr := range use.ParseErrors {
			p.errs = append(p.errs, parerr)
		}
		return p.errorf(tok, "errors in #use %s", use.How)
	}
	for td, _ := range use.Typedefs {
		if err := p.AddTypedef(tok, td); err != nil {
			inerr = true
			p.errs = append(p.errs, err)
		}
	}
	for _, usenode := range use.Nodes {
		p.nodes = append(p.nodes, usenode)
	}
	if inerr {
		return ErrUse
	} else {
		// Zero out the use contents now that we have successfully included
		// them here.
		use.Typedefs = nil
		use.Nodes = nil
	}
	return nil
}

func (p *Parser) Parse(toks *token.Tokens) error {
	p.errs = []error{}
	p.nodes = []node.Node{}
	p.typedefs = map[string]struct{}{}
	for toks.Len() > 0 {
		cur := toks.Peek()
		if newnode, err := p.GlobalDeclDef(toks); err == nil {
			p.nodes = append(p.nodes, newnode)
			switch t := newnode.(type) {
			case *node.DirectiveUse:
				p.handleUse(cur, t)
			}
		} else {
			// If we completely failed in parsing, rewind until the next ';' or
			// '}' is reached. This gives us a better chance to catch multiple
			// errors.
			toks.Find(token.Semicolon, token.RCurly)
			toks.Pop()
		}
	}
	if len(p.errs) > 0 {
		return ErrParse
	}
	return nil
}

func New() *Parser {
	return NewFile("<stdin>")
}

func NewFile(fn string) *Parser {
	return &Parser{
		fn:       fn,
		typedefs: map[string]struct{}{},
	}
}
