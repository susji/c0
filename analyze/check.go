package analyze

// Most code in this file is related to type-checking the syntax free
// consisting of Node implementation provided us by parsing. We do the whole
// thing with a single depth-first-traversal pass. This means that during
// checking, we maintain several data structures inside *Syntax, which are
// primarily maps where the unique Node identifier (NodeID) is the key.
//
// For an example, see below for function checkVariable, which has an extra
// clause for struct field access.

import (
	"errors"
	"fmt"

	"github.com/susji/c0/node"
	"github.com/susji/c0/types"
)

var (
	ErrCondType                 = errors.New("condition not boolean")
	ErrTernaryMissingCond       = errors.New("ternary operator missing '?'")
	ErrTernaryMissingValue      = errors.New("ternary operator missing ':'")
	ErrTernaryCondBool          = errors.New("ternary condition not boolean")
	ErrCompareNonInteger        = errors.New("non-integer comparison")
	ErrCompareTypes             = errors.New("types for comparison do not match")
	ErrCompareBadType           = errors.New("equality can only be evaluated for integers, booleans, characters and arrays")
	ErrVarNotDefined            = errors.New("variable has not been defined")
	ErrArithNonInteger          = errors.New("non-integer arithmetic")
	ErrArithTypes               = errors.New("types for arithmetic do not match")
	ErrAssignTypeMismatch       = errors.New("assignment type mismatch")
	ErrAssignNotLValue          = errors.New("cannot assign to a non-lvalue")
	ErrTypedefNotFound          = errors.New("typedef not found")
	ErrFuncallNotFound          = errors.New("calling non-declared function")
	ErrFuncallArgType           = errors.New("function argument type mismatch")
	ErrFuncallArgsAmount        = errors.New("wrong amount of function arguments")
	ErrFuncallWrongPtrType      = errors.New("expecting function pointer")
	ErrVarDeclShadowsFunction   = errors.New("variable declaration already a function")
	ErrVarDeclShadowsTypedef    = errors.New("variable declaration already a typedef")
	ErrAllocArrayBadExpr        = errors.New("`alloc_array' expression should result in integer")
	ErrArraySubBadExpr          = errors.New("bad array subscript expression")
	ErrArraySubNotArray         = errors.New("trying to subscript a non-array")
	ErrArraySubNotInt           = errors.New("array subscript a non-integer")
	ErrStructDecNotField        = errors.New("struct deconstruction needs a field name")
	ErrStructDecFieldNotFound   = errors.New("struct field not found")
	ErrStructNotAccessingStruct = errors.New("trying to access a field of a non-struct")
	ErrStructBadType            = errors.New("trying to access a field from bad type")
	ErrStructSizeUnknown        = errors.New("forward-declared struct size is unknown")
	ErrStructOnlyForward        = errors.New("cannot declare a non-pointer variable of struct, which is only forward-declared")
	ErrContinueOutsideLoop      = errors.New("`continue' not permitted outside loops")
	ErrBreakOutsideLoop         = errors.New("`break' not permitted outside loops")
	ErrReturnExprMissing        = errors.New("`return' expression missing for non-void function")
	ErrReturnMistyped           = errors.New("`return' expression is mistyped")
	ErrReturnMissing            = errors.New("`return' statement missing for non-void function")
	ErrFuncParamStruct          = errors.New("function parameter may not be plain struct")
	ErrVarDeclVoid              = errors.New("`void' as a variable type is unacceptable")
	ErrCastVoid                 = errors.New("cannot cast to void")
	ErrCastVoidPointer          = errors.New("cannot cast to void pointer")
	ErrNegateNonBool            = errors.New("cannot negate non-boolean")
)

