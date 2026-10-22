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

package num

import (
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

func targetShape(genArgs []ir.Element) (shapes.Shape, error) {
	axes, err := elements.AxesFromElement(genArgs[0])
	if err != nil {
		return shapes.Shape{}, err
	}
	tp, err := cast.To[ir.Type](ir.BareValue(genArgs[1]))
	if err != nil {
		return shapes.Shape{}, err
	}
	return shapes.Shape{
		DType:      tp.Kind().DType(),
		Dimensions: axes,
	}, nil
}

func evalIota(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	targetShape, err := targetShape(genArgs)
	if err != nil {
		return nil, err
	}
	axisIndex, err := elements.IntFromElement(args[0])
	if err != nil {
		return nil, err
	}
	gr := env.Engine().Graph()
	op, err := gr.Iota(targetShape, axisIndex)
	if err != nil {
		return nil, err
	}
	mat := builtin.Materialiser(env)
	return materialise.ElementFromNode(env.ExprEval(), mat, op, call.Type())
}

func evalIotaFull(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	targetShape, err := targetShape(genArgs)
	if err != nil {
		return nil, err
	}
	gr := env.Engine().Graph()
	iotaOp, err := gr.Iota(shapes.Shape{
		DType:      targetShape.DType,
		Dimensions: []int{targetShape.Size()},
	}, 0)
	if err != nil {
		return nil, err
	}
	op, err := gr.Reshape(iotaOp, targetShape.Dimensions...)
	if err != nil {
		return nil, err
	}
	mat := builtin.Materialiser(env)
	return materialise.ElementFromNode(env.ExprEval(), mat, op, call.Type())
}
