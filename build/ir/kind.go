// Copyright 2024 Google LLC
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

import "github.com/gx-org/gx/build/ir/irkind"

func toTypeSet(typ Type) (*TypeSet, bool) {
	typSet, ok := Underlying(typ).(*TypeSet)
	return typSet, ok
}

// SupportOperators returns true if the type supports unary or binary operators.
func SupportOperators(typ Type) bool {
	switch typ.Kind() {
	case irkind.IntIdx, irkind.IntLen:
		return true
	case irkind.Bool:
		return true
	case irkind.Bfloat16, irkind.Float32, irkind.Float64:
		return true
	case irkind.Int32, irkind.Int64:
		return true
	case irkind.NumberFloat, irkind.NumberInt:
		return true
	case irkind.Uint32, irkind.Uint64:
		return true
	case irkind.Interface:
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(SupportOperators)
		}
		return false
	default:
		return false
	}
}

// IsDataType returns true if the type can be stored in an array.
func IsDataType(typ Type) bool {
	switch typ.Kind() {
	case irkind.Bool:
	case irkind.Bfloat16:
	case irkind.Float32, irkind.Float64:
	case irkind.Int32, irkind.Int64:
	case irkind.Uint32, irkind.Uint64:
	case irkind.Interface:
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(IsDataType)
		}
		return false
	default:
		return false
	}
	return true
}

// IsIndexType returns true if the kind is a supported array index type.
func IsIndexType(typ Type) bool {
	switch typ.Kind() {
	case irkind.Int32:
	case irkind.Int64:
	case irkind.Uint32:
	case irkind.Uint64:
	case irkind.IntLen:
	case irkind.Interface:
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(IsIndexType)
		}
		return false
	default:
		return false
	}
	return true
}

// IsSlicingOk returns true if the type supports slicing,
// (that is value_of_type[i]).
func IsSlicingOk(typ Type) bool {
	switch typ.Kind() {
	case irkind.Slice, irkind.Array:
		return true
	case irkind.Interface:
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(IsSlicingOk)
		}
		return false
	}
	return false
}

// IsAxisLengthType returns true if a type is the type for an array axis length.
func IsAxisLengthType(typ Type) bool {
	switch typ.Kind() {
	case irkind.IntLen:
		return true
	case irkind.Slice:
		sliceType, ok := Underlying(typ).(*SliceType)
		if !ok {
			return false
		}
		return sliceType.DType.Typ.Kind() == irkind.IntLen
	case irkind.Interface:
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(IsInteger)
		}
		return false
	}
	return false
}

// IsInteger return true if kind is an integer.
func IsInteger(typ Type) bool {
	if typ.Kind() == irkind.Interface {
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(IsInteger)
		}
		return false
	}
	return irkind.IsIntegerKind(typ.Kind())
}

// IsFloat return true if type is a float.
func IsFloat(typ Type) bool {
	if typ.Kind() == irkind.Interface {
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(IsFloat)
		}
		return false
	}
	return irkind.IsFloatKind(typ.Kind())
}

// CanBeNumber returns true if the value of a kind can be a number.
func CanBeNumber(typ Type) bool {
	switch typ.Kind() {
	case irkind.IntLen, irkind.IntIdx:
		return true
	case irkind.Interface:
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(CanBeNumber)
		}
		return false
	default:
		return IsFloat(typ) || IsInteger(typ)
	}
}

// TypeFromKind returns a type from a kind.
func TypeFromKind(kind irkind.Kind) Type {
	switch kind {
	case irkind.IntIdx:
		return IntIndexType()
	case irkind.IntLen:
		return IntLenType()
	case irkind.Bool:
		return BoolType()
	case irkind.Bfloat16:
		return Bfloat16Type()
	case irkind.Float32:
		return Float32Type()
	case irkind.Float64:
		return Float64Type()
	case irkind.Int32:
		return Int32Type()
	case irkind.Int64:
		return Int64Type()
	case irkind.NumberFloat:
		return NumberFloatType()
	case irkind.NumberInt:
		return NumberIntType()
	case irkind.String:
		return StringType()
	case irkind.Uint32:
		return Uint32Type()
	case irkind.Uint64:
		return Uint64Type()
	default:
		return InvalidType()
	}
}

// DefaultNumberType returns the default GX type for a number.
func DefaultNumberType(kind irkind.Kind) Type {
	switch kind {
	case irkind.NumberFloat:
		return NumberFloat{}.DefaultType()
	case irkind.NumberInt:
		return NumberInt{}.DefaultType()
	default:
		return InvalidType()
	}
}

// AtomicFromString returns a scalar type singleton from a string.
func AtomicFromString(ident string) Type {
	return TypeFromKind(irkind.KindFromString(ident))
}
