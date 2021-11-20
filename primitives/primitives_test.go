package primitives_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	pr "github.com/susji/c0/primitives"
	"github.com/susji/c0/testers/assert"
	"github.com/susji/c0/testers/require"
)

func TestPrimitives(t *testing.T) {
	type entry struct {
		parser               pr.Parser
		src, expres, expleft string
	}

	table := []entry{
		{
			pr.Rune('z').OneOrMore(),
			"zza",
			"zz",
			"a",
		},
		{
			pr.AnyOf(
				pr.Rune('a'),
				pr.Rune('b'),
				pr.Rune('c')).OneOrMore(),
			"abc!%",
			"abc",
			"!%",
		},
		{
			pr.Rune('c').
				And(pr.Rune('b')).
				And(pr.Rune('a')),
			"cbaaz",
			"cba",
			"az",
		},
		{
			pr.Rune('a').
				Or(pr.Rune('b')).
				Or(pr.Rune('c')).OneOrMore(),
			"abczz",
			"abc",
			"zz",
		},
		{
			pr.String("foobar"),
			"foobarxfoobar",
			"foobar",
			"xfoobar",
		},
		{
			pr.ExceptRunes("ab").OneOrMore(),
			"xyzab",
			"xyz",
			"ab",
		},
		{
			pr.Rune('a').And(pr.Rune('b')),
			"abc",
			"ab",
			"c",
		},
		{
			pr.Rune('a').Optional().
				And(pr.Rune('b').Optional()).
				And(pr.Rune('c')).
				And(pr.Rune('d').Optional()),
			"bc",
			"bc",
			"",
		},
		{
			pr.Runes("abc").OneOrMore(),
			"abbbcd",
			"abbbc",
			"d",
		},
		{
			pr.Strings("eka", "toka", "kolmas").OneOrMore(),
			"ekatokakolmasekazzzzeka",
			"ekatokakolmaseka",
			"zzzzeka",
		},
		{
			pr.Rune('a').And(pr.Chomp('b')).And(pr.Rune('c')),
			"abcd",
			"ac",
			"d",
		},
	}

	for _, cur := range table {
		t.Run(cur.src, func(t *testing.T) {
			res := cur.parser.DoRunes([]rune(cur.src))
			require.Nil(t, res.Error())
			assert.Equal(t, cur.expres, string(res.State().Value()))
			assert.Equal(t, cur.expleft, string(res.State().Left()))
		})
	}
}

func TestFlush(t *testing.T) {
	var flushed1, flushed2 string
	p := pr.Rune('a').
		Or(pr.Rune('b')).
		Or(pr.Rune('c')).
		OneOrMore().
		Deliver(func(rv pr.ResultValue) error {
			flushed1 = string(rv)
			return nil
		}).
		And(pr.Rune('d').
			Or(pr.Rune('e')).
			OneOrMore().
			Deliver(func(rv pr.ResultValue) error {
				flushed2 = string(rv)
				return nil
			}))
	input := "aabbccddeeff"
	res := p.DoRunes([]rune(input))
	require.NotNil(t, res)
	assert.Equal(t, pr.ResultValue(""), res.State().Value())
	assert.Equal(t, []rune("ff"), res.State().Left())
	assert.Equal(t, "aabbcc", flushed1)
	assert.Equal(t, "ddee", flushed2)
}

func TestMap(t *testing.T) {
	mapper := func(val pr.ResultValue) (res pr.ResultValue) {
		for _, r := range val {
			res = append(res, r+1)
		}
		return res
	}
	res := pr.String("abcd").Map(mapper).DoRunes([]rune("abcd"))
	require.NotNil(t, res)
	assert.Equal(t, pr.ResultValue("bcde"), res.State().Value())
}

