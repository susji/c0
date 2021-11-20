package parse_test

import (
	"os"
	"testing"

	"github.com/susji/c0/node"
	"github.com/susji/c0/parse"
	"github.com/susji/c0/span"
	"github.com/susji/c0/testers/assert"
	"github.com/susji/c0/token"
)

func sp() span.Span {
	return span.Span{}
}

func TestMain(m *testing.M) {
	// The nodes we parse will, by default, receive unique IDs.
	//
	// Problem:  We do *not* want to start conjuring those for our tests below
	//           when we match expectations with the results.
	//
	// Solution: Disable the node pooling which disables node tagging with IDs.
	//           This means we may neglect Node IDs.
	//
	node.DisableTagging()
	os.Exit(m.Run())
}

func DumpErrors(t *testing.T, errs []error) {
	if len(errs) == 0 {
		return
	}
	t.Log("dumping errors:")
	for i, err := range errs {
		t.Logf("[%2d] %v", i+1, err)
	}
}

func TestTypeSimple(t *testing.T) {
	toks := &token.Tokens{}
	toks.Add(token.New(token.Id, sp(), "int"))
	want := node.NewKind(node.KIND_INT, 0, 0, "")

	p := parse.New()
	n, err := p.Type(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestTypeStruct(t *testing.T) {
	toks := &token.Tokens{}
	toks.Add(token.New(token.Id, sp(), "struct")).
		Add(token.New(token.Id, sp(), "name"))
	want := node.NewKind(node.KIND_STRUCT, 0, 0, "name")

	p := parse.New()
	n, err := p.Type(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestTypeComplex(t *testing.T) {
	toks := &token.Tokens{}
	toks.Add(token.New(token.Id, sp(), "struct")).
		Add(token.New(token.Id, sp(), "name")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Brackets, sp(), ""))

	want := node.NewKind(node.KIND_STRUCT, 2, 1, "name")

	p := parse.New()
	n, err := p.Type(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestTypeTypedef(t *testing.T) {
	toks := &token.Tokens{}
	toks.Add(token.New(token.Id, sp(), "something"))
	p := parse.New()
	want := node.NewKind(node.KIND_TYPEDEF, 0, 0, "something")
	p.AddTypedef(toks.Peek(), "something")
	n, err := p.Type(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestGlobalDeclStruct(t *testing.T) {
	toks := &token.Tokens{}
	toks.Add(token.New(token.Id, sp(), "struct")).
		Add(token.New(token.Id, sp(), "rakenne")).
		Add(token.New(token.Semicolon, sp(), ""))
	p := parse.New()
	want := &node.StructForwardDecl{Value: "rakenne"}
	n, err := p.GlobalDeclDef(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestGlobalDeclUse(t *testing.T) {
	toks := &token.Tokens{}
	toks.Add(token.New(token.UseLibLit, sp(), "testdata/test.h0"))
	p := parse.New()
	want := &node.DirectiveUse{
		Success:     true,
		Nodes:       []node.Node{&node.StructForwardDecl{Value: "asd"}},
		How:         &node.LibLit{Value: "testdata/test.h0"},
		LexErrors:   nil,
		ParseErrors: nil,
		Typedefs:    map[string]struct{}{},
	}
	n, err := p.GlobalDeclDef(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestGlobalDeclFuncSimple(t *testing.T) {
	toks := &token.Tokens{}
	// int foo();
	toks.Add(token.New(token.Id, sp(), "int")).
		Add(token.New(token.Id, sp(), "foo")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.Semicolon, sp(), ""))
	p := parse.New()
	want := &node.FunDecl{
		Returns: node.NewKind(node.KIND_INT, 0, 0, ""),
		Name:    "foo",
		Params:  []node.VarDecl{},
	}
	n, err := p.GlobalDeclDef(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestGlobalDeclFuncStruct(t *testing.T) {
	toks := &token.Tokens{}
	// struct s* foo(struct s*[] zap);
	toks.Add(token.New(token.Id, sp(), "struct")).
		Add(token.New(token.Id, sp(), "s")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Id, sp(), "foo")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Id, sp(), "struct")).
		Add(token.New(token.Id, sp(), "s")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Brackets, sp(), "")).
		Add(token.New(token.Id, sp(), "zap")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.Semicolon, sp(), ""))
	p := parse.New()
	want := &node.FunDecl{
		Returns: node.NewKind(node.KIND_STRUCT, 1, 0, "s"),
		Name:    "foo",
		Params: []node.VarDecl{
			node.VarDecl{
				Kind: node.Kind{
					Kind:         node.KIND_STRUCT,
					PointerLevel: 1,
					ArrayLevel:   1,
					Name:         "s",
				},
				Name: "zap",
			},
		},
	}
	n, err := p.GlobalDeclDef(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestGlobalDeclFunc(t *testing.T) {
	toks := &token.Tokens{}
	// int foo(string a, int b, bool[] c);
	toks.Add(token.New(token.Id, sp(), "int")).
		Add(token.New(token.Id, sp(), "foo")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Id, sp(), "string")).
		Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.Comma, sp(), "")).
		Add(token.New(token.Id, sp(), "int")).
		Add(token.New(token.Id, sp(), "b")).
		Add(token.New(token.Comma, sp(), "")).
		Add(token.New(token.Id, sp(), "bool")).
		Add(token.New(token.Brackets, sp(), "")).
		Add(token.New(token.Id, sp(), "c")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.Semicolon, sp(), ""))
	p := parse.New()
	want := &node.FunDecl{
		Returns: node.NewKind(node.KIND_INT, 0, 0, ""),
		Name:    "foo",
		Params: []node.VarDecl{
			{
				Name: "a",
				Kind: node.NewKind(node.KIND_STRING, 0, 0, ""),
			},
			{
				Name: "b",
				Kind: node.NewKind(node.KIND_INT, 0, 0, ""),
			},
			{
				Name: "c",
				Kind: node.NewKind(node.KIND_BOOL, 0, 1, ""),
			},
		},
	}
	n, err := p.GlobalDeclDef(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestExprSimple(t *testing.T) {
	toks := &token.Tokens{}
	// a + -b * c
	toks.Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.Plus, sp(), "")).
		Add(token.New(token.Minus, sp(), "")).
		Add(token.New(token.Id, sp(), "b")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Id, sp(), "c"))
	p := parse.New()
	want := &node.OpBinary{
		Op:   node.OPBIN_ADD,
		Left: &node.Variable{Value: "a"},
		Right: &node.OpBinary{
			Op: node.OPBIN_MUL,
			Left: &node.OpUnary{
				Op: node.OPUN_NEG,
				To: &node.Variable{Value: "b"},
			},
			Right: &node.Variable{Value: "c"},
		},
	}
	got, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, got)
	DumpErrors(t, p.Errors())
}

func TestExprFuncallSimple(t *testing.T) {
	toks := &token.Tokens{}
	// fun() + 1
	toks.Add(token.New(token.Id, sp(), "fun")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.Plus, sp(), "")).
		Add(token.New(token.DecNum, sp(), "1"))
	p := parse.New()
	want := &node.OpBinary{
		Op: node.OPBIN_ADD,
		Left: &node.OpBinary{
			Op:    node.OPBIN_FUNCALL,
			Left:  &node.Variable{Value: "fun"},
			Right: &node.Args{Value: []node.Node{}}},
		Right: &node.Numeric{Base: 10, Value: 1},
	}
	n, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestExprFuncallPointer(t *testing.T) {
	toks := &token.Tokens{}
	// (*ptr)(1, 2)
	toks.Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Id, sp(), "ptr")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.DecNum, sp(), "1")).
		Add(token.New(token.Comma, sp(), "1")).
		Add(token.New(token.DecNum, sp(), "2")).
		Add(token.New(token.RParen, sp(), ""))
	p := parse.New()
	want := &node.OpBinary{
		Op: node.OPBIN_FUNCALL,
		Left: &node.OpUnary{
			Op: node.OPUN_DEREF,
			To: &node.Variable{Value: "ptr"},
		},
		Right: &node.Args{
			Value: []node.Node{
				&node.Numeric{Base: 10, Value: 1},
				&node.Numeric{Base: 10, Value: 2},
			},
		},
	}
	n, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestExprFuncallOneArg(t *testing.T) {
	toks := &token.Tokens{}
	// fun(x)
	toks.Add(token.New(token.Id, sp(), "fun")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Id, sp(), "x")).
		Add(token.New(token.RParen, sp(), ""))
	p := parse.New()
	want := &node.OpBinary{
		Op:   node.OPBIN_FUNCALL,
		Left: &node.Variable{Value: "fun"},
		Right: &node.Args{
			Value: []node.Node{
				&node.Variable{Value: "x"},
			},
		},
	}
	n, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestExprFuncallTwoArgs(t *testing.T) {
	toks := &token.Tokens{}
	// fun(1+2, x)
	toks.Add(token.New(token.Id, sp(), "fun")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.DecNum, sp(), "1")).
		Add(token.New(token.Plus, sp(), "")).
		Add(token.New(token.DecNum, sp(), "2")).
		Add(token.New(token.Comma, sp(), "")).
		Add(token.New(token.Id, sp(), "x")).
		Add(token.New(token.RParen, sp(), ""))
	p := parse.New()
	want := &node.OpBinary{
		Op:   node.OPBIN_FUNCALL,
		Left: &node.Variable{Value: "fun"},
		Right: &node.Args{
			Value: []node.Node{
				&node.OpBinary{
					Op:    node.OPBIN_ADD,
					Left:  &node.Numeric{Base: 10, Value: 1},
					Right: &node.Numeric{Base: 10, Value: 2},
				},
				&node.Variable{Value: "x"},
			},
		},
	}
	n, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestExprFuncallNested(t *testing.T) {
	toks := &token.Tokens{}
	// one(two(1, three(3+4)), 2)
	toks.Add(token.New(token.Id, sp(), "one")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Id, sp(), "two")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.DecNum, sp(), "1")).
		Add(token.New(token.Comma, sp(), "")).
		Add(token.New(token.Id, sp(), "three")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.DecNum, sp(), "3")).
		Add(token.New(token.Plus, sp(), "")).
		Add(token.New(token.DecNum, sp(), "4")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.Comma, sp(), "")).
		Add(token.New(token.DecNum, sp(), "2")).
		Add(token.New(token.RParen, sp(), ""))
	p := parse.New()
	want := &node.OpBinary{
		Op:   node.OPBIN_FUNCALL,
		Left: &node.Variable{Value: "one"},
		Right: &node.Args{
			Value: []node.Node{
				&node.OpBinary{
					Op:   node.OPBIN_FUNCALL,
					Left: &node.Variable{Value: "two"},
					Right: &node.Args{
						Value: []node.Node{
							&node.Numeric{Base: 10, Value: 1},
							&node.OpBinary{
								Op:   node.OPBIN_FUNCALL,
								Left: &node.Variable{Value: "three"},
								Right: &node.Args{
									Value: []node.Node{
										&node.OpBinary{
											Op:    node.OPBIN_ADD,
											Left:  &node.Numeric{Base: 10, Value: 3},
											Right: &node.Numeric{Base: 10, Value: 4},
										},
									},
								},
							},
						},
					},
				},
				&node.Numeric{Base: 10, Value: 2},
			},
		},
	}
	n, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestExprArray(t *testing.T) {
	toks := &token.Tokens{}
	// (arr)[a/b], as array refs can be ``<expr> '[' <expr> ']'""
	toks.Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Id, sp(), "arr")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.LBrack, sp(), "")).
		Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.Slash, sp(), "")).
		Add(token.New(token.Id, sp(), "b")).
		Add(token.New(token.RBrack, sp(), ""))
	p := parse.New()
	want := &node.OpBinary{
		Op:   node.OPBIN_ARRSUB,
		Left: &node.Variable{Value: "arr"},
		Right: &node.OpBinary{
			Op:    node.OPBIN_DIV,
			Left:  &node.Variable{Value: "a"},
			Right: &node.Variable{Value: "b"},
		},
	}
	n, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestExprStruct(t *testing.T) {
	toks := &token.Tokens{}
	// stru->element.other
	toks.Add(token.New(token.Id, sp(), "stru")).
		Add(token.New(token.Arrow, sp(), "")).
		Add(token.New(token.Id, sp(), "element")).
		Add(token.New(token.Dot, sp(), "")).
		Add(token.New(token.Id, sp(), "other"))
	p := parse.New()
	want := &node.OpBinary{
		Op: node.OPBIN_STRUCTDEC,
		Left: &node.OpBinary{
			Op:    node.OPBIN_STRUCTPTRDEC,
			Left:  &node.Variable{Value: "stru"},
			Right: &node.Variable{Value: "element"},
		},
		Right: &node.Variable{Value: "other"},
	}
	n, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestExprTernary(t *testing.T) {
	toks := &token.Tokens{}
	// one + two ? yes + 1: no - 2
	toks.Add(token.New(token.Id, sp(), "one")).
		Add(token.New(token.Plus, sp(), "")).
		Add(token.New(token.Id, sp(), "two")).
		Add(token.New(token.Quest, sp(), "")).
		Add(token.New(token.Id, sp(), "yes")).
		Add(token.New(token.Plus, sp(), "")).
		Add(token.New(token.DecNum, sp(), "1")).
		Add(token.New(token.Colon, sp(), "")).
		Add(token.New(token.Id, sp(), "no")).
		Add(token.New(token.Minus, sp(), "")).
		Add(token.New(token.DecNum, sp(), "2"))

	p := parse.New()
	want := &node.OpBinary{
		Op: node.OPBIN_TERNARYCOND,
		Left: &node.OpBinary{
			Op:    node.OPBIN_ADD,
			Left:  &node.Variable{Value: "one"},
			Right: &node.Variable{Value: "two"},
		},
		Right: &node.OpBinary{
			Op: node.OPBIN_TERNARYVALS,
			Left: &node.OpBinary{Op: node.OPBIN_ADD,
				Left:  &node.Variable{Value: "yes"},
				Right: &node.Numeric{Base: 10, Value: 1},
			},
			Right: &node.OpBinary{Op: node.OPBIN_SUB,
				Left:  &node.Variable{Value: "no"},
				Right: &node.Numeric{Base: 10, Value: 2},
			},
		},
	}
	n, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestExprShouldFail(t *testing.T) {
	type entry struct {
		what string
		toks *token.Tokens
	}

	ctor := func() *token.Tokens {
		return &token.Tokens{}
	}

	entries := []entry{
		entry{
			what: "/5",
			toks: ctor().
				Add(token.New(token.Slash, sp(), "")).
				Add(token.New(token.DecNum, sp(), "5")),
		},
		entry{
			what: "?5",
			toks: ctor().
				Add(token.New(token.Quest, sp(), "")).
				Add(token.New(token.DecNum, sp(), "5")),
		},
		entry{
			what: "(123+456",
			toks: ctor().
				Add(token.New(token.LParen, sp(), "")).
				Add(token.New(token.DecNum, sp(), "123")).
				Add(token.New(token.Plus, sp(), "")).
				Add(token.New(token.DecNum, sp(), "456")),
		},
		entry{
			what: "fun(123,)",
			toks: ctor().
				Add(token.New(token.Id, sp(), "fun")).
				Add(token.New(token.LParen, sp(), "")).
				Add(token.New(token.DecNum, sp(), "123")).
				Add(token.New(token.Comma, sp(), "")).
				Add(token.New(token.RParen, sp(), "")),
		},
	}

	for _, cur := range entries {
		t.Run(cur.what, func(t *testing.T) {
			p := parse.New()
			n, err := p.Expr(cur.toks)
			assert.NotNil(t, err)
			assert.Nil(t, n)
			DumpErrors(t, p.Errors())
		})
	}
}

func TestExprBinops(t *testing.T) {
	ops := []token.Kind{
		token.Plus,
		token.Minus,
		token.Star,
		token.Slash,
		token.Percent,
		token.Dot,
		token.Arrow,
		token.Quest,
		token.Colon,
		token.DGt,
		token.DLt,
		token.Le,
		token.Ge,
		token.Lt,
		token.Gt,
		token.Eq,
		token.Ne,
		token.Ampersand,
		token.Hat,
		token.Pipe,
		token.DAmpersand,
		token.DPipe,
	}

	for _, op := range ops {
		t.Run(op.String(), func(t *testing.T) {
			p := parse.New()
			toks := &token.Tokens{}
			toks.Add(token.New(token.DecNum, sp(), "1")).
				Add(token.New(op, sp(), "")).
				Add(token.New(token.DecNum, sp(), "3"))
			n, err := p.Expr(toks)
			assert.Nil(t, err)
			assert.NotNil(t, n)
			DumpErrors(t, p.Errors())
		})
	}
}

func TestSimpleVarDecl(t *testing.T) {
	toks := &token.Tokens{}
	// int var = 123 + 456;
	toks.Add(token.New(token.Id, sp(), "int")).
		Add(token.New(token.Id, sp(), "var")).
		Add(token.New(token.Assign, sp(), "")).
		Add(token.New(token.DecNum, sp(), "123")).
		Add(token.New(token.Plus, sp(), "")).
		Add(token.New(token.DecNum, sp(), "456"))

	want := &node.OpAssign{
		Op: node.OPASN_PLAIN,
		To: &node.VarDecl{
			Kind: node.Kind{
				Kind:         node.KIND_INT,
				PointerLevel: 0,
				ArrayLevel:   0,
				Name:         "",
			},
			Name: "var",
		},
		What: &node.OpBinary{
			Op:    node.OPBIN_ADD,
			Left:  &node.Numeric{Base: 10, Value: 123},
			Right: &node.Numeric{Base: 10, Value: 456},
		},
	}
	p := parse.New()
	got, err := p.SimpleStmt(toks)
	assert.Nil(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, want, got)
	DumpErrors(t, p.Errors())
}

func TestSimpleAssign(t *testing.T) {
	toks := &token.Tokens{}
	// *var = fun(1)
	toks.Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Id, sp(), "var")).
		Add(token.New(token.Assign, sp(), "")).
		Add(token.New(token.Id, sp(), "fun")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.DecNum, sp(), "1")).
		Add(token.New(token.RParen, sp(), ""))

	want := &node.OpAssign{
		Op: node.OPASN_PLAIN,
		To: &node.OpUnary{
			Op: node.OPUN_DEREF,
			To: &node.Variable{Value: "var"},
		},
		What: &node.OpBinary{
			Op:    node.OPBIN_FUNCALL,
			Left:  &node.Variable{Value: "fun"},
			Right: &node.Args{Value: []node.Node{&node.Numeric{Base: 10, Value: 1}}},
		},
	}
	p := parse.New()
	got, err := p.SimpleStmt(toks)
	assert.Nil(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, want, got)
	DumpErrors(t, p.Errors())
}

