package cfg_test

import (
	"io/ioutil"
	"testing"

	"github.com/susji/c0/analyze"
	"github.com/susji/c0/cfg"
	"github.com/susji/c0/lex"
	"github.com/susji/c0/node"
	"github.com/susji/c0/parse"
	"github.com/susji/c0/testers/assert"
	"github.com/susji/c0/testers/require"
)

func nodes(t *testing.T, code string) ([]node.Node, *analyze.Analyzer) {
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
	return nn, a
}

func render(c *cfg.CFG) {
	ioutil.WriteFile("test.dot", []byte(c.Dot()), 0644)
}

func TestBasic(t *testing.T) {
	n, a := nodes(t, "void a() { return; }")
	c, cerrs := cfg.Form(n[0].(*node.FunDef))
	_ = a
	require.NotNil(t, c)
	require.Equal(t, 0, len(cerrs))
	//render(c)
}

func matchernums(i int32) []cfg.NodeCb {
	nums := []cfg.NodeCb{}
	for j := int32(0); j < i; j++ {
		nums = append(nums, matchernum(j))
	}
	return nums
}

func matchernum(i int32) cfg.NodeCb {
	return func(n node.Node) bool {
		switch t := n.(type) {
		case *node.Numeric:
			return t.Value == i
		}
		return false
	}
}

func matcherret(i int32) cfg.NodeCb {
	return func(n node.Node) bool {
		switch t := n.(type) {
		case *node.Return:
			return t.Expr.(*node.Numeric).Value == i
		}
		return false
	}
}

func TestIfElse(t *testing.T) {
	n, a := nodes(t, `
void f() {
	0;
	if (true)
		1;
	else
		2;
	3;
}`)
	c, cerrs := cfg.Form(n[0].(*node.FunDef))
	_ = a
	require.NotNil(t, c)
	require.Equal(t, 0, len(cerrs))

	nums := matchernums(4)
	assert.True(t, c.Connect(nums[0], nums[1]))
	assert.True(t, c.Connect(nums[0], nums[2]))
	assert.True(t, c.Connect(nums[0], nums[3]))
	assert.False(t, c.Connect(nums[1], nums[2]))
}

func TestIfEarlyReturn(t *testing.T) {
	n, a := nodes(t, `
int a() {
	0;
	if (true) {
		1;
		return 10;
	}
	2;
	return 20;
}`)
	c, cerrs := cfg.Form(n[0].(*node.FunDef))
	_ = a
	require.NotNil(t, c)
	require.Equal(t, 0, len(cerrs))

	nums := matchernums(3)

	retfirst := matcherret(10)
	retsecond := matcherret(20)
	assert.True(t, c.Connect(nums[0], nums[1]))
	assert.True(t, c.Connect(nums[0], nums[2]))
	assert.True(t, c.Connect(nums[1], retfirst))
	assert.False(t, c.Connect(nums[1], nums[0]))
	assert.False(t, c.Connect(nums[2], nums[0]))
	assert.False(t, c.Connect(nums[2], nums[1]))
	assert.False(t, c.Connect(nums[1], retsecond))
}

func TestIfHarder(t *testing.T) {
	n, a := nodes(t, `
int a() {
	0;
	bool zap = true;
	int ret = 0;
	if (zap) {
		1;
		zap = false;
		if (!zap) {
			2;
			zap = true;
			if (zap) {
				3;
			} else {
				4;
				return 10;
			}
			5;
		}
		6;
	} else {
		7;
		return 20;
	}
	8;
	return 30;
}`)
	nums := matchernums(9)
	retfirst := matcherret(10)
	retsecond := matcherret(20)
	retthird := matcherret(30)
	c, cerrs := cfg.Form(n[0].(*node.FunDef))
	_ = a
	require.NotNil(t, c)
	require.Equal(t, 0, len(cerrs))
	// I know...
	assert.True(t, c.Connect(nums[0], nums[4]))
	assert.True(t, c.Connect(nums[0], nums[8]))
	assert.True(t, c.Connect(nums[2], nums[5]))
	assert.True(t, c.Connect(nums[2], nums[4]))
	assert.True(t, c.Connect(nums[0], retfirst))
	assert.True(t, c.Connect(nums[0], retsecond))
	assert.True(t, c.Connect(nums[0], retthird))
	assert.True(t, c.Connect(nums[4], retfirst))
	assert.True(t, c.Connect(nums[7], retsecond))
	assert.True(t, c.Connect(nums[8], retthird))
	assert.False(t, c.Connect(nums[1], nums[7]))
	assert.False(t, c.Connect(nums[3], nums[4]))
	assert.False(t, c.Connect(nums[4], retthird))
	assert.False(t, c.Connect(nums[4], retthird))
	assert.False(t, c.Connect(nums[2], retsecond))
	assert.False(t, c.Connect(nums[4], retsecond))
	assert.False(t, c.Connect(nums[5], retsecond))
	//render(c)
}

func TestWhileSimple(t *testing.T) {
	n, a := nodes(t, `
int a() {
	int i;
	0;
	while (i < 10) {
		1;
		if (i > 5) {
			2;
		}
		3;
		i++;
	}
	return 10;
}`)
	c, cerrs := cfg.Form(n[0].(*node.FunDef))
	_ = a
	require.NotNil(t, c)
	require.Equal(t, 0, len(cerrs))

	nums := matchernums(4)
	ret := matcherret(10)
	assert.True(t, c.Connect(nil, ret))
	assert.True(t, c.Connect(nums[0], nums[1]))
	assert.True(t, c.Connect(nums[0], nums[2]))
	assert.True(t, c.Connect(nums[0], nums[3]))
	assert.True(t, c.Connect(nums[0], ret))
	assert.False(t, c.Connect(nums[3], nums[1]))
	//render(c)
}

