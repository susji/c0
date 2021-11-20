package analyze

import (
	"errors"
	"fmt"

	"github.com/susji/c0/node"
	"github.com/susji/c0/types"
)

var ErrTypeUnrecognizedTypedef = errors.New("unrecognized typedef")
var ErrTypeUnrecognizedStruct = errors.New("unrecognized struct")

func (s *Analyzer) StructFromNode(n *node.Struct) (*types.Struct, error) {
	sf, err := s.StructFieldsFromVarDecls(n.Members)
	if err != nil {
		return nil, err
	}
	return &types.Struct{
		Name:   n.Name,
		Fields: sf,
	}, nil
}

func (s *Analyzer) StructFieldsFromVarDecls(vds node.VarDecls) (types.StructFields, error) {
	ret := types.StructFields{}
	for _, vd := range vds {
		t, err := s.KindToType(&vd.Kind)
		if err != nil {
			return ret, err
		}
		if t.Type == types.TYPE_STRUCT_FWD && t.PointerLevel == 0 {
			return ret, ErrStructSizeUnknown
		}
		ret = append(ret, types.StructField{
			Name: vd.Name,
			Type: *t,
		})
	}
	return ret, nil
}

func (s *Analyzer) TypesFromVarDecls(vds node.VarDecls) (types.Types, error) {
	ret := types.Types{}
	for _, vd := range vds {
		t, err := s.KindToType(&vd.Kind)
		if err != nil {
			return ret, err
		}
		ret = append(ret, *t)
	}
	return ret, nil
}

func (s *Analyzer) TypedefFromNode(n *node.Typedef) (*types.Typedef, error) {
	t, err := s.KindToType(&n.Kind)
	if err != nil {
		return nil, err
	}
	return &types.Typedef{
		Type: *t,
	}, nil
}

// The naming...
func (s *Analyzer) FunctionFromNodeTypedefFunc(n *node.TypedefFunc) (*types.Function, error) {
	t, err := s.KindToType(&n.Returns)
	if err != nil {
		return nil, err
	}
	ts, err := s.TypesFromVarDecls(n.Params)
	if err != nil {
		return nil, err
	}
	return &types.Function{
		Returns:    *t,
		ParamTypes: ts,
	}, nil
}

func (s *Analyzer) FunctionFromNodeFunDecl(n *node.FunDecl) (*types.Function, error) {
	ts, err := s.TypesFromVarDecls(n.Params)
	if err != nil {
		return nil, err
	}
	t, err := s.KindToType(&n.Returns)
	if err != nil {
		return nil, err
	}
	return &types.Function{
		Returns:    *t,
		ParamTypes: ts,
	}, nil
}

// KindToType transforms parsed variable declarations into Types.
func (s *Analyzer) KindToType(k *node.Kind) (*types.Type, error) {
	// We have to perform some impedance matching here. For regular types (int,
	// string, bool, char, NULL), we may just map them directly to a TypeEnum
	// thing with some pointer and array levels.
	//
	// However, things become slightly harder with user-definable types, that
	// is, typedefs and structs. Upon encountering these, we have to to look
	// into our book-keeping and figure out if we have such an user-defined
	// type.
	var t types.TypeEnum
	var extra types.ExtraType
	pointerlevel := k.PointerLevel
	arraylevel := k.ArrayLevel
	switch k.Kind {
	case node.KIND_TYPEDEF:
		if len(k.Name) == 0 {
			panic(fmt.Sprintf("no name for typedef: %s", k))
		}
		// Non-function and function typedefs are stored separately as
		// function-typing has to contain information about its return value
		// AND arguments.
		if td := s.getTypedef(k.Name); td != nil {
			t = td.Type.Type
			pointerlevel += td.Type.PointerLevel
			arraylevel += td.Type.ArrayLevel
			extra = td.Type.Extra
		} else if tdf := s.getTypedefFunc(k.Name); tdf != nil {
			t = types.TYPE_FUNC
			extra = tdf
		} else {
			return nil, s.errorf(k, "%w: %q", ErrTypeUnrecognizedTypedef, k.Name)
		}
	case node.KIND_STRUCT:
		if len(k.Name) == 0 {
			panic(fmt.Sprintf("no name for struct: %s", k))
		}
		// For struct-typing, definition has precedence over forward
		// declarations. In the type-checking code, a defined struct is
		// permitted to be used in variable declarations ("struct something
		// a;"), whereas only-forward-declared structs are required to be
		// pointers. The reason is simple: If we only have a
		// forward-declaration, we do not know the struct's size.
		if st := s.getStruct(k.Name); st != nil {
			t = types.TYPE_STRUCT
			extra = st
		} else if stf := s.getStructFwd(k.Name); stf != nil {
			t = types.TYPE_STRUCT_FWD
			extra = stf
		} else {
			return nil, s.errorf(k, "%w: %q", ErrTypeUnrecognizedStruct, k.Name)
		}
	default:
		// Plain types map directly.
		t = types.KindEnumToTypeEnum(k.Kind)
	}
	return &types.Type{
		Type:         t,
		PointerLevel: pointerlevel,
		ArrayLevel:   arraylevel,
		Extra:        extra,
	}, nil
}
