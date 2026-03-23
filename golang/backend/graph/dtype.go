// Copyright 2026 Google LLC
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
)

// DType returns the builder for the dtype package.
func (g *Graph) DType() ops.DTypeBuilder {
	return g
}

// Bitcast casts a byte array into a given data type.
func (g *Graph) Bitcast(x ops.Node, target dtype.DataType) (ops.Node, error) {
	return nil, errors.Errorf("Bitcast not implemented in the Go backend")
}
