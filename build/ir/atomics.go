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

type atomicType struct {
	BaseType[ast.Expr]

	Knd irkind.Kind
}

func (*atomicType) node()   {}
func (*atomicType) atomic() {}

// Kind returns the scalar kind.
func (s *atomicType) Kind() irkind.Kind { return s.Knd }

func (s *atomicType) equalAtomic(other ArrayType) (bool, error) {
	return s.Knd == other.Kind(), nil
}

func (s *atomicType) equalArray(tpcmp TypeCmp, other ArrayType) (bool, error) {
	dtypeEq, err := s.Equal(tpcmp, other.DataType())
	if !dtypeEq || err != nil {
		return false, err
	}
	return s.Rank().Equal(tpcmp, other.Rank())
}

// Equal returns true if other is the same type.
func (s *atomicType) Equal(tpcmp TypeCmp, other Type) (bool, error) {
	otherT, ok := other.(ArrayType)
	if !ok {
		return false, nil
	}
	if otherT.Rank().IsAtomic() {
		return s.equalAtomic(otherT)
	}
	return s.equalArray(tpcmp, otherT)
}

var scalarRank = &Rank{}

// Rank of the array.
func (s *atomicType) Rank() ArrayRank { return scalarRank }

// AssignableTo reports if the type can be assigned to other.
func (s *atomicType) AssignableTo(tpcmp TypeCmp, target Type) (bool, error) {
	if assignFrom, ok := target.(assignsFrom); ok {
		return assignFrom.assignableFrom(tpcmp, s)
	}

	targetT, ok := target.(ArrayType)
	if !ok {
		return false, nil
	}
	if targetT.Rank().IsAtomic() {
		if s.Knd == irkind.NumberInt && (IsInteger(target) || IsFloat(target)) {
			return true, nil
		}
		if s.Knd == irkind.NumberFloat && IsFloat(target) {
			return true, nil
		}
		return s.equalAtomic(targetT)
	}
	dtypeEq, err := s.AssignableTo(tpcmp, targetT.DataType())
	if !dtypeEq || err != nil {
		return false, err
	}
	rankOk, err := s.Rank().AssignableTo(tpcmp, targetT.Rank())
	if !rankOk || err != nil {
		return rankOk, err
	}
	return true, nil
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *atomicType) ConvertibleTo(tpcmp TypeCmp, target Type) (bool, error) {
	if convertFrom, ok := target.(convertsFrom); ok {
		return convertFrom.convertibleFrom(tpcmp, s)
	}

	targetT, ok := target.(ArrayType)
	if !ok {
		return false, nil
	}
	return SupportOperators(s) == SupportOperators(targetT.DataType()), nil
}

// Specialise a type to a given target.
func (s *atomicType) Specialise(Specialiser) Type {
	return s
}

// Instantiate a function type.
func (s *atomicType) Instantiate(Fetcher, Specialiser) (Type, bool) {
	return s, true
}

// Node returns the source code defining the type.
// Always returns nil.
func (s *atomicType) Node() ast.Node {
	return s.ArrayType()
}

