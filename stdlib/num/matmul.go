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
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type matmul struct {
	builtin.Func
}

func (f matmul) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[matmul]("MatMul", impl.Num.MatMul, pkg), nil
}

func (f matmul) resultsType(fetcher ir.Fetcher, call *ir.CallExpr) ([]ir.Type, ir.Type, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		builtins.GenericArrayType,
		builtins.GenericArrayType,
	})
	if err != nil {
		return nil, nil, err
	}
	arrays, err := builtins.NarrowTypes[ir.ArrayType](fetcher, call, params)
	if err != nil {
		return nil, nil, err
	}
	left := arrays[0]
	right := arrays[1]
	if left.DataType().Kind() != right.DataType().Kind() {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "mismatched argument types %s and %s in %s call", left.DataType().String(), right.DataType().String(), f.Name())
	}
	leftRank := left.Rank()
	if len(leftRank.Axes()) > 2 {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "more than 2-axis for the left argument (shape: %v) not supported in %s call", left.Rank(), f.Name())
	}
	rightRank := right.Rank()
	if len(rightRank.Axes()) > 2 {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "more than 2-axis for the right argument (shape: %v) not supported in %s call", right.Rank(), f.Name())
	}
	leftDims := leftRank.Axes()
	rightDims := rightRank.Axes()
	if len(leftDims) == 0 || len(rightDims) == 0 {
		// If either argument has unknown rank, the result has unknown rank too.
		return params, ir.NewArrayType(nil, left.DataType(), left.Rank()), nil
	}
	lastLeft := leftDims[len(leftDims)-1]
	dotDimIndex := 0
	if len(rightDims) > 1 {
		dotDimIndex = len(rightDims) - 2
	}
	dotDim := rightDims[dotDimIndex]
	ok, err := lastLeft.AssignableTo(fetcher, dotDim)
	if err != nil {
		return nil, nil, fmterr.Position(fetcher.File().FileSet(), call.Source(), err)
	}
	if !ok {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "left argument (shape: %v) not compatible with right argument (shape: %v) in %s call", left.Rank(), right.Rank(), f.Name())
	}
	resultRank := ir.Rank{}
	if len(leftDims) == 2 && len(rightDims) == 2 {
		resultRank.Ax = []ir.AxisLengths{leftDims[0], rightDims[1]}
	} else if len(leftDims) == 2 {
		resultRank.Ax = []ir.AxisLengths{leftDims[0]}
	} else if len(rightDims) == 2 {
		resultRank.Ax = []ir.AxisLengths{rightDims[1]}
	}
	return params, ir.NewArrayType(nil, left.DataType(), &resultRank), nil
}

func (f matmul) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params, result, err := f.resultsType(fetcher, call)
	if err != nil {
		return nil, err
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(call, params...),
		Results:  builtins.Fields(call, result),
	}, nil
}

// num.Einsum(lhs, lhsContractingAxes, lhsBatchAxes, rhs, rhsContractingAxes, rhsBatchAxes)
// evaluates the "Einstein summation" various types of products (inner/outer/batched)
// between 2 tensors, on arbitrary numbered dimensions.
//
// The contraction axes must be specified as a pair (once each on the LHS and RHS sides). These
// axes are multiplied and summed, as in a dot product with lhs[lhsContractingAxes[n]] on one side
// and rhs[rhsContractingAxes[n]]) on the other.
//
// The batch axes specify a batch dimension.
type einsum struct {
	builtin.Func
}

func (f einsum) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[einsum]("Einsum", impl.Num.Einsum, pkg), nil
}

func (f einsum) validateAxisExpr(fetcher ir.Fetcher, call *ir.CallExpr, arg, maxRank int, seen map[int]bool) ([]int, error) {
	axisExpr, ok := call.Args[arg].(*ir.SliceLitExpr)
	if !ok {
		return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "argument %d to %s must be []intidx (got %s)", arg, f.Name(), call.Args[arg].String())
	}

	axes := make([]int, len(axisExpr.Elts))
	for n, val := range axisExpr.Elts {
		axisI64, err := compeval.EvalInt(fetcher, val)
		if err != nil {
			return nil, err
		}
		axis := int(axisI64)
		if _, exists := seen[axis]; exists {
			return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "axis %d already specified in argument %d to %s: axes may only be contracted or batched once", axis, arg, f.Name())
		}
		if axis < 0 || axis >= maxRank {
			return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "axis %d specified in argument %d to %s is out-of-range: must be in [0, %d)", axis, arg, f.Name(), maxRank)
		}
		axes[n] = axis
		seen[axis] = true
	}
	return axes, nil
}

