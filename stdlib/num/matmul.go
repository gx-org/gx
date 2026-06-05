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
	"fmt"
	"go/ast"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

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
	return builtin.IRFuncBuiltin[einsum]("Einsum", evalEinsum, pkg), nil
}

func (f einsum) validateAxisExpr(fetcher ir.Fetcher, call *ir.FuncCallExpr, arg, maxRank int, seen map[int]bool) ([]int, error) {
	axisExpr, ok := call.Args[arg].(*ir.SliceLitExpr)
	if !ok {
		return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "argument %d to %s must be []intidx (got %s)", arg, f.Name(), call.Args[arg].SourceString(fetcher.File()))
	}

	axes := make([]int, len(axisExpr.Elts))
	for n, val := range axisExpr.Elts {
		axisI64, err := elements.MustEvalInt(fetcher, val)
		if err != nil {
			return nil, err
		}
		axis := int(axisI64)
		if _, exists := seen[axis]; exists {
			return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "axis %d already specified in argument %d to %s: axes may only be contracted or batched once", axis, arg, f.Name())
		}
		if axis < 0 || axis >= maxRank {
			return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "axis %d specified in argument %d to %s is out-of-range: must be in [0, %d)", axis, arg, f.Name(), maxRank)
		}
		axes[n] = axis
		seen[axis] = true
	}
	return axes, nil
}

func (f einsum) resultsType(fetcher ir.Fetcher, call *ir.FuncCallExpr) ([]ir.Type, ir.Type, error) {
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
		from := fetcher.File()
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "mismatched argument types %s and %s in %s call", left.DataType().ReferString(from), right.DataType().ReferString(from), f.Name())
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
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(),
			"must specify the same number of lhs and rhs contracting dimensions for %s (got %d and %d)",
			f.Name(), len(lhsContractingDims), len(rhsContractingDims))
	}
	for n := range lhsContractingDims {
		lhsDim := leftDims[lhsContractingDims[n]]
		rhsDim := rightDims[rhsContractingDims[n]]
		ok, cpErr, err := lhsDim.Equal(fetcher, rhsDim)
		if err != nil {
			return nil, nil, fmterr.Error(fetcher.File().FileSet(), call.Node(), err)
		}
		if cpErr != nil {
			return nil, nil, fmterr.Error(fetcher.File().FileSet(), call.Node(), cpErr)
		}
		if !ok {
			from := fetcher.File()
			return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(),
				"left argument (shape: %s) not compatible with right argument (shape: %s) in %s call: cannot contract lhs dimension %s with rhs dimension %s",
				left.Rank().SourceString(from), right.Rank().SourceString(from), f.Name(), lhsDim.SourceString(from), rhsDim.SourceString(from))
		}
	}
	if len(lhsBatchDims) != len(rhsBatchDims) {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(),
			"must specify the same number of lhs and rhs batching dimensions for %s (got %d and %d)",
			f.Name(), len(lhsBatchDims), len(rhsBatchDims))
	}
	for n := range min(len(lhsBatchDims), len(rhsBatchDims)) {
		lhsDim := leftDims[lhsBatchDims[n]]
		rhsDim := rightDims[rhsBatchDims[n]]
		ok, cpErr, err := lhsDim.Equal(fetcher, rhsDim)
		if err != nil {
			return nil, nil, fmterr.Error(fetcher.File().FileSet(), call.Node(), err)
		}
		if cpErr != nil {
			return nil, nil, fmterr.Error(fetcher.File().FileSet(), call.Node(), cpErr)
		}
		if !ok {
			from := fetcher.File()
			return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Node(),
				"left argument (shape: %s) not compatible with right argument (shape: %s) in %s call: cannot batch lhs dimension %s with rhs dimension %s",
				left.Rank().SourceString(from), right.Rank().SourceString(from), f.Name(), lhsDim.SourceString(from), rhsDim.SourceString(from))
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
	return params, ir.NewArrayType(&ast.ArrayType{}, left.DataType(), &ir.Rank{Ax: outDims}), nil
}

func (f einsum) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
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

func evalEinsum(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	left, leftShape, err := materialise.Element(mat, args[0])
	if err != nil {
		return nil, err
	}
	lhsContractingAxes, err := elements.AxesFromElement(args[1])
	if err != nil {
		return nil, err
	}
	lhsBatchAxes, err := elements.AxesFromElement(args[2])
	if err != nil {
		return nil, err
	}
	right, rightShape, err := materialise.Element(mat, args[3])
	if err != nil {
		return nil, err
	}
	rhsContractingAxes, err := elements.AxesFromElement(args[4])
	if err != nil {
		return nil, err
	}
	rhsBatchAxes, err := elements.AxesFromElement(args[5])
	if err != nil {
		return nil, err
	}

	op, err := env.Engine().ArrayOps().Graph().Num().DotGeneral(left, right,
		[2][]int{lhsBatchAxes, rhsBatchAxes},
		[2][]int{lhsContractingAxes, rhsContractingAxes})
	if err != nil {
		return nil, fmt.Errorf("\nlhsContractingAxes: %v\nlhsBatchAxes: %v\nrhsContractingAxes: %v\nrhsBatchAxes: %v\nleft: %v\nright: %v", lhsContractingAxes, lhsBatchAxes, rhsContractingAxes, rhsBatchAxes, leftShape, rightShape)
	}
	return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
		Node: op,
		Shape: &shape.Shape{
			DType:       leftShape.DType,
			AxisLengths: op.(interface{ PJRTDims() []int }).PJRTDims(),
		},
	}, call.Type())
}
