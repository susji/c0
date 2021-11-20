package analyze_test

import (
	"errors"
	"testing"

	"github.com/susji/c0/analyze"
	"github.com/susji/c0/lex"
	"github.com/susji/c0/node"
	"github.com/susji/c0/parse"

	"github.com/susji/c0/testers/assert"
	"github.com/susji/c0/testers/require"
)

func nodes(t *testing.T, code string) ([]node.Node, *analyze.Analyzer) {
	toks, lexerrs := lex.Lex([]rune(code))
	if len(lexerrs) > 0 {
		t.Log("lex errors:", lexerrs)
	}

	p := parse.New()
	require.Equal(t, 0, len(lexerrs))
	perr := p.Parse(toks)
	n := p.Nodes()
	t.Logf("parse nodes[%d]: %s\n", len(n), n)
	if perr != nil {
		t.Log("parse errors:    ", p.Errors())
	}
	require.Nil(t, perr)
	require.NotNil(t, p.Nodes())
	return n, analyze.New(p.Fn())
}

func TestSmoke(t *testing.T) {
	n, s := nodes(t, "void a() { return; }")
	assert.Equal(t, 0, len(s.Analyze(n)))
}

func TestArith(t *testing.T) {
	n, s := nodes(t, "bool f() { int a; int b; return a < b; }")
	assert.Equal(t, 1, len(n))
	errs := s.Analyze(n)
	t.Log(errs)
	assert.Equal(t, 0, len(errs))
}

func TestFailArith(t *testing.T) {
	n, s := nodes(t, "bool f() { int a; bool b; return a < b; }")
	assert.Equal(t, 1, len(n))
	errs := s.Analyze(n)
	t.Log(errs)
	assert.True(t, len(errs) == 1)
	assert.True(t, errors.Is(errs[0], analyze.ErrCompareNonInteger))
}

func TestTernary(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{"void a() { true ? 1; }", analyze.ErrTernaryMissingValue},
		{"void b() { 1 : 0; }", analyze.ErrTernaryMissingCond},
		{"void c() { true ? 1 : 0; }", nil},
		{`void d() { "jep" ? 1 : 0; }`, analyze.ErrTernaryCondBool},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			require.Equal(t, 1, len(n))
			errs := s.Analyze(n)
			t.Log(errs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(errs))
			} else {
				require.True(t, len(errs) > 0)
				assert.True(t, errors.Is(errs[0], cur.wanterr))
			}
		})
	}
}

func TestTypedef(t *testing.T) {
	type entry struct {
		code     string
		wanterrs []error
	}
	table := []entry{
		{
			code:     "typedef int zab; void f() { zab z = 10; }",
			wanterrs: nil,
		},
		{
			code:     `typedef int zeb; void f() { zeb z = "jep"; }`,
			wanterrs: []error{analyze.ErrAssignTypeMismatch},
		},
		{
			code:     `typedef int zib; void f() { zib *z = NULL; }`,
			wanterrs: nil,
		},
		{
			code:     `typedef int* zob; void f() { zob z = NULL; }`,
			wanterrs: nil,
		},
		{
			code:     `typedef int zub; void f() { zub z = NULL; }`,
			wanterrs: []error{analyze.ErrAssignTypeMismatch},
		},
		{
			code:     `typedef int[] zyb; void f() { zyb* z = NULL; }`,
			wanterrs: nil,
		},
		{
			code: `
struct some {
	int a;
};
typedef struct some ss;
void f() {
	ss a;
	a.a = 1;
}`,
			wanterrs: nil,
		},
		{
			code: `
typedef int zub;
void f() {
	zub a = 1;
	if (a > 0) {
		return;
	} else {
		a++;
	}
}`,
			wanterrs: []error{},
		},
	}
	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			assert.True(t, len(n) >= 2)
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if len(cur.wanterrs) == 0 {
				t.Log("errors:", goterrs)
				assert.Equal(t, 0, len(goterrs))
			} else {
				for _, curerr := range cur.wanterrs {
					found := false
					for _, goterr := range goterrs {
						if errors.Is(goterr, curerr) {
							found = true
							break
						}
					}
					assert.True(t, found)
				}
			}
		})
	}
}

