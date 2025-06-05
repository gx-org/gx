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
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

// evalAtom evaluates an expression on the host and returns its result as an array.
func evalAtom[T dtype.GoDataType](ctx *context, expr ir.Expr) (val T, err error) {
	el, err := ctx.evalExpr(expr)
	if err != nil {
		var zero T
		return zero, err
	}
	return elements.ConstantScalarFromElement[T](el)
}

func evalAtomInt(ctx *context, expr ir.Expr) (int, error) {
	el, err := ctx.evalExpr(expr)
	if err != nil {
		return 0, err
	}
	return elements.ConstantIntFromElement(el)
}
