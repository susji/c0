// Package primitives implements the basic chassis for parser combinators. The
// implementation here is heavily inspired by Armin Heller's post at
//
//    https://medium.com/@armin.heller/using-parser-combinators-in-go-e63b3ad69c94
//
package primitives

import (
	"errors"
	"fmt"
	"strings"
)

var EOI = errors.New("end of input")
var noMatch = errors.New("no match")

type MapFunc func(ResultValue) ResultValue
type DeliverFunc func(ResultValue) error
type StateFunc func(*State)
type ResultValue []rune
type Parser func(*State) *Result
type ParserGeneratorFunc func() Parser

type State struct {
	left        []rune
	lineno, col int
	value       ResultValue
}

type Result struct {
	state *State
	err   error
}

func resultOk(state *State) *Result {
	return &Result{
		err:   nil,
		state: state,
	}
}

func resultErr(state *State, err error) *Result {
	return &Result{
		err:   fmt.Errorf("%d:%d: %w", state.lineno, state.col, err),
		state: state,
	}
}

func (s *State) _advance(r rune) {
	if r == '\n' {
		s.lineno++
		s.col = 1
	} else {
		s.col++
	}
}

func (s *State) advance(r rune) {
	s._advance(r)
	s.value = append(s.value, r)
}

func Epsilon() Parser {
	return func(state *State) *Result {
		return resultOk(state)
	}
}

func End() Parser {
	return func(state *State) *Result {
		if len(state.left) == 0 {
			return resultOk(state)
		}
		return resultErr(state, fmt.Errorf("runes still left"))
	}
}

func Strings(want ...string) Parser {
	if len(want) < 2 {
		panic("Strings: less than two candidates")
	}
	ps := []Parser{}
	for _, cur := range want {
		ps = append(ps, String(cur))
	}
	return AnyOf(ps...)
}

func ExceptString(donotwant string) Parser {
	return func(state *State) *Result {
		// fastpath!
		if len(state.left) < len(donotwant) {
			return resultErr(state, noMatch)
		}
		got := state.left[:len(donotwant)]
		if donotwant == string(got) {
			return resultErr(
				state,
				fmt.Errorf("wanted to avoid %q, got %q: %w", donotwant, got, noMatch))
		}
		var r rune
		r, state.left = state.left[0], state.left[1:]
		state.advance(r)
		return resultOk(state)
	}
}

func String(want string) Parser {
	return func(state *State) *Result {
		// fastpath!
		if len(state.left) < len(want) {
			return resultErr(state, noMatch)
		}
		got := state.left[:len(want)]
		if want != string(got) {
			return resultErr(
				state,
				fmt.Errorf("wanted to avoid %q, got %q: %w", want, got, noMatch))
		}
		for _, r := range got {
			state.advance(r)
		}
		state.left = state.left[len(want):]
		return resultOk(state)
	}
}

func runescmp(what string, cmp func(candidates string, against rune) bool) Parser {
	return func(state *State) *Result {
		if len(state.left) == 0 {
			return resultErr(state, EOI)
		}
		got := state.left[0]
		if !cmp(what, got) {
			return resultErr(
				state,
				noMatch)
		}
		state.left = state.left[1:]
		state.advance(got)
		return resultOk(state)
	}
}

func ExceptRunes(rs string) Parser {
	if len(rs) == 0 {
		panic("ExceptRunes: no rune candidates")
	}
	return runescmp(rs, func(candidates string, against rune) bool {
		return !strings.ContainsAny(string([]rune{against}), candidates)
	})
}

func Runes(rs string) Parser {
	if len(rs) < 2 {
		panic("Runes: less than two rune candidates")
	}
	return runescmp(rs, func(candidates string, against rune) bool {
		return strings.ContainsAny(string([]rune{against}), candidates)
	})
}

func runer(want rune, action func(rune, *State)) Parser {
	return func(state *State) *Result {
		if len(state.left) == 0 {
			return resultErr(state, EOI)
		}
		got := state.left[0]
		if got != want {
			return resultErr(
				state,
				fmt.Errorf("wanted %q, got %q: %w", want, got, noMatch))
		}
		state.left = state.left[1:]
		action(got, state)
		return resultOk(state)
	}
}

func Rune(want rune) Parser {
	return runer(want, func(r rune, state *State) { state.advance(r) })
}

func Chomp(want rune) Parser {
	return runer(want, func(r rune, state *State) { state._advance(r) })
}

func Discard(p Parser) Parser {
	return func(state *State) *Result {
		res := p(state.copy())
		if res.err == nil {
			res.state.value = state.value
			return resultOk(res.state)
		} else {
			return resultErr(state, noMatch)
		}
	}
}

func RuneRange(r1, r2 rune) Parser {
	if r1 >= r2 {
		panic("RuneRange: invalid range (r1 >= r2)")
	}
	rs := &strings.Builder{}
	for i := r1; i <= r2; i++ {
		rs.WriteRune(i)
	}
	return Runes(rs.String())
}

