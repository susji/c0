package node

import (
	"fmt"
	"strings"

	"github.com/susji/c0/token"
)

type VarDecls []VarDecl

type Common struct {
	id NodeId
}

// Node is the interface, which must be implemented by all syntax tree nodes.
// All Nodes must be able to produce their unique identifier, the Token which
// they originated from, and stringify themselves in a sexpr.
type Node interface {
	String() string
	Id() NodeId
	Tok() *token.Token
}

// Loop is a pseudo-interface used to tag valid loop constructs, namely "while"
// and "for".
type Loop interface {
	Loop()
}

type Variable struct {
	*Common
	Value string
}

type Numeric struct {
	*Common
	Value int32
	Base  int
}

type StructForwardDecl struct {
	*Common
	Value string
}

type Struct struct {
	*Common
	Name    string
	Members VarDecls
}

type StrLit struct {
	*Common
	Value string
}

type ChrLit struct {
	*Common
	Value rune
}

type LibLit struct {
	*Common
	Value string
}

type Bool struct {
	*Common
	Value bool
}

type Null struct {
	*Common
}

type Args struct {
	*Common
	Value []Node
}

type OpUnary struct {
	*Common
	Op KindOpUn
	To Node
}

type OpBinary struct {
	*Common
	Op          KindOpBin
	Left, Right Node
}

type OpAssign struct {
	*Common
	Op       KindOpAsn
	To, What Node
}

type Block struct {
	*Common
	Value []Node
}

type If struct {
	*Common
	Cond        Node
	True, False Node
}

type For struct {
	*Common
	Init, Cond, OnEach, Body Node
}

type While struct {
	*Common
	Cond, Body Node
}

type Return struct {
	*Common
	Expr Node
}

type Assert struct {
	*Common
	Expr Node
}

type Error struct {
	*Common
	Expr Node
}

type Alloc struct {
	*Common
	Kind Kind
}

type AllocArray struct {
	*Common
	Kind Kind
	N    Node
}

type Typedef struct {
	*Common
	Name string
	Kind Kind
}

type TypedefFunc struct {
	*Common
	Name    string
	Returns Kind
	Params  VarDecls
}

type Break struct {
	*Common
}

type Continue struct {
	*Common
}

type Cast struct {
	*Common
	To   Kind
	What Node
}

type KindOpBin int
type KindOpUn int
type KindOpAsn int

const (
	OPUN_NEG = iota
	OPUN_LOGNOT
	OPUN_BITNOT
	OPUN_DEREF
	OPUN_ADDONE
	OPUN_SUBONE
	OPUN_ADDROF
	OPUN_ADDONESUFFIX
	OPUN_SUBONESUFFIX
)

var opunnames = [...]string{
	"u-",
	"!",
	"~",
	"*",
	"p++",
	"p--",
	"&",
	"s++",
	"s--",
}

const (
	OPBIN_ADD = iota
	OPBIN_SUB
	OPBIN_MUL
	OPBIN_DIV
	OPBIN_MOD
	OPBIN_ARRSUB
	OPBIN_FUNCALL
	OPBIN_STRUCTDEC
	OPBIN_STRUCTPTRDEC
	OPBIN_TERNARYCOND
	OPBIN_TERNARYVALS
	OPBIN_SHIFTR
	OPBIN_SHIFTL
	OPBIN_LE
	OPBIN_GE
	OPBIN_LT
	OPBIN_GT
	OPBIN_EQ
	OPBIN_NE
	OPBIN_BAND
	OPBIN_BOR
	OPBIN_BXOR
	OPBIN_AND
	OPBIN_OR
)

var opbinnames = [...]string{
	"+",
	"-",
	"*",
	"/",
	"%",
	"[]",
	"CALL",
	".",
	"->",
	"?",
	":",
	">>",
	"<<",
	"<=",
	">=",
	"<",
	">",
	"==",
	"!=",
	"&",
	"|",
	"^",
	"&&",
	"||",
}

const (
	OPASN_PLAIN = iota
	OPASN_ADD
	OPASN_SUB
	OPASN_MUL
	OPASN_DIV
	OPASN_MOD
	OPASN_LSHIFT
	OPASN_RSHIFT
	OPASN_AND
	OPASN_XOR
	OPASN_OR
)

var opasnnames = [...]string{
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
}

type VarDecl struct {
	*Common
	Kind Kind
	Name string
}

type FunDecl struct {
	*Common
	Name    string
	Returns Kind
	Params  VarDecls
}

type FunDef struct {
	*Common
	FunDecl
	Body Block
}

type KindEnum int

const (
	KIND_INT = iota
	KIND_BOOL
	KIND_STRING
	KIND_STRUCT
	KIND_VOID
	KIND_CHAR
	KIND_TYPEDEF
)

