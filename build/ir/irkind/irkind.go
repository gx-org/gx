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

// Package irkind defines kind for the GX intermediate representation (IR).
package irkind

import "github.com/gx-org/backend/dtype"

// Kind of a type.
type Kind uint

// DefaultInt is the default kind for integer.
const DefaultInt = Int64

// Kind of data supported by GX.
const (
	Invalid = Kind(dtype.Invalid)

	Bool     = Kind(dtype.Bool)
	Int32    = Kind(dtype.Int32)
	Int64    = Kind(dtype.Int64)
	Uint32   = Kind(dtype.Uint32)
	Uint64   = Kind(dtype.Uint64)
	Bfloat16 = Kind(dtype.Bfloat16)
	Float32  = Kind(dtype.Float32)
	Float64  = Kind(dtype.Float64)

	IntIdx = Kind(iota + dtype.MaxDataType)
	IntLen

	// Unknown is a proxy type used while a type is being inferred by the compiler.
	Unknown
	// Void is a type for expression returning nothing.
	Void
	// Interface is an interface.
	Interface
	// NumberFloat is a float number with no concrete type.
	NumberFloat
	// NumberInt is an integer number with no concrete type.
	NumberInt

	Array
	Builtin
	Func
	Rank
	Slice
	String
	Struct
	Tuple
	IR
	Package
	MetaType

	// Max value for a Kind constant.
	Max
)

// String returns a string representation of a kind.
func (k Kind) String() string {
	switch k {
	case Unknown:
		return "unknown"
	case Void:
		return "void"
	case Interface:
		return "interface"
	case NumberFloat:
		return "float number"
	case NumberInt:
		return "int number"
	case IntIdx:
		return "intidx"
	case IntLen:
		return "intlen"
	case Bool:
		return "bool"
	case Int32:
		return "int32"
	case Int64:
		return "int64"
	case Uint32:
		return "uint32"
	case Uint64:
		return "uint64"
	case Bfloat16:
		return "bfloat16"
	case Float32:
		return "float32"
	case Float64:
		return "float64"
	case Array:
		return "array"
	case Func:
		return "func"
	case Rank:
		return "rank"
	case Slice:
		return "slice"
	case String:
		return "string"
	case Struct:
		return "struct"
	case Builtin:
		return "builtin"
	case MetaType:
		return "metatype"
	}
	return "invalid"
}

// DType converts a GX kind into an array data type.
func (k Kind) DType() dtype.DataType {
	if k == IntIdx || k == IntLen {
		return DefaultInt.DType()
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
		return IntIdx
	case "intlen":
		return IntLen
	case "bool":
		return Bool
	case "bfloat16":
		return Bfloat16
	case "float32":
		return Float32
	case "float64":
		return Float64
	case "int32":
		return Int32
	case "int64":
		return Int64
	case "string":
		return String
	case "uint32":
		return Uint32
	case "uint64":
		return Uint64
	default:
		return Invalid
	}
}

// KindGeneric returns the kind of a variable from its generic type.
// If the type is not supported, an invalid type is returned.
func KindGeneric[T dtype.GoDataType]() Kind {
	return Kind(dtype.Generic[T]())
}

// IsNumber returns true if the kind is a number.
func IsNumber(knd Kind) bool {
	return knd == NumberFloat || knd == NumberInt
}

// IsRangeOk returns true if the kind can be used to iterate in a for loop with a range statement.
func IsRangeOk(k Kind) bool {
	switch k {
	case IntLen:
	case NumberInt:
	default:
		return false
	}
	return true
}

// IsIntegerKind return true if kind is an integer.
func IsIntegerKind(kind Kind) bool {
	switch kind {
	case IntLen, IntIdx:
		return true
	case Int32, Int64, Uint32, Uint64:
		return true
	case NumberInt:
		return true
	}
	return false
}

// IsFloatKind returns true if kind is a float.
func IsFloatKind(kind Kind) bool {
	switch kind {
	case Bfloat16:
		return true
	case Float32, Float64:
		return true
	case NumberFloat:
		return true
	}
	return false
}
