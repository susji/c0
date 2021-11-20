package parse

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/susji/c0/analyze"
	"github.com/susji/c0/node"
	"github.com/susji/c0/token"
)

var ErrAtomTypedef = errors.New("atom is typedef")

func precedenceb(tok *token.Token) int {
	// We do not give a precedence value for assignment operators as they are
	// apparently not meant to be interpreted as binary operators within
	// expressions -- see "<simple>" vs. "<exp>". This decision has the
	// downside that our grammar will not permit chained assignments, eg.
	// "a = b = c = 10".
	switch tok.Kind() {
	// assignment operators would go here
	case token.Quest, token.Colon:
		return 0
	case token.DPipe:
		return 1
	case token.DAmpersand:
		return 2
	case token.Pipe:
		return 3
	case token.Hat:
		return 4
	case token.Ampersand:
		return 5
	case token.Eq, token.Ne:
		return 6
	case token.Lt, token.Gt, token.Le, token.Ge:
		return 7
	case token.DLt, token.DGt:
		return 8
	case token.Plus, token.Minus:
		return 9
	case token.Star, token.Slash, token.Percent:
		return 10
	case token.Arrow, token.Dot:
		// token.LParen, token.LBrack are treated as special cases and they do
		// not make use of the precedence machinery.
		return 11
	default:
		panic(fmt.Sprintf("invalid binary operator: %s", tok))
	}
}

func precedenceu(tok *token.Token) int {
	switch tok.Kind() {
	case token.Star, token.Exclam, token.Worm, token.Minus,
		token.DPlus, token.DMinus, token.Ampersand:
		return 10
	default:
		panic(fmt.Sprintf("invalid unary operator: %s", tok))
	}
}

func isleftassocb(tok *token.Token) bool {
	switch tok.Kind() {
	case token.Quest, token.Colon,
		token.Assign, token.AssignPlus, token.AssignMinus,
		token.AssignStar, token.AssignSlash, token.AssignPercent,
		token.AssignAmpersand, token.AssignHat, token.AssignPipe,
		token.AssignDLt, token.AssignDGt:
		return false
	default:
		return true
	}
}

var tok_to_unop = map[token.Kind]node.KindOpUn{
	token.Minus:     node.OPUN_NEG,
	token.Exclam:    node.OPUN_LOGNOT,
	token.Worm:      node.OPUN_BITNOT,
	token.Star:      node.OPUN_DEREF,
	token.DPlus:     node.OPUN_ADDONE,
	token.DMinus:    node.OPUN_SUBONE,
	token.Ampersand: node.OPUN_ADDROF,
}

var tok_to_binop = map[token.Kind]node.KindOpBin{
	token.Plus:       node.OPBIN_ADD,
	token.Minus:      node.OPBIN_SUB,
	token.Star:       node.OPBIN_MUL,
	token.Slash:      node.OPBIN_DIV,
	token.Percent:    node.OPBIN_MOD,
	token.LBrack:     node.OPBIN_ARRSUB,
	token.LParen:     node.OPBIN_FUNCALL,
	token.Dot:        node.OPBIN_STRUCTDEC,
	token.Arrow:      node.OPBIN_STRUCTPTRDEC,
	token.Quest:      node.OPBIN_TERNARYCOND,
	token.Colon:      node.OPBIN_TERNARYVALS,
	token.DGt:        node.OPBIN_SHIFTR,
	token.DLt:        node.OPBIN_SHIFTL,
	token.Le:         node.OPBIN_LE,
	token.Ge:         node.OPBIN_GE,
	token.Lt:         node.OPBIN_LT,
	token.Gt:         node.OPBIN_GT,
	token.Eq:         node.OPBIN_EQ,
	token.Ne:         node.OPBIN_NE,
	token.Ampersand:  node.OPBIN_BAND,
	token.Pipe:       node.OPBIN_BOR,
	token.Hat:        node.OPBIN_BXOR,
	token.DAmpersand: node.OPBIN_AND,
	token.DPipe:      node.OPBIN_OR,
}

