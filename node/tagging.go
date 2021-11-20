package node

import (
	"reflect"

	"github.com/susji/c0/token"
)

const NODEID_INVALID = 0

type NodeId uint64
type TokenTags map[NodeId]*token.Token

var globalid NodeId = NODEID_INVALID
var toktags TokenTags = TokenTags{}

// Store does all the relevant book-keeping for a Node.
var Store = func(tok *token.Token, n Node) Node {
	if n == nil {
		panic("nil node")
	}
	if tok == nil {
		panic("nil token")
	}
	globalid++
	// For a new Node, we initialize the embedded *Common via reflection.
	f := reflect.ValueOf(n).Elem().FieldByName("Common")
	f.Set(reflect.ValueOf(&Common{id: globalid}))
	toktags[globalid] = tok
	return n
}

// Tok retrieves the connected Token for a previous Store'd Node.
var Tok = func(id NodeId) *token.Token {
	if id == NODEID_INVALID {
		panic("invalid nodeid used")
	}
	return toktags[id]
}

// DisableTagging permanently disables the node tagging & pooling completely.
// Store and Tok will not function correctly after calling this. Only used when
// testing.
func DisableTagging() {
	Store = func(_ *token.Token, n Node) Node {
		return n
	}

	Tok = func(_ NodeId) *token.Token {
		return nil
	}
}