func TestStmtIf(t *testing.T) {
	toks := &token.Tokens{}
	// if (true) {} return 123;
	toks.Add(token.New(token.Id, sp(), "if")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.True, sp(), "")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.LCurly, sp(), "")).
		Add(token.New(token.RCurly, sp(), "")).
		Add(token.New(token.Id, sp(), "return")).
		Add(token.New(token.DecNum, sp(), "123")).
		Add(token.New(token.Semicolon, sp(), ""))

	want := &node.If{
		Cond:  &node.Bool{Value: true},
		True:  &node.Block{Value: []node.Node{}},
		False: nil,
	}
	p := parse.New()
	got, err := p.Stmt(toks)
	assert.Nil(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, want, got)
	DumpErrors(t, p.Errors())
}

func TestStmtIfNoBlock(t *testing.T) {
	toks := &token.Tokens{}
	// if (true) 1; else 2; 3;
	toks.Add(token.New(token.Id, sp(), "if")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.True, sp(), "")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.DecNum, sp(), "1")).
		Add(token.New(token.Semicolon, sp(), "")).
		Add(token.New(token.Id, sp(), "else")).
		Add(token.New(token.DecNum, sp(), "2")).
		Add(token.New(token.Semicolon, sp(), "")).
		Add(token.New(token.DecNum, sp(), "3")).
		Add(token.New(token.Semicolon, sp(), ""))

	want := &node.If{
		Cond:  &node.Bool{Value: true},
		True:  &node.Numeric{Value: 1, Base: 10},
		False: &node.Numeric{Value: 2, Base: 10},
	}
	p := parse.New()
	got, err := p.Stmt(toks)
	assert.Nil(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, want, got)
	DumpErrors(t, p.Errors())
}

