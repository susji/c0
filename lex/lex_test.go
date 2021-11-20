package lex_test

import (
	"fmt"
	"testing"

	"github.com/susji/c0/lex"
	pr "github.com/susji/c0/primitives"
	"github.com/susji/c0/span"
	"github.com/susji/c0/testers/assert"
	"github.com/susji/c0/testers/require"
	"github.com/susji/c0/token"
)

func TestStrLit(t *testing.T) {
	type entry struct {
		give, want, left string
	}

	table := []entry{
		{`"string literal"`, `string literal`, ""},
		{`"\nmore\nlines\t\n" rest`, "\nmore\nlines\t\n", " rest"},
	}

	for _, cur := range table {
		t.Run(cur.give, func(t *testing.T) {
			res := lex.StrLit.Do(pr.NewState([]rune(cur.give)))
			require.NotNil(t, res)
			require.Nil(t, res.Error())
			assert.Equal(t, cur.want, res.State().String())
			assert.Equal(t, []rune(cur.left), res.State().Left())
		})
	}
}

func TestIdentifier(t *testing.T) {
	one := "this_is_identifier1"
	two := " and it follows-with-something more!"

	res := lex.Identifier.Do(pr.NewState([]rune(one + two)))
	require.NotNil(t, res)
	require.Nil(t, res.Error())
	assert.Equal(t, one, res.State().String())
	assert.Equal(t, []rune(two), res.State().Left())
}

func TestNumeric(t *testing.T) {
	type entry struct {
		give, left, want string
	}

	table := []entry{
		{"0", "", "0"},
		{"0  ", "  ", "0"},
		{"0123", "123", "0"},
		{"123", "", "123"},
		{"0xcafe", "", "0xcafe"},
	}

	for _, cur := range table {
		t.Run(cur.give, func(t *testing.T) {
			res := lex.DecNum.Or(lex.HexNum).Do(pr.NewState([]rune(cur.give)))
			require.NotNil(t, res)
			require.Nil(t, res.Error())
			assert.Equal(t, []rune(cur.left), res.State().Left())
			assert.Equal(t, cur.want, res.State().String())
		})
	}
}

func TestNumericFail(t *testing.T) {
	table := []string{
		"z1",
		"0xz",
	}

	for _, cur := range table {
		t.Run(cur, func(t *testing.T) {
			res := lex.HexNum.Or(lex.DecNum).Do(pr.NewState([]rune(cur)))
			require.NotNil(t, res)
			require.NotNil(t, res.Error())
		})
	}
}

// TestChrLitEmpty validates that a lexing bug causing infinite looping is
// fixed.
func TestChrLitEmpty(t *testing.T) {
	res, errs := lex.Lex([]rune("''"))
	require.NotNil(t, res)
	require.True(t, len(errs) > 0)
}

func TestChrLit(t *testing.T) {
	type entry struct {
		give string
		want rune
	}
	table := []entry{
		{`'a'`, 'a'},
		{`'\n'`, '\n'},
		{`'\0'`, 0},
	}
	for _, cur := range table {
		t.Run(cur.give, func(t *testing.T) {
			res := lex.ChrLit.Do(pr.NewState([]rune(cur.give)))
			require.NotNil(t, res)
			require.Nil(t, res.Error())
			assert.Equal(t, []rune(""), res.State().Left())
			assert.Equal(t, cur.want, res.State().Value()[0])
		})
	}
}

func TestLibLit(t *testing.T) {
	type entry struct {
		give, want string
	}
	table := []entry{
		{`<jep.h>`, `jep.h`},
		{`<ネコ>`, `ネコ`},
	}
	for _, cur := range table {
		t.Run(cur.give, func(t *testing.T) {
			res := lex.LibLit.Do(pr.NewState([]rune(cur.give)))
			require.NotNil(t, res)
			require.Nil(t, res.Error())
			assert.Equal(t, cur.want, res.State().String())
		})
	}
}

func TestLexSmoke(t *testing.T) {
	table := []string{
		`#use <stdio.h>
int funcer(string one, int *two) {
	print("%s=%d", one, *two);
	return *two;
}

int main() {
	int var = 123;
	var++;
	var -= 5;
	return funcer("this is a string", &var);
}
`,
	}
	for i, cur := range table {
		t.Run(fmt.Sprintf("#%d", i+1), func(t *testing.T) {
			toks, errs := lex.Lex([]rune(cur))
			assert.Equal(t, 0, len(errs))
			if len(errs) > 0 {
				for _, curerr := range errs {
					t.Logf("%v", curerr)
				}
			}
			assert.NotNil(t, toks)
			assert.True(t, 50 < toks.Len())
			t.Log(toks)
		})
	}
}