func TestFuncalls(t *testing.T) {
	type entry struct {
		code     string
		wanterrs []error
	}

	table := []entry{
		{
			code:     "void f(int a) { f(a+1); }",
			wanterrs: []error{},
		},
		{
			code:     "void f() { g(); }",
			wanterrs: []error{analyze.ErrFuncallNotFound},
		},
		{
			code:     `void a(int b, bool c) { a("jep", false); }`,
			wanterrs: []error{analyze.ErrFuncallArgType},
		},
		{
			code:     `void a(int b, bool c) { a(1); }`,
			wanterrs: []error{analyze.ErrFuncallArgsAmount},
		},
		{
			code:     `void z() { int *z; (*z)(); }`,
			wanterrs: []error{analyze.ErrVarDeclShadowsFunction},
		},
		{
			code:     `void x() { int *z; (*z)(); }`,
			wanterrs: []error{analyze.ErrFuncallWrongPtrType},
		},
		{
			code: `
struct st {int a;};
void x(struct st* a) { x(a); }
`,
			wanterrs: nil,
		},
		{
			code: `
struct st {int a;};
void x(struct st a) { x(a); }
`,
			wanterrs: []error{analyze.ErrFuncParamStruct},
		},
		{
			code: `
struct st {int a;};
void x(struct st *a) { struct st b; x(b); }
`,
			wanterrs: []error{analyze.ErrFuncallArgType},
		},
		{
			// This one is almost straight out of the C0 Reference.
			code: `
typedef bool cmp(void* p, void* q);
bool lesserer(void *a, void *b) {
	return *(int *)a < *(int *)b;
}
bool f() {
	cmp *ptr = &lesserer;
	int *a;
	int *b;
	return (*ptr)((void*)a, (void*)b);
}
`,
			wanterrs: []error{},
		},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if len(cur.wanterrs) == 0 {
				assert.Equal(t, 0, len(goterrs))
			} else {
				for _, curerr := range cur.wanterrs {
					found := false
					for _, goterr := range goterrs {
						if errors.Is(goterr, curerr) {
							found = true
							break
						}
					}
					assert.Truef(t, found, "did not see error: %s", curerr)
				}
			}
		})
	}
}

func TestLValues(t *testing.T) {
	type entry struct {
		code     string
		wanterrs []error
	}

	table := []entry{
		{
			code:     "void f(int a) { a = 1; }",
			wanterrs: nil,
		},
		{
			code:     "void f() { f = f; }",
			wanterrs: []error{analyze.ErrAssignNotLValue},
		},
		{
			code:     "void f() { 1 = 2; }",
			wanterrs: []error{analyze.ErrAssignNotLValue},
		},
		{
			code:     "void f() { int a = 'd'; }",
			wanterrs: []error{analyze.ErrAssignTypeMismatch},
		},
		{
			code:     "typedef void ptr(); void f() { ptr *zap = &f; }",
			wanterrs: nil,
		},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if len(cur.wanterrs) == 0 {
				assert.Equal(t, 0, len(goterrs))
			} else {
				for _, curerr := range cur.wanterrs {
					found := false
					for _, goterr := range goterrs {
						if errors.Is(goterr, curerr) {
							found = true
							break
						}
					}
					assert.Truef(t, found, "did not see error: %s", curerr)
				}
			}
		})
	}
}