var kindnames = [...]string{
	"Int",
	"Bool",
	"String",
	"Struct",
	"Void",
	"Char",
	"Typedef",
	"Null",
	"Function",
}

type Kind struct {
	*Common
	Kind         KindEnum
	PointerLevel int
	ArrayLevel   int
	Name         string // struct or typedef name
}

func (k *Kind) String() string {
	fp := "***************************************************"
	fa := "[][][][][][][][][][][][][][][][][][][][][][][][][]"
	var pp, pa string
	if k.PointerLevel > len(fp) {
		pp = fp[:len(fp)-3] + "..."
	} else {
		pp = fp[:k.PointerLevel]
	}
	if k.ArrayLevel*2 > len(fa) {
		pa = fa[:len(fp)-4] + "..."
	} else {
		pa = fa[:k.ArrayLevel*2]
	}
	var pn string
	switch k.Kind {
	case KIND_TYPEDEF:
		pn = fmt.Sprintf("typedef: %s", k.Name)
	case KIND_STRUCT:
		pn = fmt.Sprintf("struct %s", k.Name)
	default:
		pn = kindnames[k.Kind]
	}
	return fmt.Sprintf("(kind \"%s%s%s\")", pn, pp, pa)
}

func validkind(kind KindEnum) bool {
	return int(kind) >= 0 && int(kind) <= len(kindnames)-1
}

// NewKind is a validating constructors for Kinds.
func NewKind(kind KindEnum, pointerlevel, arraylevel int, name string) Kind {
	if !validkind(kind) {
		panic(fmt.Sprintf("invalid kind: %d", kind))
	}
	if pointerlevel < 0 {
		panic("pointerlevel < 0")
	}
	if arraylevel < 0 {
		panic("arraylevel < 0")
	}
	switch kind {
	case KIND_TYPEDEF, KIND_STRUCT:
		if len(name) == 0 {
			panic("typedef/struct without name")
		}
	default:
		if len(name) > 0 {
			panic("atomic type with a name")
		}
	}
	return Kind{
		Kind:         kind,
		PointerLevel: pointerlevel,
		ArrayLevel:   arraylevel,
		Name:         name,
	}
}

type DirectiveUse struct {
	*Common
	Success                bool
	How                    Node
	Nodes                  []Node
	LexErrors, ParseErrors []error
	Typedefs               map[string]struct{}
}

func (n *Numeric) String() string {
	return fmt.Sprintf("%d", n.Value)
}

func (n *StrLit) String() string {
	return fmt.Sprintf("%q", n.Value)
}

func (n *ChrLit) String() string {
	return fmt.Sprintf("%q", n.Value)
}

func (n *LibLit) String() string {
	return fmt.Sprintf("<%s>", n.Value)
}

func (n *DirectiveUse) String() string {
	return fmt.Sprintf(
		"(#use %t %s %v %s %s %s)",
		n.Success, n.How, n.Typedefs, n.Nodes, n.LexErrors, n.ParseErrors)
}

func (n *VarDecl) String() string {
	return fmt.Sprintf("(vardecl %q %s)", n.Name, &n.Kind)
}

func (n *OpBinary) String() string {
	return fmt.Sprintf("(%s %s %s)", opbinnames[n.Op], n.Left, n.Right)
}

func (n *OpUnary) String() string {
	return fmt.Sprintf("(%s %s)", opunnames[n.Op], n.To)
}

func (n *OpAssign) String() string {
	what := "nil"
	if n.What != nil {
		what = n.What.String()
	}
	return fmt.Sprintf("(assign%s %s %s)", opasnnames[n.Op], n.To, what)
}

func (n *Null) String() string {
	return "NULL"
}

func (n *Block) String() string {
	b := &strings.Builder{}
	b.WriteString("(begin")
	for _, stmt := range n.Value {
		b.WriteString(fmt.Sprintf(" %s", stmt))
	}
	b.WriteString(")")
	return b.String()
}

func (n *If) String() string {
	b := &strings.Builder{}
	b.WriteString(fmt.Sprintf("(if %s", n.Cond))
	b.WriteString(fmt.Sprintf(" %s", n.True))
	if n.False != nil {
		b.WriteString(fmt.Sprintf(" %s", n.False))
	} else {
		b.WriteString(" 'noelse")
	}
	b.WriteString(")")
	return b.String()
}

func (n *While) String() string {
	return fmt.Sprintf("(while %s %s)", n.Cond, n.Body)
}

func (n *For) String() string {
	return fmt.Sprintf("(for %s %s %s %s)", n.Init, n.Cond, n.OnEach, n.Body)
}