// Value returns a value pointing to the receiver.
func (s *atomicType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// ElementType returns an invalid type because an atomic type cannot be sliced.
func (s *atomicType) ElementType() (Type, bool) {
	return InvalidType(), false
}

// DataType returns the type of the element.
func (s *atomicType) DataType() Type {
	return s
}

// ArrayType returns the source code defining the type.
// Always returns nil.
func (s *atomicType) ArrayType() ast.Expr {
	return &ast.Ident{Name: s.Knd.String()}
}

func (*atomicType) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

func (s *atomicType) IndexForVarArgs(int) Type {
	return s
}

// Zero returns a zero expression of the same type.
func (s *atomicType) Zero() Expr {
	return &NumberCastExpr{
		X:   zero,
		Typ: s,
	}
}

// ReferString returns the string representation of the node in an error message.
func (s *atomicType) ReferString(from *File) string {
	return s.DefineString(from)
}

// DefineString returns the GX source code of the node.
func (s *atomicType) DefineString(*File) string {
	return s.Kind().String()
}

type rankType struct {
	atomicType
}

var rankT = &rankType{atomicType: atomicType{Knd: irkind.Rank}}

// RankType returns the rank type.
func RankType() Type {
	return rankT
}

type boolType struct {
	atomicType
}

var boolT = &boolType{atomicType: atomicType{Knd: irkind.Bool}}

func (s *boolType) Specialise(spec Specialiser) Type { return boolT }

// BoolType returns the type for a boolean.
func BoolType() Type {
	return boolT
}

type bfloat16Type struct {
	atomicType
}

var bfloat16T = &bfloat16Type{atomicType: atomicType{Knd: irkind.Bfloat16}}

func (s *bfloat16Type) Specialise(spec Specialiser) Type {
	return bfloat16T
}

// Bfloat16Type returns the type for a bfloat16.
func Bfloat16Type() Type {
	return bfloat16T
}

type float32Type struct {
	atomicType
}

var float32T = &float32Type{atomicType: atomicType{Knd: irkind.Float32}}

func (s *float32Type) Specialise(spec Specialiser) Type {
	return float32T
}

// Float32Type returns the type for a float32.
func Float32Type() Type {
	return float32T
}

type float64Type struct {
	atomicType
}

var float64T = &float64Type{atomicType: atomicType{Knd: irkind.Float64}}

func (s *float64Type) Specialise(spec Specialiser) Type {
	return float64T
}

// Float64Type returns the type for a float64.
func Float64Type() Type {
	return float64T
}

type int32Type struct {
	atomicType
}

var int32T = &int32Type{atomicType: atomicType{Knd: irkind.Int32}}

func (s *int32Type) Specialise(spec Specialiser) Type {
	return int32T
}

// Int32Type returns the type for a int32.
func Int32Type() Type {
	return int32T
}

type int64Type struct {
	atomicType
}

var int64T = &int64Type{atomicType: atomicType{Knd: irkind.Int64}}

func (s *int64Type) Specialise(spec Specialiser) Type {
	return int64T
}

// Int64Type returns the type for a int64.
func Int64Type() Type {
	return int64T
}

type numberFloatType struct {
	atomicType
}

var numberFloatT = &numberFloatType{atomicType: atomicType{Knd: irkind.NumberFloat}}

type intidxType struct {
	atomicType
}

var (
	intidxT         = &intidxType{atomicType: atomicType{Knd: irkind.IntIdx}}
	axisIndicesType = &SliceType{
		DType: TypeExpr(nil, IntIndexType()),
		Rank:  1,
	}
)

// IntIndexType returns the type for intidx, that is the length of an axis.
func IntIndexType() Type {
	return intidxT
}

// IntIndexSliceType returns a slice of axis lengths type.
func IntIndexSliceType() *SliceType {
	return axisIndicesType
}

type intlenType struct {
	atomicType
}

var (
	intlenT         = &intlenType{atomicType: atomicType{Knd: irkind.IntLen}}
	axisLengthsType = &SliceType{
		BaseType: BaseType[ast.Expr]{Src: &ast.ArrayType{}},
		DType:    TypeExpr(nil, IntLenType()),
		Rank:     1,
	}
)

func (s *intlenType) Specialise(spec Specialiser) Type {
	return intlenT
}

// IntLenType returns the type for intlen, that is the length of an axis.
func IntLenType() Type {
	return intlenT
}

// IntLenSliceType returns a slice of axis lengths type.
func IntLenSliceType() *SliceType {
	return axisLengthsType
}

// NumberFloatType returns the type for a float number.
func NumberFloatType() Type {
	return numberFloatT
}

type numberIntType struct {
	atomicType
}

var numberIntT = &numberIntType{atomicType: atomicType{Knd: irkind.NumberInt}}

// NumberIntType returns the type for an integer number.
func NumberIntType() Type {
	return numberIntT
}

type stringType struct {
	atomicType
}

var stringT = &stringType{atomicType: atomicType{Knd: irkind.String}}

func (s *stringType) Specialise(spec Specialiser) Type {
	return stringT
}

// StringType returns the type for a string.
func StringType() Type {
	return stringT
}

type uint32Type struct {
	atomicType
}

var uint32T = &uint32Type{atomicType: atomicType{Knd: irkind.Uint32}}

func (s *uint32Type) Specialise(spec Specialiser) Type {
	return uint32T
}

// Uint32Type returns the type for a uint32.
func Uint32Type() Type {
	return uint32T
}

type uint64Type struct {
	atomicType
}

var uint64T = &uint64Type{atomicType: atomicType{Knd: irkind.Uint64}}

func (s *uint64Type) Specialise(spec Specialiser) Type {
	return uint64T
}

// Uint64Type returns the type for a uint64.
func Uint64Type() ArrayType {
	return uint64T
}
