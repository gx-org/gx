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
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

func evalPhilox(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element, dtyp dtypes.DType) ([]ir.Element, error) {
	philox, err := elements.ToNamedType(recv)
	if err != nil {
		return nil, err
	}
	mat := builtin.Materialiser(env)
	philoxStruct := ir.Underlying(philox.Type()).(*ir.StructType)
	stateArray := philoxStruct.Fields.FindField("state")
	field, err := philox.Select(env, &ir.SelectorExpr{
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
	dimensions, err := elements.AxesFromElement(genArgs[0])
	if err != nil {
		return nil, err
	}
	bckGraph := eng.Graph()
	targetShape := shapes.Shape{DType: dtyp, Dimensions: dimensions}
	newState, values, err := bckGraph.RNGBitGenerator(stateNode, targetShape)
	if err != nil {
		return nil, err
	}

	philoxStateElement, err := materialise.ElementFromNode(env.ExprEval(), mat, newState, stateArray.Storage().Type())
	if err != nil {
		return nil, err
	}
	valuesElement, err := materialise.ElementFromNode(env.ExprEval(), mat, values, call.ExprFromResult(1).Type())
	if err != nil {
		return nil, err
	}
	return []ir.Element{
		elements.NewNamedType(philox.TypeMethods(), elements.NewStruct(
			philoxStruct,
			map[string]ir.Element{"state": philoxStateElement[0]},
		)),
		valuesElement[0],
	}, nil
}

func evalPhiloxUint32(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	return evalPhilox(env, call, recv, genArgs, args, dtypes.Uint32)
}

func evalPhiloxUint64(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	return evalPhilox(env, call, recv, genArgs, args, dtypes.Uint64)
}
