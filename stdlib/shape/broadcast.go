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

package shape

import (
	"go/ast"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

type broadcast struct {
	builtin.Func
}

func (f broadcast) BuildFuncIR(pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[broadcast]("Broadcast", evalBroadcast, pkg), nil
}

var oneAxisLength = numbers.OneInt()

func checkBroadcastRanks(tpcmp ir.TypeCmp, call *ir.FuncCallExpr, src ir.ArrayRank, target ir.ArrayRank, targetElmts []cmp.Canonical) error {
	if src == nil || target == nil {
		return nil
	}
	srcAxes := src.Axes()
	if len(srcAxes) != len(targetElmts) {
		return fmterr.Errorf(tpcmp.File().FileSet(), call.Node(), "cannot broadcast array from %d to %d axes (expect an equal number)", len(srcAxes), len(targetElmts))
	}

	for i, targetElt := range targetElmts {
		srcElt, err := tpcmp.EvalExpr(srcAxes[i].AsExpr())
		if err != nil {
			return err
		}
		srcCan, ok := srcElt.(cmp.Canonical)
		if !ok {
			return fmterr.InternalAt(tpcmp.File().FileSet(), call.Node(), "expression evaluation axis %d=%s did not return a canonical expression", i, srcAxes[i])
		}
		tgOk, err := tpcmp.Compare(targetElt, srcCan)
		if err != nil {
			return fmterr.Internal(err)
		}
		oneAxisOk, err := tpcmp.Compare(oneAxisLength, srcCan)
		if err != nil {
			return fmterr.Internal(err)
		}
		if !tgOk && !oneAxisOk {
			from := tpcmp.File()
			return fmterr.Errorf(from.FileSet(), srcAxes[i].Node(), "cannot broadcast axis %d from length %s to %s (expect a source length of 1 or %s)", i, srcCan.ShortString(from), targetElt.ShortString(from), targetElt.ShortString(from))
		}
	}
	return nil
}

func (f broadcast) BuildFuncType(tpcmp ir.TypeCmp, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	params, err := builtins.BuildFuncParams(tpcmp, call, f.Name(), []ir.Type{
		builtins.GenericArrayType,
		ir.IntSliceType(),
	})
	if err != nil {
		return nil, err
	}
	arrayType, err := builtins.NarrowType[ir.ArrayType](tpcmp, call, call.Args[0].Type())
	if err != nil {
		return nil, err
	}
	targetRank, targetElmts, err := elements.EvalRank(tpcmp, call.Args[1])
	if err != nil {
		return nil, err
	}
	if err := checkBroadcastRanks(tpcmp, call, arrayType.Rank(), targetRank, targetElmts); err != nil {
		return nil, err
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Node().Pos()}},
		Params:   builtins.Fields(call, params...),
		Results:  builtins.Fields(call, ir.NewArrayType(&ast.ArrayType{}, arrayType.DataType(), targetRank)),
	}, nil
}

func evalBroadcast(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	targetAxes, err := elements.AxesFromElement(args[0])
	if err != nil {
		return nil, err
	}
	broadcastAxes := make([]int, len(targetAxes))
	for i := range targetAxes {
		broadcastAxes[i] = i
	}
	mat := builtin.Materialiser(env)
	x, xShape, err := materialise.Element(mat, args[3])
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       xShape.DType,
		AxisLengths: targetAxes,
	}
	op, err := env.Engine().ArrayOps().Graph().Core().BroadcastInDim(x, targetShape, broadcastAxes)
	if err != nil {
		return nil, err
	}
	tp, err := concrete.Concrete(env.ExprEval(), call.Type())
	if err != nil {
		return nil, fmterr.Error(env.File().FileSet(), call.Src, err)
	}
	return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
		Node:  op,
		Shape: targetShape,
	}, tp)
}
