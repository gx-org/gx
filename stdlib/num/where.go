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

package num

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/internal/undef"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

func evalWhereShape(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	var shape ir.Element
	var shapeSlice *elements.Slice
	hasUnresolved := false
	for _, arg := range args {
		under, err := coreiface.Underlying(arg)
		if err != nil {
			return nil, err
		}
		sl, isSlice := under.(*elements.Slice)
		if isSlice && sl.Len() == 0 {
			continue
		}
		if shape == nil {
			shape = arg
			shapeSlice = sl
			continue
		}
		same, err := cmp.Equal(env.ExprEval(), shape, arg)
		if err != nil {
			return nil, err
		}
		if same {
			continue
		}
		if !isSlice || shapeSlice == nil {
			hasUnresolved = true
			if isSlice {
				shape = arg
				shapeSlice = sl
			}
			continue
		}
		shapeStr := ir.ExprString(env.ExprEval(), call.Expr(), shapeSlice)
		argStr := ir.ExprString(env.ExprEval(), call.Expr(), sl)
		return nil, ir.CompileErrorF("cannot use array of shape %s as scalar or array of shape %s (expect same shape)", argStr, shapeStr)
	}
	if hasUnresolved {
		return nil, undef.Err()
	}
	if shape == nil {
		return []ir.Element{args[0]}, nil
	}
	return []ir.Element{shape}, nil
}

func evalWhere(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	cond, err := materialise.Element(mat, args[0])
	if err != nil {
		return nil, err
	}
	onTrue, err := materialise.Element(mat, args[1])
	if err != nil {
		return nil, err
	}
	onFalse, err := materialise.Element(mat, args[2])
	if err != nil {
		return nil, err
	}
	op, err := env.Engine().Graph().Where(cond, onTrue, onFalse)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.ExprEval(), mat, op, call.Type())
}
