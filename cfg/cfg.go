// Package cfg contains everything relevant for representing a C0 program's
// control-flow graph. The graph's nodes are basic blocks and the edges are
// branching statements. A basic block is a linear sequence of statements
// without any branches.
//
// When considering the compiler pipeline, successful CFG formation relies on
// previous type-checking, syntax-checking, and determining variable scopes.
//
// A CFG is independently formed for each function definition.
//
// A CFG should contain all the relevant information from previous passes to
// continue with code generation.
//
package cfg

import (
	"github.com/susji/c0/node"
)

type BlockId uint64
type BranchId uint64

type Stmts []node.Node

const (
	BLOCKID_ENTRY = 0
	BLOCKID_EXIT  = 1
)

var blockid BlockId = BLOCKID_EXIT
var branchid BranchId = 0

// CFG represents the control-flow path for a single function
type CFG struct {
	first  BasicBlock
	fundef *node.FunDef
}

// BasicBlock contains all permitted statements except branches.
type BasicBlock struct {
	Id         BlockId
	Stmts      Stmts
	Successors []*Branch
}

type Branch struct {
	Id       BranchId
	From, To *BasicBlock
	Kind     Kind
}

type BranchKind int

const (
	BK_INVALID = iota
	BK_IFTRUE
	BK_IFFALSE
	BK_IFNOELSE
	BK_WHILETRUE
	BK_WHILEFALSE
	BK_FORTRUE
	BK_FORFALSE
	BK_ALWAYS
)

var branchkindnames = [...]string{
	"invalid",
	"if-true",
	"if-false",
	"if-no-else",
	"while-true",
	"while-false",
	"for-true",
	"for-false",
	"always",
}

func (bk BranchKind) String() string {
	return branchkindnames[bk]
}

type Kind struct {
	Kind BranchKind
	Node node.Node
}

func newblock() *BasicBlock {
	blockid++
	return &BasicBlock{
		Id: blockid,
	}
}

func (c *CFG) First() *BasicBlock {
	return &c.first
}

func (c *CFG) Definition() *node.FunDef {
	return c.fundef
}