func TestStmtFor(t *testing.T) {
	toks := &token.Tokens{}
	// for (a = 1; a < 5; a++) { printf("%d\n", a); }
	toks.Add(token.New(token.Id, sp(), "for")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.Assign, sp(), "")).
		Add(token.New(token.DecNum, sp(), "1")).
		Add(token.New(token.Semicolon, sp(), "")).
		Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.Lt, sp(), "")).
		Add(token.New(token.DecNum, sp(), "5")).
		Add(token.New(token.Semicolon, sp(), "")).
		Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.DPlus, sp(), "")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.LCurly, sp(), "")).
		Add(token.New(token.Id, sp(), "printf")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.StrLit, sp(), "%d\n")).
		Add(token.New(token.Comma, sp(), "")).
		Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.Semicolon, sp(), "")).
		Add(token.New(token.RCurly, sp(), ""))

	want := &node.For{
		Init: &node.OpAssign{
			Op:   node.OPASN_PLAIN,
			To:   &node.Variable{Value: "a"},
			What: &node.Numeric{Base: 10, Value: 1},
		},
		Cond: &node.OpBinary{
			Op:    node.OPBIN_LT,
			Left:  &node.Variable{Value: "a"},
			Right: &node.Numeric{Base: 10, Value: 5},
		},
		OnEach: &node.OpUnary{
			Op: node.OPUN_ADDONESUFFIX,
			To: &node.Variable{Value: "a"},
		},
		Body: &node.Block{
			Value: []node.Node{
				&node.OpBinary{
					Op:   node.OPBIN_FUNCALL,
					Left: &node.Variable{Value: "printf"},
					Right: &node.Args{Value: []node.Node{
						&node.StrLit{Value: "%d\n"},
						&node.Variable{Value: "a"},
					}},
				},
			},
		},
	}
	p := parse.New()
	got, err := p.Stmt(toks)
	assert.Nil(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, want, got)
	DumpErrors(t, p.Errors())
}

