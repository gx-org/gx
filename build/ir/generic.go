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
	"fmt"
	"go/ast"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir/irkind"
)

type (
	// Specialiser provides methods to specialise a type.
	Specialiser interface {
		ErrSource
		IsDefined(int) bool
		NonTypeFor(GenericParam) *NonTypeGenericValue
		TypeFor(GenericParam) *TypeGenericValue
		Values() []GenericValue
		InstantiateError(error) bool
	}

	// Unifier provides methods to unify types.
	Unifier interface {
		fmterr.ErrAppender
		DefineType(genType *GenericTypeParam, typ Type) bool
		DefineNonType(genAxis *GenericNonTypeParam, expr Expr) bool
	}

	// GenericParam is a storage pointing to a type parameter field.
	// In a function declared as:
	//   func F[S []int, T Ints](array [S]T)
	// then S and T will be resolved as Ident using GenericParam as storage
	// pointing to their respective fields.
	//
	// GenericParam has two implementations:
	//   - GenericTypeParam for type parameter (T in the example above)
	//   - GenericNonTypeParam for non-type parameter (S in the example above)
	GenericParam interface {
		genericIdent()
		Storage
		// Always returns the field in the original function type
		// because function types are cloned but not generic parameters
		OrigField() *Field
		Assign(Fetcher, Expr) (GenericValue, bool)
		IsTypeParam() bool
	}

	genericParamCore struct {
		typeParam bool
		field     *Field
	}
)

func newGenericParamCore(field *Field, typeParam bool) genericParamCore {
	return genericParamCore{
		typeParam: typeParam,
		field:     field,
	}
}

func (*genericParamCore) node()           {}
func (*genericParamCore) storage()        {}
func (s *genericParamCore) genericIdent() {}

func (s *genericParamCore) pos() int {
	return s.field.Pos
}

func (s *genericParamCore) OrigField() *Field {
	return s.field
}

// Node defining the type.
func (s *genericParamCore) Node() ast.Node {
	return s.field.Node()
}

// NameDef returns the identifier identifying the storage.
func (s *genericParamCore) NameDef() *ast.Ident {
	return s.field.Name
}

// Same returns true if the other storage is this storage.
func (s *genericParamCore) Same(o Storage) bool {
	return false
}

func (s *genericParamCore) IsTypeParam() bool {
	return s.typeParam
}

// GenericTypeParam is a type mapping to a field name in a function type parameters.
type GenericTypeParam struct {
	genericParamCore
}

var (
	_ Type         = (*GenericTypeParam)(nil)
	_ GenericParam = (*GenericTypeParam)(nil)
)

func (*GenericTypeParam) storageValue() {}

// NewGenericTypeParam creates a new identifier to a generic type.
func NewGenericTypeParam(field *Field) *GenericTypeParam {
	return &GenericTypeParam{
		genericParamCore: newGenericParamCore(field, true),
	}
}

func (s *GenericTypeParam) typ() Type {
	return s.field.Type()
}

// Kind of the type.
func (s *GenericTypeParam) Kind() irkind.Kind {
	return s.typ().Kind()
}

func (s *GenericTypeParam) equal(tpcmp TypeCmp, typ Type) (bool, error) {
	switch typT := typ.(type) {
	case *GenericTypeParam:
		if s.pos() == typT.pos() {
			return true, nil
		}
	case *NamedType:
		return s.typ().Equal(tpcmp, typ)
	case ArrayType:
		if !typT.Rank().IsAtomic() {
			return false, nil
		}
		return s.Equal(tpcmp, typT.DataType())
	}
	return false, nil
}

// Equal returns true if other is the same type.
func (s *GenericTypeParam) Equal(tpcmp TypeCmp, typ Type) (bool, error) {
	return s.equal(tpcmp, typ)
}

