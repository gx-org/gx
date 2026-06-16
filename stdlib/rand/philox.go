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

package rand

import (
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

var philoxStateShape = &shape.Shape{
	DType:       dtype.Uint64,
	AxisLengths: []int{3},
}

func evalPhilox(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element, dtyp dtype.DataType) ([]ir.Element, error) {
	philox, err := fun.ToNamedType(recv)
	if err != nil {
		return nil, err
	}
	mat := builtin.Materialiser(env)
	philoxStruct := ir.Underlying(philox.Type()).(*ir.StructType)
	stateArray := philoxStruct.Fields.FindField("state")
	field, err := philox.Select(&ir.SelectorExpr{
		X:    call,
		Stor: stateArray.Storage(),
	})
	if err != nil {
		return nil, err
	}
	eng := env.Engine()
	stateNode, _, err := materialise.Element(mat, field)
	if err != nil {
		return nil, err
	}
	dimensions, err := elements.AxesFromElement(args[0])
	if err != nil {
		return nil, err
	}
	bckGraph := eng.ArrayOps().Graph()
	targetShape := &shape.Shape{DType: dtyp, AxisLengths: dimensions}
	newState, values, err := bckGraph.Random().RngBitGenerator(stateNode, targetShape)
	if err != nil {
		return nil, err
	}

	philoxStateElement, err := materialise.ElementFromNode(env.File(), mat,
		&ops.OutputNode{
			Node:  newState,
			Shape: philoxStateShape,
		},
		stateArray.Storage().Type())
	if err != nil {
		return nil, err
	}
	valuesElement, err := materialise.ElementFromNode(
		env.File(), mat,
		&ops.OutputNode{
			Node:  values,
			Shape: targetShape,
		},
		call.ExprFromResult(1).Type(),
	)
	if err != nil {
		return nil, err
	}
	return []ir.Element{
		fun.NewNamedType(interp.NewRunFunc, philox.TypeMethods(), elements.NewStruct(
			philoxStruct,
			map[string]ir.Element{"state": philoxStateElement[0]},
		)),
		valuesElement[0],
	}, nil
}

func evalPhiloxUint32(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	return evalPhilox(env, call, recv, args, dtype.Uint32)
}

func evalPhiloxUint64(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	return evalPhilox(env, call, recv, args, dtype.Uint64)
}