var binop_tail = map[token.Kind]token.Kind{
	token.LBrack: token.RBrack,
}

func (p *Parser) expratom(toks *token.Tokens) (node.Node, error) {
	this := toks.Peek()
	if this == nil {
		return nil, EOT
	}
	if unop, ok := tok_to_unop[this.Kind()]; ok {
		// All unary operators bind right, hence the +1 to their precedence.
		nextminprec := precedenceu(this) + 1
		toks.Pop()
		n, err := p.exprparse(toks, nextminprec)
		if err != nil {
			return nil, err
		}
		return node.Store(this, &node.OpUnary{
			Op: unop,
			To: n,
		}), nil
	}
	switch this.Kind() {
	case token.LParen:
		// For an expression atom, '(' can mean two things:
		//   - casting, eg. "(int *)"
		//   - a subexpression, eg. "(...)"
		toks.Pop()
		castkind, err := p.Type(toks)
		if err == nil {
			if err := toks.Accept(token.RParen); err != nil {
				return nil, p.errorf(this, "invalid cast: %w", err)
			}
			castwhat, err := p.expratom(toks)
			if err != nil {
				return nil, err
			}
			return node.Store(this, &node.Cast{
				To:   castkind,
				What: castwhat,
			}), nil
		}
		parexpr, err := p.exprparse(toks, 0)
		if err != nil {
			return nil, err
		}
		if err := toks.Accept(token.RParen); err != nil {
			return nil, p.errorf(this, "unbalanced parentheses: %w", err)
		}
		return parexpr, nil
	case token.DecNum, token.HexNum:
		toks.Pop()
		base := 10
		val := this.Value()
		if this.Kind() == token.HexNum {
			base = 16
			if val != "0" {
				val = val[2:]
			}
		}
		pi, err := strconv.ParseInt(val, base, 32)
		if err != nil {
			return nil, p.errorf(this, "invalid integer: %w", err)
		}
		return node.Store(
				this, &node.Numeric{Value: int32(pi), Base: base}),
			nil
	case token.Id:
		iv := this.Value()
		if p.IsTypedef(iv) {
			return nil, ErrAtomTypedef
		}
		switch iv {
		case "void":
			// As "void" is not accepted in expressions, then this must not be
			// a valid expression parse.
			return nil, errors.New("`void' not permitted in expressions")
		case "alloc", "alloc_array":
			toks.Pop()
			if err := toks.Accept(token.LParen); err != nil {
				return nil, p.errorf(this, "%s missing '('", iv)
			}
			ak, err := p.Type(toks)
			if err != nil {
				return nil, p.errorf(this, "invalid type for %s: %w", iv, err)
			}
			var ret node.Node
			if iv == "alloc_array" {
				if err := toks.Accept(token.Comma); err != nil {
					return nil, p.errorf(this,
						"alloc_array missing size expression: %w", err)
				}
				n, err := p.Expr(toks)
				if err != nil {
					return nil, p.errorf(this,
						"invalid size expression for alloc_array: %w", err)
				}
				ret = node.Store(this, &node.AllocArray{
					Kind: ak,
					N:    n,
				})
			} else {
				ret = node.Store(this, &node.Alloc{Kind: ak})
			}
			if err := toks.Accept(token.RParen); err != nil {
				return nil, p.errorf(this, "%s missing ')'", iv)
			}
			return ret, nil
		default:
			if analyze.IsReserved(this.Value()) {
				return nil, fmt.Errorf(
					"reserved identifier %q in expression", iv)
			}
			toks.Pop()
			return node.Store(this, &node.Variable{Value: iv}), nil
		}
	case token.True, token.False:
		toks.Pop()
		return node.Store(this, &node.Bool{Value: this.Kind() == token.True}), nil
	case token.Null:
		toks.Pop()
		return node.Store(this, &node.Null{}), nil
	case token.StrLit:
		toks.Pop()
		return node.Store(this, &node.StrLit{Value: this.Value()}), nil
	case token.ChrLit:
		toks.Pop()
		return node.Store(this, &node.ChrLit{Value: []rune(this.Value())[0]}), nil
	default:
		return nil, p.errorf(this, "invalid expression atom: %q", this.Kind())
	}
}

