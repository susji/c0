// package analyze is responsible for variable scoping & syntax and type
// checking.
package analyze

import (
	"errors"
	"fmt"

	"github.com/susji/c0/node"
	"github.com/susji/c0/types"
)

var (
	ErrTypedefAlreadyDefined = errors.New("typedef already defined")
	ErrStructAlreadyDefined  = errors.New("struct already defined")
	ErrFuncDifferentType     = errors.New("function redefined with different type")
	ErrFuncDeclInvalid       = errors.New("invalid function declaration")
)

type ternaryCheck struct {
	n    node.Node
	seen int
}

// Analyzer maintains the state when we check our forest of ASTs. This state
// means mainly information about user-defined types (structs, typedefs) and
// type-checking (what is some node's type).
type Analyzer struct {
	fn   string
	errs []error

	// res will contain everything that it's meant to be passed onwards after
	// the analysis stage.
	res *Results

	// The stuff below this comment is used to maintain state while
	// syntax-checking.

	// scope is the current, parent-linked & nested variable scope
	scope *scope
	// curfunc stores the type of the function we're currently analyzing
	curfunc *types.Function

	// loops is a LIFO of loops used to connect "break" and "continue"
	loops []node.Loop
	// canassign keeps track of valid lvalues
	canassign map[node.NodeId]struct{}
	// ternaryvals is used to match pairs of "?" and ":"
	ternaryvals map[node.NodeId]*ternaryCheck
	// structaccess is used to propagate struct information for "." and "->"
	structaccess map[node.NodeId]*types.Struct
	// returns tracks how many valid return statements each function has
	returns map[*types.Function]int
}

func (s *Analyzer) Results() *Results {
	return s.res
}

func (s *Analyzer) reset() {
	s.errs = []error{}
	s.scope = newScope(nil, nil)
	s.res = &Results{
		Functions:    Functions{},
		Typedefs:     Typedefs{},
		TypedefFuncs: Functions{},
		Structs:      Structs{},
		StructFwds:   StructFwds{},
		NodeTypes:    NodeTypes{},
	}
	s.canassign = map[node.NodeId]struct{}{}
	s.ternaryvals = map[node.NodeId]*ternaryCheck{}
	s.structaccess = map[node.NodeId]*types.Struct{}
	s.returns = map[*types.Function]int{}
}

func New(fn string) *Analyzer {
	ret := &Analyzer{fn: fn}
	ret.reset()
	return ret
}

func (s *Analyzer) setAssignable(n node.Node) {
	if _, ok := s.canassign[n.Id()]; ok {
		panic(fmt.Sprintf("node %s is assigning too hard", n))
	}
	s.canassign[n.Id()] = struct{}{}
}

func (s *Analyzer) isAssignable(n node.Node) bool {
	_, ok := s.canassign[n.Id()]
	return ok
}

func (s *Analyzer) setStructAccess(n node.Node, st *types.Struct) {
	if st == nil {
		return
	}
	s.structaccess[n.Id()] = st
}

func (s *Analyzer) getStructAccess(n node.Node) *types.Struct {
	return s.structaccess[n.Id()]
}

func (s *Analyzer) setType(n node.Node, k *types.Type) {
	if _, ok := s.res.NodeTypes[n.Id()]; ok {
		panic(fmt.Sprintf("nodetype defined twice for %s", n))
	}
	s.res.NodeTypes[n.Id()] = k
}

func (s *Analyzer) getType(n node.Node) *types.Type {
	return s.res.NodeTypes[n.Id()]
}

func (s *Analyzer) setFunction(fn *node.FunDecl) error {
	f, err := s.FunctionFromNodeFunDecl(fn)
	if err != nil {
		return err
	}
	if ff, ok := s.res.Functions[fn.Name]; ok && !ff.Matches(f) {
		return fmt.Errorf("%w: %q", ErrFuncDifferentType, fn.Name)
	}
	s.res.Functions[fn.Name] = f
	return nil
}

func (s *Analyzer) getFunction(name string) *types.Function {
	return s.res.Functions[name]
}

func (s *Analyzer) addTypedef(n *node.Typedef) error {
	if _, ok := s.res.Typedefs[n.Name]; ok {
		return fmt.Errorf("%w: %q", ErrTypedefAlreadyDefined, n.Name)
	}
	td, err := s.TypedefFromNode(n)
	if err != nil {
		return err
	}
	s.res.Typedefs[n.Name] = td
	return nil
}

func (s *Analyzer) getTypedef(name string) *types.Typedef {
	return s.res.Typedefs[name]
}

func (s *Analyzer) addTypedefFunc(n *node.TypedefFunc) error {
	if _, ok := s.res.TypedefFuncs[n.Name]; ok {
		return fmt.Errorf("%w: %q", ErrTypedefAlreadyDefined, n.Name)
	}
	f, err := s.FunctionFromNodeTypedefFunc(n)
	if err != nil {
		return err
	}
	s.res.TypedefFuncs[n.Name] = f
	return nil
}

func (s *Analyzer) getTypedefFunc(name string) *types.Function {
	return s.res.TypedefFuncs[name]
}

func (s *Analyzer) addStruct(n *node.Struct) error {
	if _, ok := s.res.Structs[n.Name]; ok {
		return fmt.Errorf("%w: %q", ErrStructAlreadyDefined, n.Name)
	}
	st, err := s.StructFromNode(n)
	if err != nil {
		return err
	}
	s.res.Structs[n.Name] = st
	return nil
}

func (s *Analyzer) getStruct(name string) *types.Struct {
	return s.res.Structs[name]
}

func (s *Analyzer) setStructFwd(n *node.StructForwardDecl) {
	s.res.StructFwds[n.Value] = &types.StructForward{Name: n.Value}
}

func (s *Analyzer) getStructFwd(name string) *types.StructForward {
	return s.res.StructFwds[name]
}

func (p *Analyzer) errorf(n node.Node, format string, a ...interface{}) error {
	err := &SyntaxError{
		Node:    n,
		Fn:      p.fn,
		Wrapped: fmt.Errorf(format, a...),
	}
	p.errs = append(p.errs, err)
	return err
}

// Analyze finds syntax errors and does type-checking. It uses depth-first
// traversal of the syntax tree defined by the given root node.
func (s *Analyzer) Analyze(nodes []node.Node) (errs []error) {
	for _, node := range nodes {
		s.check(node)
		s.checkTernaries()
	}
	return s.errs
}

func (s *Analyzer) withScope(n node.Node, what func()) {
	s.scope = newScope(s.scope, n)
	what()
	s.scope = s.scope.parent
}

func (s *Analyzer) withFunction(f *node.FunDef, what func()) {
	// Since we do not support closures or nested functions of any kind, we can
	// keep a track of the current function through one pointer.
	s.curfunc = s.getFunction(f.Name)
	if s.curfunc == nil {
		s.errorf(f, "%w: %q", ErrFuncDeclInvalid, f.Name)
		return
	}
	what()
	s.curfunc = nil
}

func (s *Analyzer) curFunction() *types.Function {
	return s.curfunc
}

func (s *Analyzer) withLoop(l node.Loop, what func()) {
	s.loops = append(s.loops, l)
	what()
	s.loops = s.loops[:len(s.loops)-1]
}

func (s *Analyzer) currentLoop() node.Loop {
	if len(s.loops) == 0 {
		return nil
	}
	return s.loops[len(s.loops)-1]
}