func TestQuotedString(t *testing.T) {
	type escpair struct {
		src string
		dst rune
	}
	escpairs := []escpair{
		{`\n`, '\n'},
		{`\t`, '\t'},
		{`\v`, '\v'},
		{`\b`, '\b'},
		{`\r`, '\r'},
		{`\f`, '\f'},
		{`\a`, '\a'},
		{`\\`, '\\'},
		{`\"`, '"'},
	}
	eps := []pr.Parser{}
	for _, cur := range escpairs {
		c := cur
		this := pr.String(c.src).
			Map(func(from pr.ResultValue) pr.ResultValue {
				from = from[:len(from)-2]
				from = append(from, c.dst)
				return from
			})
		eps = append(eps, this)
	}
	ep := pr.AnyOf(eps...)
	q0 := pr.Rune('"').Error("missing start quote")
	q1 := pr.Rune('"').Error("missing end quote")
	ch := pr.ExceptRunes("\"\\")
	str := q0.And(ch.Or(ep).ZeroOrMore()).And(q1)

	type entry struct {
		src string
		exp string
	}

	table := []entry{
		{`"jep"`, `"jep"`},
		{`"jep\n\t"`, "\"jep\n\t\""},
	}

	for _, cur := range table {
		t.Run(cur.src, func(t *testing.T) {
			res := str.DoRunes([]rune(cur.src))
			require.NotNil(t, res)
			require.Nil(t, res.Error())
			require.Equal(t, cur.exp, string(res.State().Value()))
		})
	}
}

func TestDeliverError(t *testing.T) {
	e := errors.New("example error")
	res := pr.Rune('a').
		Deliver(func(val pr.ResultValue) error {
			return e
		}).
		DoRunes([]rune("a"))
	require.NotNil(t, res)
	assert.Equal(t, e, res.Error())
}

func TestIdentifier(t *testing.T) {
	plow := pr.RuneRange('a', 'z')
	pupp := pr.RuneRange('A', 'Z')
	pdig := pr.RuneRange('0', '9')
	pus := pr.Rune('_')
	pid := plow.Or(pupp).Or(pus).Or(plow).
		And(pupp.Or(pus).Or(plow).Or(pdig).ZeroOrMore())

	id := "ThisIsAnIdentifier_1"
	res := pid.DoRunes([]rune(id))
	require.NotNil(t, res)
	require.Equal(t, string(res.State().Value()), id)
}

func TestAst(t *testing.T) {
	type binaryOp struct {
		op       rune
		lhs, rhs int
	}

	ws := pr.Runes(" \t").ZeroOrMore().Discard()

	dp := &strings.Builder{}
	for i := int32(0); i < 10; i++ {
		dp.WriteRune(i + '0')
	}
	digits := pr.Runes(dp.String()).OneOrMore()

	op := pr.Runes("+-*/")

	node := binaryOp{}
	expr := ws.
		And(digits.Error("lhs").
			Deliver(func(lhs pr.ResultValue) error {
				vali, err := strconv.Atoi(string(lhs))
				node.lhs = vali
				return err
			})).
		And(ws).
		And(op.Error("op").
			Deliver(func(opr pr.ResultValue) error {
				node.op = opr[0]
				return nil
			})).
		And(ws).
		And(digits.Error("rhs").
			Deliver(func(rhs pr.ResultValue) error {
				vali, err := strconv.Atoi(string(rhs))
				node.rhs = vali
				return err
			}))

	res := expr.DoRunes([]rune(" 123 + 987 "))
	require.NotNil(t, res)
	assert.Nil(t, res.Error())
	assert.Equal(t, 123, node.lhs)
	assert.Equal(t, 987, node.rhs)
	assert.Equal(t, '+', node.op)
}

// TestStrings tests a case that turned up when the implementation of `String'
// was erroneously returning EOI (end of input) if the candidate string was
// longer than what remained. This was passed back to `AnyOf' which promptly
// determined that there is nothing left to parse even though one of the other
// `Strings' matchers might have succeeded.
func TestStrings(t *testing.T) {
	short := "short"
	long := "looooooooooooong"
	res := pr.Strings(long, short).DoRunes([]rune(short))
	require.NotNil(t, res)
	require.Nil(t, res.Error())
	require.Equal(t, short, res.State().String())
	require.Equal(t, []rune(""), res.State().Left())
}

