// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ir

import (
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/src/gx/ast"
)

// FieldIndexType is the builtin fieldindex type returned by functions traversing structures.
type FieldIndexType struct {
	genParams GenericParams
}

var fieldIndexIdent = &ast.Ident{Name: "fieldindex"}

var _ SpecialisableType = (*FieldIndexType)(nil)

var origin = newFieldIndexType(TypeExpr(nil, &FuncType{}))

func newFieldIndexType(keyToValueFunc Expr) *FieldIndexType {
	rootGroup := &FieldGroup{Type: TypeExpr(nil, AnyType())}
	rootGroup.Fields = []*Field{
		{Group: rootGroup, Pos: 0, Name: &ast.Ident{Name: "root"}},
	}
	keyValueGroup := &FieldGroup{Type: TypeExpr(keyToValueFunc.Expr(), keyToValueFunc.Type())}
	keyValueGroup.Fields = []*Field{
		{Group: keyValueGroup, Pos: 1, Name: &ast.Ident{Name: "keyToValue"}},
	}
	tp := &FieldIndexType{}
	tp.genParams.SetFields(&FieldList{List: []*FieldGroup{rootGroup, keyValueGroup}})
	genFuncParam := NewGenericNonTypeParam(keyValueGroup.Fields[0])
	genFuncValue := NewNonTypeGenericValue(genFuncParam, keyToValueFunc)
	tp.genParams.Values = []GenericValue{nil, genFuncValue}
	return tp
}

// SpecialiseForFunc returns a fieldindex type with the type of the
// keyToValue generic parameter set to the type of the function fn.
// Returns an error if fn is not a function.
func (*FieldIndexType) SpecialiseForFunc(fn Expr) (*FieldIndexType, error) {
	if fn.Type().Kind() != irkind.Func {
		return nil, CompileErrorF("%s is not a function", fn.SourceString(nil))
	}
	return newFieldIndexType(fn), nil
}

// GenericFieldIndexType returns the builtin fieldindex type.
func GenericFieldIndexType() Type {
	return origin
}

func (*FieldIndexType) node()         {}
func (*FieldIndexType) storage()      {}
func (*FieldIndexType) storageValue() {}

func (*FieldIndexType) Node() ast.Node { return fieldIndexIdent }

// root returns the type argument of the fieldindex type
// or nil if the type has not been specialised.
func (t *FieldIndexType) root() *TypeValExpr {
	typeVal, ok := t.genParams.Values[0].(*TypeGenericValue)
	if !ok {
		return nil
	}
	return typeVal.tp
}

func (t *FieldIndexType) Refer(file *File) ast.Expr {
	root := t.root()
	if root == nil {
		return fieldIndexIdent
	}
	return &ast.IndexExpr{X: fieldIndexIdent, Index: root.Val().Refer(file)}
}

func (*FieldIndexType) Same(Storage) bool {
	return true
}

func (t *FieldIndexType) Equal(tpcmp TypeEval, other Type) (bool, error) {
	otherT, ok := other.(*FieldIndexType)
	if !ok {
		return false, nil
	}
	return t.genParams.Equal(tpcmp, &otherT.genParams)
}

func (t *FieldIndexType) AssignableTo(tpcmp TypeEval, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

func (t *FieldIndexType) assignableFrom(tpcmp TypeEval, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

func (t *FieldIndexType) ConvertibleTo(tpcmp TypeEval, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

func (t *FieldIndexType) Instantiate(_ Fetcher, spec Instantiator) (Type, bool) {
	return t.Specialise(spec)
}

func (*FieldIndexType) Kind() irkind.Kind { return irkind.FieldIndex }

// Origin returns the builtin fieldindex type.
func (*FieldIndexType) Origin() GenericType { return origin }

// Generics returns the generic parameters and their values.
func (t *FieldIndexType) Generics() *GenericParams { return &t.genParams }

func (*FieldIndexType) specialisableType() {}

func (*FieldIndexType) NameDef() *ast.Ident { return fieldIndexIdent }

func (*FieldIndexType) Type() Type { return MetaType() }

func (*FieldIndexType) Value(Expr) Expr { return nil }

func (t *FieldIndexType) DefineString(from *File) string {
	root := t.root()
	if root == nil {
		return irkind.FieldIndex.String()
	}
	return irkind.FieldIndex.String() + "[" + root.SourceString(from) + "]"
}

func (t *FieldIndexType) ReferString(from *File) string {
	return t.DefineString(from)
}

// Specialise a type to a given target.
func (t *FieldIndexType) Specialise(spec Specialiser) (Type, bool) {
	genParams, ok := t.genParams.Specialise(spec, spec.Target() == t)
	return &FieldIndexType{genParams: genParams}, ok
}

// UnifyWith recursively unifies a type parameters with types.
func (*FieldIndexType) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

// Unroll returns an error because fieldindex types do not support unrolling.
func (t *FieldIndexType) Unroll(ev Fetcher, urlr Unroller) (ast.Expr, bool) {
	return nil, ev.Err().AppendInternalf(fieldIndexIdent, "%T does not support unrolling", t)
}

func (t *FieldIndexType) IndexForVarArgs(ErrSource, int) (Type, bool) {
	return t, true
}

// Visit traverses the type argument of the fieldindex type.
func (t *FieldIndexType) Visit(f Finder) {
	t.root().Visit(f)
}