func (f einsum) resultsType(fetcher ir.Fetcher, call *ir.CallExpr) ([]ir.Type, ir.Type, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		builtins.GenericArrayType,
		ir.IntIndexSliceType(),
		ir.IntIndexSliceType(),
		builtins.GenericArrayType,
		ir.IntIndexSliceType(),
		ir.IntIndexSliceType(),
	})
	if err != nil {
		return nil, nil, err
	}

	left := params[0].(ir.ArrayType)
	right := params[3].(ir.ArrayType)
	if left.DataType().Kind() != right.DataType().Kind() {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "mismatched argument types %s and %s in %s call", left.DataType().String(), right.DataType().String(), f.Name())
	}
	leftRank := left.Rank()
	rightRank := right.Rank()
	if leftRank == nil || rightRank == nil {
		// If either argument has unknown rank, the result has unknown rank too.
		return params, ir.NewArrayType(nil, left.DataType(), &ir.RankInfer{}), nil
	}
	leftDims := leftRank.Axes()
	rightDims := rightRank.Axes()

	leftSeen := make(map[int]bool)
	rightSeen := make(map[int]bool)
	lhsContractingDims, err := f.validateAxisExpr(fetcher, call, 1, len(leftDims), leftSeen)
	if err != nil {
		return nil, nil, err
	}
	lhsBatchDims, err := f.validateAxisExpr(fetcher, call, 2, len(leftDims), leftSeen)
	if err != nil {
		return nil, nil, err
	}
	rhsContractingDims, err := f.validateAxisExpr(fetcher, call, 4, len(rightDims), rightSeen)
	if err != nil {
		return nil, nil, err
	}
	rhsBatchDims, err := f.validateAxisExpr(fetcher, call, 5, len(rightDims), rightSeen)
	if err != nil {
		return nil, nil, err
	}
	if len(lhsContractingDims) != len(rhsContractingDims) {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(),
			"must specify the same number of lhs and rhs contracting dimensions for %s (got %d and %d)",
			f.Name(), len(lhsContractingDims), len(rhsContractingDims))
	}
	for n := range lhsContractingDims {
		lhsDim := leftDims[lhsContractingDims[n]]
		rhsDim := rightDims[rhsContractingDims[n]]
		ok, err := lhsDim.AssignableTo(fetcher, rhsDim)
		if err != nil {
			return nil, nil, fmterr.Position(fetcher.File().FileSet(), call.Source(), err)
		}
		if !ok {
			return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(),
				"left argument (shape: %v) not compatible with right argument (shape: %v) in %s call: cannot contract lhs dimension %s with rhs dimension %s",
				left.Rank(), right.Rank(), f.Name(), lhsDim, rhsDim)
		}
	}
	if len(lhsBatchDims) != len(rhsBatchDims) {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(),
			"must specify the same number of lhs and rhs batching dimensions for %s (got %d and %d)",
			f.Name(), len(lhsBatchDims), len(rhsBatchDims))
	}
	for n := range min(len(lhsBatchDims), len(rhsBatchDims)) {
		lhsDim := leftDims[lhsBatchDims[n]]
		rhsDim := rightDims[rhsBatchDims[n]]
		ok, err := lhsDim.AssignableTo(fetcher, rhsDim)
		if err != nil {
			return nil, nil, fmterr.Position(fetcher.File().FileSet(), call.Source(), err)
		}
		if !ok {
			return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(),
				"left argument (shape: %v) not compatible with right argument (shape: %v) in %s call: cannot batch lhs dimension %s with rhs dimension %s",
				left.Rank(), right.Rank(), f.Name(), lhsDim, rhsDim)
		}
	}
	// Infer output dimensions: batch dimensions (in the LHS order) become the outermost dimensions,
	// followed by LHS dimensions not used for batching nor contracting, then RHS dimensions not used
	// for batching nor contracting.
	var outDims []ir.AxisLengths
	for _, lhsBatchDim := range lhsBatchDims {
		outDims = append(outDims, leftDims[lhsBatchDim])
	}
	for n, lhsDim := range leftDims {
		if !leftSeen[n] {
			outDims = append(outDims, lhsDim)
		}
	}
	for n, rhsDim := range rightDims {
		if !rightSeen[n] {
			outDims = append(outDims, rhsDim)
		}
	}
	return params, ir.NewArrayType(nil, left.DataType(), &ir.Rank{Ax: outDims}), nil
}

func (f einsum) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params, result, err := f.resultsType(fetcher, call)
	if err != nil {
		return nil, err
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(call, params...),
		Results:  builtins.Fields(call, result),
	}, nil
}
