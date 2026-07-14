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

// Package num provides the functions in the num GX standard library.
package num

import (
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
	gxerrors "github.com/gx-org/gx/stdlib/errors"
)

// Package description of the GX num package.
var Package = builtin.PackageBuilder{
	FullPath: "num",
	Builders: []builtin.Builder{
		builtin.ParseSource("num.gx"),
		builtin.BuildFunc(einsum{}),
		builtin.ImplementBuiltin("ArgMax", evalArgMax),
		builtin.ImplementBuiltin("Iota", evalIota),
		builtin.ImplementBuiltin("IotaFull", evalIotaFull),
		builtin.ImplementBuiltin("ReduceMax", evalReduceMax),
		builtin.ImplementBuiltin("MatMul", evalMatMul),
		builtin.ImplementBuiltin("Sum", evalReduceSum),
		builtin.ImplementBuiltin("MatMulAxes", evalMatMulAxes),
	},
}

func vecVecMatMul(env engine.Env, left, right ir.Element) ([]ir.Element, error) {
	eq, err := ir.ElementEqual(left, right)
	if err != nil {
		return nil, err
	}
	if !eq {
		from := env.File()
		gxErr, err := gxerrors.Errorf(env, "incompatible axis lengths: %s!=%s", ir.ShortString(from, left), ir.ShortString(from, right))
		return []ir.Element{builtin.NilShape, gxErr}, err
	}
	return builtin.ToShapeResult()
}

func vecMatMatMul(env engine.Env, left ir.Element, right []ir.Element) ([]ir.Element, error) {
	eq, err := ir.ElementEqual(left, right[0])
	if err != nil {
		return nil, err
	}
	if !eq {
		from := env.File()
		gxErr, err := gxerrors.Errorf(env, "incompatible axis lengths: %s!=%s in %s,%s", ir.ShortString(from, left), ir.ShortString(from, right[0]), builtin.ToShapeString([]ir.Element{left}), builtin.ToShapeString(right))
		return []ir.Element{builtin.NilShape, gxErr}, err
	}
	return builtin.ToShapeResult(right[1])
}

func matVecMatMul(env engine.Env, left []ir.Element, right ir.Element) ([]ir.Element, error) {
	eq, err := ir.ElementEqual(left[1], right)
	if err != nil {
		return nil, err
	}
	if !eq {
		from := env.File()
		gxErr, err := gxerrors.Errorf(env, "incompatible axis lengths: %s!=%s in %s,%s", ir.ShortString(from, left[0]), ir.ShortString(from, right), builtin.ToShapeString(left), builtin.ToShapeString([]ir.Element{right}))
		return []ir.Element{builtin.NilShape, gxErr}, err
	}
	return builtin.ToShapeResult(left[0])
}

func evalMatMulAxes(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	leftSlice, err := elements.SliceFromElement(args[0])
	if err != nil {
		return nil, err
	}
	rightSlice, err := elements.SliceFromElement(args[1])
	if err != nil {
		return nil, err
	}
	left, right := leftSlice.Elements(), rightSlice.Elements()
	if len(left) > 2 {
		gxErr, err := gxerrors.Errorf(env, "expects no more than 2 axes but got %d axes (%s)", len(left), builtin.ToShapeString(left))
		return []ir.Element{builtin.NilShape, gxErr}, err
	}
	if len(right) > 2 {
		gxErr, err := gxerrors.Errorf(env, "expects no more than 2 axes but got %d axes (%s)", len(right), builtin.ToShapeString(right))
		return []ir.Element{builtin.NilShape, gxErr}, err
	}
	if len(left) == 1 && len(right) == 1 {
		return vecVecMatMul(env, left[0], right[0])
	}
	if len(left) == 1 {
		return vecMatMatMul(env, left[0], right)
	}
	if len(right) == 1 {
		return matVecMatMul(env, left, right[0])
	}
	eq, err := ir.ElementEqual(left[1], right[0])
	if err != nil {
		return nil, err
	}
	if !eq {
		gxErr, err := gxerrors.Errorf(env, "invalid axis length: %s!=%s in %s,%s", left[1], right[0], builtin.ToShapeString(left), builtin.ToShapeString(right))
		if err != nil {
			return nil, err
		}
		shape, err := elements.NewSlice(ir.IntSliceType(), []ir.Element{left[0], right[1]})
		if err != nil {
			return nil, err
		}
		return []ir.Element{shape, gxErr}, nil
	}
	return builtin.ToShapeResult(left[0], right[1])
}

func evalMatMul(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	xNode, xShape, err := materialise.Element(mat, args[3])
	if err != nil {
		return nil, err
	}
	yNode, _, err := materialise.Element(mat, args[4])
	if err != nil {
		return nil, err
	}
	gr := env.Engine().ArrayOps().Graph()
	op, err := gr.Num().Dot(xNode, yNode)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
		Node: op,
		Shape: &shape.Shape{
			DType:       xShape.DType,
			AxisLengths: op.(interface{ PJRTDims() []int }).PJRTDims(),
		},
	}, call.Type())
}

func evalArgMax(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	argNode, _, err := materialise.Element(mat, args[3])
	if err != nil {
		return nil, err
	}
	axisIndex, err := elements.ConstantScalarFromElement[ir.Int](args[0])
	if err != nil {
		return nil, err
	}
	gr := env.Engine().ArrayOps().Graph()
	op, err := gr.Num().ArgMinMax(argNode, int(axisIndex), irkind.Int.DType(), false)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
		Node: op,
		Shape: &shape.Shape{
			DType:       irkind.Int.DType(),
			AxisLengths: op.(interface{ PJRTDims() []int }).PJRTDims(),
		},
	}, call.Type())
}

func evalReduce(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element, reduce func(x ops.Node, axes []int) (ops.Node, error)) ([]ir.Element, error) {
	axisIndices, err := elements.Map(elements.ConstantIntFromElement, args[0])
	if err != nil {
		return nil, err
	}
	if len(axisIndices) == 0 {
		return []ir.Element{args[3]}, nil
	}
	mat := builtin.Materialiser(env)
	x, xShape, err := materialise.Element(mat, args[3])
	if err != nil {
		return nil, err
	}
	op, err := reduce(x, axisIndices)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
		Node: op,
		Shape: &shape.Shape{
			DType:       xShape.DType,
			AxisLengths: op.(interface{ PJRTDims() []int }).PJRTDims(),
		},
	}, call.Type())
}

func evalReduceSum(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	return evalReduce(env, call, recv, args, env.Engine().ArrayOps().Graph().Num().ReduceSum)
}

func evalReduceMax(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	return evalReduce(env, call, recv, args, env.Engine().ArrayOps().Graph().Num().ReduceMax)
}
