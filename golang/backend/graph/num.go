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

package graph

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Num returns the builder for the num package.
func (g *Graph) Num() ops.NumBuilder {
	return g
}

// ReduceMax is a shortcut for Reduce with the proper computation and initial value to reduce x
// on the given axes, by taking the max value.
//
// If no axes are given, it reduces the full array.
func (g *Graph) ReduceMax(x ops.Node, axes []int) (ops.Node, error) {
	return nil, errors.Errorf("ReduceMax not implemented in the Go backend")
}

// ReduceSum sums over axes.
func (g *Graph) ReduceSum(x ops.Node, axes []int) (ops.Node, error) {
	return nil, errors.Errorf("ReduceSum not implemented in the Go backend")
}

// Transpose an array.
func (g *Graph) Transpose(x ops.Node, permutation []int) (ops.Node, error) {
	return nil, errors.Errorf("Transpose not implemented in the Go backend")
}

// Dot product between x and y.
func (g *Graph) Dot(x, y ops.Node) (ops.Node, error) {
	return nil, errors.Errorf("Dot not implemented in the Go backend")
}

// Gather data from an array.
func (g *Graph) Gather(x ops.Node, startIndices ops.Node, indexVectorAxis int, offsetAxes []int, collapsedSliceAxes []int, startIndexMap []int, sliceSizes []int, indicesAreSorted bool) (ops.Node, error) {
	return nil, errors.Errorf("Gather not implemented in the Go backend")
}

type iotaNode struct {
	node
}

var _ execNode = (*iotaNode)(nil)

func (n *iotaNode) exec(*executor) (kernels.Array, error) {
	data := make([]int64, n.sh.Size())
	for i := range data {
		data[i] = int64(i)
	}
	return kernels.ToIntegerArray(data, n.sh.AxisLengths), nil
}

// Iota returns a node filling an array with values from 0 to number of elements-1.
func (g *Graph) Iota(sh *shape.Shape, iotaAxis int) (ops.Node, error) {
	if iotaAxis != 0 {
		return nil, errors.Errorf("op Iota: got axis %d but Iota only implemented axis 0", iotaAxis)
	}
	if sh.DType != dtype.Int64 {
		return nil, errors.Errorf("op Iota: shape has datatype %s but want %s", sh.DType, dtype.Int64)
	}
	factory, err := kernels.FactoryFor(dtype.Int64)
	if err != nil {
		return nil, err
	}
	return &iotaNode{
		node: node{
			g:  g,
			k:  factory,
			sh: sh,
		},
	}, nil
}

// ArgMinMax applies a min or max operator over an axis
func (g *Graph) ArgMinMax(x ops.Node, axis int, outputDType dtype.DataType, isMin bool) (ops.Node, error) {
	return nil, errors.Errorf("ArgMinMax not implemented in the Go backend")
}