func TestLexSimple(t *testing.T) {
	type tok struct {
		kind  token.Kind
		value string
	}
	type entry struct {
		give string
		want []tok
	}
	table := []entry{
		{
			give: `
int main() {
	int var = 123;
}
`,
			want: []tok{
				{token.Id, "int"},
				{token.Id, "main"},
				{token.LParen, ""},
				{token.RParen, ""},
				{token.LCurly, ""},
				{token.Id, "int"},
				{token.Id, "var"},
				{token.Assign, ""},
				{token.DecNum, "123"},
				{token.Semicolon, ""},
				{token.RCurly, ""},
			},
		},
	}
	for i, cur := range table {
		t.Run(fmt.Sprintf("#%d", i+1), func(t *testing.T) {
			toks, errs := lex.Lex([]rune(cur.give))
			assert.Equal(t, 0, len(errs))
			t.Log(errs)
			require.NotNil(t, toks)
			assert.True(t, len(cur.want) == toks.Len())

			var tok *token.Token
			i := 0
			for tok = toks.Pop(); tok != nil; tok = toks.Pop() {
				wanttok := cur.want[i]
				assert.Equal(t, wanttok.kind, tok.Kind())
				if len(wanttok.value) > 0 {
					assert.Equal(t, wanttok.value, tok.Value())
				}
				i++
			}
		})
	}
}

func TestLexComments(t *testing.T) {
	type entry struct {
		give, wantmsg string
		wantkind      token.Kind
		wantspan      span.Span
	}
	table := []entry{
		{
			give:     `// one line`,
			wantmsg:  ` one line`,
			wantkind: token.CommentOne,
			wantspan: span.Span{
				Lineno0: 1,
				Col0:    1,
				Lineno:  1,
				Col:     12,
			},
		},
		{
			give: `  /* multi
line
comment*/`,
			wantmsg: ` multi
line
comment`,
			wantkind: token.CommentMulti,
			wantspan: span.Span{
				Lineno0: 1,
				Col0:    3,
				Lineno:  3,
				Col:     10,
			},
		},
	}
	for _, cur := range table {
		t.Run(cur.give, func(t *testing.T) {
			toks, errs := lex.Lex([]rune(cur.give))
			assert.Equal(t, 0, len(errs))
			t.Log(errs)
			require.NotNil(t, toks)
			assert.Equal(t, 1, toks.Len())
			com := toks.PeekAll()
			assert.Equal(t, cur.wantmsg, com.Value())
			assert.Equal(t, cur.wantspan, com.Span())
		})
	}
}

func TestLexPrecedences(t *testing.T) {
	type entry struct {
		give string
		want token.Kind
	}

	table := []entry{
		{"--", token.DMinus},
		{"-", token.Minus},
		{"++", token.DPlus},
		{"==", token.Eq},
		{"=", token.Assign},
		{"+=", token.AssignPlus},
		{"-=", token.AssignMinus},
		{"&=", token.AssignAmpersand},
		{"->", token.Arrow},
		{"&&", token.DAmpersand},
		{"&", token.Ampersand},
	}

	for _, cur := range table {
		t.Run(cur.give, func(t *testing.T) {
			toks, errs := lex.Lex([]rune(cur.give))
			t.Log(errs)
			assert.Equal(t, 0, len(errs))
			require.Equal(t, cur.want, toks.Peek().Kind())
		})
	}
}

func TestLexUse(t *testing.T) {
	type entry struct {
		give     string
		wantkind token.Kind
		wantval  string
	}

	table := []entry{
		{"#use <lib>\n", token.UseLibLit, "lib"},
		{`#use "file.c0"` + "\n", token.UseStrLit, "file.c0"},
	}

	for _, cur := range table {
		t.Run(cur.give, func(t *testing.T) {
			toks, errs := lex.Lex([]rune(cur.give))
			t.Log(errs)
			require.Equal(t, 0, len(errs))
			require.NotNil(t, toks)
			require.Equal(t, cur.wantkind, toks.Peek().Kind())
			require.Equal(t, cur.wantval, toks.Peek().Value())
		})
	}
}

func TestLexLiterals(t *testing.T) {
	type entry struct {
		give     string
		wantkind token.Kind
		wantval  string
	}

	table := []entry{
		{`"zaf.fib"`, token.StrLit, "zaf.fib"},
		{"'c'", token.ChrLit, "c"},
	}

	for _, cur := range table {
		t.Run(cur.give, func(t *testing.T) {
			toks, errs := lex.Lex([]rune(cur.give))
			t.Log(errs)
			require.Equal(t, 0, len(errs))
			require.NotNil(t, toks)
			require.Equal(t, cur.wantkind, toks.Peek().Kind())
			require.Equal(t, cur.wantval, toks.Peek().Value())
		})
	}
}

func TestLexNotLibLit(t *testing.T) {
	code := `i < 0 && j > 10`

	type tok struct {
		kind  token.Kind
		value string
	}

	want := []tok{
		{token.Id, "i"},
		{token.Lt, ""},
		{token.HexNum, "0"},
		{token.DAmpersand, ""},
		{token.Id, "j"},
		{token.Gt, ""},
		{token.DecNum, "10"},
	}

	toks, errs := lex.Lex([]rune(code))
	t.Log(errs)
	require.Equal(t, 0, len(errs))
	require.NotNil(t, toks)
	t.Log(toks)

	i := 0
	for tok := toks.Pop(); tok != nil; tok = toks.Pop() {
		wanttok := want[i]
		assert.Equal(t, wanttok.kind, tok.Kind())
		if len(wanttok.value) > 0 {
			assert.Equal(t, wanttok.value, tok.Value())
		}
		i++
	}

}
