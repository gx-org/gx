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
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type iotaWithAxis struct {
	builtin.Func
}

func (f iotaWithAxis) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[iotaWithAxis]("Iota", impl.Num.Iota, pkg), nil
}

func (f iotaWithAxis) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		ir.IntLenSliceType(),
		ir.IntIndexType(),
	})
	if err != nil {
		return nil, err
	}
	rank, _, err := compeval.EvalRank(fetcher, call.Args[0])
	if err != nil {
		return nil, err
	}
	return &ir.FuncType{
		Params:  builtins.Fields(call, params...),
		Results: builtins.Fields(call, ir.NewArrayType(nil, ir.DefaultIntType, rank)),
	}, nil
}

func evalIotaFull(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element) ([]elements.Element, error) {
	axes, err := elements.AxesFromElement(args[0])
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       ir.DefaultIntKind.DType(),
		AxisLengths: axes,
	}
	gr := ctx.Evaluation().Evaluator().ArrayOps().Graph()
	iotaOp, err := gr.Num().Iota(&shape.Shape{
		DType:       ir.DefaultIntKind.DType(),
		AxisLengths: []int{targetShape.Size()},
	}, 0)
	if err != nil {
		return nil, err
	}
	op, err := gr.Core().Reshape(iotaOp, targetShape.AxisLengths)
	if err != nil {
		return nil, err
	}
	return grapheval.ElementsFromNode(call.ToExprAt(), &ops.OutputNode{
		Node:  op,
		Shape: targetShape,
	})
}
