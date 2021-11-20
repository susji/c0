// Package ir contains the full description of a program's intermediate
// representation. These constructs are the connection between the previous
// AST-based description and subsequent machine code.
//
// We mostly ape a subset of the LLVM IR approach here: This is a RISC machine,
// that is, all ALU operands are registers, and LOAD & STORE must be used to
// get or store memory contents, respectively. We assume an infinite amount of
// registers and leave allocation for later stages to worry about.
//
package ir

import "fmt"

const (
	TYPE_INT32 = iota
)

type Type struct {
	Kind, PointerLevel, Elements int
}

func (n *Type) String() string {
	var kind string
	switch n.Kind {
	case TYPE_INT32:
		kind = "i32"
	default:
		panic("unrecognized Type")
	}
	ptr := "*************************************************************"
	ptri := n.PointerLevel
	if ptri >= len(ptr) {
		ptri = len(ptr) - 1
	}
	var e string
	if n.Elements > 0 {
		e = fmt.Sprintf("%d x ", n.Elements)
	}
	return fmt.Sprintf("[%s%s%s]", e, ptr[:ptri], kind)
}

type Instruction interface {
	String() string
	Instruction()
}

type Value interface {
	IsValue()
}

type Variable struct {
	Name  string
	Count int
}

type Function struct {
	Name   string
	Params []string
}

type Load struct {
	Type     *Type
	To, From *Variable
}

type Store struct {
	Type     *Type
	To, From *Variable
}

type Add struct {
	Type        *Type
	To          *Variable
	Left, Right Value
}

type Mul struct {
	Type        *Type
	To          *Variable
	Left, Right Value
}

type Xor struct {
	Type        *Type
	To          *Variable
	Left, Right Value
}

type Mov struct {
	Type *Type
	To   *Variable
	What Value
}

type Return struct {
	Type *Type
	With Value
}

type Alloca struct {
	Type  *Type
	Align uint
	To    *Variable
}

type Label struct {
	Name string
}

type FunCall struct {
}

type Jump struct {
}

type JumpZero struct {
}

type JumpNonZero struct {
}

type Numeric32i struct {
	Value int32
}

func (r *Variable) String() string {
	if r.Name == "" {
		return fmt.Sprintf("%%%d", r.Count)
	}
	return fmt.Sprintf("%%%s_%d", r.Name, r.Count)
}

func (i Numeric32i) String() string {
	return fmt.Sprintf("%d [32i]", i.Value)
}

func (i Store) String() string {
	return fmt.Sprintf("STORE<%s> %s, [%s]", i.Type, i.From, i.To)
}

func (i Load) String() string {
	return fmt.Sprintf("LOAD<%s> [%s], %s", i.Type, i.From, i.To)
}

func (i Add) String() string {
	return fmt.Sprintf("%s = ADD<%s> %s, %s", i.To, i.Type, i.Left, i.Right)
}

func (i Mul) String() string {
	return fmt.Sprintf("%s = MUL<%s> %s, %s", i.To, i.Type, i.Left, i.Right)
}

func (i Xor) String() string {
	return fmt.Sprintf("%s = XOR<%s> %s, %s", i.To, i.Type, i.Left, i.Right)
}

func (i Mov) String() string {
	return fmt.Sprintf("MOV<%s> %s, %s", i.Type, i.What, i.To)
}

func (i Return) String() string {
	return fmt.Sprintf("RET<%s> %s", i.Type, i.With)
}

func (i Alloca) String() string {
	return fmt.Sprintf("%s = ALLOCA %s, align %d", i.To, i.Type, i.Align)
}

func (i Label) String() string {
	return fmt.Sprintf("%s:", i.Name)
}

func (i Load) Instruction()   {}
func (i Store) Instruction()  {}
func (i Add) Instruction()    {}
func (i Mul) Instruction()    {}
func (i Return) Instruction() {}
func (i Alloca) Instruction() {}
func (i Xor) Instruction()    {}
func (i Mov) Instruction()    {}
func (i Label) Instruction()  {}

func (v *Variable) IsValue()   {}
func (i *Numeric32i) IsValue() {}