func TestDefTypedef(t *testing.T) {
	toks := &token.Tokens{}
	toks.Add(token.New(token.Id, sp(), "typedef")).
		Add(token.New(token.Id, sp(), "string")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Id, sp(), "somename")).
		Add(token.New(token.Semicolon, sp(), ""))
	want := &node.Typedef{
		Kind: node.NewKind(node.KIND_STRING, 1, 0, ""),
		Name: "somename",
	}
	p := parse.New()
	got, err := p.GlobalDeclDef(toks)
	assert.Nil(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, want, got)
	DumpErrors(t, p.Errors())
}

func TestDefTypedefFunctionPointer(t *testing.T) {
	toks := &token.Tokens{}
	// typedef string* name(int a, bool[] b);
	toks.Add(token.New(token.Id, sp(), "typedef")).
		Add(token.New(token.Id, sp(), "string")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Id, sp(), "name")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Id, sp(), "int")).
		Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.Comma, sp(), "")).
		Add(token.New(token.Id, sp(), "bool")).
		Add(token.New(token.Brackets, sp(), "")).
		Add(token.New(token.Id, sp(), "b")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.Semicolon, sp(), ""))
	want := &node.TypedefFunc{
		Returns: node.NewKind(node.KIND_STRING, 1, 0, ""),
		Name:    "name",
		Params: []node.VarDecl{
			node.VarDecl{
				Name: "a",
				Kind: node.NewKind(node.KIND_INT, 0, 0, ""),
			},
			node.VarDecl{
				Name: "b",
				Kind: node.NewKind(node.KIND_BOOL, 0, 1, ""),
			},
		},
	}
	p := parse.New()
	got, err := p.GlobalDeclDef(toks)
	assert.Nil(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, want, got)
	DumpErrors(t, p.Errors())
}

