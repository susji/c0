package parse

import (
	"errors"

	"github.com/susji/c0/node"
	"github.com/susji/c0/token"
)

var tok_to_asnop = map[token.Kind]node.KindOpAsn{
	token.Assign:          node.OPASN_PLAIN,
	token.AssignPlus:      node.OPASN_ADD,
	token.AssignMinus:     node.OPASN_SUB,
	token.AssignStar:      node.OPASN_MUL,
	token.AssignSlash:     node.OPASN_DIV,
	token.AssignPercent:   node.OPASN_MOD,
	token.AssignDLt:       node.OPASN_LSHIFT,
	token.AssignDGt:       node.OPASN_RSHIFT,
	token.AssignAmpersand: node.OPASN_AND,
	token.AssignHat:       node.OPASN_XOR,
	token.AssignPipe:      node.OPASN_OR,
}

var tok_to_stmtsuffix = map[token.Kind]node.KindOpUn{
	token.DPlus:  node.OPUN_ADDONESUFFIX,
	token.DMinus: node.OPUN_SUBONESUFFIX,
}

// SimpleStmt roughly implements "<simple>". As lvalues are a limited subset of
// expressions, we again adhere to a different parsing grammar. Later on, in
// syntax-checking, we make sure that we have received acceptable lvalues.
//
// This is our modified grammar:
//
// <simple> = <tp> <vid> [ "="" <exp> ]
//          | <exp> <asnop> <exp>
//          | <exp> "++"
//          | <exp> "--"
//          | <exp>
//
func (p *Parser) SimpleStmt(toks *token.Tokens) (node.Node, error) {
	first := toks.Peek()
	if first == nil {
		return nil, EOT
	}
	// The first branch here covers the case, where we have an
	// expression-looking thing, which may be a lvalue. At this stage, we'll
	// parse any expression and consider it a potential lvalue, as lvalues form
	// a subset of expressions. This must then be syntax-checked later on.
	lv, exprerr := p.Expr(toks)
	if exprerr == nil {
		next := toks.Peek()
		if next == nil {
			return lv, nil
		}
		if ak, ok := tok_to_asnop[next.Kind()]; ok {
			// Looks like an assignment statement.
			toks.Pop()
			rv, err := p.Expr(toks)
			if err != nil {
				return nil, p.errorf(next, "invalid rvalue: %w", err)
			}
			return node.Store(first, &node.OpAssign{
				Op:   ak,
				To:   lv,
				What: rv,
			}), nil
		} else if ak, ok := tok_to_stmtsuffix[next.Kind()]; ok {
			// Suffix-operation statement.
			toks.Pop()
			return node.Store(next, &node.OpUnary{
				Op: ak,
				To: lv,
			}), nil
		}
		// A plain expression-looking thing.
		return lv, nil
	}
	// <tp> <vid> ["="" <exp>]
	if vd, err := p.VarDecl(toks); err == nil {
		var av node.Node
		if toks.Peek() != nil && toks.Peek().Kind() == token.Assign {
			toks.Pop()
			av, err = p.Expr(toks)
			if err != nil {
				return nil, p.errorf(
					first,
					"erroneous variable assignment: %w", err)
			}
		}
		return node.Store(first, &node.OpAssign{
			Op:   node.OPASN_PLAIN,
			To:   vd,
			What: av,
		}), nil
	}
	// We prefer the expression error, if nothing else was found. For instance,
	// a reserved word might have been encountered.
	return nil, exprerr
}

func (p *Parser) Block(toks *token.Tokens) (*node.Block, error) {
	first := toks.Peek()
	if first == nil {
		return nil, EOT
	}
	if first.Kind() == token.LCurly {
		toks.Pop()
	} else {
		return nil, errors.New("not a block")
	}
	stmts := []node.Node{}
	inerror := false
	for toks.Peek() != nil && toks.Peek().Kind() != token.RCurly {
		stmt, err := p.Stmt(toks)
		if err != nil {
			inerror = true
			// Attempt finding next statement for the block for more errors.
			toks.Find(token.Semicolon, token.RCurly)
			toks.Pop()
		}
		stmts = append(stmts, stmt)
	}
	if err := toks.Accept(token.RCurly); err != nil {
		return nil, p.errorf(
			first,
			"block not terminated: %w", err)
	}
	if inerror {
		return nil, errors.New("block contained errors")
	}
	return node.Store(first, &node.Block{Value: stmts}).(*node.Block), nil
}