func (n *Bool) String() string {
	if n.Value {
		return "#t"
	} else {
		return "#f"
	}
}

func (n *StructForwardDecl) String() string {
	return fmt.Sprintf("(struct-fwd %s)", n.Value)
}

func (n *FunDecl) _string() string {
	b := &strings.Builder{}
	b.WriteString(fmt.Sprintf("(fundecl %q %s (", n.Name, &n.Returns))
	for i, param := range n.Params {
		b.WriteString(param.String())
		if (i + 1) != len(n.Params) {
			b.WriteString(" ")
		}
	}
	b.WriteString(")")
	return b.String()
}

func (n *FunDecl) String() string {
	return n._string() + ")"
}

func (n *FunDef) String() string {
	b := &strings.Builder{}
	b.WriteString("(fundef ")
	b.WriteString(n.FunDecl._string())
	b.WriteString(") ")
	b.WriteString(n.Body.String())
	b.WriteString(")")
	return b.String()
}

func (n *Variable) String() string {
	return fmt.Sprintf("%s", n.Value)
}

func (n *Args) String() string {
	return fmt.Sprintf("%v", n.Value)
}

func (n *Return) String() string {
	if n.Expr == nil {
		return "(return)"
	}
	return fmt.Sprintf("(return %s)", n.Expr)
}

func (n *Assert) String() string {
	return fmt.Sprintf("(assert %s)", n.Expr)
}

func (n *Error) String() string {
	return fmt.Sprintf("(error %s)", n.Expr)
}

func (n *Alloc) String() string {
	return fmt.Sprintf("(alloc %s)", &n.Kind)
}

func (n *AllocArray) String() string {
	return fmt.Sprintf("(alloc-array %s %s)", &n.Kind, n.N)
}

func (n *Break) String() string {
	return "(break)"
}

func (n *Continue) String() string {
	return "(continue)"
}

func (n *Typedef) String() string {
	return fmt.Sprintf("(typedef %q %s)", n.Name, &n.Kind)
}

func (n *TypedefFunc) String() string {
	return fmt.Sprintf("(typedef-func %q %s %v)", n.Name, &n.Returns, n.Params)
}

func (n *Cast) String() string {
	return fmt.Sprintf("(cast %s %s)", &n.To, n.What)
}

func (n *Struct) String() string {
	b := &strings.Builder{}
	b.WriteString("(")
	for _, member := range n.Members {
		b.WriteString(fmt.Sprintf("%s ", member.String()))
	}
	b.WriteString(")")
	return fmt.Sprintf("(def-struct %q %s)", n.Name, b.String())
}

func (p VarDecls) String() string {
	b := &strings.Builder{}
	b.WriteString("(vardecls")
	for _, cur := range p {
		b.WriteString(fmt.Sprintf(" %s", cur.String()))
	}
	b.WriteString(")")
	return b.String()
}

func (c *Common) Id() NodeId {
	return c.id
}

func (c *Common) Tok() *token.Token {
	return Tok(c.id)
}

// NodeCallback is called by Walk for each individual Node encountered. The
// integer argument is the current recursion depth. NodeCallback has to return
// a boolean, which indicates whether to continue recursion for the present
// path.
type NodeCallback func(Node, int) bool

func walk(node Node, cb NodeCallback, depth int) {
	if !cb(node, depth) {
		return
	}
	sub := []Node{}
	a := func(n Node) {
		sub = append(sub, n)
	}
	switch t := node.(type) {
	case *OpUnary:
		a(t.To)
	case *OpBinary:
		a(t.Left)
		a(t.Right)
	case *OpAssign:
		a(t.To)
		a(t.What)
	case *VarDecl:
		a(&t.Kind)
	case *Args:
		for _, arg := range t.Value {
			a(arg)
		}
	case *FunDecl:
		a(&t.Returns)
		for _, param := range t.Params {
			a(&param)
		}
	case *FunDef:
		a(&t.Returns)
		for _, param := range t.Params {
			a(&param)
		}
		a(&t.Body)
	case *Block:
		for _, param := range t.Value {
			a(param)
		}
	case *If:
		a(t.True)
		a(t.False)
	case *For:
		a(t.Init)
		a(t.Cond)
		a(t.OnEach)
		a(t.Body)
	case *While:
		a(t.Cond)
		a(t.Body)
	case *Return:
		a(t.Expr)
	case *Assert:
		a(t.Expr)
	case *Error:
		a(t.Expr)
	default:
	}
	for _, n := range sub {
		walk(n, cb, depth+1)
	}
}

// Walk performs a pre-order traversal of a syntax tree defined by node.
func Walk(node Node, cb NodeCallback) {
	walk(node, cb, 0)
}

func (l *While) Loop() {}
func (l *For) Loop()   {}