func TestGlobalDefFunc(t *testing.T) {
	toks := &token.Tokens{}
	// int foo(string a, int b, bool[] c) { a = "jep"; return b; }
	toks.Add(token.New(token.Id, sp(), "int")).
		Add(token.New(token.Id, sp(), "foo")).
		Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Id, sp(), "string")).
		Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.Comma, sp(), "")).
		Add(token.New(token.Id, sp(), "int")).
		Add(token.New(token.Id, sp(), "b")).
		Add(token.New(token.Comma, sp(), "")).
		Add(token.New(token.Id, sp(), "bool")).
		Add(token.New(token.Brackets, sp(), "")).
		Add(token.New(token.Id, sp(), "c")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.LCurly, sp(), "")).
		Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.Assign, sp(), "")).
		Add(token.New(token.StrLit, sp(), "jep")).
		Add(token.New(token.Semicolon, sp(), "")).
		Add(token.New(token.Id, sp(), "return")).
		Add(token.New(token.Id, sp(), "b")).
		Add(token.New(token.Semicolon, sp(), "")).
		Add(token.New(token.RCurly, sp(), ""))
	p := parse.New()
	want := &node.FunDef{
		FunDecl: node.FunDecl{
			Returns: node.NewKind(node.KIND_INT, 0, 0, ""),
			Name:    "foo",
			Params: []node.VarDecl{
				{
					Name: "a",
					Kind: node.NewKind(node.KIND_STRING, 0, 0, ""),
				},
				{
					Name: "b",
					Kind: node.NewKind(node.KIND_INT, 0, 0, ""),
				},
				{
					Name: "c",
					Kind: node.NewKind(node.KIND_BOOL, 0, 1, ""),
				},
			},
		},
		Body: node.Block{
			Value: []node.Node{
				&node.OpAssign{
					Op:   node.OPASN_PLAIN,
					To:   &node.Variable{Value: "a"},
					What: &node.StrLit{Value: "jep"},
				},
				&node.Return{Expr: &node.Variable{Value: "b"}},
			},
		},
	}
	n, err := p.GlobalDeclDef(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestGlobalDefStruct(t *testing.T) {
	toks := &token.Tokens{}
	// struct test { int a; bool b; };
	toks.Add(token.New(token.Id, sp(), "struct")).
		Add(token.New(token.Id, sp(), "test")).
		Add(token.New(token.LCurly, sp(), "")).
		Add(token.New(token.Id, sp(), "int")).
		Add(token.New(token.Id, sp(), "a")).
		Add(token.New(token.Semicolon, sp(), "")).
		Add(token.New(token.Id, sp(), "bool")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Id, sp(), "b")).
		Add(token.New(token.Semicolon, sp(), "")).
		Add(token.New(token.RCurly, sp(), "")).
		Add(token.New(token.Semicolon, sp(), ""))
	p := parse.New()
	want := &node.Struct{
		Name: "test",
		Members: []node.VarDecl{
			node.VarDecl{
				Name: "a",
				Kind: node.NewKind(node.KIND_INT, 0, 0, ""),
			},
			node.VarDecl{
				Name: "b",
				Kind: node.NewKind(node.KIND_BOOL, 1, 0, ""),
			},
		},
	}
	n, err := p.GlobalDeclDef(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestCasting(t *testing.T) {
	toks := &token.Tokens{}
	// (int*)voidy
	toks.Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Id, sp(), "int")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.Id, sp(), "voidy"))
	p := parse.New()
	want := &node.Cast{
		To:   node.NewKind(node.KIND_INT, 1, 0, ""),
		What: &node.Variable{Value: "voidy"},
	}
	n, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestExprChrLit(t *testing.T) {
	toks := &token.Tokens{}
	// 'r'
	toks.Add(token.New(token.ChrLit, sp(), "r"))
	p := parse.New()
	want := &node.ChrLit{Value: 'r'}
	n, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equal(t, want, n)
	DumpErrors(t, p.Errors())
}

