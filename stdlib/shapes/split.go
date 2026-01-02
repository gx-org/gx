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
	"go/token"

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type split struct {
	builtin.Func
}

func (f split) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[split]("Split", impl.Shapes.Split, pkg), nil
}

func (f split) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		ir.IntIndexType(),
		builtins.GenericArrayType,
		ir.IntLenType(),
	})
	if err != nil {
		return nil, err
	}

	axis, err := elements.EvalInt(fetcher, call.Args[0])
	if err != nil {
		return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "unable to evaluate axis argument to %s: %s", f.Func.Name(), err)
	}
	arrayType, ok := params[1].(ir.ArrayType)
	if !ok {
		return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "expected second argument to be an array in call to %s, but got %s", f.Func.Name(), call.Args[0].Type().String())
	}
	rank := arrayType.Rank()
	// Determine dimensions after split.
	dims := rank.Axes()
	numSplit := call.Args[2]
	if irkind.IsNumber(numSplit.Type().Kind()) {
		numSplit = &ir.NumberCastExpr{
			X:   numSplit,
			Typ: ir.IntLenType(),
		}
	}
	outputDims := append([]ir.AxisLengths{&ir.AxisExpr{Src: call.Expr(), X: numSplit}}, dims...)
	splitDimExpr, ok := dims[axis].(*ir.AxisExpr)
	if !ok {
		return nil, fmterr.Internalf(fetcher.File().FileSet(), call.Source(), "cannot split axis %s (%T)", dims[axis], dims[axis])
	}
	outputDims[axis+1] = &ir.AxisExpr{
		Src: call.Expr(),
		X:   builtins.ToBinaryExpr(token.QUO, splitDimExpr.X, numSplit),
	}
	out := ir.NewArrayType(&ast.ArrayType{}, arrayType.DataType(), &ir.Rank{Ax: outputDims})
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(call, params...),
		Results:  builtins.Fields(call, out),
	}, nil
}
