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
	"go/ast"
	"reflect"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir/irkind"
)

var (
	numberFloatT = &numberFloatType{}
	numberIntT   = &numberIntType{}
)

func init() {
	numberFloatT.numberType = newNumberT(irkind.NumberFloat, numberFloatT)
	numberIntT.numberType = newNumberT(irkind.NumberInt, numberIntT)
}

type numberType struct {
	BaseType[ast.Expr]
	kind irkind.Kind
	tp   ArrayType
}

func newNumberT(k irkind.Kind, tp ArrayType) *numberType {
	return &numberType{
		BaseType: BaseType[ast.Expr]{
			Src: &ast.Ident{Name: k.String()},
		},
		kind: k,
		tp:   tp,
	}
}

func (*numberType) node()      {}
func (*numberType) arrayType() {}

func (t *numberType) ArrayType() ast.Expr {
	return nil
}

func (t *numberType) Rank() ArrayRank {
	return scalarRank
}

func (t *numberType) Kind() irkind.Kind {
	return t.kind
}

func (t *numberType) Zero() Expr {
	return zero
}

func (t *numberType) equalArray(tpcmp TypeCmp, other ArrayType) (bool, error) {
	if !other.Rank().IsAtomic() {
		return false, nil
	}
	return other.DataType() == t.tp, nil
}

func (t *numberType) Equal(tpcmp TypeCmp, other Type) (bool, error) {
	return t.tp == other, nil
}

func (t *numberType) ConvertibleTo(tpcmp TypeCmp, target Type) (bool, error) {
	return false, nil
}

func (t *numberType) Specialise(Specialiser) (Type, bool) {
	return t.tp, true
}

func (t *numberType) Instantiate(Fetcher, Specialiser) (Type, bool) {
	return t.tp, true
}

func (t *numberType) UnifyWith(uni Unifier, typ Type) bool {
	return true
}

func (t *numberType) ReferString(from *File) string {
	return t.DefineString(from)
}

func (t *numberType) DefineString(*File) string {
	return t.Kind().String()
}

func (t *numberType) IndexForVarArgs(ErrSource, int) (Type, bool) {
	return t.tp, true
}

func (t *numberType) ElementType() (Type, bool) {
	return InvalidType(), false
}

func (t *numberType) DataType() Type {
	return t.tp
}

// numberFloatType is the type returned by function with no results.
type numberFloatType struct {
	*numberType
}

var _ ArrayType = (*numberFloatType)(nil)

func (t *numberFloatType) AssignableTo(tpcmp TypeCmp, target Type) (bool, error) {
	return IsFloat(target), nil
}

// NumberFloatType returns the numberFloat type.
func NumberFloatType() Type {
	return numberFloatT
}

// numberIntType is the type returned by function with no results.
type numberIntType struct {
	*numberType
}

var _ ArrayType = (*numberIntType)(nil)

func (t *numberIntType) AssignableTo(tpcmp TypeCmp, target Type) (bool, error) {
	return IsInteger(target) || IsFloat(target), nil
}

// NumberIntType returns the numberInt type.
func NumberIntType() Type {
	return numberIntT
}

// FileWithError is an interface able to return a file context and an error appender.
type FileWithError interface {
	File() *File
	fmterr.ErrAppender
}

// CastNumber builds a number cast expression.
func CastNumber(fetcher FileWithError, expr Expr, target Type) (*NumberCastExpr, bool) {
	cast := &NumberCastExpr{
		X:   expr,
		Typ: target,
	}
	if cast.Typ.Kind() == irkind.Unknown {
		// No specification on what we want.
		// For example:
		//   a := 5.2
		// Then we cast the number, 5.2 in this example, to a default type.
		cast.Typ = DefaultNumberType(expr.Type().Kind())
	}
	if cast.Typ.Kind() == irkind.Array {
		// The required type is an array. For example:
		//   a := 5 * [2]float32{1, 2}
		// We cast the number, 5 in this example, to the data type of the array.
		underlying := Underlying(cast.Typ)
		arrayType, ok := underlying.(ArrayType)
		if !ok {
			return cast, fetcher.Err().AppendInternalf(expr.Node(), "type %T has %s but does not implement %s", underlying, irkind.Array.String(), reflect.TypeFor[ArrayType]().Name())
		}
		cast.Typ = arrayType.DataType()
	}
	if !CanBeNumber(cast.Typ) {
		from := fetcher.File()
		// Return an error for code like:
		// func f() string { return 3 }
		return cast, fetcher.Err().Appendf(expr.Node(), "cannot use a number as %v", cast.Typ.ReferString(from))
	}
	if IsFloat(expr.Type()) && IsInteger(cast.Typ) {
		from := fetcher.File()
		return cast, fetcher.Err().Appendf(expr.Node(), "cannot use %s (untyped FLOAT constant) as %s value", expr.SourceString(from), cast.Typ.ReferString(from))
	}
	return cast, true
}
