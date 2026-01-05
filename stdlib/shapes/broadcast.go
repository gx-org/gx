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

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/interp/numbers"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type broadcast struct {
	builtin.Func
}

func (f broadcast) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[broadcast]("Broadcast", evalBroadcast, pkg), nil
}

var oneAxisLength = numbers.NewInt(elements.NewExprAt(nil, &ir.NumberInt{}), big.NewInt(1))

func checkBroadcastRanks(fetcher ir.Fetcher, call *ir.FuncCallExpr, src ir.ArrayRank, target ir.ArrayRank, targetElmts []canonical.Canonical) error {
	if src == nil || target == nil {
		return nil
	}
	srcAxes := src.Axes()
	if len(srcAxes) != len(targetElmts) {
		return fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "cannot broadcast array with %d axes to %d axes: the same number of axes is required", len(srcAxes), len(targetElmts))
	}

	for i, targetElt := range targetElmts {
		srcElt, err := fetcher.EvalExpr(srcAxes[i].AsExpr())
		if err != nil {
			return err
		}
		srcCan, ok := srcElt.(canonical.Canonical)
		if !ok {
			return fmterr.Internalf(fetcher.File().FileSet(), call.Node(), "expression evaluation axis %d=%s did not return a canonical expression", i, srcAxes[i])
		}
		tgOk, err := targetElt.Compare(srcCan)
		if err != nil {
			return fmterr.Internal(err)
		}
		oneAxisOk, err := oneAxisLength.Compare(srcCan)
		if err != nil {
			return fmterr.Internal(err)
		}
		if !tgOk && !oneAxisOk {
			return fmterr.Errorf(fetcher.File().FileSet(), srcAxes[i].Node(), "cannot broadcast array with axis %d of size %d: size of 1 or %d required", i, srcElt, targetElt)
		}
	}
	return nil
}

func (f broadcast) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
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
	targetRank, targetElmts, err := elements.EvalRank(fetcher, call.Args[1])
	if err != nil {
		return nil, err
	}
	if err := checkBroadcastRanks(fetcher, call, arrayType.Rank(), targetRank, targetElmts); err != nil {
		return nil, err
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Node().Pos()}},
		Params:   builtins.Fields(call, params...),
		Results:  builtins.Fields(call, ir.NewArrayType(&ast.ArrayType{}, arrayType.DataType(), targetRank)),
	}, nil
}

func evalBroadcast(env evaluator.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	targetAxes, err := elements.AxesFromElement(args[1])
	if err != nil {
		return nil, err
	}
	broadcastAxes := make([]int, len(targetAxes))
	for i := range targetAxes {
		broadcastAxes[i] = i
	}
	mat := builtin.Materialiser(env)
	x, xShape, err := materialise.Element(mat, args[0])
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       xShape.DType,
		AxisLengths: targetAxes,
	}
	op, err := env.Evaluator().ArrayOps().Graph().Core().BroadcastInDim(x, targetShape, broadcastAxes)
	if err != nil {
		return nil, err
	}
	return mat.ElementsFromNodes(call.File(), call.Node(), &ops.OutputNode{
		Node:  op,
		Shape: targetShape,
	})
}