func TestArrayAlloc(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`void f() { int[] a = alloc_array(int, 1+2+3); }`, nil},
		{`void f() { int[][] a = alloc_array(int[], 1+2+3); }`, nil},
		{`void f() { int[] a = alloc_array(int, true); }`, analyze.ErrAllocArrayBadExpr},
		{`void f() { int[] a = alloc_array(int[], 1); }`, analyze.ErrAssignTypeMismatch},
		{`void f() { int a = alloc_array(int, 1); }`, analyze.ErrAssignTypeMismatch},
		{`void f() { int[][] a = alloc_array(int, 1); }`, analyze.ErrAssignTypeMismatch},
		{`void f() { string[] a = alloc_array(int, 1); }`, analyze.ErrAssignTypeMismatch},
		{`
struct zap {
	int[] ai;
	string[] as;
};
struct zapzap {
	struct zap nested;
};
void f() {
	struct zapzap arr;
	arr.nested.ai = alloc_array(int, 1);
	arr.nested.as = alloc_array(string, 1);
}
`,
			nil},
		{`
struct zap {
	string[] as;
};
struct zapzap {
	struct zap nested;
};
void f() {
	struct zapzap arr;
	arr.nested.as = alloc_array(int, 1);
}
`,
			analyze.ErrAssignTypeMismatch},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			require.True(t, len(n) >= 1)
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(goterrs))
			} else {
				require.True(t, len(goterrs) > 0)
				assert.True(t, errors.Is(goterrs[0], cur.wanterr))
			}
		})
	}
}

func TestAlloc(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`void f() { int *a = alloc(int); }`, nil},
		{`void h() { int **a = alloc(int*); }`, nil},
		{`void g() { int **a = alloc(int); }`, analyze.ErrAssignTypeMismatch},
		{`void g() { int *a = alloc(bool); }`, analyze.ErrAssignTypeMismatch},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			require.Equal(t, 1, len(n))
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(goterrs))
			} else {
				require.True(t, len(goterrs) > 0)
				assert.True(t, errors.Is(goterrs[0], cur.wanterr))
			}
		})
	}
}

func TestPointer(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`void f() { int *a; *a = 10; }`, nil},
		{`void g() { int *a; a = 10; }`, analyze.ErrAssignTypeMismatch},
		{`void h() { int **a; *a = 10; }`, analyze.ErrAssignTypeMismatch},
		{`void i() { int **a; **a = 10; }`, nil},
		{`void j() { bool *a; *a = 10; }`, analyze.ErrAssignTypeMismatch},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			require.Equal(t, 1, len(n))
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(goterrs))
			} else {
				require.True(t, len(goterrs) > 0)
				assert.True(t, errors.Is(goterrs[0], cur.wanterr))
			}
		})
	}
}

func TestArraySub(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`void f() { int[] a; a[0] = 1; }`, nil},
		{`void f() { int[] a; a = 1; }`, analyze.ErrAssignTypeMismatch},
		{`void f() { int[] a; a[0][1] = 1; }`, analyze.ErrArraySubNotArray},
		{`void f() { int[] a; int b; b = a; }`, analyze.ErrAssignTypeMismatch},
		{`void f() { int[] a; int[][] b; b[0] = a; }`, nil},
		{`void f() { int[] a; int b; b = a[9]; }`, nil},
		{`void f() { int[] a; int b; b = a[1+2*3]; }`, nil},
		{`void f() { int[][][] a; int b; b = a[0][1][2]; }`, nil},
		{`void f() { int[] a; string b; b = a[9]; }`, analyze.ErrAssignTypeMismatch},
		{`void f() { int[] a; int b; b = a["jep"]; }`, analyze.ErrArraySubNotInt},
		{`void f() { int[] a; a[0] = 1; }`, nil},
		{`void f() { int[] a; a[0][0] = 1; }`, analyze.ErrArraySubNotArray},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			require.Equal(t, 1, len(n))
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(goterrs))
			} else {
				require.True(t, len(goterrs) > 0)
				assert.True(t, errors.Is(goterrs[0], cur.wanterr))
			}
		})
	}
}

