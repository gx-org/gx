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
	"math/big"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
	gxerrors "github.com/gx-org/gx/stdlib/errors"
)

func gatherAxis(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) (_ []ir.Element, err error) {
	inputSlice, err := elements.SliceFromElement(args[0])
	if err != nil {
		return nil, err
	}
	input := inputSlice.Elements()
	indicesSlice, err := elements.SliceFromElement(args[1])
	if err != nil {
		return nil, err
	}
	indices := indicesSlice.Elements()
	if len(indices) < 1 {
		gxErr, err := gxerrors.Errorf(env, "expect indices to have at least one axis but got zero")
		return []ir.Element{builtin.NilShape, gxErr}, err
	}
	if len(indices) > 2 {
		gxErr, err := gxerrors.Errorf(env, "expect indices to have maximum 2 axes but got %d", len(indices))
		return []ir.Element{builtin.NilShape, gxErr}, err
	}
	numPositions := 1
	if len(indices) == 2 {
		numPosBig, err := togo.ValueT[*big.Int](indices[1])
		if err != nil {
			return nil, err
		}
		numPositions = int(numPosBig.Int64())
	}
	out := []ir.Element{indices[0]}
	if numPositions > len(input) {
		from := env.File()
		gxErr, err := gxerrors.Errorf(env, "cannot specify %s position(s) with %d axis indices (in %s) to specify values in input array of %d axis length(s)", ir.SourceString(from, indices[0]), numPositions, builtin.ToShapeString(indices), len(input))
		return []ir.Element{builtin.NilShape, gxErr}, err
	}
	out = append(out, input[numPositions:]...)
	return builtin.ToShapeResult(out...)
}

func evalGather(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	inputShape, err := elements.Map(elements.ConstantIntFromElement, args[0])
	if err != nil {
		return nil, err
	}
	indicesShape, err := elements.Map(elements.ConstantIntFromElement, args[1])
	if err != nil {
		return nil, err
	}
	paramsRank := len(inputShape)
	indicesRank := len(indicesShape)
	indexedSubRank := indicesShape[indicesRank-1] // N from documentation.
	slicesSubRank := paramsRank - indexedSubRank  // S from documentation, the slices dimensions.
	if slicesSubRank < 0 {
		return nil, fmt.Errorf("Gather params are \"over-indexed\": params has only rank %d and "+
			"indexed rank is %d (last dimension of indices)", paramsRank, indexedSubRank)
	}
	outputSubRank := indicesRank - 1

	// * indexVectorDim is always the last one.
	indexVectorDim := indicesRank - 1
	// * startIndexMap is sequential and sorted
	startIndexMap := make([]int, indexedSubRank)
	for ii := 0; ii < indexedSubRank; ii++ {
		startIndexMap[ii] = ii
	}
	// * sliceSizes are 1 everywhere but on the sliced dimensions.
	// * collapsedSliceDims is set to collapse all dimensions set to 1.
	sliceSizes := make([]int, paramsRank)
	collapsedSliceDims := make([]int, indexedSubRank)
	for ii := 0; ii < paramsRank; ii++ {
		if ii < indexedSubRank {
			sliceSizes[ii] = 1
			collapsedSliceDims[ii] = ii
		} else {
			sliceSizes[ii] = inputShape[ii]
		}
	}
	// * offsetDims are the dimensions indexed.
	offsetDims := make([]int, paramsRank-indexedSubRank)
	for ii := range offsetDims {
		offsetDims[ii] = outputSubRank + ii
	}

	mat := builtin.Materialiser(env)
	x, xShape, err := materialise.Element(mat, args[3])
	if err != nil {
		return nil, err
	}
	indicesNode, _, err := materialise.Element(mat, args[4])
	if err != nil {
		return nil, err
	}
	gr := env.Engine().ArrayOps().Graph()
	op, err := gr.Shape().Gather(x, indicesNode, indexVectorDim, offsetDims, collapsedSliceDims, startIndexMap, sliceSizes, false)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.File(), mat, &ops.OutputNode{
		Node: op,
		Shape: &shape.Shape{
			DType:       xShape.DType,
			AxisLengths: op.(interface{ PJRTDims() []int }).PJRTDims(),
		},
	}, call.Type())
}
