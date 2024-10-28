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

// ScalarTypeK returns a scalar type singleton from a kind.
func ScalarTypeK(kind Kind) *AtomicType {
	switch kind {
	case AxisIndexKind:
		return AxisIndexType()
	case AxisLengthKind:
		return AxisLengthType()
	case BoolKind:
		return &AtomicType{Knd: BoolKind}
	case Int32Kind:
		return &AtomicType{Knd: Int32Kind}
	case Int64Kind:
		return &AtomicType{Knd: Int64Kind}
	case Uint32Kind:
		return &AtomicType{Knd: Uint32Kind}
	case Uint64Kind:
		return &AtomicType{Knd: Uint64Kind}
	case Float32Kind:
		return &AtomicType{Knd: Float32Kind}
	case Float64Kind:
		return &AtomicType{Knd: Float64Kind}
	case NumberKind:
		return &AtomicType{Knd: UnknownKind}
	default:
		return &AtomicType{Knd: InvalidKind}
	}
}

// ScalarTypeS returns a scalar type singleton from a string.
func ScalarTypeS(ident string) *AtomicType {
	switch ident {
	case "bool":
		return ScalarTypeK(BoolKind)
	case "int32":
		return ScalarTypeK(Int32Kind)
	case "int64":
		return ScalarTypeK(Int64Kind)
	case "uint32":
		return ScalarTypeK(Uint32Kind)
	case "uint64":
		return ScalarTypeK(Uint64Kind)
	case "float32":
		return ScalarTypeK(Float32Kind)
	case "float64":
		return ScalarTypeK(Float64Kind)
	default:
		return ScalarTypeK(InvalidKind)
	}
}