func TestStruct(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`
struct s { int a; bool b; };
void f() {
	struct s zap;
	zap.a = 1;
	zap.b = true;
}`,
			nil,
		},
		{`
struct p { int a; bool b; };
void f() {
	struct p *zap;
	zap.a = 1;
}`,
			analyze.ErrStructBadType,
		},
		{`
struct s { int a; bool b; };
void f() {
	struct s zap;
	zap->a = 1;
}`,
			analyze.ErrStructBadType,
		},
		{`
struct s { int a; bool b; };
void f() {
	struct s[] zap;
	zap->a = 1;
}`,
			analyze.ErrStructBadType,
		},
		{`
struct s { int a; bool b; };
void f() {
	struct s[] zap;
	zap[0].a = 1;
}`,
			nil,
		},
		{`
struct p { int a; bool b; };
void f() {
	struct p *zap;
	zap->a = 1;
}`,
			nil,
		},
		{`
struct p { int a; bool b; };
void f() {
	struct p ***zap;
	(**zap)->a = 1;
}`,
			nil,
		},
		{`
struct p { int a; bool b; };
void f() {
	struct p ***zap;
	**zap->a = 1;
}`,
			analyze.ErrStructBadType,
		},

		{`
struct p { int a; bool b; };
void f() {
	struct p ***zap;
	zap->a = 1;
}`,
			analyze.ErrStructBadType,
		},
		{`
struct s { int a; bool b; };
void f() {
	struct s zap;
	zap.a = true;
	zap.b = true;
}`,
			analyze.ErrAssignTypeMismatch,
		},
		{`
struct s { int a; bool b; };
void f() {
	struct s zap;
	zap.a = 1;
	zap.c = 2;
}`,
			analyze.ErrStructDecFieldNotFound,
		},
		{`
struct s { int a; bool b; };
void f() {
	struct s zap;
	zap.123 = 1;
}`,
			analyze.ErrStructDecNotField,
		},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			require.Equal(t, 2, len(n))
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(goterrs))
			} else {
				require.Truef(t, len(goterrs) > 0, "no errors found")
				found := false
				for _, goterr := range goterrs {
					if errors.Is(goterr, cur.wanterr) {
						found = true
					}
				}
				assert.True(t, found)
			}
		})
	}
}

func TestStructTypedefCall(t *testing.T) {
	code := `
struct somestruct {
	int a;
	bool b;
	string[] c;
};

typedef struct somestruct* s;

int structer(s st) {
	return st->a;
}

int f() {
	s st = alloc(struct somestruct);
	return structer(st);
}`
	n, s := nodes(t, code)
	require.Equal(t, 4, len(n))
	goterrs := s.Analyze(n)
	assert.Equal(t, 0, len(goterrs))
}

// Struct forward-declarations mean we should be able to use them as pointer
// types only.
func TestStructForward(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`
struct fwd;
struct fwd * zapper(struct fwd *zap) {
	return zap;
}
void f() {
	struct fwd *zap;
	zap = zapper(zap);
}`,
			nil},
		{`
struct fwd;
void f() {
	struct fwd zap;
}`,
			analyze.ErrStructOnlyForward},
		{`
struct fwd * zapper(struct fwd *zap) {
	return zap;
}`,
			analyze.ErrTypeUnrecognizedStruct},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(goterrs))
			} else {
				require.True(t, len(goterrs) > 0)
				assert.True(t, errors.Is(goterrs[0], cur.wanterr))
			}
		})
	}
}

func TestStructNested(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`
struct nested {
	int a;
};
struct nester {
	struct nested wrapped;
};
int f() {
	struct nester n;
	return n.wrapped.a;
}
`,
			nil,
		},
		{`
struct nested;
typedef struct nested* nestedptr;
struct nester {
	struct nested *a;
	nestedptr b;
};
`,
			nil},
		{`
struct nested;
typedef struct nested* nestedptr;
struct nester {
	struct nested a;
};`,
			analyze.ErrStructSizeUnknown},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(goterrs))
			} else {
				require.True(t, len(goterrs) > 0)
				assert.True(t, errors.Is(goterrs[0], cur.wanterr))
			}
		})
	}
}

func TestLoop(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`
int f() {
	int i = 10;
	while (i > 0) {
		i--;
		break;
		continue;
	}
	return 0;
}
`,
			nil,
		},
		{`
int f() {
	break;
}
`,
			analyze.ErrBreakOutsideLoop,
		},
		{`
int f() {
	continue;
}
`,
			analyze.ErrContinueOutsideLoop,
		},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(goterrs))
			} else {
				require.True(t, len(goterrs) > 0)
				assert.True(t, errors.Is(goterrs[0], cur.wanterr))
			}
		})
	}
}