func TestPrecedenceUnary(t *testing.T) {
	toks := &token.Tokens{}
	// *s.f
	// This should be parsed as (* (. s f)) because '.' has higher precedence.
	toks.Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Id, sp(), "s")).
		Add(token.New(token.Dot, sp(), "")).
		Add(token.New(token.Id, sp(), "f"))
	p := parse.New()
	want := &node.OpUnary{
		Op: node.OPUN_DEREF,
		To: &node.OpBinary{
			Op:    node.OPBIN_STRUCTDEC,
			Left:  &node.Variable{Value: "s"},
			Right: &node.Variable{Value: "f"},
		},
	}
	got, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equalf(t, want, got, "want: %s, got %s", want, got)
	DumpErrors(t, p.Errors())
}

func TestExprVoid(t *testing.T) {
	toks := &token.Tokens{}
	// 1+void;
	toks.Add(token.New(token.DecNum, sp(), "1")).
		Add(token.New(token.Plus, sp(), "void")).
		Add(token.New(token.Id, sp(), "void"))
	p := parse.New()
	_, err := p.Expr(toks)
	assert.NotNil(t, err)
	DumpErrors(t, p.Errors())
}

func TestPrecedenceUnaryParens(t *testing.T) {
	toks := &token.Tokens{}
	// (*s).f This should be parsed as (. (* s) f)) due to parens -- see
	// TestPrecedenceUnary.
	toks.Add(token.New(token.LParen, sp(), "")).
		Add(token.New(token.Star, sp(), "")).
		Add(token.New(token.Id, sp(), "s")).
		Add(token.New(token.RParen, sp(), "")).
		Add(token.New(token.Dot, sp(), "")).
		Add(token.New(token.Id, sp(), "f"))
	p := parse.New()
	want := &node.OpBinary{
		Op: node.OPBIN_STRUCTDEC,
		Left: &node.OpUnary{
			Op: node.OPUN_DEREF,
			To: &node.Variable{Value: "s"},
		},
		Right: &node.Variable{Value: "f"},
	}
	got, err := p.Expr(toks)
	assert.Nil(t, err)
	assert.Equalf(t, want, got, "want: %s, got %s", want, got)
	DumpErrors(t, p.Errors())
}
