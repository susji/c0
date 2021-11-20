// Package ssa is responsible for turning a control-flow graph (CFG) of a
// function into an intermediate representation (IR), which adheres to static
// single assignment (SSA).
//
// While generating the IR, We have two noteworthy challenges:
//
//    1) Maintain variable usage counts (x_0, x_1, ... x_n)
//    2) Insert phi nodes after branch points
//
// As C0 does not permit variable shadowing, scoping issues are easier to
// manage. Our first approach to phi nodes will be wasteful as we are more
// interested about a working implementation than efficiency.
//
// Note: The SSA generation is meant to be called on per-function basis. The
// aggregate instructions of these separate runs forms the complete program.
//
package ssa

import (
	"fmt"
	"strings"

	"github.com/susji/c0/cfg"
	"github.com/susji/c0/ir"
)

type generations map[string]int

func (g generations) increase(name string) int {
	fmt.Print("increase:", name, "->")
	if _, ok := g[name]; !ok {
		g[name] = 0
		fmt.Println("0")
		return 0
	}
	g[name] += 1
	fmt.Println(g[name])
	return g[name]
}

func (g generations) get(name string) int {
	ret, ok := g[name]
	if !ok {
		panic(fmt.Sprintf("unknown generation for %q", name))
	}
	return ret
}

type SSA struct {
	cfg          *cfg.CFG
	reggen       int
	generations  generations
	Instructions []ir.Instruction
	Errors       []error
}

func (s *SSA) emit(inst ir.Instruction) {
	s.Instructions = append(s.Instructions, inst)
}

func (s *SSA) registerNew() *ir.Variable {
	s.reggen++
	return s.register()
}

func (s *SSA) register() *ir.Variable {
	return &ir.Variable{Name: "", Count: s.reggen}
}

func (s *SSA) Dump() string {
	b := &strings.Builder{}
	for i, instr := range s.Instructions {
		b.WriteString(fmt.Sprintf("[%03d] %s\n", i, instr))
	}
	return b.String()
}

func New(c *cfg.CFG) *SSA {
	ret := &SSA{
		cfg:         c,
		generations: generations{},
	}
	ret.build()
	return ret
}
