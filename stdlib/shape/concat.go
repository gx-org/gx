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

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
	gxerrors "github.com/gx-org/gx/stdlib/errors"
)

func checkConsistent(env engine.Env, call *ir.FuncCallExpr, ref ir.Element, axisIdx int, refShape []engine.NumericalElement, argNum int, el ir.Element) (engine.NumericalElement, ir.Element, error) {
	argShape, err := elements.Map(elements.ToNumericalElement, el)
	if err != nil {
		return nil, nil, err
	}
	if len(argShape) != len(refShape) {
		gxErr, err := gxerrors.Errorf(env, "argument %d has %d axe(s) but argument 1 has %d axe(s)", argNum, len(argShape), len(refShape))
		return nil, gxErr, err
	}
	var concatAxis engine.NumericalElement
	for i, axLen := range argShape {
		if i == axisIdx {
			concatAxis = axLen
			continue
		}
		eq, err := ir.ElementEqual(refShape[i], axLen)
		if err != nil {
			return nil, nil, err
		}
		if !eq {
			refShapeS := ir.ExprString(env.ExprEval(), call.Expr(), ref)
			iRefS := ir.ExprString(env.ExprEval(), call.Expr(), refShape[i])
			argShapeS := ir.ExprString(env.ExprEval(), call.Expr(), el)
			iArgS := ir.ExprString(env.ExprEval(), call.Expr(), axLen)
			gxErr, err := gxerrors.Errorf(env, "argument %d shape (%s) incompatible with argument 1 shape (%s): [%s] != [%s] for axis %d", argNum, argShapeS, refShapeS, iArgS, iRefS, axisIdx)
			return nil, gxErr, err
		}
	}
	return concatAxis, nil, nil
}

func irAdd(env engine.Env, src ast.Expr, x, y engine.NumericalElement) (engine.NumericalElement, error) {
	xExpr, err := ir.ToSingleExpr(env.ExprEval(), src, x)
	if err != nil {
		return nil, err
	}
	yExpr, err := ir.ToSingleExpr(env.ExprEval(), src, y)
	if err != nil {
		return nil, err
	}
	return x.BinaryOp(
		env,
		&ir.BinaryExpr{
			Src: &ast.BinaryExpr{
				Op: token.ADD,
				X:  xExpr.Expr(),
				Y:  yExpr.Expr(),
			},
			X:   xExpr,
			Y:   yExpr,
			Typ: ir.IntType(),
		},
		x, y,
	)
}

func concatAxis(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) (_ []ir.Element, err error) {
	idx, err := elements.ConstantIntFromElement(args[0])
	if err != nil {
		return nil, err
	}
	varargsSlice, err := elements.ToWithElements(args[1])
	if err != nil {
		return nil, err
	}
	varargsElts := varargsSlice.Elements()
	if len(varargsElts) == 0 {
		gxErr, err := gxerrors.Errorf(env, "not enough arguments for concat() (expected 1, found 0)")
		return []ir.Element{nil, gxErr}, err
	}
	firstShapeElts, err := elements.SliceFromElement(varargsElts[0])
	if err != nil {
		return nil, err
	}
	if idx >= firstShapeElts.Len() {
		gxErr, err := outOfBoundAxis(env, call, firstShapeElts, idx, "concat")
		return []ir.Element{args[0], gxErr}, err
	}
	firstShape, err := elements.Map(elements.ToNumericalElement, firstShapeElts)
	if err != nil {
		return nil, err
	}
	concatAxis := firstShape[idx]
	for i, el := range varargsElts[1:] {
		toAdd, gxErr, err := checkConsistent(env, call, firstShapeElts, idx, firstShape, i+2, el)
		if err != nil {
			return nil, err
		}
		if gxErr != nil {
			return []ir.Element{args[0], gxErr}, err
		}
		concatAxis, err = irAdd(env, call.Expr(), concatAxis, toAdd)
		if err != nil {
			return nil, err
		}
	}
	// Final shape.
	final := append([]ir.Element{}, firstShapeElts.Elements()...)
	final[idx] = concatAxis
	finalShape, err := elements.NewSlice(ir.IntSliceType(), final)
	if err != nil {
		return nil, err
	}
	return []ir.Element{finalShape, elements.NilError()}, nil
}

func evalConcat(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	axis, err := elements.ConstantIntFromElement(args[0])
	if err != nil {
		return nil, err
	}
	arrays, err := elements.SliceFromElement(args[3])
	if err != nil {
		return nil, err
	}
	mat := builtin.Materialiser(env)
	nodes, firstShape, err := materialise.All(mat, arrays.Elements())
	if err != nil {
		return nil, err
	}
	op, err := env.Engine().ArrayOps().Graph().Shape().Concat(axis, nodes)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
		Node: op,
		Shape: &shape.Shape{
			DType:       firstShape.DType,
			AxisLengths: op.(interface{ PJRTDims() []int }).PJRTDims(),
		},
	}, call.Type())
}