func (s *GenericTypeParam) assignableFrom(tpcmp TypeCmp, other Type) (bool, error) {
	return other.AssignableTo(tpcmp, s.typ())
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (s *GenericTypeParam) AssignableTo(tpcmp TypeCmp, typ Type) (bool, error) {
	return s.Equal(tpcmp, typ)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *GenericTypeParam) ConvertibleTo(tpcmp TypeCmp, typ Type) (bool, error) {
	return s.typ().ConvertibleTo(tpcmp, typ)
}

// convertibleFrom reports whether a value of some type can be converted to the receiver.
func (s *GenericTypeParam) convertibleFrom(tpcmp TypeCmp, from Type) (bool, error) {
	return from.ConvertibleTo(tpcmp, s.typ())
}

// Instantiate the generic type parameter.
func (s *GenericTypeParam) Instantiate(_ Fetcher, spec Specialiser) (Type, bool) {
	return s.Specialise(spec)
}

// Type of the type.
func (s *GenericTypeParam) Type() Type {
	return MetaType()
}

// Value returns a value pointing to the receiver.
func (s *GenericTypeParam) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// DefineString returns the GX source code of the node.
func (s *GenericTypeParam) DefineString(from *File) string {
	return fmt.Sprintf("%s %s", s.NameDef().Name, s.typ().ReferString(from))
}

// ReferString returns the string representation of the node in an error message.
func (s *GenericTypeParam) ReferString(from *File) string {
	name := s.field.Name
	if name == nil {
		return ""
	}
	return name.Name
}

// Specialise a type to a given target.
func (s *GenericTypeParam) Specialise(spec Specialiser) (Type, bool) {
	tp := spec.TypeFor(s)
	if tp == nil {
		return s, true
	}
	return tp.Type(), true
}

// UnifyWith recursively unifies a type parameters with types.
func (s *GenericTypeParam) UnifyWith(uni Unifier, typ Type) bool {
	return uni.DefineType(s, typ)
}

// IndexForVarArgs returns a type specific to a given index in varargs.
func (s *GenericTypeParam) IndexForVarArgs(ErrSource, int) (Type, bool) {
	return s, true
}

func (s *GenericTypeParam) invalidValue() GenericValue {
	return NewTypeGenericValue(s, InvalidType())
}

// Assign an expression to the generic type parameter.
func (s *GenericTypeParam) Assign(fetcher Fetcher, x Expr) (GenericValue, bool) {
	typeValExpr := TypeFromExpr(x)
	if typeValExpr == nil {
		return s.invalidValue(), fetcher.Err().Appendf(x.Node(), "%s is not a type", x.SourceString(fetcher.File()))
	}
	gotType, wantType := typeValExpr.Val(), s.field.Group.Type.Val()
	assignedOk, err := gotType.AssignableTo(fetcher, wantType)
	if err != nil {
		return s.invalidValue(), fetcher.Err().AppendAt(x.Node(), err)
	}
	if !assignedOk {
		return s.invalidValue(), fetcher.Err().Appendf(x.Node(), "%s does not satisfy %s", gotType.ReferString(fetcher.File()), wantType.ReferString(fetcher.File()))
	}
	return NewTypeExprGenericValue(s, typeValExpr), true
}

// GenericNonTypeParam is a reference to a non-type type parameter.
type GenericNonTypeParam struct {
	genericParamCore
}

var _ GenericParam = (*GenericNonTypeParam)(nil)

// NewGenericNonTypeParam creates a new identifier to a generic axis set.
func NewGenericNonTypeParam(field *Field) *GenericNonTypeParam {
	return &GenericNonTypeParam{
		genericParamCore: newGenericParamCore(field, false),
	}
}

// Type of the type.
func (s *GenericNonTypeParam) Type() Type {
	return s.field.Type()
}

func (s *GenericNonTypeParam) invalidValue() GenericValue {
	return NewAxisGenericValue(s, TypeExpr(nil, InvalidType()))
}

// Assign an expression to the generic non-type parameter.
func (s *GenericNonTypeParam) Assign(fetcher Fetcher, x Expr) (GenericValue, bool) {
	src := x.Type()
	dst := s.Type()
	assignable, err := AssignableTo(fetcher, src, dst)
	if err != nil {
		return s.invalidValue(), fetcher.Err().AppendAt(x.Node(), err)
	}
	if !assignable {
		from := fetcher.File()
		return s.invalidValue(), fetcher.Err().Appendf(x.Node(), "cannot use %s as %s generic value %s", src.ReferString(from), dst.ReferString(from), s.OrigField().Name.Name)
	}
	xEval, err := CompEvalExpr(fetcher, x)
	if err != nil {
		return s.invalidValue(), fetcher.Err().AppendAt(x.Expr(), err)
	}
	return NewAxisGenericValue(s, xEval[0]), true
}

type (
	// GenericValue is a value of a generic type parameter.
	GenericValue interface {
		genericValue()
		StringSourcer
		Name() string
		Generic() GenericParam
		DefinedType() Type
		Value() Expr
	}

	coreGenericValue struct {
		tparam GenericParam
	}

	// NonTypeGenericValue assigns a value to an axis length.
	NonTypeGenericValue struct {
		coreGenericValue
		x Expr
	}
	// TypeGenericValue assigns a type to a field of a more generic type.
	TypeGenericValue struct {
		coreGenericValue
		tp *TypeValExpr
	}
)

var (
	_ GenericValue = (*NonTypeGenericValue)(nil)
	_ GenericValue = (*TypeGenericValue)(nil)
)

func (coreGenericValue) genericValue()           {}
func (v coreGenericValue) Generic() GenericParam { return v.tparam }

// Name of the generic value defined by its type parameter field.
func (v *coreGenericValue) Name() string {
	return v.tparam.NameDef().Name
}

// NewAxisGenericValue returns a new axis value from IR expressions and an interpreter element.
func NewAxisGenericValue(tparam *GenericNonTypeParam, x Expr) *NonTypeGenericValue {
	return &NonTypeGenericValue{
		coreGenericValue: coreGenericValue{
			tparam: tparam,
		},
		x: x,
	}
}

// Equal returns true if the set of axes is the same than the already attached rank.
func (v *NonTypeGenericValue) Equal(tpcmp TypeCmp, x Expr) (bool, error) {
	return areEqual(tpcmp, v.x, x)
}

// DefinedType returns the type of the type param.
func (v *NonTypeGenericValue) DefinedType() Type {
	return v.tparam.Type()
}

// SourceString returns a string representing all axes.
func (v *NonTypeGenericValue) SourceString(from *File) string {
	return fmt.Sprintf("[%s]", v.x.SourceString(from))
}

// Value returns the value of the generic.
func (v *NonTypeGenericValue) Value() Expr {
	return v.x
}

func (v *NonTypeGenericValue) String() string {
	return fmt.Sprintf("(%d:%s %s)", v.Generic().OrigField().Pos, v.Name(), v.DefinedType().ReferString(nil))
}

// AxisLengthsFromExpr converts an expression to an axis length.
func AxisLengthsFromExpr(x Expr) []AxisLengths {
	return []AxisLengths{&AxisExpr{X: x}}
}

// NewTypeGenericValue returns a generic type value directly from a type instead of a type expression.
// Use when the type is inferred.
func NewTypeGenericValue(tparam *GenericTypeParam, tp Type) *TypeGenericValue {
	return NewTypeExprGenericValue(tparam, TypeExpr(nil, tp))
}

// NewTypeExprGenericValue returns a new type generic value.
func NewTypeExprGenericValue(tparam *GenericTypeParam, tp *TypeValExpr) *TypeGenericValue {
	return &TypeGenericValue{
		coreGenericValue: coreGenericValue{
			tparam: tparam,
		},
		tp: tp,
	}
}

// Type returns the set of axis length defining the type parameter.
func (v *TypeGenericValue) Type() Type {
	return v.tp.Val()
}

// SourceString returns a string representing of the value of type parameter.
func (v *TypeGenericValue) SourceString(from *File) string {
	return fmt.Sprintf("[%s]", v.tp.SourceString(from))
}

// DefinedType returns the type defined for the type param.
func (v *TypeGenericValue) DefinedType() Type {
	return v.tp.Val()
}

// Value returns type assigned to the generic.
func (v *TypeGenericValue) Value() Expr {
	return v.tp
}

// ExprWithSpecialise is an expression that can be specialise itself.
// Only used in axis expression in function signature.
type ExprWithSpecialise interface {
	Expr
	// SpecialiseForAxis an expression.
	Specialise(Specialiser) (Expr, bool)
}

func specialiseExpr(spec Specialiser, x Expr) (Expr, bool) {
	xSpec, canSpecialise := x.(ExprWithSpecialise)
	if !canSpecialise {
		return x, true
	}
	return xSpec.Specialise(spec)
}

// ExprWithUnify is an expression that can be unify its type.
// Only used in axis expression in function signature.
type ExprWithUnify interface {
	Expr
	UnifyWith(uni Unifier, targets []AxisLengths) ([]AxisLengths, bool)
}

func unifyExpr(uni Unifier, targets []AxisLengths, x Expr) ([]AxisLengths, bool) {
	xUni, canSpecialise := x.(ExprWithUnify)
	if !canSpecialise {
		return targets, true
	}
	return xUni.UnifyWith(uni, targets)
}

// IsNonTypeGeneric returns true if the type is a non-type generic.
func IsNonTypeGeneric(tp Type) bool {
	return tp.Kind() != irkind.Interface
}
