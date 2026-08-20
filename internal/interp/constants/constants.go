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

// Package constants provides elements for constants used by the interpreter.
package constants

import (
	"math/big"

	"github.com/pkg/errors"
	"google3/third_party/golang/github_com/gomlx/gopjrt/v/v0/dtypes/bfloat16/bfloat16"
	"github.com/gx-org/backend/dtypes"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/interp/engine"
)

// Constant element provided by this package.
type Constant interface {
	engine.Constant
	cmp.Comparable
	cmp.Canonical
}

// ToUInt64 converts an element to uint64, if possible.
func ToUInt64(cst engine.Constant) (uint64, bool) {
	nb, isNumber := cst.(engine.ScalarConstant)
	if !isNumber {
		return 0, false
	}
	u64, acc := nb.Number().Float().Uint64()
	if acc != big.Exact {
		return 0, false
	}
	return u64, true
}

// ToInt64 converts an element to int64, if possible.
func ToInt64(cst engine.Constant) (int64, bool) {
	nb, isNumber := cst.(engine.ScalarConstant)
	if !isNumber {
		return 0, false
	}
	i64, acc := nb.Number().Float().Int64()
	if acc != big.Exact {
		return 0, false
	}
	return i64, true
}

// ToFloat64 converts an element to float64, if possible.
func ToFloat64(cst engine.Constant) (float64, bool) {
	nb, isNumber := cst.(engine.ScalarConstant)
	if !isNumber {
		return 0, false
	}
	f64, _ := nb.Number().Float().Float64()
	return f64, true
}

// ToBool converts a constant into a boolean value.
func ToBool(cst engine.Constant) (bool, bool) {
	bEl, isBool := cst.(engine.BoolConstant)
	if !isBool {
		return false, false
	}
	return bEl.Bool(), true
}

// ToInt converts a constant to integer, if possible.
func ToInt[T dtypes.Signed](cst engine.Constant) (T, bool) {
	val, ok := ToInt64(cst)
	return T(val), ok
}

// ToUInt converts a constant to integer, if possible.
func ToUInt[T dtypes.Unsigned](cst engine.Constant) (T, bool) {
	val, ok := ToUInt64(cst)
	return T(val), ok
}

// ToFloat32 converts a constant to a float32, if possible.
func ToFloat32(cst engine.Constant) (float32, bool) {
	val, ok := ToFloat64(cst)
	return float32(val), ok
}

// ToBFloat16 converts a constant into bfloat16.
func ToBFloat16(cst engine.Constant) (bfloat16.BFloat16, bool) {
	val, ok := ToFloat64(cst)
	return bfloat16.FromFloat64(val), ok
}

// ConvertOk converts an element to a Go value if possible.
func ConvertOk[T dtypes.Supported](cvt ConverterT[T], el ir.Element) (T, bool) {
	cstEl, ok := el.(engine.ConstantElement)
	if !ok {
		var zero T
		return zero, false
	}
	return cvt.ToGo(cstEl.Constant())
}

// Convert a constant into a Go value.
func Convert[T dtypes.Supported](cvt ConverterT[T], el ir.Element) (T, error) {
	cstEl, err := coreiface.Cast[engine.ConstantElement](el)
	if err != nil {
		var zero T
		return zero, err
	}
	return cvt.ConvertT(cstEl.Constant())
}

type (
	// Converter converts constants to Go type.
	Converter interface {
		Convert(engine.Constant) (any, error)
		ConvertOk(engine.Constant) (any, bool)
		convertSlice(total int, cstEls []engine.AtomConstant) (any, error)
	}

	// ConverterT converts constant element to Go.
	ConverterT[T dtypes.Supported] struct {
		Kind irkind.Kind
		ToGo func(engine.Constant) (T, bool)
	}
)

// ConvertT to the target type.
func (c ConverterT[T]) ConvertT(cst engine.Constant) (T, error) {
	val, ok := c.ToGo(cst)
	if !ok {
		var zero T
		return zero, errors.Errorf("cannot convert %T to %T", cst, zero)
	}
	return val, nil
}

// Convert a constant to a Go value.
// Returns false if this is not possible.
func (c ConverterT[T]) Convert(cst engine.Constant) (any, error) {
	return c.ConvertT(cst)
}

// ConvertOk a constant to a Go value.
// Returns false if this is not possible.
func (c ConverterT[T]) ConvertOk(cst engine.Constant) (any, bool) {
	return c.ToGo(cst)
}

func (c ConverterT[T]) convertSlice(total int, cstEls []engine.AtomConstant) (any, error) {
	r := make([]T, total)
	if len(cstEls) == 0 {
		return r, nil
	}
	for i, cstEl := range cstEls {
		var err error
		r[i], err = c.ConvertT(cstEl)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Converters
var (
	CBool     = ConverterT[bool]{Kind: irkind.Bool, ToGo: ToBool}
	CBFloat16 = ConverterT[bfloat16.BFloat16]{Kind: irkind.Bfloat16, ToGo: ToBFloat16}
	CFloat32  = ConverterT[float32]{Kind: irkind.Float32, ToGo: ToFloat32}
	CFloat64  = ConverterT[float64]{Kind: irkind.Float64, ToGo: ToFloat64}
	CInt      = ConverterT[int]{Kind: irkind.Int, ToGo: ToInt[int]}
	CInt32    = ConverterT[int32]{Kind: irkind.Int32, ToGo: ToInt[int32]}
	CInt64    = ConverterT[int64]{Kind: irkind.Int64, ToGo: ToInt64}
	CUint32   = ConverterT[uint32]{Kind: irkind.Uint32, ToGo: ToUInt[uint32]}
	CUint64   = ConverterT[uint64]{Kind: irkind.Uint64, ToGo: ToUInt64}
)

// NewConverter returns a new converter given a kind.
func NewConverter(kind irkind.Kind) (v Converter, err error) {
	switch kind {
	case irkind.Bool:
		return CBool, nil
	case irkind.Bfloat16:
		return CBFloat16, nil
	case irkind.Float32:
		return CFloat32, nil
	case irkind.Float64:
		return CFloat64, nil
	case irkind.Int:
		return CInt, nil
	case irkind.Int32:
		return CInt32, nil
	case irkind.Int64:
		return CInt64, nil
	case irkind.Uint32:
		return CUint32, nil
	case irkind.Uint64:
		return CUint64, nil
	default:
		err = fmterr.Internalf("%s cannot be converted to backend numerical: not supported", kind)
	}
	return
}