func (p *Parser) exprparse(toks *token.Tokens, minprec int) (node.Node, error) {
	lhs, err := p.expratom(toks)
	if err != nil {
		return nil, err
	}
	op := toks.Peek()
out:
	for {
		op = toks.Peek()
		if op == nil {
			break out
		}
		binop, ok := tok_to_binop[op.Kind()]
		if !ok {
			break out
		}
		// We treat function calls () and array subscripts [] as special,
		// maximally greedy postfix operators -- the C0 Reference specifies
		// them with the highest precedence. All other binary operators are
		// treated with the precedence-climbing machinery.
		//
		// NB: We also do not validate whether '?' is followed by ':' here.
		//     That is done at a later stage in syntax checking.
		switch op.Kind() {
		case token.LBrack:
			// Array subscript.
			toks.Pop()
			index, err := p.exprparse(toks, 0)
			if err != nil {
				return nil, p.errorf(
					op,
					"invalid function argument: %w", err)
			}
			if err := toks.Accept(token.RBrack); err != nil {
				return nil, p.errorf(op, "unbalanced array subscript: %w", err)
			}
			lhs = node.Store(op, &node.OpBinary{
				Op:    binop,
				Left:  lhs,
				Right: index,
			})
			continue out
		case token.LParen:
			// Function call.
			toks.Pop()
			args := []node.Node{}
			// We may have the case without arguments, ie. "()".
			if err := toks.Accept(token.RParen); err != nil {
				for toks.Peek() != nil {
					arg, err := p.exprparse(toks, 0)
					if err != nil {
						return nil, p.errorf(
							op,
							"invalid function argument: %w", err)
					}
					args = append(args, arg)
					if err := toks.Accept(token.Comma); err == nil {
						// ',' -> more args
						continue
					} else if err := toks.Accept(token.RParen); err == nil {
						// ')' => end of args
						break
					} else {
						// no ')' or ',' => error
						return nil, p.errorf(
							op,
							"unbalanced parentheses in function call: %w", err)
					}
				}
			}
			lhs = node.Store(op, &node.OpBinary{
				Op:    binop,
				Left:  lhs,
				Right: &node.Args{Value: args},
			})
			continue out
		}
		// All of this is just vanilla precedence-climbing.
		prec := precedenceb(op)
		if prec < minprec {
			break out
		}
		nextminprec := prec
		if isleftassocb(op) {
			nextminprec++
		}
		toks.Pop()
		rhs, err := p.exprparse(toks, nextminprec)
		if err != nil {
			return nil, err
		}
		lhs = node.Store(op, &node.OpBinary{
			Op:    binop,
			Left:  lhs,
			Right: rhs,
		})
	}
	return lhs, nil
}

func (p *Parser) Expr(toks *token.Tokens) (node.Node, error) {
	// Grammar taken from the C0 Reference and modified to ease parsing.
	//
	// <exp>    = <prefix> <suffix>
	// <prefix> = <num> | <strlit> | <chrlit> | true | false | NULL
	//          | "(" <exp> ")"
	//          | <unop> <exp>
	//          | <exp> "[" <exp> "]"
	//          | <exp> "(" [ <exp> ("," <exp> )*] ")"
	//          | <exp> "." <fid>
	//          | <exp> "->" <fid>
	//          | <vid>
	// <suffix> =
	//          | <binop> <exp>
	//          | "?" <exp> ":" <exp>
	//          | ε
	//
	// Our precedence climbing is implemented mainly by following Norvell at
	//
	//   https://www.engr.mun.ca/~theo/Misc/exp_parsing.htm#climbing
	//
	// NB: We do not handle assignment operators here.
	return p.exprparse(toks, 0)
}
