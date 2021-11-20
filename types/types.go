// Package types captures everything we need to know about a C0 AST node's
// type.
package types

import (
	"fmt"
	"strings"

	"github.com/susji/c0/node"
)

type TypeEnum int
type Types []Type

const (
	TYPE_INT = iota
	TYPE_BOOL
	TYPE_STRING
	TYPE_STRUCT
	TYPE_STRUCT_FWD
	TYPE_VOID
	TYPE_CHAR
	TYPE_FUNC
	TYPE_NULL
)

var typenames = [...]string{
	"int",
	"bool",
	"string",
	"struct",
	"struct (fwd)",
	"void",
	"char",
	"func",
	"null",
}

// Type is used to propagate type information from variable declarations to
// expressions.
type Type struct {
	Type         TypeEnum
	PointerLevel int
	ArrayLevel   int
	Extra        ExtraType // Used for structs and function pointers
}

type ExtraType interface {
	IsExtra()
}

type Function struct {
	Returns    Type
	ParamTypes Types
}

type StructField struct {
	Name string
	Type Type
}

type StructFields []StructField

type Struct struct {
	Name   string
	Fields StructFields
}

type StructForward struct {
	Name string
}

type Typedef struct {
	Type Type
}

func NewType(t TypeEnum, pointerlevel, arraylevel int) *Type {
	return &Type{
		Type:         t,
		PointerLevel: pointerlevel,
		ArrayLevel:   arraylevel,
	}
}

func NewTypeExtra(t TypeEnum, pointerlevel, arraylevel int, extra ExtraType) *Type {
	ret := NewType(t, pointerlevel, arraylevel)
	ret.Extra = extra
	return ret
}

func (k *Type) Matches(k2 *Type) bool {
	if k.Type != k2.Type || k.PointerLevel != k2.PointerLevel ||
		k.ArrayLevel != k2.ArrayLevel {
		return false
	}
	// If the type assertions fail, then it's correct to panic because it's a
	// bug somewhere, and the result of matching would be nonsensical.
	switch k.Type {
	case TYPE_STRUCT:
		s := k.Extra.(*Struct)
		s2 := k2.Extra.(*Struct)
		return s.Matches(s2)
	case TYPE_STRUCT_FWD:
		sf := k.Extra.(*StructForward)
		sf2 := k2.Extra.(*StructForward)
		return sf.Name == sf2.Name
	case TYPE_FUNC:
		f := k.Extra.(*Function)
		f2 := k2.Extra.(*Function)
		return f.Matches(f2)
	default:
		return true
	}
}

func (f *Function) Matches(f2 *Function) bool {
	return f.Returns.Matches(&f2.Returns) && f.ParamTypes.Matches(f2.ParamTypes)
}

func (s *Struct) Matches(s2 *Struct) bool {
	return s.Name == s2.Name && s.Fields.Matches(s2.Fields)
}

func (t Types) Matches(t2 Types) bool {
	if len(t) != len(t2) {
		return false
	}
	for i := range t {
		if !t[i].Matches(&t2[i]) {
			return false
		}
	}
	return true
}

func (sm *StructField) Matches(sm2 *StructField) bool {
	return sm.Name == sm2.Name && sm.Type.Matches(&sm2.Type)
}

func (sm StructFields) Matches(sm2 StructFields) bool {
	for i := range sm {
		if !sm[i].Matches(&sm2[i]) {
			return false
		}
	}
	return true
}

var kinds_to_types = map[node.KindEnum]TypeEnum{
	node.KIND_INT:    TYPE_INT,
	node.KIND_BOOL:   TYPE_BOOL,
	node.KIND_STRING: TYPE_STRING,
	node.KIND_STRUCT: TYPE_STRUCT,
	node.KIND_VOID:   TYPE_VOID,
	node.KIND_CHAR:   TYPE_CHAR,
}

func KindEnumToTypeEnum(k node.KindEnum) TypeEnum {
	if t, ok := kinds_to_types[k]; ok {
		return t
	} else {
		panic(fmt.Sprintf("unrecognized kind enum: %d", k))
	}
}

func (t *Type) string() string {
	fp := "***************************************************"
	fa := "[][][][][][][][][][][][][][][][][][][][][][][][][]"
	var pp, pa string
	if t.PointerLevel > len(fp) {
		pp = fp[:len(fp)-3] + "..."
	} else {
		pp = fp[:t.PointerLevel]
	}
	if t.ArrayLevel*2 > len(fa) {
		pa = fa[:len(fp)-4] + "..."
	} else {
		pa = fa[:t.ArrayLevel*2]
	}
	pn := typenames[t.Type]
	return fmt.Sprintf("%s%s%s", pn, pp, pa)
}

func (t *Type) String() string {
	return t.string()
}

func (t *Type) Long() string {
	basis := t.string()
	switch t.Type {
	case TYPE_STRUCT:
		var st *Struct
		if _st, ok := t.Extra.(*Struct); ok {
			st = _st
		}
		return fmt.Sprintf(
			"(type-struct %s %q)", st, basis)
	case TYPE_FUNC:
		var f *Function
		if _f, ok := t.Extra.(*Function); ok {
			f = _f
		}
		return fmt.Sprintf(
			"(type-func %s %q)", f, basis)
	}
	return fmt.Sprintf(`(type %q)`, basis)
}

func (t *Typedef) String() string {
	return fmt.Sprintf(
		"(typedef %s)", &t.Type)
}

func (f *Function) String() string {
	if f == nil {
		return "(function nil)"
	}
	return fmt.Sprintf("(def-function %s %s)", &f.Returns, f.ParamTypes)
}

func (f *Struct) String() string {
	if f == nil {
		return "(def-struct nil)"
	}
	return fmt.Sprintf("(def-struct %q %s)", f.Name, f.Fields)
}

func (p Types) String() string {
	b := &strings.Builder{}
	b.WriteString("(types")
	for _, cur := range p {
		b.WriteString(fmt.Sprintf(" %s", cur.String()))
	}
	b.WriteString(")")
	return b.String()
}

func (sm StructFields) String() string {
	b := &strings.Builder{}
	b.WriteString("(struct-members")
	for _, cur := range sm {
		b.WriteString(fmt.Sprintf(" %s", cur.String()))
	}
	b.WriteString(")")
	return b.String()
}

func (sm StructFields) Find(name string) *StructField {
	for _, cur := range sm {
		if cur.Name == name {
			return &cur
		}
	}
	return nil
}

func (sm *StructField) String() string {
	return fmt.Sprintf("(struct-member %q %s)", sm.Name, &sm.Type)
}

func (t *Type) Copy() *Type {
	ret := *t
	return &ret
}

func (t TypeEnum) String() string {
	return typenames[t]
}

func (t *Type) DecPtr() {
	t.PointerLevel--
	if t.PointerLevel < 0 {
		panic("PointerLevel < 0")
	}
}

func (t *Type) IncPtr() {
	t.PointerLevel++
}

func (t *Type) IncArray() {
	t.ArrayLevel++
}

func (t *Type) DecArray() {
	t.ArrayLevel--
	if t.ArrayLevel < 0 {
		panic("ArrayLevel < 0")
	}
}

func (ie *Function) IsExtra()      {}
func (ie *Struct) IsExtra()        {}
func (ie *StructForward) IsExtra() {}
