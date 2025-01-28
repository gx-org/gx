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

type boolType struct {
	atomicType
}

var boolT = &boolType{atomicType: atomicType{Knd: BoolKind}}

// BoolType returns the type for a boolean.
func BoolType() Type {
	return boolT
}

type float32Type struct {
	atomicType
}

var float32T = &float32Type{atomicType: atomicType{Knd: Float32Kind}}

// Float32Type returns the type for a float32.
func Float32Type() Type {
	return float32T
}

type float64Type struct {
	atomicType
}

var float64T = &float64Type{atomicType: atomicType{Knd: Float64Kind}}

// Float64Type returns the type for a float64.
func Float64Type() Type {
	return float64T
}

type int32Type struct {
	atomicType
}

var int32T = &int32Type{atomicType: atomicType{Knd: Int32Kind}}

// Int32Type returns the type for a int32.
func Int32Type() Type {
	return int32T
}

type int64Type struct {
	atomicType
}

var int64T = &int64Type{atomicType: atomicType{Knd: Int64Kind}}

// Int64Type returns the type for a int64.
func Int64Type() Type {
	return int64T
}

type numberFloatType struct {
	atomicType
}

var numberFloatT = &numberFloatType{atomicType: atomicType{Knd: NumberFloatKind}}

type intidxType struct {
	atomicType
}

var (
	intidxT         = &intidxType{atomicType: atomicType{Knd: IntIdxKind}}
	axisIndicesType = &SliceType{DType: IntIndexType(), Rank: 1}
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
	intlenT         = &intlenType{atomicType: atomicType{Knd: IntLenKind}}
	axisLengthsType = &SliceType{DType: IntLenType(), Rank: 1}
)

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

var numberIntT = &numberIntType{atomicType: atomicType{Knd: NumberIntKind}}

// NumberIntType returns the type for an integer number.
func NumberIntType() Type {
	return numberIntT
}

type stringType struct {
	atomicType
}

var stringT = &stringType{atomicType: atomicType{Knd: StringKind}}

// StringType returns the type for a string.
func StringType() Type {
	return stringT
}

type uint32Type struct {
	atomicType
}

var uint32T = &uint32Type{atomicType: atomicType{Knd: Uint32Kind}}

// Uint32Type returns the type for a uint32.
func Uint32Type() Type {
	return uint32T
}

type uint64Type struct {
	atomicType
}

var uint64T = &uint64Type{atomicType: atomicType{Knd: Uint64Kind}}

// Uint64Type returns the type for a uint64.
func Uint64Type() Type {
	return uint64T
}
