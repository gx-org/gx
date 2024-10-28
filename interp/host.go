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

package interp

import (
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/interp/state"
)

// evalExprOnHost evaluates an expression on the host.
func evalExprOnHost(ctx *context, expr ir.Expr) (*values.HostArray, error) {
	el, err := evalExpr(ctx, expr)
	if err != nil {
		return nil, err
	}
	// First, check if the value is a static value.
	value := state.ConstantFromElement(el)
	if value != nil {
		return value, nil
	}
	// If not, check if it is an argument.
	value, err = state.HostValueFromContext(ctx, el)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func evalScalarCastOnHost(ctx *context, expr *ir.CastExpr, val *values.HostArray, target ir.Kind) (state.Element, error) {
	nativeHandle := val.Handle().(kernels.Handle).KernelValue()
	kernelFactory := nativeHandle.Factory()
	castOp, _, _, err := kernelFactory.Cast(target.DType(), nil)
	if err != nil {
		return nil, err
	}
	return evalLocalKernel(ctx, expr, castOp, nativeHandle, target)
}

func evalLocalKernel(ctx *context, expr *ir.CastExpr, kernel kernels.Unary, handle kernels.Array, target ir.Kind) (state.Element, error) {
	outArray, err := kernel(handle)
	if err != nil {
		return nil, err
	}
	valuer, err := newValuer(ctx, expr, target)
	if err != nil {
		return nil, err
	}
	return valuer.toLiteral(ctx, expr.Type(), outArray)
}
