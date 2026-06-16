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
	"go/token"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

func splitAxis(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) (_ []ir.Element, err error) {
	// Fetch the axis index.
	idx, err := elements.ConstantIntFromElement(args[0])
	if err != nil {
		return nil, err
	}
	// Fetch the number of splits.
	numSplit, err := elements.ToNumericalElement(args[1])
	if err != nil {
		return nil, err
	}
	// Fetch the shape of the array.
	axes, isSlice := args[2].(*elements.Slice)
	if !isSlice {
		return nil, errors.Errorf("cannot convert %T to %s", args[0], reflect.TypeFor[*elements.Slice]().String())
	}
	if idx >= axes.Len() {
		gxErr, err := outOfBoundAxis(env, call, axes, idx, "split")
		return []ir.Element{axes, gxErr}, err
	}
	axLens := append([]ir.Element{}, axes.Elements()...)
	splitAxis, err := elements.ToNumericalElement(axLens[idx])
	if err != nil {
		return nil, err
	}
	numSplitExpr, err := ir.ToSingleExpr(env.ExprEval(), call.Src, numSplit)
	if err != nil {
		return nil, err
	}
	splitAxisExpr, err := ir.ToSingleExpr(env.ExprEval(), call.Src, splitAxis)
	if err != nil {
		return nil, err
	}
	axLens[idx], err = splitAxis.BinaryOp(env, &ir.BinaryExpr{
		Src: &ast.BinaryExpr{
			Op: token.QUO,
		},
		X:   splitAxisExpr,
		Y:   numSplitExpr,
		Typ: ir.IntLenType(),
	}, splitAxis, numSplit)
	if err != nil {
		return nil, err
	}
	axLens = append([]ir.Element{numSplit}, axLens...)
	out, err := elements.NewSlice(axes.Type(), axLens)
	return []ir.Element{out, elements.NilError()}, err
}

func evalSplit(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	node, firstArgShape, err := materialise.Element(mat, args[4])
	if err != nil {
		return nil, err
	}
	axis, err := elements.ConstantScalarFromElement[ir.Int](args[0])
	if err != nil {
		return nil, err
	}
	numSplits, err := elements.ConstantScalarFromElement[ir.Int](args[1])
	if err != nil {
		return nil, err
	}
	op, err := env.Engine().ArrayOps().Graph().Shape().Split(node, int(axis), int(numSplits))
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
		Node: op,
		Shape: &shape.Shape{
			DType:       firstArgShape.DType,
			AxisLengths: op.(interface{ PJRTDims() []int }).PJRTDims(),
		},
	}, call.Type())
}
