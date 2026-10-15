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
	"github.com/gomlx/compute"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

// Package description of the GX num package.
var Package = builtin.PackageBuilder{
	FullPath: "num",
	Builders: []builtin.Builder{
		builtin.ParseSource("num.gx"),
		builtin.ImplementBuiltin("ArgMax", evalArgMax),
		builtin.ImplementBuiltin("Iota", evalIota),
		builtin.ImplementBuiltin("IotaFull", evalIotaFull),
		builtin.ImplementBuiltin("ReduceMax", evalReduceMax),
		builtin.ImplementBuiltin("MatMul", evalMatMul),
		builtin.ImplementBuiltin("MatVecMul", evalMatVecMul),
		builtin.ImplementBuiltin("VecMatMul", evalVecMatMul),
		builtin.ImplementBuiltin("VecMul", evalVecMul),
		builtin.ImplementBuiltin("Sum", evalReduceSum),
		builtin.ImplementBuiltin("einsumAxes", evalEinsumAxes),
		builtin.ImplementBuiltin("Einsum", evalEinsum),
	},
}

func evalVecMul(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	return evalDot(env, call, recv, args[0], args[1])
}

func evalVecMatMul(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	return evalDot(env, call, recv, args[0], args[1])
}

func evalMatVecMul(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	return evalDot(env, call, recv, args[0], args[1])
}

func evalMatMul(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	return evalDot(env, call, recv, args[0], args[1])
}

func evalDot(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, x ir.Element, y ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	xNode, xShape, err := materialise.ElementWithShape(mat, x)
	if err != nil {
		return nil, err
	}
	yNode, err := materialise.Element(mat, y)
	if err != nil {
		return nil, err
	}
	gr := env.Engine().Graph()
	op, err := gr.DotGeneral(xNode, []int{len(xShape.Dimensions) - 1}, nil, yNode, []int{0}, nil, compute.DotGeneralConfig{})
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.ExprEval(), mat, op, call.Type())
}

func evalArgMax(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	argNode, err := materialise.Element(mat, args[0])
	if err != nil {
		return nil, err
	}
	axisIndex, err := elements.IntFromElement(genArgs[0])
	if err != nil {
		return nil, err
	}
	gr := env.Engine().Graph()
	op, err := gr.ArgMinMax(argNode, int(axisIndex), irkind.Int.DType(), false)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.ExprEval(), mat, op, call.Type())
}

func evalReduce(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element, reduce func(x compute.Value, axes ...int) (compute.Value, error)) ([]ir.Element, error) {
	axisIndices, err := elements.Map(elements.IntFromElement, genArgs[0])
	if err != nil {
		return nil, err
	}
	if len(axisIndices) == 0 {
		return []ir.Element{args[0]}, nil
	}
	mat := builtin.Materialiser(env)
	x, err := materialise.Element(mat, args[0])
	if err != nil {
		return nil, err
	}
	op, err := reduce(x, axisIndices...)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.ExprEval(), mat, op, call.Type())
}

func evalReduceSum(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	return evalReduce(env, call, recv, genArgs, args, env.Engine().Graph().ReduceSum)
}

func evalReduceMax(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	return evalReduce(env, call, recv, genArgs, args, env.Engine().Graph().ReduceMax)
}