var (
	typeBool = types.NewType(types.TYPE_BOOL, 0, 0)
	typeInt  = types.NewType(types.TYPE_INT, 0, 0)
	typeChar = types.NewType(types.TYPE_CHAR, 0, 0)
	typeVoid = types.NewType(types.TYPE_VOID, 0, 0)
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Stores all ternary value nodes (':') we have reached from a condition node
// ('?)'. We use this to make sure we have valid ':' ... '?' pairs.
func (s *Analyzer) checkTernaryCond(tc *node.OpBinary) {
	k := s.getType(tc.Left)
	if k == nil {
		return
	}
	if !k.Matches(typeBool) {
		s.errorf(tc, "%w: got %s", ErrTernaryCondBool, k)
	}
	tv, ok := tc.Right.(*node.OpBinary)
	if !ok || tv.Op != node.OPBIN_TERNARYVALS {
		s.errorf(tc, "%w", ErrTernaryMissingValue)
		return
	} else if _, ok := s.ternaryvals[tv.Id()]; !ok {
		s.errorf(tv, "%w", ErrTernaryMissingValue)
		return
	}
	s.ternaryvals[tv.Id()].seen++
}

// MarkTernaryVal is the other half of ternary checking. Once we meet a ':'
// node, we store a count of 1 with its ID. Later on, if we find the respective
// '?', it will do a +1 with the same ID. After we have parsed the expression,
// all valid pairs of ('?', ':') indexed with the ':' ID will have a count of
// 2.
func (s *Analyzer) MarkTernaryVal(tv *node.OpBinary) {
	s.ternaryvals[tv.Id()] = &ternaryCheck{n: tv, seen: 1}
}

func (s *Analyzer) checkArraySub(b *node.OpBinary) {
	// For array subscripts, the left node must be an array. The right has to
	// be an int.
	tl := s.getType(b.Left)
	tr := s.getType(b.Right)
	if tl == nil {
		s.errorf(b, "%w: array", ErrArraySubBadExpr)
		return
	} else {
		if tl.ArrayLevel < 1 || tl.PointerLevel != 0 {
			s.errorf(b.Left, "%w: got %s", ErrArraySubNotArray, tl)
		}
	}
	if tr == nil {
		s.errorf(b, "%w: subscript", ErrArraySubBadExpr)
		return
	}
	if !tr.Matches(typeInt) {
		s.errorf(b.Right, "%w: got %s", ErrArraySubNotInt, tr)
	}
	if tl.ArrayLevel == 0 {
		s.errorf(b.Left, "%w: got %s", ErrArraySubNotArray, tl)
		return
	}
	nt := tl.Copy()
	nt.DecArray()
	s.setType(b, nt)
	s.setAssignable(b)
	// See the comment in checkVariable about propagating this flag.
	if st := s.getStructAccess(b.Left); st != nil {
		s.setStructAccess(b, st)
	}
}

func (s *Analyzer) checkComp(b *node.OpBinary) {
	// Arithmetic comparison unconditionally results in boolean.
	s.setType(b, typeBool.Copy())

	kl := s.getType(b.Left)
	kr := s.getType(b.Right)
	if kl == nil || kr == nil {
		return
	}
	if !kl.Matches(kr) || !(kl.Matches(typeInt) || kl.Matches(typeChar)) {
		s.errorf(b.Left, "%w: %s vs. %s", ErrCompareNonInteger, kl, kr)
		return
	}
	if !kl.Matches(kr) {
		s.errorf(
			b,
			"%w: %s vs %s",
			ErrCompareTypes, kl, kr)
	}
}

func (s *Analyzer) checkArith(b *node.OpBinary) {
	kl := s.getType(b.Left)
	kr := s.getType(b.Right)
	if kl == nil || kr == nil {
		return
	}
	if !kl.Matches(kr) || !kr.Matches(typeInt) {
		s.errorf(b.Left, "%w: %s vs. %s", ErrArithNonInteger, kl, kr)
		return
	}
	if !kl.Matches(kr) {
		s.errorf(
			b,
			"%w: %s vs %s",
			ErrArithTypes,
			kl, kr)
	}
	s.setType(b, kl)
}

func (s *Analyzer) checkAtom(n node.Node, k types.TypeEnum) {
	nk := types.NewType(k, 0, 0)
	s.setType(n, nk)
}

func (s *Analyzer) checkAssign(n *node.OpAssign) {
	// For an lvalue to be valid, it has to fulfill two conditions:
	//   - it has to be suitably typed
	//   - its node type has to be a thing in memory, ie. struct member or
	//     a plain variable
	if !s.isAssignable(n.To) {
		s.errorf(n.To, "%w: %s", ErrAssignNotLValue, n.To)
		return
	}
	if n.What == nil {
		// Assignment may be without first value.
		return
	}
	kt := s.getType(n.To)
	kw := s.getType(n.What)
	if kt == nil || kw == nil {
		return
	}
	// We have different cases where we permit assignment:
	//   - a trivially matching rvalue to matching lvalue, ie. same types
	//   - NULL to anything which is a pointer
	//   - an `alloc_array` to something which is an array
	//
	if !kt.Matches(kw) &&
		!(kt.PointerLevel > 0 && kw.Type == types.TYPE_NULL) {
		s.errorf(n, "%w: %s vs %s", ErrAssignTypeMismatch, kt, kw)
	}
	s.setType(n, kt)
}

func (s *Analyzer) getStructFieldType(n *node.Variable, st *types.Struct) *types.Type {
	if st == nil {
		return nil
	}
	f := st.Fields.Find(n.Value)
	if f == nil {
		s.errorf(
			n,
			"%w: %s does not have %q",
			ErrStructDecFieldNotFound,
			st,
			n.Value)
		return nil
	}
	return &f.Type
}

func (s *Analyzer) checkVariable(n *node.Variable) {
	// All Variable things are leaf-nodes in the tree, by definition.
	if fd := s.getFunction(n.Value); fd != nil {
		s.setType(n, types.NewTypeExtra(types.TYPE_FUNC, 0, 0, fd))
		return
	}
	t := s.scope.get(n.Value)
	if t == nil {
		s.errorf(n, "%w: %q", ErrVarNotDefined, n.Value)
		return
	}
	s.setType(n, t)
	if t.Type == types.TYPE_STRUCT {
		// If our leaf is a struct, someone may be attempting to perform field
		// access via "." or "->". If the syntax is correct, this means we are
		// in LHS of either binary operator. A simple case of
		// "(*somestruct).somefield" might then look like this:
		//
		//          "." (3)
		//         /   \
		//       /       \
		//     /           \
		//    "*" (2)   "somefield" (4)
		//     |
		//     |
		// somestruct (1)
		//
		// We are doing a single-pass DFS here, and the LHS is always taken
		// first, so we first arrive at the variable node here, which defines a
		// variable of some struct type. See the numbers in the figure -- the
		// ascending value stands for order of node visitation.
		//
		// Problem #1: How to make the field-access operator aware that its LHS
		//             indeed eventually defines some valid struct-type?
		//
		// Problem #2: When we end up looking at the identifier of "somefield",
		//             how can we can know a) it is supposed to be mean struct
		//             field name and b) what is this struct's type, if any?
		//
		// Solution:   We propagate "struct access" information upwards from
		//             this node until we reach a field-access operator. The
		//             field-access operator may then query "getStructAccess"
		//             to figure out what struct it is operating on. See calls
		//             to "getStructAccess" and "setStructAccess" and
		//             checkStructFieldAccess().
		//
		// Notably, nested structs are just separate sub-tree verifications as
		// shown in the simple example here.
		//
		s.setStructAccess(n, t.Extra.(*types.Struct))
	}
	s.setAssignable(n)
}

func (s *Analyzer) isNameShadowed(n node.Node, name string) bool {
	if fd := s.getFunction(name); fd != nil {
		s.errorf(n, "%w: %q", ErrVarDeclShadowsFunction, name)
		return true
	}
	if td := s.getTypedef(name); td != nil {
		s.errorf(n, "%w: %q", ErrVarDeclShadowsTypedef, name)
		return true
	}
	if tdf := s.getTypedefFunc(name); tdf != nil {
		s.errorf(n, "%w: %q", ErrVarDeclShadowsTypedef, name)
		return true
	}
	return false
}

func (s *Analyzer) checkVarDecl(n *node.VarDecl) {
	if s.isNameShadowed(n, n.Name) {
		return
	}
	t, err := s.KindToType(&n.Kind)
	if err != nil {
		return
	}
	if err := s.scope.add(n.Name, t); err != nil {
		s.errorf(n, "variable %q has already been defined", n.Name)
		return
	}
	if t.Type == types.TYPE_VOID && t.PointerLevel == 0 {
		s.errorf(n, "%w", ErrVarDeclVoid)
		return
	}
	// If we only have a struct forward-declaration, we do not know the
	// struct's size. This means we may only declare pointers to it.
	if t.Type == types.TYPE_STRUCT_FWD && t.PointerLevel == 0 {
		s.errorf(n, "%w: %q", ErrStructOnlyForward, n.Name)
		return
	}
	s.setType(n, t)
	s.setAssignable(n)
}

func (s *Analyzer) checkFunCall(n *node.OpBinary) {
	var want types.Types
	var got []node.Node
	var returns *types.Type
	switch t := n.Left.(type) {
	case *node.Variable:
		// Regular function calls via a Variable.
		fd := s.getFunction(t.Value)
		if fd == nil {
			s.errorf(n, "%w: %q", ErrFuncallNotFound, t.Value)
			return
		}
		returns = &fd.Returns
		want = fd.ParamTypes
		switch tt := n.Right.(type) {
		case *node.Args:
			got = tt.Value
		default:
			panic(fmt.Sprintf("invalid function parameters: %s", t))
		}
	default:
		// This means an arbitrary expression Node. We only accept something
		// which eventually types into a function pointer.
		ct := s.getType(t)
		if ct == nil {
			return
		}
		if ct.Type != types.TYPE_FUNC {
			s.errorf(t, "%w: got %s", ErrFuncallWrongPtrType, ct)
			return
		}
		if ct.Extra == nil {
			panic(fmt.Sprintf("no FuncPtr for %s", ct))
		}
		returns = &ct.Extra.(*types.Function).Returns
		want = ct.Extra.(*types.Function).ParamTypes
		switch tt := n.Right.(type) {
		case *node.Args:
			got = tt.Value
		default:
			panic(fmt.Sprintf("invalid function parameters: %s", t))
		}
	}
	ngot := len(got)
	nwant := len(want)
	if ngot != nwant {
		s.errorf(n, "%w: wanted %d, got %d",
			ErrFuncallArgsAmount, nwant, ngot)
	}
	for i := 0; i < min(ngot, nwant); i++ {
		typegot := s.getType(got[i])
		typewant := want[i]
		if !typewant.Matches(typegot) {
			s.errorf(n, "%w: wanted %s, got %s",
				ErrFuncallArgType, &typewant, typegot)
		}
	}
	s.setType(n, returns)
}

func (s *Analyzer) checkFunDecl(n *node.FunDecl) {
	if s.isNameShadowed(n, n.Name) {
		return
	}
	for _, param := range n.Params {
		pt, err := s.KindToType(&param.Kind)
		if err != nil {
			return
		}
		if pt.ArrayLevel < 1 &&
			(pt.Type == types.TYPE_STRUCT || pt.Type == types.TYPE_STRUCT_FWD) &&
			pt.PointerLevel == 0 {
			s.errorf(&param, "%w", ErrFuncParamStruct)
		}
	}
	if err := s.setFunction(n); err != nil {
		s.errorf(n, "%w", err)
	}
}

func (s *Analyzer) checkCast(n *node.Cast) {
	// Any pointer can be cast to "void *" and "void *" can be cast to any
	// pointer.
	kc := &n.To
	kw := s.getType(n.What)
	if kc.Kind == types.TYPE_VOID && kc.PointerLevel < 1 {
		s.errorf(n, "%w", ErrCastVoid)
	}
	// If kw is nil, its traversal failed and produced no usable type.
	if kw == nil {
		goto end
	}
	// NULL, "the default value of type void*" is immune to casting.
	switch n.What.(type) {
	case *node.Null:
		s.errorf(n, "NULL cannot be cast")
	default:
		if kw.PointerLevel < 1 && kc.PointerLevel > 0 {
			s.errorf(n, "%w: %s is %s", ErrCastVoidPointer, n.What, kw)
		}
	}
end:
	t, err := s.KindToType(&n.To)
	if err != nil {
		return
	}
	s.setType(n, t)
}

func (s *Analyzer) checkUnary(n *node.OpUnary) {
	kt := s.getType(n.To)
	if kt == nil {
		return
	}
	switch n.Op {
	case node.OPUN_DEREF:
		if kt.Type == types.TYPE_NULL {
			s.errorf(n, "derefencing NULL")
			return
		} else if kt.PointerLevel < 1 {
			s.errorf(n, "dereferencing non-pointer %q", n.To)
			return
		}
		nt := kt.Copy()
		nt.DecPtr()
		s.setType(n, nt)
		// As we do not permit pointer arithmetics, deferencing should only
		// appear in front of pointers.
		if s.isAssignable(n.To) {
			s.setAssignable(n)
		}
		// See the comment in checkVariable about propagating this flag.
		if st := s.getStructAccess(n.To); st != nil {
			s.setStructAccess(n, st)
		}
	case node.OPUN_ADDROF:
		if kt.Type != types.TYPE_FUNC {
			s.errorf(n, "cannot get address of non-function %s", n.To)
		}
		nt := kt.Copy()
		nt.IncPtr()
		s.setType(n, nt)
	case node.OPUN_LOGNOT:
		if !kt.Matches(typeBool) {
			s.errorf(n, "%w: %q", ErrNegateNonBool, n.To)
		}
		s.setType(n, kt)
	default:
		// The default case covers all integer operations.
		if !kt.Matches(typeInt) {
			s.errorf(n, "integer operation for %s %s", kt, n.To)
		}
		s.setType(n, kt)
	}
}

func (s *Analyzer) checkEq(n *node.OpBinary) {
	// Equality comparison unconditionally results in the boolean type.
	s.setType(n, typeBool.Copy())
	kl := s.getType(n.Left)
	kr := s.getType(n.Right)
	if kl == nil || kr == nil {
		return
	}
	v := func(k *types.Type) bool {
		return k.Matches(typeInt) || k.Matches(typeBool) || k.Matches(typeChar) ||
			k.ArrayLevel > 0
	}
	if !v(kl) || !v(kr) {
		s.errorf(n, "%w: got %s and %s", ErrCompareBadType, kl, kr)
	}
	if !kl.Matches(kr) {
		s.errorf(n,
			"%w: %s vs. %s",
			ErrCompareTypes,
			kl,
			kr)
	}
}

func (s *Analyzer) checkCond(cond node.Node, name string) {
	if cond == nil {
		return
	}
	k := s.getType(cond)
	if k == nil {
		panic(fmt.Sprintf("no type for %s", name))
	}
	if !k.Matches(typeBool) {
		s.errorf(cond, "%w for %s: got %s", ErrCondType, name, k)
	}
}

func (s *Analyzer) checkAllocArray(n *node.AllocArray) {
	at, err := s.KindToType(&n.Kind)
	if err != nil {
		return
	}
	at.IncArray()
	s.setType(n, at)

	nt := s.getType(n.N)
	if !nt.Matches(typeInt) {
		s.errorf(n.N, "%w: got %s", ErrAllocArrayBadExpr, nt)
	}
}

func (s *Analyzer) checkAlloc(n *node.Alloc) {
	at, err := s.KindToType(&n.Kind)
	if err != nil {
		return
	}
	at.IncPtr()
	s.setType(n, at)
}

func (s *Analyzer) checkBreak(n *node.Break) {
	cl := s.currentLoop()
	if cl == nil {
		s.errorf(n, "%w", ErrBreakOutsideLoop)
		return
	}
}

func (s *Analyzer) checkContinue(n *node.Continue) {
	cl := s.currentLoop()
	if cl == nil {
		s.errorf(n, "%w", ErrContinueOutsideLoop)
		return
	}
}

func (s *Analyzer) checkReturn(n *node.Return) {
	cf := s.curFunction()
	if cf == nil {
		return
	}
	s.returns[cf]++
	if n.Expr == nil {
		if !cf.Returns.Matches(typeVoid) {
			s.errorf(n, "%w", ErrReturnExprMissing)
		}
		return
	}
	rt := s.getType(n.Expr)
	if rt == nil {
		return
	}
	if !cf.Returns.Matches(rt) {
		s.errorf(
			n,
			"%w: wanted %s, got %s",
			ErrReturnMistyped,
			&cf.Returns,
			rt)
		return
	}
}

func (s *Analyzer) checkStructFieldAccess(n *node.OpBinary) {
	var explvl int
	switch n.Op {
	case node.OPBIN_STRUCTPTRDEC:
		explvl = 1
	case node.OPBIN_STRUCTDEC:
		explvl = 0
	default:
		panic(fmt.Sprintf("expecting struct op, got %s", n))
	}
	tl := s.getType(n.Left)
	if tl == nil {
		return
	}
	if tl.PointerLevel != explvl || tl.ArrayLevel != 0 {
		s.errorf(n, "%w: got %s", ErrStructBadType, tl)
	}
	// As we are doing DFS, struct field access needs extra help. The
	// lhs traversal should produce the *Struct involved in this
	// attempt, which we will then pass on towards the rhs, so we may
	// check whether the field inquiry is correct.
	sm, ok := n.Right.(*node.Variable)
	if !ok {
		s.errorf(n.Right, "%w: got %s", ErrStructDecNotField, n.Right)
		return
	}
	st := s.getStructAccess(n.Left)
	if st == nil {
		s.errorf(n, "%w: %s", ErrStructNotAccessingStruct, n.Left)
		return
	}
	ft := s.getStructFieldType(sm, st)
	if ft == nil {
		return
	}
	// In the case of nested struct access, eg. "a->b->c.d", we need to
	// propagate the struct type upwards. See the example in checkVariable().
	if ft.Type == types.TYPE_STRUCT {
		s.setStructAccess(n, ft.Extra.(*types.Struct))
	}
	// Now we know we have
	//   1) a valid struct on the LHS, and
	//   2) a valid field of that struct on the RHS.
	s.setType(n, ft)
	s.setAssignable(n)
}

func (s *Analyzer) checkBinary(n *node.OpBinary) {
	switch n.Op {
	case node.OPBIN_TERNARYCOND:
		s.checkTernaryCond(n)
	case node.OPBIN_TERNARYVALS:
		s.MarkTernaryVal(n)
	case node.OPBIN_ARRSUB:
		s.checkArraySub(n)
	case node.OPBIN_EQ, node.OPBIN_NE:
		s.checkEq(n)
	case node.OPBIN_FUNCALL:
		s.checkFunCall(n)
	case node.OPBIN_LE, node.OPBIN_GE, node.OPBIN_LT, node.OPBIN_GT:
		s.checkComp(n)
	case node.OPBIN_BAND, node.OPBIN_BOR, node.OPBIN_BXOR,
		node.OPBIN_AND, node.OPBIN_OR,
		node.OPBIN_SHIFTR, node.OPBIN_SHIFTL,
		node.OPBIN_ADD, node.OPBIN_SUB, node.OPBIN_MUL, node.OPBIN_DIV,
		node.OPBIN_MOD:
		s.checkArith(n)
	case node.OPBIN_STRUCTDEC, node.OPBIN_STRUCTPTRDEC:
		panic("struct decomposition should be handled elsewhere")
	default:
		panic("unhandled binary operator")
	}
}

func (s *Analyzer) check(n node.Node) {
	a := s.check
	switch t := n.(type) {
	case *node.Variable:
		s.checkVariable(t)
	case *node.Bool:
		s.checkAtom(t, types.TYPE_BOOL)
	case *node.LibLit:
		panic(fmt.Sprintf("unexpected liblit: %s", t))
	case *node.StrLit:
		s.checkAtom(t, types.TYPE_STRING)
	case *node.ChrLit:
		s.checkAtom(t, types.TYPE_CHAR)
	case *node.Numeric:
		s.checkAtom(t, types.TYPE_INT)
	case *node.Null:
		s.checkAtom(t, types.TYPE_NULL)
	case *node.Struct:
		if err := s.addStruct(t); err != nil {
			s.errorf(n, "%w", err)
		}
	case *node.StructForwardDecl:
		s.setStructFwd(t)
	case *node.Typedef:
		if err := s.addTypedef(t); err != nil {
			s.errorf(n, "%w", err)
		}
	case *node.TypedefFunc:
		if err := s.addTypedefFunc(t); err != nil {
			s.errorf(n, "%w", err)
		}
	case *node.OpUnary:
		a(t.To)
		s.checkUnary(t)
	case *node.OpBinary:
		a(t.Left)
		switch t.Op {
		case node.OPBIN_STRUCTDEC, node.OPBIN_STRUCTPTRDEC:
			s.checkStructFieldAccess(t)
		default:
			a(t.Right)
			s.checkBinary(t)
		}
	case *node.OpAssign:
		a(t.What)
		a(t.To)
		s.checkAssign(t)
	case *node.VarDecl:
		s.checkVarDecl(t)
	case *node.Args:
		for _, arg := range t.Value {
			a(arg)
		}
	case *node.FunDecl:
		s.withScope(t, func() {
			a(&t.Returns)
			for _, param := range t.Params {
				a(&param)
			}
			s.checkFunDecl(t)
		})
	case *node.FunDef:
		a(&t.Returns)
		s.withScope(t, func() {
			for _, param := range t.Params {
				a(&param)
			}
			// We need to check the declaration first in case the body tries to
			// shadow the function name with some variable.
			s.checkFunDecl(&t.FunDecl)
			// To enable return-type checking, we have to know which function
			// we are currently defining when checking the body.
			s.withFunction(t, func() {
				a(&t.Body)
				cf := s.curFunction()
				if cf == nil {
					s.errorf(n, "invalid function definition: %q", t.Name)
				}
				if !cf.Returns.Matches(typeVoid) && s.returns[cf] == 0 {
					s.errorf(t, "%w", ErrReturnMissing)
				}
			})
		})
	case *node.Block:
		s.withScope(t, func() {
			for _, param := range t.Value {
				a(param)
			}
		})
	case *node.If:
		a(t.Cond)
		a(t.True)
		a(t.False)
		s.checkCond(t.Cond, "if")
	case *node.For:
		s.withLoop(t, func() {
			a(t.Init)
			a(t.Cond)
			a(t.OnEach)
			a(t.Body)
			s.checkCond(t.Cond, "for")
		})
	case *node.While:
		s.withLoop(t, func() {
			a(t.Cond)
			a(t.Body)
			s.checkCond(t.Cond, "while")
		})
	case *node.Return:
		a(t.Expr)
		s.checkReturn(t)
	case *node.Assert:
		a(t.Expr)
		s.checkCond(t.Expr, "assert")
	case *node.Error:
		a(t.Expr)
		panic("implement error check")
	case *node.Cast:
		a(t.What)
		s.checkCast(t)
	case *node.AllocArray:
		a(t.N)
		s.checkAllocArray(t)
	case *node.Alloc:
		s.checkAlloc(t)
	case *node.Break:
		s.checkBreak(t)
	case *node.Continue:
		s.checkContinue(t)
	case nil, *node.Kind, *node.DirectiveUse:
		// these are no-action
	default:
		panic(fmt.Sprintf("check: unhandled %T: %s", t, t))
	}
}

func (s *Analyzer) checkTernaries() {
	for _, tc := range s.ternaryvals {
		if tc.seen != 2 {
			s.errorf(tc.n, "%w", ErrTernaryMissingCond)
		}
	}
}
