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

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

// evalEinsum evaluates an Einsum op.
// Arguments:
//
//	0: XContract []int
//	1: XBatch []int
//	2: YContract []int
//	3: YBatch []int
//	4: XShape []int
//	5: YShape []int
//	6: T dtype.Num
//	7: x [unpack(XShape)]T
//	8: y [unpack(YShape)]T
func evalEinsum(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	left, leftShape, err := materialise.Element(mat, args[7])
	if err != nil {
		return nil, err
	}
	lhsContractingAxes, err := elements.AxesFromElement(args[0])
	if err != nil {
		return nil, err
	}
	lhsBatchAxes, err := elements.AxesFromElement(args[1])
	if err != nil {
		return nil, err
	}
	right, rightShape, err := materialise.Element(mat, args[8])
	if err != nil {
		return nil, err
	}
	rhsContractingAxes, err := elements.AxesFromElement(args[2])
	if err != nil {
		return nil, err
	}
	rhsBatchAxes, err := elements.AxesFromElement(args[3])
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

func validateAxisExpr(env *engine.Env, call *ir.FuncCallExpr, arg ir.Element, maxRank int, seen map[int]bool) ([]int, error) {
	argSlice, err := elements.SliceFromElement(arg)
	if err != nil {
		return nil, err
	}
	axes := make([]int, argSlice.Len())
	for n, val := range argSlice.Elements() {
		axis, err := elements.IntFromElement(val)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[axis]; exists {
			return nil, ir.CompileErrorF("axis %d already specified in argument %d: axes may only be contracted or batched once", axis, arg)
		}
		if axis < 0 || axis >= maxRank {
			return nil, ir.CompileErrorF("axis %d specified in argument %d is out-of-range: must be in [0, %d)", axis, arg, maxRank)
		}
		axes[n] = axis
		seen[axis] = true
	}
	return axes, nil
}

// evalEinsumAxes implement:
//
//	func einsumAxes(xS, xContract, xBatch, yS, yContract, yBatch)
//
// The contraction axes must be specified as a pair (once each on the LHS and RHS sides). These
// axes are multiplied and summed, as in a dot product with lhs[lhsContractingAxes[n]] on one side
// and rhs[rhsContractingAxes[n]]) on the other.
//
// The batch axes specify a batch dimension.
func evalEinsumAxes(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	leftDims, err := elements.SliceFromElement(args[0])
	if err != nil {
		return nil, err
	}
	rightDims, err := elements.SliceFromElement(args[3])
	if err != nil {
		return nil, err
	}
	leftSeen := make(map[int]bool)
	rightSeen := make(map[int]bool)
	lhsContractingDims, err := validateAxisExpr(env, call, args[1], leftDims.Len(), leftSeen)
	if err != nil {
		return nil, err
	}
	lhsBatchDims, err := validateAxisExpr(env, call, args[2], leftDims.Len(), leftSeen)
	if err != nil {
		return nil, err
	}
	rhsContractingDims, err := validateAxisExpr(env, call, args[4], rightDims.Len(), rightSeen)
	if err != nil {
		return nil, err
	}
	rhsBatchDims, err := validateAxisExpr(env, call, args[5], rightDims.Len(), rightSeen)
	if err != nil {
		return nil, err
	}
	if len(lhsContractingDims) != len(rhsContractingDims) {
		return nil, ir.CompileErrorF(
			"must specify the same number of lhs and rhs contracting dimensions (got %d and %d)",
			len(lhsContractingDims), len(rhsContractingDims))
	}
	for n := range lhsContractingDims {
		lhsDim := leftDims.Elements()[lhsContractingDims[n]]
		rhsDim := rightDims.Elements()[rhsContractingDims[n]]
		eq, err := cmp.Equal(env.ExprEval(), lhsDim, rhsDim)
		if err != nil {
			return nil, err
		}
		if !eq {
			return nil, ir.CompileErrorF(
				"left argument (shape: %v) not compatible with right argument (shape: %v): cannot contract lhs dimension %v with rhs dimension %v",
				leftDims, rightDims, lhsDim, rhsDim)
		}
	}
	if len(lhsBatchDims) != len(rhsBatchDims) {
		return nil, ir.CompileErrorF(
			"must specify the same number of lhs and rhs batching dimensions (got %d and %d)",
			len(lhsBatchDims), len(rhsBatchDims))
	}
	for n := range min(len(lhsBatchDims), len(rhsBatchDims)) {
		lhsDim := leftDims.Elements()[lhsBatchDims[n]]
		rhsDim := rightDims.Elements()[rhsBatchDims[n]]
		eq, err := cmp.Equal(env.ExprEval(), lhsDim, rhsDim)
		if err != nil {
			return nil, err
		}
		if !eq {
			return nil, ir.CompileErrorF(
				"left argument (shape: %v) not compatible with right argument (shape: %v): cannot batch lhs dimension %v with rhs dimension %v",
				leftDims, rightDims, lhsDim, rhsDim)
		}
	}
	// Infer output dimensions: batch dimensions (in the LHS order) become the outermost dimensions,
	// followed by LHS dimensions not used for batching nor contracting, then RHS dimensions not used
	// for batching nor contracting.
	var outDims []ir.Element
	for _, lhsBatchDim := range lhsBatchDims {
		outDims = append(outDims, leftDims.Elements()[lhsBatchDim])
	}
	for n, lhsDim := range leftDims.Elements() {
		if !leftSeen[n] {
			outDims = append(outDims, lhsDim)
		}
	}
	for n, rhsDim := range rightDims.Elements() {
		if !rightSeen[n] {
			outDims = append(outDims, rhsDim)
		}
	}
	return builtin.ToShapeResult(outDims...)
}
