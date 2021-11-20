package token_test

import (
	"testing"

	"github.com/susji/c0/span"
	"github.com/susji/c0/testers/assert"
	"github.com/susji/c0/token"
)

func sp() span.Span {
	return span.Span{}
}

func TestTokensFind(t *testing.T) {
	toks := &token.Tokens{}
	toks.Add(token.New(token.DecNum, sp(), "1")).
		Add(token.New(token.DecNum, sp(), "2")).
		Add(token.New(token.DecNum, sp(), "3")).
		Add(token.New(token.DecNum, sp(), "4")).
		Add(token.New(token.Id, sp(), "one")).
		Add(token.New(token.DecNum, sp(), "5")).
		Add(token.New(token.DecNum, sp(), "6")).
		Add(token.New(token.Id, sp(), "two")).
		Add(token.New(token.Id, sp(), "three")).
		Add(token.New(token.DecNum, sp(), "7")).
		Add(token.New(token.HexNum, sp(), "0x123"))

	first := toks.Find(token.Id)
	toks.Pop()
	second := toks.Find(token.Id)
	toks.Pop()
	third := toks.Find(token.Id)
	toks.Pop()
	fourth := toks.Find(token.HexNum, token.DecNum)
	toks.Pop()
	fifth := toks.Find(token.HexNum, token.DecNum)
	toks.Pop()
	assert.Nil(t, toks.Peek())

	assert.NotNil(t, first)
	assert.NotNil(t, second)
	assert.NotNil(t, third)
	assert.NotNil(t, fourth)
	assert.NotNil(t, fifth)
	assert.Equal(t, "one", first.Value())
	assert.Equal(t, "two", second.Value())
	assert.Equal(t, "three", third.Value())
	assert.Equal(t, "7", fourth.Value())
	assert.Equal(t, "0x123", fifth.Value())
}
