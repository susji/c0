package lex

import (
	"fmt"

	pr "github.com/susji/c0/primitives"
	"github.com/susji/c0/span"
	"github.com/susji/c0/token"
)

// Whitespace-related helpers
var Whitespace = pr.Runes(" \t\r\v")
var WhitespaceN = Whitespace.OneOrMore()
var Whitespace0 = Whitespace.ZeroOrMore()
var Linefeed = pr.Rune('\n')

// Comments
var CommentOneline = pr.Discard(pr.String("//")).
	And(pr.ExceptString("\n").ZeroOrMore())
var CommentMultiline = pr.String("/*").Discard().
	And(pr.ExceptString("*/").ZeroOrMore()).
	And(pr.Discard(pr.String("*/").Fatal(`no matching "*/" for comment`)))

// Identifiers
var plow = pr.RuneRange('a', 'z')
var pupp = pr.RuneRange('A', 'Z')
var pdig = pr.RuneRange('0', '9')
var pus = pr.Rune('_')
var Identifier = plow.Or(pupp).Or(pus).Or(plow).
	And(pupp.Or(pus).Or(plow).Or(pdig).ZeroOrMore())

// String literals
// escapebuilder is also used for character literals
var escapebuilder = func(wantstring bool) pr.Parser {
	type escpair struct {
		src string
		dst rune
	}
	eps := []pr.Parser{}
	escpairs := []escpair{
		{`\n`, '\n'},
		{`\t`, '\t'},
		{`\v`, '\v'},
		{`\b`, '\b'},
		{`\r`, '\r'},
		{`\f`, '\f'},
		{`\a`, '\a'},
		{`\\`, '\\'},
	}
	if wantstring {
		escpairs = append(escpairs, escpair{`\"`, '"'})
	} else {
		escpairs = append(escpairs, escpair{`\'`, '\''})
		escpairs = append(escpairs, escpair{`\0`, 0})
	}
	for _, cur := range escpairs {
		c := cur
		this := pr.String(c.src).
			Map(func(from pr.ResultValue) pr.ResultValue {
				from = from[:len(from)-2]
				from = append(from, c.dst)
				return from
			})
		eps = append(eps, this)
	}
	return pr.AnyOf(eps...)
}
var pstrlitq1 = pr.Chomp('"')
var pstrlitq2 = pr.Chomp('"').Fatal("missing closing '\"'")
var pstrlitch = pr.ExceptRunes("\"\\")
var StrLit = pr.Discard(pstrlitq1).
	And(pstrlitch.Or(escapebuilder(true)).ZeroOrMore()).
	And(pr.Discard(pstrlitq2))

// Character (rune) literal
var pchrlitq1 = pr.Chomp('\'')
var pchrlitq2 = pr.Chomp('\'').Fatal(`missing closing "'"`)
var pchrlitch = pr.ExceptRunes("'\\")
var pchrlitesc = escapebuilder(false).Fatal("invalid character literal")
var ChrLit = pr.Discard(pchrlitq1).
	And(pchrlitch.Or(pchrlitesc)).
	And(pr.Discard(pchrlitq2))

// Library literal
var pliblitq1 = pr.Chomp('<')
var pliblitq2 = pr.Chomp('>')
var pliblitch = pr.ExceptRunes(">")
var LibLit = pr.Discard(pliblitq1).
	And(pliblitch.OneOrMore()).
	And(pr.Discard(pliblitq2))

// Separators and operators
var Separators = pr.Runes("()[]{},;")

// '*' and '-' are lexed as binary ops
var OpUnary = pr.Runes("!~").Or(pr.String("[]"))

// Separated for precedence.
var OpSet = pr.Rune('=')

// Note the greediness issue when parsing, eg. '<' vs `<='.
var OpBinary = pr.Strings("<<", ">>", "<=", ">=", "==", "!=", "&&", "||", "->").
	Or(pr.Runes(".*/%+-<>&^|?:"))

