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
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"github.com/pkg/errors"

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type concat struct {
	builtin.Func
}

func (f concat) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[concat]("Concat", impl.Shapes.Concat, pkg), nil
}

func checkConsistent[R any](values []ir.ArrayType, extractFn func(v ir.ArrayType) (R, func() string, error), equalityFn func(lhs R, rhs R) bool) (result R, err error) {
	var empty R
	if len(values) == 0 {
		return empty, errors.Errorf("must have at least one value")
	}
	var resultStringer func() string
	result, resultStringer, err = extractFn(values[0])
	if err != nil {
		return empty, err
	}
	for i := 1; i < len(values); i++ {
		thisResult, thisStringer, err := extractFn(values[i])
		if err != nil {
			return empty, err
		}
		if !equalityFn(thisResult, result) {
			return empty, fmt.Errorf("inconsistent values, %s vs %s", resultStringer(), thisStringer())
		}
	}
	return result, nil
}

func numAxes(fetcher ir.Fetcher, call *ir.FuncCallExpr) func(ir.ArrayType) (int, func() string, error) {
	return func(a ir.ArrayType) (int, func() string, error) {
		val := len(a.Rank().Axes())
		return val, func() string {
			return strconv.Itoa(val)
		}, nil
	}
}

func (f concat) resultsType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (params []ir.Type, out ir.Type, err error) {
	want := []ir.Type{
		ir.IntIndexType(),
		builtins.GenericArrayType,
		builtins.GenericArrayType,
	}
	if len(call.Args) > 3 {
		for range len(call.Args) - 3 {
			want = append(want, builtins.GenericArrayType)
		}
	}
	params, err = builtins.BuildFuncParams(fetcher, call, f.Name(), want)
	if err != nil {
		return nil, nil, err
	}
	axis, err := elements.MustEvalInt(fetcher, call.Args[0])
	if err != nil {
		return nil, nil, err
	}
	arrayTypes, err := builtins.NarrowTypes[ir.ArrayType](fetcher, call, params[1:])
	if err != nil {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "expected all arguments but the first to be arrays in call to %s, but %s", f.Func.Name(), err)
	}
	arrayDataType := func(t ir.ArrayType) (ir.Type, func() string, error) {
		return t.DataType(), func() string {
			return t.DataType().ReferString(fetcher.File())
		}, nil
	}
	isSameKind := func(lhs, rhs ir.Type) bool { return lhs.Kind() == rhs.Kind() }
	dtype, err := checkConsistent(arrayTypes, arrayDataType, isSameKind)
	if err != nil {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "expected arrays of the same data type in call to %s, but %s", f.Func.Name(), err)
	}
	isEqual := func(lhs, rhs int) bool { return lhs == rhs }
	_, err = checkConsistent(arrayTypes, numAxes(fetcher, call), isEqual)
	if err != nil {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "expected all arguments to be arrays of the same rank in call to %s, but %s", f.Func.Name(), err)
	}

	// Check that all but the concatenated dimension match, determine dimensions after concat.
	firstDims := arrayTypes[0].Rank().Axes()
	if axis < 0 || int(axis) >= len(firstDims) {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "axis %d is out of bounds for array of rank %d in call to Concat", axis, len(firstDims))
	}
	var outputDims []ir.AxisLengths = make([]ir.AxisLengths, len(firstDims))
	copy(outputDims, firstDims)
	var outputExpr ir.Expr = firstDims[axis].AsExpr()

	for i := 1; i < len(arrayTypes); i++ {
		rank := arrayTypes[i].Rank()
		for j, axJ := range rank.Axes() {
			if j == int(axis) {
				// Ignore the concatenated dimension, it doesn't have to match.
				continue
			}
			ok, cpErr, err := axJ.AssignableTo(fetcher, outputDims[j])
			if err != nil {
				return nil, nil, fmterr.Error(fetcher.File().FileSet(), call.Node(), err)
			}
			if cpErr != nil {
				return nil, nil, fmterr.Error(fetcher.File().FileSet(), call.Node(), cpErr)
			}
			if !ok {
				from := fetcher.File()
				return nil, nil, fmterr.Errorf(from.FileSet(), call.Node(),
					"argument %d (shape: %v) incompatible with initial shape (%s) in %s call: dimension %d, %s != %s",
					i+1, arrayTypes[i].Rank().SourceString(from), arrayTypes[0].Rank().SourceString(from),
					f.Name(), j, axJ.SourceString(from), outputDims[j].SourceString(from))
			}
		}
		outputExpr = builtins.ToBinaryExpr(token.ADD, outputExpr, rank.Axes()[axis].AsExpr())
	}
	outputDims[axis] = &ir.AxisExpr{X: outputExpr}
	return params, ir.NewArrayType(&ast.ArrayType{}, dtype, &ir.Rank{
		Ax: outputDims,
	}), nil
}

func (f concat) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	params, result, err := f.resultsType(fetcher, call)
	if err != nil {
		return nil, err
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Node().Pos()}},
		Params:   builtins.Fields(call, params...),
		Results:  builtins.Fields(call, result),
	}, nil
}
