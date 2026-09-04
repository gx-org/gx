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

// Package numbers implement elements representing numbers for the interpreter.
package numbers

import (
	"go/token"
	"math/big"

	"github.com/pkg/errors"
	"github.com/gomlx/compute/dtypes/bfloat16"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/constants"
	"github.com/gx-org/gx/interp/engine"
)

// NewElement returns an engine constant from a Go value.
func NewElement(env *engine.Env, x ir.Type, val any) (engine.ConstantElement, error) {
	cst, err := NewConstant(x, val)
	if err != nil {
		return nil, err
	}
	return env.Engine().ArrayOps().ElementFromHostValue(env.ExprEval(), cst)
}

// NewConstant returns a new constant from an IR type and a Go value.
func NewConstant(typ ir.Type, val any) (engine.Constant, error) {
	var nb engine.ScalarNumber
	switch valT := val.(type) {
	case bfloat16.BFloat16:
		nb = newFloat(big.NewFloat(valT.Float64()))
	case float32:
		nb = newFloat(big.NewFloat(float64(valT)))
	case float64:
		nb = newFloat(big.NewFloat(valT))
	case int:
		nb = newInt(big.NewInt(int64(valT)))
	case int32:
		nb = newInt(big.NewInt(int64(valT)))
	case int64:
		nb = newInt(big.NewInt(valT))
	case uint32:
		nb = newInt((&big.Int{}).SetUint64(uint64(valT)))
	case uint64:
		nb = newInt((&big.Int{}).SetUint64(valT))
	default:
		return nil, errors.Errorf("cannot convert %T(%v) to %s: not supported", val, val, typ.ReferString(nil))
	}
	return constants.NewScalar(typ, nb), nil
}

var zero = newInt(&big.Int{})

// NewZero returns a new zero constant for a given type.
func NewZero(tp ir.Type) constants.Scalar {
	return constants.NewScalar(tp, zero)
}

var one = newInt(big.NewInt(1))

// One returns a constant of 1.
func One(tp ir.Type) constants.Scalar {
	return constants.NewScalar(tp, one)
}

var oneInt = constants.NewScalar(ir.IntType(), one)

// OneInt returns an integer element equals to 1.
func OneInt() constants.Scalar {
	return oneInt
}

// NewConstantFromFloat build a new scalar from a big float value.
func NewConstantFromFloat(typ ir.Type, f *big.Float) constants.Scalar {
	var nb engine.ScalarNumber
	intVal, acc := f.Int(nil)
	if acc == big.Exact {
		nb = newInt(intVal)
	} else {
		nb = newFloat(f)
	}
	return constants.NewScalar(typ, nb)
}

// NewInt returns a new int constant.
func NewInt(val int) engine.Constant {
	nb := newInt(big.NewInt(int64(val)))
	return constants.NewScalar(ir.IntType(), nb)
}

func compare(op token.Token, x, y *big.Float) engine.BoolConstant {
	var val bool
	switch op {
	case token.LSS: // <
		val = x.Cmp(y) < 0
	case token.GTR: // >
		val = x.Cmp(y) > 0
	case token.LEQ: // <=
		val = x.Cmp(y) <= 0
	case token.GEQ: // >=
		val = x.Cmp(y) >= 0
	case token.NEQ: // !=
		val = x.Cmp(y) != 0
	case token.EQL: // ==
		val = x.Cmp(y) == 0
	default:
		return nil
	}
	return constants.NewBool(val)
}
