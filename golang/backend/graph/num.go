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
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
)

// Num returns the builder for the num package.
func (g *Graph) Num() graph.NumBuilder {
	return g
}

// Iota returns a node filling an array with values from 0 to number of elements-1.
func (g *Graph) Iota(sh *shape.Shape, iotaAxis int) (graph.Node, error) {
	return nil, errors.Errorf("not implemented")
}
