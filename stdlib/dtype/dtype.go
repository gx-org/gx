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

// Package dtype provides the functions in the dtype GX standard library.
package dtype

import (
	"fmt"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

// Package description of the GX dtype package.
var Package = builtin.PackageBuilder{
	FullPath: "dtype",
	Builders: []builtin.Builder{
		builtin.ParseSource(),
		builtin.ImplementGraphFunc("Reinterpret", evalReinterpret),
	},
}

func evalReinterpret(env evaluator.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	argNode, _, err := materialise.Element(mat, args[0])
	if err != nil {
		return nil, err
	}
	retType := call.Node().Callee.FuncType().Results.List[0].Type.Val()
	arrayType, ok := ir.Underlying(retType).(ir.ArrayType)
	if !ok {
		return nil, fmt.Errorf("%T is not an array type", retType)
	}
	dtype := arrayType.DataType().Kind().DType()
	ev := env.Evaluator().(*grapheval.Evaluator)
	gr := ev.ArrayOps().Graph()
	op, err := gr.DType().Bitcast(argNode, dtype)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(call.File(), mat, &ops.OutputNode{
		Node: op,
		Shape: &shape.Shape{
			DType:       dtype,
			AxisLengths: op.(interface{ PJRTDims() []int }).PJRTDims(),
		},
	}, call.Node().Type())
}