func TestReturn(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`
int f() {
	return 1;
}
`,
			nil,
		},
		{`
void f() {
	return;
}
`,
			nil,
		},
		{`
int f() {
	return;
}
`,
			analyze.ErrReturnExprMissing,
		},
		{`
void f() {
	return 123;
}
`,
			analyze.ErrReturnMistyped,
		},
		{`
int f() {
}
`,
			analyze.ErrReturnMissing,
		},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(goterrs))
			} else {
				require.True(t, len(goterrs) > 0)
				assert.True(t, errors.Is(goterrs[0], cur.wanterr))
			}
		})
	}
}

func TestComparison(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`
void f() {
	int a; int b;
	a == b;
}
`,
			nil,
		},
		{`
void g() {
	char a; char b;
	a == b;
}
`,
			nil,
		},
		{`
void h() {
	string[] a; string[] b;
	a != b;
}
`,
			nil,
		},
		{`
void h() {
	string a; string b;
	a != b;
}
`,
			analyze.ErrCompareBadType,
		},
		{`
void h() {
	int *a; int *b;
	a != b;
}
`,
			analyze.ErrCompareBadType,
		},
		{`
void h() {
	int a; int *b;
	a != b;
}
`,
			analyze.ErrCompareBadType,
		},
		{`
struct st {
	int x;
};
void h() {
	struct st a; struct st b;
	a.x == b.x;
}
`,
			nil,
		},
		{`
struct st {
	int x;
};
void h() {
	struct st a; struct st b;
	a == b;
}
`,
			analyze.ErrCompareBadType,
		},
		{`
struct st {
	int x;
};
void h() {
	struct st* a; struct st *b;
	a == b;
}
`,
			analyze.ErrCompareBadType,
		},
		{`
struct st {
	int x;
};
void h() {
	struct st[] a; struct st[] b;
	a == b;
}
`,
			nil,
		},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			goterrs := s.Analyze(n)
			t.Log(goterrs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(goterrs))
			} else {
				require.True(t, len(goterrs) > 0)
				assert.True(t, errors.Is(goterrs[0], cur.wanterr))
			}
		})
	}
}

func TestVoid(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`
void f() {
	void a;
}
`,
			analyze.ErrVarDeclVoid,
		},
		{`
void f() {
	void *a;
}
`,
			nil,
		},
		{`
void f() {
	void []a;
}
`,
			analyze.ErrVarDeclVoid,
		},
		{`
void f() {
	int *zap;
	void *a = (void *)zap;
}
`,
			nil,
		},
		{`
void f() {
	int zap;
	void *a = (void *)zap;
}
`,
			analyze.ErrCastVoidPointer,
		},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			errs := s.Analyze(n)
			t.Log(errs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(errs))
			} else {
				require.True(t, len(errs) > 0)
				assert.True(t, errors.Is(errs[0], cur.wanterr))
			}
		})
	}
}

func TestUnary(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`
void f() {
	bool b;
	!b;
}
`,
			nil,
		},
		{`
void f() {
	int b;
	!b;
}
`,
			analyze.ErrNegateNonBool,
		},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			errs := s.Analyze(n)
			t.Log(errs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(errs))
			} else {
				require.True(t, len(errs) > 0)
				assert.True(t, errors.Is(errs[0], cur.wanterr))
			}
		})
	}
}

func TestAssign(t *testing.T) {
	type entry struct {
		code    string
		wanterr error
	}

	table := []entry{
		{`
void f() {
	int i;
}
`,
			nil,
		},
		{`
void f() {
	int i = 1;
}
`,
			nil,
		},
		{`
void g() {
	int i = i;
}
`,
			analyze.ErrVarNotDefined,
		},
		{`
void g() {
	string i = 123;
}
`,
			analyze.ErrAssignTypeMismatch,
		},
	}

	for _, cur := range table {
		t.Run(cur.code, func(t *testing.T) {
			n, s := nodes(t, cur.code)
			errs := s.Analyze(n)
			t.Log(errs)
			if cur.wanterr == nil {
				assert.Equal(t, 0, len(errs))
			} else {
				require.True(t, len(errs) > 0)
				assert.True(t, errors.Is(errs[0], cur.wanterr))
			}
		})
	}
}