func TestForSimple(t *testing.T) {
	n, a := nodes(t, `
int a() {
	0;
	int zap = 0;
	for (int i = 0; i < 10; i++) {
		1;
		if (i > 5) {
			2;
			zap++;
		}
		3;
	}
	4;
	return 10;
}`)
	c, cerrs := cfg.Form(n[0].(*node.FunDef))
	_ = a
	nums := matchernums(5)
	ret := matcherret(10)
	require.NotNil(t, c)
	require.Equal(t, 0, len(cerrs))
	assert.True(t, c.Connect(nil, ret))
	for i := 0; i < 4; i++ {
		assert.True(t, c.Connect(nil, nums[i]))
	}
	assert.True(t, c.Connect(nums[1], nums[4]))
	assert.True(t, c.Connect(nums[3], nums[4]))
	//render(c)
}

func TestComplex(t *testing.T) {
	n, a := nodes(t, `
int a() {
	0;
	int i;
	if (i == 0) {
		1;
		while (i < 10) {
			2;
			i++;
		}
		3;
		i;
	} else {
		4;
		for (int j = 0; j < 5; j++) {
			5;
			i--;
		}
		6;
		j;
	}
	7;
	return 10;
}`)
	c, cerrs := cfg.Form(n[0].(*node.FunDef))
	_ = a
	require.NotNil(t, c)
	require.Equal(t, 0, len(cerrs))
	nums := matchernums(8)
	for i := 0; i < 7; i++ {
		assert.True(t, c.Connect(nil, nums[i]))
	}
	assert.True(t, c.Connect(nums[3], nums[7]))
	assert.True(t, c.Connect(nums[6], nums[7]))
	assert.False(t, c.Connect(nums[1], nums[4]))
	assert.False(t, c.Connect(nums[3], nums[6]))
	//render(c)
}

func TestForBreak(t *testing.T) {
	n, a := nodes(t, `
int a() {
	0;
	for (int i = 0; i < 10; i++) {
		if (i > 5) {
			1;
			break;
			2;
		} else {
			3;
		}
	}
	4;
	return 10;
}`)
	c, cerrs := cfg.Form(n[0].(*node.FunDef))
	_ = a
	nums := matchernums(5)
	ret := matcherret(10)
	require.NotNil(t, c)
	require.Equal(t, 0, len(cerrs))
	assert.True(t, c.Connect(nil, ret))
	assert.True(t, c.Connect(nums[0], nums[1]))
	assert.True(t, c.Connect(nums[0], nums[3]))
	assert.True(t, c.Connect(nums[0], nums[4]))
	assert.True(t, c.Connect(nums[1], nums[4]))
	assert.True(t, c.Connect(nums[3], nums[4]))
	assert.False(t, c.Connect(nums[0], nums[2]))
	assert.False(t, c.Connect(nums[1], nums[2]))
	assert.False(t, c.Connect(nums[1], nums[3]))
	assert.False(t, c.Connect(nums[2], nums[3]))
	assert.False(t, c.Connect(nums[2], nums[4]))
	//render(c)
}

func TestForBreakNoElse(t *testing.T) {
	n, a := nodes(t, `
int a() {
	0;
	for (int i = 0; i < 10; i++) {
		if (i > 5) {
			1;
			break;
			2;
		}
		3;
	}
	4;
	return 10;
}`)
	c, cerrs := cfg.Form(n[0].(*node.FunDef))
	_ = a
	nums := matchernums(5)
	ret := matcherret(10)
	require.NotNil(t, c)
	require.Equal(t, 0, len(cerrs))
	assert.True(t, c.Connect(nil, ret))
	assert.True(t, c.Connect(nums[0], nums[1]))
	assert.True(t, c.Connect(nums[0], nums[3]))
	assert.True(t, c.Connect(nums[0], nums[4]))
	assert.True(t, c.Connect(nums[1], nums[4]))
	assert.True(t, c.Connect(nums[3], nums[4]))
	assert.False(t, c.Connect(nums[0], nums[2]))
	assert.False(t, c.Connect(nums[1], nums[2]))
	assert.False(t, c.Connect(nums[1], nums[3]))
	assert.False(t, c.Connect(nums[2], nums[3]))
	assert.False(t, c.Connect(nums[2], nums[4]))
	//render(c)
}

func TestForContinue(t *testing.T) {
	n, a := nodes(t, `
int a() {
	0;
	for (int i = 0; i < 10; i++) {
		if (i > 5) {
			1;
			continue;
			2;
		}
		3;
	}
	4;
	return 10;
}`)
	c, cerrs := cfg.Form(n[0].(*node.FunDef))
	_ = a
	nums := matchernums(5)
	ret := matcherret(10)
	require.NotNil(t, c)
	require.Equal(t, 0, len(cerrs))
	assert.True(t, c.Connect(nil, ret))
	assert.True(t, c.Connect(nums[0], nums[1]))
	assert.True(t, c.Connect(nums[0], nums[3]))
	assert.True(t, c.Connect(nums[0], nums[4]))
	assert.True(t, c.Connect(nums[1], nums[4]))
	assert.True(t, c.Connect(nums[3], nums[4]))
	assert.True(t, c.Connect(nums[1], nums[3]))
	assert.False(t, c.Connect(nums[0], nums[2]))
	assert.False(t, c.Connect(nums[1], nums[2]))
	assert.False(t, c.Connect(nums[2], nums[3]))
	assert.False(t, c.Connect(nums[2], nums[4]))
	//render(c)
}
