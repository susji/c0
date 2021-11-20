package analyze

import (
	"github.com/susji/c0/node"
	"github.com/susji/c0/types"
)

type Functions map[string]*types.Function
type Typedefs map[string]*types.Typedef
type TypedefFuncs map[string]*types.Function
type Structs map[string]*types.Struct
type StructFwds map[string]*types.StructForward
type NodeTypes map[node.NodeId]*types.Type

// Results should contain everything that should be passed onwards from the
// analysis stage. This means at least the following things:
//
//   1) How the AST nodes are typed
//   2) What kind of user-defined data (typedefs, structs) we understood
//
type Results struct {
	Functions    Functions
	Typedefs     Typedefs
	TypedefFuncs Functions
	Structs      Structs
	StructFwds   StructFwds
	NodeTypes    NodeTypes
}
