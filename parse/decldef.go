package parse

import (
	"errors"
	"fmt"

	"github.com/susji/c0/analyze"
	"github.com/susji/c0/node"
	"github.com/susji/c0/token"
)

func (p *Parser) VarDecl(toks *token.Tokens) (*node.VarDecl, error) {
	first := toks.Peek()
	if first == nil {
		return nil, EOT
	}
	kind, err := p.Type(toks)
	if err != nil {
		return nil, err
	}
	next := toks.Peek()
	if next == nil {
		return nil, EOT
	}
	if next.Kind() != token.Id {
		return nil, p.errorf(
			first,
			"not a var declaration, expecting identifier, got %v",
			next)
	}
	if analyze.IsReserved(next.Value()) {
		return nil, p.errorf(next,
			"reserved identifier %q for variable declaration", next.Value())
	}
	toks.Pop()
	return node.Store(first, &node.VarDecl{
		Name: next.Value(),
		Kind: kind,
	}).(*node.VarDecl), nil
}

func (p *Parser) TopVarDecl(toks *token.Tokens) (node.Node, error) {
	first := toks.Peek()
	if first == nil {
		return nil, EOT
	}
	kind, err := p.Type(toks)
	if err != nil {
		return nil, err
	}
	next := toks.Peek()
	if next == nil {
		return nil, EOT
	}
	if kind.Kind == node.KIND_STRUCT &&
		kind.PointerLevel == 0 && kind.ArrayLevel == 0 &&
		next.Kind() == token.Semicolon {
		toks.Pop()
		return &node.StructForwardDecl{Value: kind.Name}, nil
	}
	switch next.Kind() {
	case token.Id, token.LCurly:
	default:
		return nil, p.errorf(
			first,
			"not a var declaration or a struct definition, got %v",
			next)

	}
	if kind.Kind == node.KIND_STRUCT {
		if sd, err := p.StructDef(toks, kind.Name); err == nil {
			return sd, nil
		}
	}
	if analyze.IsReserved(next.Value()) {
		return nil, p.errorf(next,
			"reserved identifier %q for variable declaration", next.Value())
	}
	toks.Pop()
	return node.Store(first, &node.VarDecl{
		Name: next.Value(),
		Kind: kind,
	}).(*node.VarDecl), nil
}

func (p *Parser) FuncParams(toks *token.Tokens) ([]node.VarDecl, error) {
	first := toks.Peek()
	if first == nil {
		return nil, EOT
	}
	params := []node.VarDecl{}
	if err := toks.Accept(token.RParen); err == nil {
		return params, nil
	}
params:
	for {
		pdecl, err := p.VarDecl(toks)
		if err != nil {
			return nil,
				p.errorf(first, "unexpected parameter list contents: %w", err)
		}
		params = append(params, *pdecl)

		parorcomma := toks.Peek()
		if parorcomma == nil {
			return nil,
				p.errorf(first, "unexpected end of parameter list")
		}
		switch parorcomma.Kind() {
		case token.RParen:
			break params
		case token.Comma:
			toks.Pop()
		}
	}
	if err := toks.Accept(token.RParen); err != nil {
		return nil,
			p.errorf(first, "unterminated parameter list: %w", err)
	}
	return params, nil
}

func (p *Parser) FuncDecl(toks *token.Tokens, vd *node.VarDecl) (*node.FunDecl, error) {
	first := toks.Peek()
	if first == nil {
		return nil, EOT
	}
	if err := toks.Accept(token.LParen); err != nil {
		return nil, p.errorf(first, "invalid function declaration: %w", err)
	}
	ret := node.Store(first, &node.FunDecl{
		Name:    vd.Name,
		Returns: vd.Kind,
		Params:  nil,
	}).(*node.FunDecl)
	params, err := p.FuncParams(toks)
	if err != nil {
		return nil, err
	}
	ret.Params = params
	return ret, nil
}

func (p *Parser) StructDef(toks *token.Tokens, name string) (node.Node, error) {
	first := toks.Peek()
	if first == nil {
		return nil, EOT
	}
	if first.Kind() != token.LCurly {
		return nil, errors.New("not a struct definition")
	}
	toks.Pop()
	ms := []node.VarDecl{}
	cur := toks.Peek()
	for cur != nil && cur.Kind() != token.RCurly {
		mk, err := p.Type(toks)
		if err != nil {
			return nil, p.errorf(cur, "expecting struct member type, got %s", cur)
		}
		mid := toks.Peek()
		if mid != nil && mid.Kind() != token.Id {
			return nil, p.errorf(cur, "expecting struct member name, got %s", mid)
		}
		if analyze.IsReserved(mid.Value()) {
			return nil,
				p.errorf(mid, "struct member %q is a reserved identifier", mid.Value())
		}
		toks.Pop()
		if err := toks.Accept(token.Semicolon); err != nil {
			return nil,
				p.errorf(cur, "struct definition member missing ';'")
		}
		ms = append(ms, node.VarDecl{Kind: mk, Name: mid.Value()})
		cur = toks.Peek()
	}
	if err := toks.Accept(token.RCurly); err != nil {
		return nil,
			p.errorf(first,
				"struct definition missing '}'")
	}
	if len(ms) == 0 {
		return nil, p.errorf(first, "struct without any members")
	}
	if err := toks.Accept(token.Semicolon); err != nil {
		return nil, p.errorf(first, "struct definition missing ';'")
	}
	return node.Store(first, &node.Struct{
		Name:    name,
		Members: ms,
	}), nil
}

