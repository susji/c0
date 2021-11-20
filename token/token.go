package token

import (
	"errors"
	"fmt"
	"strings"

	"github.com/susji/c0/span"
)

var EOT = errors.New("end of tokens")

// Tokens implements a FIFO for individual tokens.
type Tokens struct {
	toks []Token
}

type Token struct {
	span  span.Span
	kind  Kind
	value string
}

func New(kind Kind, span span.Span, value string) Token {
	if !validkind(kind) {
		panic(fmt.Sprintf("invalid token kind: %v", kind))
	}
	return Token{
		kind:  kind,
		value: value,
		span:  span,
	}
}

type Kind int

const (
	Id = iota
	DecNum
	HexNum
	StrLit
	LibLit
	ChrLit
	LParen
	RParen
	LBrack
	RBrack // 10
	LCurly
	RCurly
	Comma
	Semicolon
	Exclam
	Worm
	Plus
	Minus
	Star
	Dot // 20
	Arrow
	Slash
	Percent
	Lt
	Gt
	DLt
	DGt
	Le
	Ge
	Eq // 30
	Ne
	Assign
	AssignPlus
	AssignMinus
	AssignStar
	AssignSlash
	AssignPercent
	AssignDLt
	AssignDGt
	AssignAmpersand // 40
	AssignHat
	AssignPipe
	Ampersand
	Hat
	Pipe
	DAmpersand
	DPipe
	Quest
	Colon
	DPlus // 50
	DMinus
	UseStrLit
	UseLibLit
	Brackets
	True
	False
	Null
	CommentOne
	CommentMulti
)

var toknames = [...]string{
	"id",
	"decnum",
	"hexnum",
	"strlit",
	"liblit",
	"chrlit",
	"(",
	")",
	"[",
	"]",
	"{",
	"}",
	",",
	";",
	"!",
	"~",
	"+",
	"-",
	"*",
	".",
	"->",
	"/",
	"%",
	"<",
	">",
	"<<",
	">>",
	"<=",
	">=",
	"==",
	"!=",
	"=",
	"+=",
	"-=",
	"*=",
	"/=",
	"%=",
	"<<=",
	">>=",
	"&=",
	"^=",
	"|=",
	"&",
	"^",
	"|",
	"&&",
	"||",
	"?",
	":",
	"++",
	"--",
	"#usestr",
	"#uselib",
	"[]",
	"true",
	"false",
	"NULL",
	"//comment",
	"/* comment */",
}

func (k Kind) String() string {
	return toknames[k]
}

func validkind(kind Kind) bool {
	return kind >= 0 && int(kind) <= (len(toknames)-1)
}

func (tok *Token) String() string {
	switch tok.kind {
	case Id, HexNum, DecNum:
		return fmt.Sprintf("%s", tok.value)
	case StrLit, ChrLit:
		return fmt.Sprintf("%q", tok.value)
	case LibLit:
		return fmt.Sprintf("<%s>", tok.value)
	case CommentOne:
		return fmt.Sprintf("// %s", tok.value)
	case CommentMulti:
		return fmt.Sprintf("/* %s */", tok.value)
	default:
		return fmt.Sprintf("%q", toknames[tok.kind])
	}
}

func (tok *Token) Value() string {
	return tok.value
}

func (tok *Token) Kind() Kind {
	return tok.kind
}

func (tok *Token) Lineno() int {
	return tok.span.Lineno0
}

func (tok *Token) Col() int {
	return tok.span.Col0
}

func (tok *Token) Span() span.Span {
	return tok.span
}

func (toks *Tokens) Add(tok Token) *Tokens {
	toks.toks = append(toks.toks, tok)
	return toks
}

func (toks *Tokens) String() string {
	b := &strings.Builder{}
	for _, tok := range toks.toks {
		b.WriteString(
			fmt.Sprintf("[%d:%d] %s\n", tok.Lineno(), tok.Col(), tok.String()))
	}
	return b.String()
}

func (toks *Tokens) Len() int {
	return len(toks.toks)
}

func (toks *Tokens) Pop() *Token {
	if toks.Len() == 0 {
		return nil
	}
	if toks.Len() == 1 {
		tok := &toks.toks[0]
		toks.toks = nil
		return tok
	}
	var tok Token
	tok, toks.toks = toks.toks[0], toks.toks[1:]
	return &tok
}

// Peek returns the current token-to-be-parsed. It never returns comment
// tokens.
func (toks *Tokens) Peek() *Token {
nocoms:
	for {
		if toks.Len() == 0 {
			return nil
		}
		switch toks.toks[0].Kind() {
		case CommentOne, CommentMulti:
			toks.Pop()
			continue nocoms
		default:
			return &toks.toks[0]
		}
	}
}

// PeekAll returns the current token-to-be-parsed. Unlike Peek, it never
// discriminates based on token kind.
func (toks *Tokens) PeekAll() *Token {
	if toks.Len() == 0 {
		return nil
	}
	return &toks.toks[0]
}

func (toks *Tokens) Accept(kind Kind) error {
	cur := toks.Peek()
	if cur == nil {
		return EOT
	}
	got := cur.Kind()
	if got != kind {
		return fmt.Errorf("expecting %q, got %v", toknames[kind], cur)
	}
	toks.Pop()
	return nil
}

func (toks *Tokens) Find(kinds ...Kind) *Token {
	find := map[Kind]struct{}{}
	for _, kind := range kinds {
		find[kind] = struct{}{}
	}
	for {
		cur := toks.Peek()
		if cur == nil {
			return nil
		}
		if _, ok := find[cur.Kind()]; ok {
			return cur
		}
		toks.Pop()
	}
}
