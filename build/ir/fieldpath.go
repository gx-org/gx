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

// FieldPathType is the builtin fieldindex type returned by functions traversing structures.
type FieldPathType struct {
	genParams GenericParams
}

var fieldPathIdent = &ast.Ident{Name: "fieldpath"}

var _ SpecialisableType = (*FieldPathType)(nil)

var origin = newFieldPathType(TypeExpr(nil, &FuncType{}))

func newFieldPathType(keyToValueFunc Expr) *FieldPathType {
	group := &FieldGroup{Type: TypeExpr(nil, AnyType())}
	group.Fields = []*Field{
		{Group: group, Pos: 0, Name: &ast.Ident{Name: "root"}},
		{Group: group, Pos: 1, Name: &ast.Ident{Name: "value"}},
	}
	tp := &FieldPathType{}
	tp.genParams.SetFields(&FieldList{List: []*FieldGroup{group}})
	return tp
}

// GenericFieldPathType returns the builtin fieldindex type.
func GenericFieldPathType() Type {
	return origin
}

func (*FieldPathType) node()         {}
func (*FieldPathType) storage()      {}
func (*FieldPathType) storageValue() {}

// Node returns the node in the AST tree.
func (*FieldPathType) Node() ast.Node { return fieldPathIdent }

func (t *FieldPathType) root() *TypeValExpr {
	if len(t.genParams.Values) < 1 {
		return nil
	}
	typeVal, ok := t.genParams.Values[0].(*TypeGenericValue)
	if !ok {
		return nil
	}
	return typeVal.tp
}

// Root returns the type indexed by the fieldindex, or nil if not set.
func (t *FieldPathType) Root() Type {
	if len(t.genParams.Values) < 1 {
		return nil
	}
	typeVal, ok := t.genParams.Values[0].(*TypeGenericValue)
	if !ok {
		return nil
	}
	return typeVal.tp.Val()
}

// Refer returns a source expression to refer to the type.
func (t *FieldPathType) Refer(file *File) ast.Expr {
	root := t.Root()
	if root == nil {
		return fieldPathIdent
	}
	return &ast.IndexExpr{X: fieldPathIdent, Index: root.Refer(file)}
}

// Same always returns true.
func (t *FieldPathType) Same(s Storage) bool {
	return t == s
}

// Equal returns true if other is the same type.
func (t *FieldPathType) Equal(tpcmp TypeEval, other Type) (bool, error) {
	otherT, ok := other.(*FieldPathType)
	if !ok {
		return false, nil
	}
	return t.genParams.Equal(tpcmp, &otherT.genParams)
}

// AssignableTo returns true if other is the same type.
func (t *FieldPathType) AssignableTo(tpcmp TypeEval, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

func (t *FieldPathType) assignableFrom(tpcmp TypeEval, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

// ConvertibleTo returns true if other is the same type.
func (t *FieldPathType) ConvertibleTo(tpcmp TypeEval, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

// Instantiate specialises the type to a given target.
func (t *FieldPathType) Instantiate(_ Fetcher, spec Instantiator) (Type, bool) {
	return t.Specialise(spec)
}

// Kind returns the fieldpath kind.
func (*FieldPathType) Kind() irkind.Kind { return irkind.FieldPath }

// Origin returns the builtin fieldpath type.
func (*FieldPathType) Origin() GenericType { return origin }

// Generics returns the generic parameters and their values.
func (t *FieldPathType) Generics() *GenericParams { return &t.genParams }

func (*FieldPathType) specialisableType() {}

// NameDef returns the fieldindex identifier.
func (*FieldPathType) NameDef() *ast.Ident { return fieldPathIdent }

// Type of a type: always return metatype.
func (*FieldPathType) Type() Type { return MetaType() }

// Value returns nil.
func (*FieldPathType) Value(Expr) Expr { return nil }

// DefineString returns the GX source code to define the type.
func (t *FieldPathType) DefineString(from *File) string {
	root := t.root()
	if root == nil {
		return irkind.FieldPath.String()
	}
	return irkind.FieldPath.String() + "[" + root.SourceString(from) + "]"
}

// ReferString returns the GX source to refer to the type.
func (t *FieldPathType) ReferString(from *File) string {
	return t.DefineString(from)
}

// Specialise a type to a given target.
func (t *FieldPathType) Specialise(spec Specialiser) (Type, bool) {
	isTarget := spec.Target() == t
	genParams, ok := t.genParams.Specialise(spec, isTarget)
	if isTarget {
		// The values are now the type arguments of the specialiser:
		// specialise them as well (e.g. to compute [...] ranks from the field path).
		genParams, ok = genParams.Specialise(spec, false)
	}
	return &FieldPathType{genParams: genParams}, ok
}

// UnifyWith recursively unifies a type parameters with types.
func (*FieldPathType) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

// Unroll returns an error because fieldindex types do not support unrolling.
func (t *FieldPathType) Unroll(ev Fetcher, urlr Unroller) (ast.Expr, bool) {
	return nil, ev.Err().AppendInternalf(fieldPathIdent, "%T does not support unrolling", t)
}

// IndexForVarArgs returns the type itself.
func (t *FieldPathType) IndexForVarArgs(ErrSource, int) (Type, bool) {
	return t, true
}

// Visit traverses the type argument of the fieldindex type.
func (t *FieldPathType) Visit(f Finder) {
	t.root().Visit(f)
}
