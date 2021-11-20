package ssa

import (
	"fmt"

	"github.com/susji/c0/cfg"
	"github.com/susji/c0/ir"
	"github.com/susji/c0/node"
)

var typeInt = &ir.Type{Kind: ir.TYPE_INT32, Elements: 0, PointerLevel: 0}
var valueZero = &ir.Numeric32i{Value: 0}

func (s *SSA) emitOpBinary(n *node.OpBinary) {
	fmt.Println("emitOpBinary:", n)
	left := s.emitLoadable(n.Left)
	right := s.emitLoadable(n.Right)
	to := s.registerNew()
	switch n.Op {
	case node.OPBIN_ADD:
		s.emit(ir.Add{Type: typeInt, To: to, Left: left, Right: right})
	case node.OPBIN_MUL:
		s.emit(ir.Mul{Type: typeInt, To: to, Left: left, Right: right})
	default:
		fmt.Println("XXX UNHANDLED OP BINARY:", n.String())
	}
}

func (s *SSA) emitOpUnary(n *node.OpUnary) {
	s.emitNode(n.To)
	switch n.Op {
	default:
		fmt.Println("XXX UNHANDLED OP UNARY:", n.String())
	}
}

func (s *SSA) emitReturn(n *node.Return) {
	s.emit(ir.Return{Type: typeInt, With: s.emitLoadable(n.Expr)})
}

func (s *SSA) getNumeric32i(n *node.Numeric) *ir.Variable {
	s.emit(ir.Mov{
		Type: typeInt,
		What: &ir.Numeric32i{Value: n.Value},
		To:   s.registerNew(),
	})
	return s.register()
}

func (s *SSA) getNewVariable(name string) *ir.Variable {
	n := &ir.Variable{Name: name, Count: s.generations.increase(name)}
	s.emit(ir.Alloca{Type: typeInt, Align: 4, To: n})
	return n
}

func (s *SSA) getCurrentVariable(name string) *ir.Variable {
	return &ir.Variable{Name: name, Count: s.generations.get(name)}
}

func (s *SSA) getNewStorable(n node.Node) *ir.Variable {
	switch t := n.(type) {
	case *node.Variable:
		return s.getCurrentVariable(t.Value)
	case *node.VarDecl:
		return s.getNewVariable(t.Name)
	default:
		panic(fmt.Sprintf("XXX unhandled storable: %s", t))
	}
}

func (s *SSA) emitLoadable(n node.Node) *ir.Variable {
	switch t := n.(type) {
	case *node.Variable:
		s.emit(ir.Load{
			Type: typeInt,
			From: s.getCurrentVariable(t.Value),
			To:   s.registerNew(),
		})
	case *node.VarDecl:
		s.emit(ir.Load{
			Type: typeInt,
			From: s.getNewVariable(t.Name),
			To:   s.registerNew(),
		})
	case *node.Numeric:
		s.getNumeric32i(t)
	case *node.OpBinary:
		s.emitOpBinary(t)
	default:
		panic(fmt.Sprintf("XXX unhandled assignment source: %s", t))
	}
	return s.register()
}

func (s *SSA) emitAssign(n *node.OpAssign) {
	fmt.Println("emitAssign:", n)
	// each assignment means a new variable generation
	to := s.getNewStorable(n.To)
	s.emit(ir.Store{Type: typeInt, From: s.emitLoadable(n.What), To: to})
}

func (s *SSA) emitNode(n node.Node) {
	switch t := n.(type) {
	case *node.OpAssign:
		s.emitAssign(t)
	case *node.OpBinary:
		s.emitOpBinary(t)
	case *node.OpUnary:
		s.emitOpUnary(t)
	case *node.Return:
		s.emitReturn(t)
	default:
		// XXX array subs will have to be handled somehow
		panic(fmt.Sprintf("XXX unhandled node: %s", t))
	}
}

func (s *SSA) emitBlock(bb *cfg.BasicBlock) {
	for _, stmt := range bb.Stmts {
		s.emitNode(stmt)
	}
	for _, succ := range bb.Successors {
		s.emitBlock(succ.To)
	}
}

func (s *SSA) build() {
	s.emit(ir.Label{Name: "entry"})
	s.emitBlock(s.cfg.First())
}
