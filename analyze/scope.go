package analyze

import (
	"errors"

	"github.com/susji/c0/node"
	"github.com/susji/c0/types"
)

var errVarAlreadyDefined = errors.New("variable is already defined")

type scope struct {
	parent *scope
	node   node.Node
	vars   map[string]*types.Type
}

func newScope(parent *scope, from node.Node) *scope {
	return &scope{
		parent: parent,
		vars:   map[string]*types.Type{},
		node:   from,
	}
}

func (s *scope) add(name string, kind *types.Type) error {
	// As C0 does not permit any kind of variable shadowing, we have to do a
	// recursive search before agreeing.
	if s.get(name) != nil {
		return errVarAlreadyDefined
	}
	s.vars[name] = kind
	return nil
}

func (s *scope) get(name string) *types.Type {
	cur := s
	for cur != nil {
		if kind, ok := cur.vars[name]; ok {
			return kind
		}
		cur = cur.parent
	}
	return nil
}