// Note the greediness with '='
var OpAssign = pr.Strings(
	"+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "^=", "|=").
	Error("not an assignment operator")
var OpPostfix = pr.Strings("--", "++")

// Compiler directives
var dirend = pr.Discard(pr.Rune('\n').Fatal("#use directive missing newline"))
var usestart = pr.String("#use").And(Whitespace).Discard()
var DirectiveUseLib = usestart.
	And(LibLit).
	And(pr.Discard(Whitespace0)).
	And(dirend)
var DirectiveUseStr = usestart.
	And(StrLit).
	And(pr.Discard(Whitespace0)).
	And(dirend)

// Numeric values
var pdig1 = pr.RuneRange('1', '9')
var DecNum = pdig1.And(pdig.ZeroOrMore())
var HexNum = pr.Rune('0').
	And(pr.Runes("xX").
		And(pdig.
			Or(pr.RuneRange('a', 'f')).
			Or(pr.RuneRange('A', 'F')).
			OneOrMore().Fatal("invalid hexnum")).
		Or(pr.Epsilon()))

// Special identifiers
var SpecialIds = pr.Strings("true", "false", "NULL")

func Lex(what []rune) (*token.Tokens, []error) {
	toks := &token.Tokens{}
	state := pr.NewState(what)
	var lineno0, col0 int

	nt := func(st *pr.State, kind token.Kind) {
		lineno, col := st.Pos()
		span := span.Span{
			Lineno0: lineno0,
			Col0:    col0,
			Lineno:  lineno,
			Col:     col,
		}
		toks.Add(token.New(kind, span, st.String()))
	}
	// Precedence has to be considered here as `Identifier' will be the final
	// catch-all for plain wordy things.
	all := WhitespaceN.Pipe(func(curstate *pr.State) {
		// Whitespace is ignored.
	}).
		Or(Linefeed.Pipe(func(curstate *pr.State) {
			// Lone linefeeds are also ignored.
		})).
		Or(CommentOneline.Pipe(func(curstate *pr.State) {
			nt(curstate, token.CommentOne)
		})).
		Or(CommentMultiline.Pipe(func(curstate *pr.State) {
			nt(curstate, token.CommentMulti)
		})).
		Or(HexNum.Pipe(func(curstate *pr.State) {
			nt(curstate, token.HexNum)
		})).
		Or(DecNum.Pipe(func(curstate *pr.State) {
			nt(curstate, token.DecNum)
		})).
		Or(StrLit.Pipe(func(curstate *pr.State) {
			nt(curstate, token.StrLit)
		})).
		Or(ChrLit.Pipe(func(curstate *pr.State) {
			nt(curstate, token.ChrLit)
		})).
		Or(OpPostfix.Pipe(func(curstate *pr.State) {
			got := curstate.String()
			switch got {
			case "++":
				nt(curstate, token.DPlus)
			case "--":
				nt(curstate, token.DMinus)
			default:
				panic(fmt.Sprintf("unrecognized postfix operator: %s", got))
			}
		})).
		Or(OpAssign.Pipe(func(curstate *pr.State) {
			got := curstate.String()
			switch got {
			case "+=":
				nt(curstate, token.AssignPlus)
			case "-=":
				nt(curstate, token.AssignMinus)
			case "*=":
				nt(curstate, token.AssignStar)
			case "/=":
				nt(curstate, token.AssignSlash)
			case "%=":
				nt(curstate, token.AssignPercent)
			case "<<=":
				nt(curstate, token.AssignDLt)
			case ">>=":
				nt(curstate, token.AssignDGt)
			case "&=":
				nt(curstate, token.AssignAmpersand)
			case "^=":
				nt(curstate, token.AssignHat)
			case "|=":
				nt(curstate, token.AssignPipe)
			default:
				panic(fmt.Sprintf("unrecognized assignment operator: %s", got))
			}
		})).
		Or(OpBinary.Pipe(func(curstate *pr.State) {
			got := curstate.String()
			switch got {
			case "<<":
				nt(curstate, token.DLt)
			case ">>":
				nt(curstate, token.DGt)
			case "<=":
				nt(curstate, token.Le)
			case ">=":
				nt(curstate, token.Ge)
			case "==":
				nt(curstate, token.Eq)
			case "!=":
				nt(curstate, token.Ne)
			case "&&":
				nt(curstate, token.DAmpersand)
			case "||":
				nt(curstate, token.DPipe)
			case ".":
				nt(curstate, token.Dot)
			case "*":
				nt(curstate, token.Star)
			case "/":
				nt(curstate, token.Slash)
			case "%":
				nt(curstate, token.Percent)
			case "+":
				nt(curstate, token.Plus)
			case "-":
				nt(curstate, token.Minus)
			case "<":
				nt(curstate, token.Lt)
			case ">":
				nt(curstate, token.Gt)
			case "&":
				nt(curstate, token.Ampersand)
			case "^":
				nt(curstate, token.Hat)
			case "|":
				nt(curstate, token.Pipe)
			case "?":
				nt(curstate, token.Quest)
			case ":":
				nt(curstate, token.Colon)
			case "->":
				nt(curstate, token.Arrow)
			default:
				panic(fmt.Sprintf("unrecognized binary operator: %q", got))
			}
		})).
		Or(OpSet.Pipe(func(curstate *pr.State) {
			nt(curstate, token.Assign)
		})).
		Or(OpUnary.Pipe(func(curstate *pr.State) {
			got := curstate.String()
			switch got {
			case "!":
				nt(curstate, token.Exclam)
			case "~":
				nt(curstate, token.Worm)
			case "[]":
				nt(curstate, token.Brackets)
			default:
				panic(fmt.Sprintf("unrecognized unary operator: %q", got))
			}
		})).
		Or(Separators.Pipe(func(curstate *pr.State) {
			got := curstate.String()
			switch got {
			case "(":
				nt(curstate, token.LParen)
			case ")":
				nt(curstate, token.RParen)
			case "[":
				nt(curstate, token.LBrack)
			case "]":
				nt(curstate, token.RBrack)
			case "{":
				nt(curstate, token.LCurly)
			case "}":
				nt(curstate, token.RCurly)
			case ",":
				nt(curstate, token.Comma)
			case ";":
				nt(curstate, token.Semicolon)
			default:
				panic(fmt.Sprintf("unrecognized separator: %s", got))
			}
		})).
		Or(DirectiveUseLib.Pipe(func(curstate *pr.State) {
			nt(curstate, token.UseLibLit)
		})).
		Or(DirectiveUseStr.Pipe(func(curstate *pr.State) {
			nt(curstate, token.UseStrLit)
		})).
		Or(SpecialIds.Pipe(func(curstate *pr.State) {
			got := curstate.String()
			switch got {
			case "true":
				nt(curstate, token.True)
			case "false":
				nt(curstate, token.False)
			case "NULL":
				nt(curstate, token.Null)
			default:
				panic(fmt.Sprintf("unknown special identifier: %s", got))
			}
		})).
		Or(Identifier.Pipe(func(curstate *pr.State) {
			nt(curstate, token.Id)
		})).Discard()

	prevlen := len(state.Left())
	var errs []error
	for state.LenLeft() > 0 {
		lineno0, col0 = state.Pos()
		res := all.Do(state)
		err := res.Error()
		switch err {
		case nil:
		default:
			errs = append(errs, err)
		}
		state = res.State()
		curlen := len(state.Left())
		// If we managed to lex nothing, we need to bail.
		if prevlen == curlen {
			break
		}
		prevlen = curlen
	}
	return toks, errs
}
