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

import (
	"github.com/gx-org/backend/dtype"
)

// Kind of a type.
type Kind uint

// Kind of data supported by GX.
const (
	InvalidKind = Kind(dtype.Invalid)

	BoolKind     = Kind(dtype.Bool)
	Int32Kind    = Kind(dtype.Int32)
	Int64Kind    = Kind(dtype.Int64)
	Uint32Kind   = Kind(dtype.Uint32)
	Uint64Kind   = Kind(dtype.Uint64)
	Bfloat16Kind = Kind(dtype.Bfloat16)
	Float32Kind  = Kind(dtype.Float32)
	Float64Kind  = Kind(dtype.Float64)

	IntIdxKind = Kind(iota + dtype.MaxDataType)
	IntLenKind

	// UnknownKind is a proxy type used while a type is being inferred by the compiler.
	UnknownKind
	// VoidKind is a type for expression returning nothing.
	VoidKind
	// InterfaceKind is an interface.
	InterfaceKind
	// NumberFloatKind is a float number with no concrete type.
	NumberFloatKind
	// NumberIntKind is an integer number with no concrete type.
	NumberIntKind

	ArrayKind
	BuiltinKind
	FuncKind
	RankKind
	SliceKind
	StringKind
	StructKind
	TupleKind
	IRKind
	PackageKind
	MetaTypeKind

	// Maximum value for a Kind constant.
	MaxKind
)

// String returns a string representation of a kind.
func (k Kind) String() string {
	switch k {
	case UnknownKind:
		return "unknown"
	case VoidKind:
		return "void"
	case InterfaceKind:
		return "interface"
	case NumberFloatKind:
		return "float number"
	case NumberIntKind:
		return "int number"
	case IntIdxKind:
		return "intidx"
	case IntLenKind:
		return "intlen"
	case BoolKind:
		return "bool"
	case Int32Kind:
		return "int32"
	case Int64Kind:
		return "int64"
	case Uint32Kind:
		return "uint32"
	case Uint64Kind:
		return "uint64"
	case Bfloat16Kind:
		return "bfloat16"
	case Float32Kind:
		return "float32"
	case Float64Kind:
		return "float64"
	case ArrayKind:
		return "array"
	case FuncKind:
		return "func"
	case RankKind:
		return "rank"
	case SliceKind:
		return "slice"
	case StringKind:
		return "string"
	case StructKind:
		return "struct"
	case BuiltinKind:
		return "builtin"
	case MetaTypeKind:
		return "metatype"
	}
	return "invalid"
}

// DType converts a GX kind into an array data type.
func (k Kind) DType() dtype.DataType {
	if k == IntIdxKind || k == IntLenKind {
		return DefaultIntKind.DType()
	}
	if k >= dtype.MaxDataType {
		return dtype.Invalid
	}
	return dtype.DataType(k)
}

// KindFromString returns a kind given an identifier.
// It only works for the basic types that don't take other parameters -- so it doesn't
// work for tensors, tuples or functions.
//
// The "unknown" string returns an invalid kind because it is invalid to write:
//
//	var name unknown
func KindFromString(ident string) Kind {
	switch ident {
	case "intidx":
		return IntIdxKind
	case "intlen":
		return IntLenKind
	case "bool":
		return BoolKind
	case "bfloat16":
		return Bfloat16Kind
	case "float32":
		return Float32Kind
	case "float64":
		return Float64Kind
	case "int32":
		return Int32Kind
	case "int64":
		return Int64Kind
	case "string":
		return StringKind
	case "uint32":
		return Uint32Kind
	case "uint64":
		return Uint64Kind
	default:
		return InvalidKind
	}
}

// KindGeneric returns the kind of a variable from its generic type.
// If the type is not supported, an invalid type is returned.
func KindGeneric[T dtype.GoDataType]() Kind {
	return Kind(dtype.Generic[T]())
}

// IsNumber returns true if the kind is a number.
func IsNumber(knd Kind) bool {
	return knd == NumberFloatKind || knd == NumberIntKind
}

func toTypeSet(typ Type) (*TypeSet, bool) {
	typSet, ok := Underlying(typ).(*TypeSet)
	return typSet, ok
}