func (p *Parser) Stmt(toks *token.Tokens) (node.Node, error) {
	first := toks.Peek()
	if first == nil {
		return nil, EOT
	}
	// Plain block?
	if block, err := p.Block(toks); err == nil {
		return block, nil
	}
	switch first.Value() {
	case "if":
		toks.Pop()
		if err := toks.Accept(token.LParen); err != nil {
			return nil, p.errorf(first, "`if' condition missing '('")
		}
		cond, err := p.Expr(toks)
		if err != nil {
			return nil, err
		}
		if err := toks.Accept(token.RParen); err != nil {
			return nil, p.errorf(first, "`if' condition missing ')'")
		}
		bodytrue, err := p.Stmt(toks)
		if err != nil {
			return nil, err
		}
		ret := node.Store(first, &node.If{
			Cond:  cond,
			True:  bodytrue,
			False: nil,
		}).(*node.If)
		next := toks.Peek()
		if next == nil || !(next.Kind() == token.Id && next.Value() == "else") {
			return ret, nil
		}
		toks.Pop()
		bodyfalse, err := p.Stmt(toks)
		if err != nil {
			return nil, err
		}
		ret.False = bodyfalse
		return ret, nil
	case "while":
		toks.Pop()
		if err := toks.Accept(token.LParen); err != nil {
			return nil, p.errorf(first, "`while' condition missing '('")
		}
		cond, err := p.Expr(toks)
		if err != nil {
			return nil, err
		}
		if err := toks.Accept(token.RParen); err != nil {
			return nil, p.errorf(first, "`while' condition missing ')'")
		}
		body, err := p.Stmt(toks)
		if err != nil {
			return nil, err
		}
		return node.Store(first, &node.While{
			Cond: cond,
			Body: body,
		}), nil
	case "for":
		toks.Pop()
		if err := toks.Accept(token.LParen); err != nil {
			return nil, p.errorf(first, "`for' missing '('")
		}
		init, err := p.SimpleStmt(toks)
		if err != nil {
			return nil, err
		}
		if err := toks.Accept(token.Semicolon); err != nil {
			return nil, p.errorf(first, "`for' missing ';' after initializer")
		}
		cond, err := p.Expr(toks)
		if err != nil {
			return nil, err
		}
		if err := toks.Accept(token.Semicolon); err != nil {
			return nil, p.errorf(first, "`for' missing ';' after condition")
		}
		oneach, err := p.SimpleStmt(toks)
		if err != nil {
			return nil, err
		}
		if err := toks.Accept(token.RParen); err != nil {
			return nil, p.errorf(first, "`for' missing ')'")
		}
		body, err := p.Stmt(toks)
		if err != nil {
			return nil, err
		}
		return node.Store(first, &node.For{
			Init:   init,
			Cond:   cond,
			OnEach: oneach,
			Body:   body,
		}), nil
	case "return":
		toks.Pop()
		if err := toks.Accept(token.Semicolon); err == nil {
			return node.Store(first, &node.Return{Expr: nil}), nil
		}
		expr, err := p.Expr(toks)
		if err != nil {
			return nil, p.errorf(first, "invalid return expression: %w", err)
		}
		if err := toks.Accept(token.Semicolon); err != nil {
			return nil, p.errorf(first, "return missing ';'")
		}
		return node.Store(first, &node.Return{Expr: expr}), nil
	case "assert", "error":
		which := first.Value()
		toks.Pop()
		if err := toks.Accept(token.LParen); err != nil {
			return nil, p.errorf(first, "%s missing '('", which)
		}
		expr, err := p.Expr(toks)
		if err != nil {
			return nil, p.errorf(first, "invalid %s statement: %w", which, err)
		}
		if err := toks.Accept(token.RParen); err != nil {
			return nil, p.errorf(first, "%s statement missing ')'", which)
		}
		if err := toks.Accept(token.Semicolon); err != nil {
			return nil, p.errorf(first, "%s statement missing ';'", which)
		}
		var ret node.Node
		switch which {
		case "assert":
			ret = node.Store(first, &node.Assert{Expr: expr})
		case "error":
			ret = node.Store(first, &node.Error{Expr: expr})
		default:
			panic("not happening")
		}
		return ret, nil
	case "break":
		toks.Pop()
		if err := toks.Accept(token.Semicolon); err != nil {
			return nil, p.errorf(first, "break statement missing ';'")
		}
		return node.Store(first, &node.Break{}), nil
	case "continue":
		toks.Pop()
		if err := toks.Accept(token.Semicolon); err != nil {
			return nil, p.errorf(first, "continue statement missing ';'")
		}
		return node.Store(first, &node.Continue{}), nil
	default:
		if ss, err := p.SimpleStmt(toks); err == nil {
			if err := toks.Accept(token.Semicolon); err != nil {
				return nil, p.errorf(first, "statement missing ';'")
			}
			return ss, nil
		} else {
			return nil, errors.New("not a simple statement")
		}
	}
}
