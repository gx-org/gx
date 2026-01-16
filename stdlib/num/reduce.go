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

package num

import (
	"go/ast"

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

var reductionArgs = []ir.Type{
	builtins.GenericArrayType,
	ir.IntIndexSliceType(),
}

func reductionFuncSig(fetcher ir.Fetcher, f builtin.Func, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	// Perform basic signature check; we know we have (tensor, slice) arguments afterwards.
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), reductionArgs)
	if err != nil {
		return nil, err
	}
	arrayType := call.Args[0].Type().(ir.ArrayType)
	rank := arrayType.Rank()
	reduceAxes, err := builtins.UniqueAxesFromExpr(fetcher, call.Args[1])
	if err != nil {
		return nil, err
	}

	// Infer the result tensor shape by knocking out reduced axes.
	resultRank := ir.Rank{}
	result := ir.NewArrayType(&ast.ArrayType{}, arrayType.DataType(), &resultRank)
	var resultDims []ir.AxisLengths
	for axis := range reduceAxes {
		if len(rank.Axes()) > 0 && (axis < 0 || axis >= len(rank.Axes())) {
			return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(),
				"invalid reduction axis in call to %s: axis %d does not exist in input %s",
				f.Name(), axis, arrayType)
		}
	}

	for n, dim := range rank.Axes() {
		if _, reduce := reduceAxes[n]; !reduce {
			resultDims = append(resultDims, dim)
		}
	}
	resultRank.Ax = resultDims
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Node().Pos()}},
		Params:   builtins.Fields(call, params...),
		Results:  builtins.Fields(call, result),
	}, nil
}

type reduceMax struct {
	builtin.Func
}

func (f reduceMax) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[reduceMax]("ReduceMax", impl.Num.ReduceMax, pkg), nil
}

func (f reduceMax) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	return reductionFuncSig(fetcher, f.Func, call)
}

type reduceSum struct {
	builtin.Func
}

func (f reduceSum) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[reduceSum]("Sum", impl.Num.Sum, pkg), nil
}

func (f reduceSum) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	return reductionFuncSig(fetcher, f.Func, call)
}

type argmax struct {
	builtin.Func
}

func (f argmax) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[argmax]("Argmax", impl.Num.Argmax, pkg), nil
}

func (f argmax) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	// Perform basic signature check; we know we have (tensor, slice) arguments afterwards.
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		builtins.GenericArrayType,
		ir.IntIndexType(),
	})
	if err != nil {
		return nil, err
	}
	arrayType := call.Args[0].Type().(ir.ArrayType)
	rank := arrayType.Rank()
	reduceAxis, err := elements.MustEvalInt(fetcher, call.Args[1])
	if err != nil {
		return nil, err
	}

	// Infer the result tensor shape by knocking out reduced axis.
	resultRank := ir.Rank{}
	result := ir.NewArrayType(&ast.ArrayType{}, ir.TypeFromKind(irkind.DefaultInt), &resultRank)
	if len(rank.Axes()) > 0 && (reduceAxis < 0 || int(reduceAxis) >= len(rank.Axes())) {
		return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(),
			"invalid reduction axis in call to %s: axis %d does not exist in input %s",
			f.Name(), reduceAxis, arrayType)
	}

	var resultDims []ir.AxisLengths
	for n, dim := range rank.Axes() {
		if n != int(reduceAxis) {
			resultDims = append(resultDims, dim)
		}
	}
	resultRank.Ax = resultDims
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Node().Pos()}},
		Params:   builtins.Fields(call, params...),
		Results:  builtins.Fields(call, result),
	}, nil
}