func (p *Parser) FuncDeclDef(toks *token.Tokens, vd *node.VarDecl) (node.Node, error) {
	first := toks.Peek()
	if first == nil {
		return nil, EOT
	}
	fd, err := p.FuncDecl(toks, vd)
	if err != nil {
		return nil, err
	}
	if err := toks.Accept(token.Semicolon); err == nil {
		return fd, nil
	}
	ret := node.Store(first, &node.FunDef{FunDecl: *fd}).(*node.FunDef)
	if body, err := p.Block(toks); err == nil {
		ret.Body = *body
		return ret, nil
	} else {
		return nil, p.errorf(first,
			"invalid function body for %q: %w", ret.FunDecl.Name, err)
	}
}

func (p *Parser) TypedefDef(toks *token.Tokens) (node.Node, error) {
	first := toks.Peek()
	if first == nil || first.Kind() != token.Id || first.Value() != "typedef" {
		return nil, fmt.Errorf("not a typedef definition")
	}
	toks.Pop()
	tk, err := p.Type(toks)
	if err != nil {
		return nil, p.errorf(first, "invalid typedef kind: %w", err)
	}
	aidtok := toks.Peek()
	if aidtok == nil || aidtok.Kind() != token.Id {
		return nil, p.errorf(first, "expecting typedef identifier, got %s", aidtok)
	}
	aid := aidtok.Value()
	if analyze.IsReserved(aid) {
		return nil, p.errorf(aidtok, "typedef identifier %q is reserved", aid)
	}
	toks.Pop()
	var ret node.Node
	// Is it a typedef'd function pointer?
	if err := toks.Accept(token.LParen); err == nil {
		pp, err := p.FuncParams(toks)
		if err != nil {
			return nil, err
		}
		ret = &node.TypedefFunc{
			Name:    aid,
			Returns: tk,
			Params:  pp,
		}
	}
	if err := toks.Accept(token.Semicolon); err != nil {
		return nil, p.errorf(aidtok, "typedef missing ';'")
	}
	if err := p.AddTypedef(aidtok, aid); err != nil {
		return nil, p.errorf(aidtok, "invalid typedef: %w", err)
	}
	if ret == nil {
		ret = &node.Typedef{
			Kind: tk,
			Name: aid,
		}
	}
	return node.Store(first, ret), nil
}

func (p *Parser) GlobalDeclDef(toks *token.Tokens) (node.Node, error) {
	first := toks.Peek()
	if first == nil {
		return nil, EOT
	}
	// Here we combine the top-level production
	//
	//    <prog> :== (<gdecl> | <gdefn>)*
	//
	// There are thus four possible cases:
	//   - compiler directive ("#" <directive> ... "\n")
	//   - struct forward-declaration/definition ("struct" <sid> ...)
	//   - type definition ("typedef" <tp> <aid>)
	//   - global variable/function declaration/definition (<tp> <vid> ... ";")
	//
	var ret node.Node
	switch first.Kind() {
	case token.UseLibLit, token.UseStrLit:
		du, err := p.DirectiveUse(toks)
		if err != nil {
			return nil, err
		}
		ret = du
	case token.Id:
		switch first.Value() {
		case "typedef":
			td, err := p.TypedefDef(toks)
			if err != nil {
				return nil, err
			}
			ret = td
		default:
			first := toks.Peek()
			if tvd, err := p.TopVarDecl(toks); err == nil {
				switch t := tvd.(type) {
				case *node.StructForwardDecl, *node.Struct:
					ret = tvd
				case *node.VarDecl:
					if fd, err := p.FuncDeclDef(toks, t); err == nil {
						ret = fd
					} else {
						p.errorf(first,
							"invalid function definition/declaration: %w",
							err)
					}
				default:
					panic(fmt.Sprintf("unrecognized top var decl result: %s", t))
				}
			} else {
				return nil, p.errorf(first, "invalid statement")
			}
		}
	default:
		return nil, p.errorf(first, "unexpected statement token: %s", first)
	}
	return ret, nil
}
