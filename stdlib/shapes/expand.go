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

package shapes

import (
	"go/ast"
	"math/big"

	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp/numbers"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type expand struct {
	builtin.Func
}

func (f expand) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[expand]("Expand", evalExpand, pkg), nil
}

var oneAxisLength = numbers.NewInt(elements.NewExprAt(nil, &ir.NumberInt{}), big.NewInt(1))

func checkExpandRanks(fetcher ir.Fetcher, call *ir.CallExpr, src ir.ArrayRank, target ir.ArrayRank, targetElmts []canonical.Canonical) error {
	if src == nil || target == nil {
		return nil
	}
	srcAxes := src.Axes()
	if len(srcAxes) != len(targetElmts) {
		return fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "cannot expand array with %d axes to %d axes: the same number of axes is required", len(srcAxes), len(targetElmts))
	}

	for i, targetElt := range targetElmts {
		srcElt, err := fetcher.Eval(srcAxes[i])
		if err != nil {
			return err
		}
		if !srcElt.Compare(targetElt) && !oneAxisLength.Compare(srcElt) {
			return fmterr.Errorf(fetcher.File().FileSet(), srcAxes[i].Source(), "cannot expand array with axis %d of size %d: size of 1 or %d required", i, srcElt, targetElt)
		}
	}
	return nil
}

func (f expand) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		builtins.GenericArrayType,
		ir.IntLenSliceType(),
	})
	if err != nil {
		return nil, err
	}
	arrayType, err := builtins.NarrowType[ir.ArrayType](fetcher, call, call.Args[0].Type())
	if err != nil {
		return nil, err
	}
	targetRank, targetElmts, err := compeval.EvalRank(fetcher, call.Args[1])
	if err != nil {
		return nil, err
	}
	if err := checkExpandRanks(fetcher, call, arrayType.Rank(), targetRank, targetElmts); err != nil {
		return nil, err
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(params...),
		Results:  builtins.Fields(ir.NewArrayType(nil, arrayType.DataType(), targetRank)),
	}, nil
}

func evalExpand(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element) ([]elements.Element, error) {
	targetAxes, err := elements.AxesFromElement(args[1])
	if err != nil {
		return nil, err
	}
	expandAxes := make([]int, len(targetAxes))
	for i := range targetAxes {
		expandAxes[i] = i
	}
	ao := ctx.Evaluation().Evaluator().ArrayOps()
	x, xShape, err := grapheval.NodeFromElement(ao, args[0])
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       xShape.DType,
		AxisLengths: targetAxes,
	}
	op, err := ao.Graph().Core().NewBroadcastInDim(x, targetShape, expandAxes)
	if err != nil {
		return nil, err
	}
	return grapheval.ElementsFromNode(call.ToExprAt(), &graph.OutputNode{
		Node:  op,
		Shape: targetShape,
	})
}
