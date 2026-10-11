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
	"go/token"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/src/gx/ast"
	"github.com/gx-org/gx/stdlib/builtin"
)

func splitAxis(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) (_ []ir.Element, err error) {
	// Fetch the axis index.
	idx, err := elements.IntFromElement(args[0])
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
		err := outOfBoundAxis(env, call, axes, idx, "split")
		return []ir.Element{axes}, err
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
			X:  splitAxisExpr.Expr(),
			Y:  numSplitExpr.Expr(),
		},
		X:   splitAxisExpr,
		Y:   numSplitExpr,
		Typ: ir.IntType(),
	}, numSplit)
	if err != nil {
		return nil, err
	}
	axLens = append([]ir.Element{numSplit}, axLens...)
	out, err := elements.NewSlice(axes.Type(), axLens)
	return []ir.Element{out}, err
}

func evalSplit(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	node, firstArgShape, err := materialise.Element(mat, args[0])
	if err != nil {
		return nil, err
	}
	axis, err := elements.IntFromElement(genArgs[0])
	if err != nil {
		return nil, err
	}
	numSplits, err := elements.IntFromElement(genArgs[1])
	if err != nil {
		return nil, err
	}
	rank := len(firstArgShape.Dimensions)
	if axis < 0 || axis >= rank {
		return nil, errors.Errorf("axis %d is out of bounds for rank %d", axis, rank)
	}
	if firstArgShape.Dimensions[axis]%numSplits != 0 {
		return nil, errors.Errorf("axis %d has size %d which is not divisible by %d numSplits", axis, firstArgShape.Dimensions[axis], numSplits)
	}
	gr := env.Engine().Graph()
	stride := firstArgShape.Dimensions[axis] / numSplits
	slicedNodes := make([]compute.Value, numSplits)
	for i := range numSplits {
		starts := make([]int, rank)
		limits := make([]int, rank)
		strides := make([]int, rank)
		for ax, axisSize := range firstArgShape.Dimensions {
			limits[ax] = axisSize
			strides[ax] = 1
		}
		starts[axis] = i * stride
		limits[axis] = i*stride + stride
		slicedNode, err := gr.Slice(node, starts, limits, strides)
		if err != nil {
			return nil, err
		}
		slicedNodes[i] = slicedNode
	}
	outputDims := append([]int{1}, firstArgShape.Dimensions...)
	outputDims[axis+1] = stride
	reshapedNodes := make([]compute.Value, numSplits)
	for i := range slicedNodes {
		reshapedNode, err := gr.Reshape(slicedNodes[i], outputDims...)
		if err != nil {
			return nil, err
		}
		reshapedNodes[i] = reshapedNode
	}
	op, err := gr.Concatenate(0, reshapedNodes...)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.ExprEval(), mat, op, call.Type())
}