func AnyOf(parsers ...Parser) Parser {
	if len(parsers) < 2 {
		panic("AnyOf: less than two parsers")
	}

	return func(state *State) *Result {
		for _, p := range parsers {
			res := p(state)
			switch {
			case errors.Is(res.err, EOI):
				fallthrough
			case res.err == nil:
				return res
			default:
				continue
			}
		}
		return resultErr(state, noMatch)
	}
}

func (r *Result) Error() error {
	return r.err
}

func (r *Result) State() *State {
	return r.state
}

func (left Parser) And(right Parser) Parser {
	return func(state *State) *Result {
		res := left(state.copy())
		if res.err != nil {
			return res
		}
		state = res.state
		return right(res.state)
	}
}

func (left Parser) AndLazy(lazygen ParserGeneratorFunc) Parser {
	return func(state *State) *Result {
		res := left(state)
		if res.err != nil {
			return res
		}
		return lazygen()(res.state)
	}
}

func (left Parser) Or(right Parser) Parser {
	return func(state *State) *Result {
		res := left(state)
		if res.err == nil {
			return res
		}
		return right(state)
	}
}

func (left Parser) Discard() Parser {
	return func(state *State) *Result {
		res := left(state)
		res.state.value = ResultValue{}
		return res
	}
}

func (left Parser) Deliver(f DeliverFunc) Parser {
	return func(state *State) *Result {
		res := left(state)
		if res.err != nil {
			return res
		}
		err := f(res.state.Value())
		if err != nil {
			res.err = err
			return res
		}
		res.state.discard()
		return res
	}
}

func (left Parser) Pipe(sf StateFunc) Parser {
	return func(state *State) *Result {
		res := left(state)
		if res.err != nil {
			return res
		}
		sf(res.state)
		return res
	}
}

func (what Parser) Map(mapper MapFunc) Parser {
	return func(state *State) *Result {
		res := what(state)
		if res.err != nil {
			return res
		}
		res.state.value = mapper(res.state.value)
		return res
	}
}

func (what Parser) Optional() Parser {
	return func(state *State) *Result {
		res := what(state)
		switch {
		case errors.Is(res.err, EOI):
			fallthrough
		case errors.Is(res.err, noMatch):
			res.err = nil
			fallthrough
		case res.err == nil:
			fallthrough
		default:
			return res
		}
	}
}

func (what Parser) OneOrMore() Parser {
	return func(state *State) *Result {
		got := 0
		for {
			res := what(state.copy())
			if res.err == nil {
				got++
				state = res.state
			} else {
				break
			}
		}
		if got == 0 {
			return resultErr(state, noMatch)
		}
		return resultOk(state)
	}
}

func (what Parser) ZeroOrMore() Parser {
	return func(state *State) *Result {
		for {
			res := what(state.copy())
			if res.err == nil {
				state = res.state
			} else {
				break
			}
		}
		return resultOk(state)
	}
}

func (what Parser) Error(msg string) Parser {
	return what.ErrorRaw(errors.New(msg))
}

func (what Parser) ErrorRaw(err error) Parser {
	return func(state *State) *Result {
		res := what(state)
		if res.err == nil {
			return res
		}
		if err == nil {
			res.err = err
		} else {
			res.err = fmt.Errorf("%w: %v", err, res.err)
		}
		return res
	}
}

func (what Parser) Fatal(msg string) Parser {
	return what.FatalRaw(errors.New(msg))
}

func (what Parser) FatalRaw(err error) Parser {
	return func(state *State) *Result {
		res := what(state)
		if res.err == nil {
			return res
		}
		if err == nil {
			res.err = err
		} else {
			panic(err)
		}
		return res
	}
}

func (what Parser) Do(state *State) (ret *Result) {
	defer func() {
		if r := recover(); r != nil {
			ret = resultErr(state, r.(error))
		}
	}()
	ret = what(state)
	return ret
}

func (what Parser) DoRunes(runes []rune) (ret *Result) {
	state := NewState(runes)
	defer func() {
		if r := recover(); r != nil {
			ret = resultErr(state, r.(error))
		}
	}()
	ret = what(state)
	return ret
}

func (s *State) Left() []rune {
	return s.left
}

func (s *State) LenLeft() int {
	return len(s.left)
}

func (s *State) Value() ResultValue {
	return s.value
}

func (s *State) String() string {
	return string(s.value)
}

func (s *State) Pos() (int, int) {
	return s.lineno, s.col
}

func (s *State) discard() {
	s.value = ResultValue{}
}

func (s *State) copy() *State {
	return &State{
		lineno: s.lineno,
		col:    s.col,
		left:   s.left,
		value:  s.value,
	}
}

func NewState(what []rune) *State {
	return &State{
		left:   what,
		lineno: 1,
		col:    1,
	}
}