// SupportOperators returns true if the type supports unary or binary operators.
func SupportOperators(typ Type) bool {
	switch typ.Kind() {
	case IntIdxKind, IntLenKind:
		return true
	case BoolKind:
		return true
	case Bfloat16Kind, Float32Kind, Float64Kind:
		return true
	case Int32Kind, Int64Kind:
		return true
	case NumberFloatKind, NumberIntKind:
		return true
	case Uint32Kind, Uint64Kind:
		return true
	case InterfaceKind:
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
	case BoolKind:
	case Bfloat16Kind:
	case Float32Kind, Float64Kind:
	case Int32Kind, Int64Kind:
	case Uint32Kind, Uint64Kind:
	case InterfaceKind:
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
	case Int32Kind:
	case Int64Kind:
	case Uint32Kind:
	case Uint64Kind:
	case IntLenKind:
	case InterfaceKind:
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
	case SliceKind, ArrayKind:
		return true
	case InterfaceKind:
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(IsSlicingOk)
		}
		return false
	}
	return false
}

// IsRangeOk returns true if the kind can be used to iterate in a for loop with a range statement.
func IsRangeOk(k Kind) bool {
	switch k {
	case IntLenKind:
	case NumberIntKind:
	default:
		return false
	}
	return true
}

// IsIntegerKind return true if kind is an integer.
func IsIntegerKind(kind Kind) bool {
	switch kind {
	case IntLenKind, IntIdxKind:
		return true
	case Int32Kind, Int64Kind, Uint32Kind, Uint64Kind:
		return true
	case NumberIntKind:
		return true
	}
	return false
}

// IsAxisLengthType returns true if a type is the type for an array axis length.
func IsAxisLengthType(typ Type) bool {
	switch typ.Kind() {
	case IntLenKind:
		return true
	case SliceKind:
		sliceType, ok := Underlying(typ).(*SliceType)
		if !ok {
			return false
		}
		return sliceType.DType.Typ.Kind() == IntLenKind
	case InterfaceKind:
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(IsInteger)
		}
		return false
	}
	return false
}

// IsInteger return true if kind is an integer.
func IsInteger(typ Type) bool {
	if typ.Kind() == InterfaceKind {
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(IsInteger)
		}
		return false
	}
	return IsIntegerKind(typ.Kind())
}

// IsFloatKind returns true if kind is a float.
func IsFloatKind(kind Kind) bool {
	switch kind {
	case Bfloat16Kind:
		return true
	case Float32Kind, Float64Kind:
		return true
	case NumberFloatKind:
		return true
	}
	return false
}

// IsFloat return true if type is a float.
func IsFloat(typ Type) bool {
	if typ.Kind() == InterfaceKind {
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(IsFloat)
		}
		return false
	}
	return IsFloatKind(typ.Kind())
}

// CanBeNumber returns true if the value of a kind can be a number.
func CanBeNumber(typ Type) bool {
	switch typ.Kind() {
	case IntLenKind, IntIdxKind:
		return true
	case InterfaceKind:
		if typSet, ok := toTypeSet(typ); ok {
			return typSet.hasCapability(CanBeNumber)
		}
		return false
	default:
		return IsFloat(typ) || IsInteger(typ)
	}
}

// DefaultNumberType returns the default GX type for a number.
func DefaultNumberType(kind Kind) Type {
	switch kind {
	case NumberFloatKind:
		return NumberFloat{}.DefaultType()
	case NumberIntKind:
		return NumberInt{}.DefaultType()
	default:
		return InvalidType()
	}
}

// TypeFromKind returns a type from a kind.
func TypeFromKind(kind Kind) Type {
	switch kind {
	case IntIdxKind:
		return IntIndexType()
	case IntLenKind:
		return IntLenType()
	case BoolKind:
		return BoolType()
	case Bfloat16Kind:
		return Bfloat16Type()
	case Float32Kind:
		return Float32Type()
	case Float64Kind:
		return Float64Type()
	case Int32Kind:
		return Int32Type()
	case Int64Kind:
		return Int64Type()
	case NumberFloatKind:
		return NumberFloatType()
	case NumberIntKind:
		return NumberIntType()
	case StringKind:
		return StringType()
	case Uint32Kind:
		return Uint32Type()
	case Uint64Kind:
		return Uint64Type()
	default:
		return InvalidType()
	}
}

// AtomicFromString returns a scalar type singleton from a string.
func AtomicFromString(ident string) Type {
	return TypeFromKind(KindFromString(ident))
}