// TestPartialParse exemplifies the issue with partial parsing. If the
// combinators do not properly maintain a copied state, the partial parse of
// the last "ab" is included in the result.
//
func TestPartialParse(t *testing.T) {
	pb := pr.Rune('a').And(pr.Rune('b')).And(pr.Rune('c'))
	for i, pc := range []pr.Parser{pb.OneOrMore(), pb.ZeroOrMore()} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			res := pc.DoRunes([]rune("abcabcab"))
			require.NotNil(t, res)
			require.Nil(t, res.Error())
			assert.Equal(t, "ab", string(res.State().Left()))
			assert.Equal(t, "abcabc", res.State().String())
		})
	}
}

func LazyRecExpr() pr.Parser {
	return pr.Rune('a').OneOrMore().
		Or(pr.Chomp('(').AndLazy(LazyRecExpr).And(pr.Chomp(')')))
}

// TestLazy makes sure we can use AndLazy to describe recursive productions
// such as
//
//     <program> = '>', <expr>
//     <expr>    = ( 'a', { 'a' } ) | '(' <expr> ')'
//
func TestLazy(t *testing.T) {
	give := ">((((aaa))))"
	p := pr.Chomp('>').And(LazyRecExpr())

	res := p.DoRunes([]rune(give))

	require.NotNil(t, res)
	assert.Nil(t, res.Error())
	assert.Equal(t, []rune(""), res.State().Left())
	assert.Equal(t, "aaa", res.State().String())
}

// TestAndOr makes sure we are properly state-preserving. If we handle
// state-preservation wrong, the first branch matching "ab" will consume the
// first a, and the latter `Or' branch will fail.
func TestAndOr(t *testing.T) {
	res := pr.Rune('a').And(pr.Rune('c')).
		Or(pr.Rune('a').And(pr.Rune('a').And(pr.Rune('b')))).
		DoRunes([]rune("aab"))

	require.NotNil(t, res)
	assert.Nil(t, res.Error())
	assert.Equal(t, []rune(""), res.State().Left())
	assert.Equal(t, "aab", res.State().String())
}

func TestEnd(t *testing.T) {
	ok := "ab"
	nok := ok + " "
	p := pr.Rune('a').And(pr.Rune('b')).And(pr.End())

	res := p.DoRunes([]rune(ok))
	require.NotNil(t, res)
	assert.Nil(t, res.Error())
	assert.Equal(t, ok, res.State().String())
	assert.Equal(t, []rune(""), res.State().Left())

	res = p.DoRunes([]rune(nok))
	require.NotNil(t, res)
	assert.NotNil(t, res.Error())
	assert.Equal(t, ok, res.State().String())
}

func TestExceptString(t *testing.T) {
	contents := `
 * multi
 * line
 * comment `
	test := "/*" + contents + "*/ something else"
	p := pr.String("/*").Discard().
		And(pr.ExceptString("*/").ZeroOrMore()).
		And(pr.Discard(pr.String("*/")))

	res := p.DoRunes([]rune(test))
	require.NotNil(t, res)
	assert.Nil(t, res.Error())
	assert.Equal(t, contents, res.State().String())
}

func TestCaptureMiddle(t *testing.T) {
	val := "our-value-is-this"
	test := `{prefix  ` + val + `   }`
	ws := pr.Runes(" \t\r\v").OneOrMore()
	ab := pr.RuneRange('a', 'z').Or(pr.Rune('-'))
	p := pr.Rune('{').
		And(pr.String("prefix")).
		And(ws).Discard().
		And(ab.OneOrMore()).
		And(pr.Discard(ws)).
		And(pr.Discard(pr.String("}")))
	res := p.DoRunes([]rune(test))
	require.NotNil(t, res)
	assert.Nil(t, res.Error())
	assert.Equal(t, val, res.State().String())
}

func TestFatal(t *testing.T) {
	fat := errors.New("this is a fatal error")
	res := pr.String("abc").FatalRaw(fat).DoRunes([]rune("not abc"))
	require.NotNil(t, res)
	assert.True(t, errors.Is(res.Error(), fat))
}
