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
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

func evalIota(ctx engine.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	axes, err := elements.AxesFromElement(args[0])
	if err != nil {
		return nil, err
	}
	axisIndex, err := elements.ConstantIntFromElement(args[1])
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       irkind.DefaultInt.DType(),
		AxisLengths: axes,
	}
	ev := ctx.Evaluator().(*grapheval.Evaluator)
	gr := ev.ArrayOps().Graph()
	op, err := gr.Num().Iota(targetShape, axisIndex)
	if err != nil {
		return nil, err
	}
	mat := builtin.Materialiser(ctx)
	return materialise.ElementFromNode(call.File(), mat, &ops.OutputNode{
		Node:  op,
		Shape: targetShape,
	}, call.Node().Type())
}

func evalIotaFull(ctx engine.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	axes, err := elements.AxesFromElement(args[0])
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       irkind.DefaultInt.DType(),
		AxisLengths: axes,
	}
	ev := ctx.Evaluator().(*grapheval.Evaluator)
	gr := ev.ArrayOps().Graph()
	iotaOp, err := gr.Num().Iota(&shape.Shape{
		DType:       irkind.DefaultInt.DType(),
		AxisLengths: []int{targetShape.Size()},
	}, 0)
	if err != nil {
		return nil, err
	}
	op, err := gr.Core().Reshape(iotaOp, targetShape.AxisLengths)
	if err != nil {
		return nil, err
	}
	mat := builtin.Materialiser(ctx)
	return materialise.ElementFromNode(call.File(), mat, &ops.OutputNode{
		Node:  op,
		Shape: targetShape,
	}, call.Node().Type())
}
