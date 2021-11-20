package parse

import (
	"errors"

	"github.com/susji/c0/analyze"
	"github.com/susji/c0/node"
	"github.com/susji/c0/token"
)

func (p *Parser) Type(toks *token.Tokens) (node.Kind, error) {
	// The C0 Reference grammar for `<tp>' produces also non-sensical type
	// definitions, which are (assumedly) then syntax-checked after parsing. In
	// our case, we modify the grammar to explicitly produce only acceptable
	// type declarations:
	//
	// <tp>        = <tp-atomic> <tp-suffix>
	// <tp-atomic> = "int" | "bool" | "string" | "char" | "void"
	//             | "struct" <sid>
	//             | <aid>
	// <tp-suffix> = [ "*" { "*" } ] [ "[]" { "[]" } ]
	//
	// Note: This grammar still permits declaring unacceptable types via
	//       typedefs such as a typedef'd array of arrays. This means that upon
	//       during later stages, ie. type-checking, we have to look at the
	//       typedef-based declarations and see if they make sense.
	//
	atom := toks.Peek()
	if atom == nil {
		return node.Kind{}, EOT
	}
	if atom.Kind() != token.Id {
		return node.Kind{}, errors.New("not a type declaration")
	}
	if !analyze.IsValidPrimitive(atom.Value()) {
		if _, ok := p.typedefs[atom.Value()]; !ok {
			return node.Kind{}, p.errorf(atom, "typedef %q not defined", atom)
		}
	}

	toks.Pop()

	var pointerlevel, arraylevel int
	var name string
	var kind node.KindEnum
	// <tp-atomic>
	//
	// A type declaration either needs to be one of the primitive types, a
	// "struct", or an user-defined type. We don't care about typedefs yet so
	// parsing *will* accept any identifier.
	switch atom.Value() {
	case "int":
		kind = node.KIND_INT
	case "bool":
		kind = node.KIND_BOOL
	case "string":
		kind = node.KIND_STRING
	case "char":
		kind = node.KIND_CHAR
	case "void":
		kind = node.KIND_VOID
	case "struct":
		sid := toks.Pop()
		if sid == nil || sid.Kind() != token.Id {
			return node.Kind{}, p.errorf(sid, "expected struct name, got %s", sid.String())
		}
		kind = node.KIND_STRUCT
		name = sid.Value()
	default:
		// If it's not a primitive type or a struct, it must be a typedef. Or
		// invalid, but this will be resolved later.
		kind = node.KIND_TYPEDEF
		name = atom.Value()
	}

	// <tp-suffix>
	// pointer level?
	for {
		ptr := toks.Peek()
		if ptr == nil || ptr.Kind() != token.Star {
			break
		}
		pointerlevel++
		toks.Pop()
	}

	// array level?
	for {
		bra := toks.Peek()
		if bra == nil || bra.Kind() != token.Brackets {
			break
		}
		arraylevel++
		toks.Pop()
	}
	k := node.NewKind(kind, pointerlevel, arraylevel, name)
	ret := node.Store(atom, &k).(*node.Kind)
	return *ret, nil
}
