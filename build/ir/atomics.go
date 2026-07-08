// Copyright 2025 Google LLC
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

	"github.com/gx-org/gx/build/ir/irkind"
)

var (
	boolT = &boolType{}

	bfloat16T = &bfloat16Type{}
	float32T  = &float32Type{}
	float64T  = &float64Type{}

	intT    = &intType{}
	int32T  = &int32Type{}
	int64T  = &int64Type{}
	uint32T = &uint32Type{}
	uint64T = &uint64Type{}

	intSliceType *SliceType
)

func init() {
	boolT.atomicArrayType = newAtomicArrayT(irkind.Bool, boolT)

	bfloat16T.atomicArrayType = newAtomicArrayT(irkind.Bfloat16, bfloat16T)
	float32T.atomicArrayType = newAtomicArrayT(irkind.Float32, float32T)
	float64T.atomicArrayType = newAtomicArrayT(irkind.Float64, float64T)

	intT.atomicArrayType = newAtomicArrayT(irkind.Int, intT)
	int32T.atomicArrayType = newAtomicArrayT(irkind.Int32, int32T)
	int64T.atomicArrayType = newAtomicArrayT(irkind.Int64, int64T)
	uint32T.atomicArrayType = newAtomicArrayT(irkind.Uint32, uint32T)
	uint64T.atomicArrayType = newAtomicArrayT(irkind.Uint64, uint64T)

	intSliceType = &SliceType{
		BaseType: BaseType[ast.Expr]{Src: &ast.ArrayType{}},
		DType:    TypeExpr(nil, IntType()),
		Rank:     1,
	}
}

var scalarRank = &Rank{}

type atomicArrayType struct {
	BaseType[ast.Expr]
	kind irkind.Kind
	tp   ArrayType
}

func newAtomicArrayT(k irkind.Kind, tp ArrayType) *atomicArrayType {
	return &atomicArrayType{
		BaseType: BaseType[ast.Expr]{
			Src: &ast.Ident{Name: k.String()},
		},
		kind: k,
		tp:   tp,
	}
}

func (*atomicArrayType) node()      {}
func (*atomicArrayType) arrayType() {}

// Kind returns the scalar kind.
func (s *atomicArrayType) Kind() irkind.Kind {
	return s.kind
}

func (s *atomicArrayType) ArrayType() ast.Expr {
	return nil
}

// Zero returns a zero expression of the same type.
func (s *atomicArrayType) Zero() Expr {
	return &NumberCastExpr{
		X:   zero,
		Typ: s,
	}
}

// ElementType returns the type of an element.
func (s *atomicArrayType) ElementType() (Type, bool) {
	return InvalidType(), false
}

// DataType returns the element type of the array.
func (s *atomicArrayType) DataType() Type {
	return s.tp
}

func (s *atomicArrayType) equalArray(tpcmp TypeCmp, other ArrayType) (bool, error) {
	if !other.Rank().IsAtomic() {
		return false, nil
	}
	return other.DataType() == s.tp, nil
}

// Equal returns true if other is the same type.
func (s *atomicArrayType) Equal(tpcmp TypeCmp, other Type) (bool, error) {
	otherT, ok := other.(ArrayType)
	if !ok {
		return false, nil
	}
	return s.equalArray(tpcmp, otherT)
}

// Rank of the array.
func (s *atomicArrayType) Rank() ArrayRank {
	return scalarRank
}

// AssignableTo reports if the type can be assigned to other.
func (s *atomicArrayType) AssignableTo(tpcmp TypeCmp, target Type) (bool, error) {
	if assignFrom, ok := target.(assignsFrom); ok {
		return assignFrom.assignableFrom(tpcmp, s.tp)
	}
	return s.Equal(tpcmp, target)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *atomicArrayType) ConvertibleTo(tpcmp TypeCmp, target Type) (bool, error) {
	if convertFrom, ok := target.(convertsFrom); ok {
		return convertFrom.convertibleFrom(tpcmp, s.tp)
	}

	targetT, ok := target.(ArrayType)
	if !ok {
		return false, nil
	}
	return SupportOperators(s.tp) == SupportOperators(targetT.DataType()), nil
}

// Specialise a type to a given target.
func (s *atomicArrayType) Specialise(Specialiser) (Type, bool) {
	return s.tp, true
}

// Instantiate a function type.
func (s *atomicArrayType) Instantiate(Fetcher, Specialiser) (Type, bool) {
	return s.tp, true
}

// Node returns the source code defining the type.
// Always returns nil.
func (s *atomicArrayType) Node() ast.Node {
	return &ast.Ident{Name: s.kind.String()}
}

func (s *atomicArrayType) UnifyWith(uni Unifier, typ Type) bool {
	other, ok := typ.(ArrayType)
	if !ok {
		return true
	}
	return scalarRank.UnifyWith(uni, other.Rank())
}

func (s *atomicArrayType) IndexForVarArgs(ErrSource, int) (Type, bool) {
	return s.tp, true
}

// ReferString returns the string representation of the node in an error message.
func (s *atomicArrayType) ReferString(from *File) string {
	return s.DefineString(from)
}

// DefineString returns the GX source code of the node.
func (s *atomicArrayType) DefineString(*File) string {
	return s.Kind().String()
}

type boolType struct {
	*atomicArrayType
}

// BoolType returns the type for a boolean.
func BoolType() ArrayType {
	return boolT
}

type bfloat16Type struct {
	*atomicArrayType
}

// Bfloat16Type returns the type for a bfloat16.
func Bfloat16Type() ArrayType {
	return bfloat16T
}

type float32Type struct {
	*atomicArrayType
}

// Float32Type returns the type for a float32.
func Float32Type() ArrayType {
	return float32T
}

type float64Type struct {
	*atomicArrayType
}

// Float64Type returns the type for a float64.
func Float64Type() ArrayType {
	return float64T
}

type intType struct {
	*atomicArrayType
}

func (s *intType) Specialise(spec Specialiser) (Type, bool) {
	return intT, true
}

// IntType returns the type for a int32.
func IntType() ArrayType {
	return intT
}

type int32Type struct {
	*atomicArrayType
}

// Int32Type returns the type for a int32.
func Int32Type() ArrayType {
	return int32T
}

type int64Type struct {
	*atomicArrayType
}

// Int64Type returns the type for a int64.
func Int64Type() ArrayType {
	return int64T
}

type uint32Type struct {
	*atomicArrayType
}

// Uint32Type returns the type for a uint32.
func Uint32Type() ArrayType {
	return uint32T
}

type uint64Type struct {
	*atomicArrayType
}

// Uint64Type returns the type for a uint64.
func Uint64Type() ArrayType {
	return uint64T
}

// IntSliceType returns a slice of axis lengths type.
func IntSliceType() *SliceType {
	return intSliceType
}
