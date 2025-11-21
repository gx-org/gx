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

type iotaNode struct {
	node
}

var _ execNode = (*iotaNode)(nil)

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

func (n *iotaNode) exec(*executor) (kernels.Array, error) {
	data := make([]int64, n.sh.Size())
	for i := range data {
		data[i] = int64(i)
	}
	return kernels.ToIntegerArray(data, n.sh.AxisLengths), nil
}
