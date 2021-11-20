package ssa_test

import (
	"testing"

	"github.com/susji/c0/analyze"
	"github.com/susji/c0/cfg"
	"github.com/susji/c0/lex"
	"github.com/susji/c0/node"
	"github.com/susji/c0/parse"
	"github.com/susji/c0/ssa"
	"github.com/susji/c0/ssa/vm"
	"github.com/susji/c0/testers/require"
)

func do(t *testing.T, code string) *cfg.CFG {
	toks, lexerrs := lex.Lex([]rune(code))
	require.Equal(t, 0, len(lexerrs))
	p := parse.New()
	perr := p.Parse(toks)
	require.Nil(t, perr)
	nn := p.Nodes()
	require.NotNil(t, nn)
	a := analyze.New(p.Fn())
	aerrs := a.Analyze(nn)
	t.Log("analysis errors:", aerrs)
	require.Equal(t, 0, len(aerrs))
	c, cerrs := cfg.Form(nn[0].(*node.FunDef))
	require.Equal(t, 0, len(cerrs))
	return c
}

func TestSimple(t *testing.T) {
	cfg := do(t, `
int f() {
	int a = 1;
	int b = a + 3; // b = 4
	a = a * 2 + b; // a = 1 * 2 + 4 = 6
	return a + 1; // 7
}
`)
	s := ssa.New(cfg)
	require.Equal(t, 0, len(s.Errors))
	//fmt.Println(s.Dump())
	v := vm.New()
	v.Insert("f", s)
	require.Equal(t, int32(7), *v.Run(true))
}
