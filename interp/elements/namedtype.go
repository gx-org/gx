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

package elements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// NamedType references a type exported by an imported package.
type NamedType struct {
	errFmt fmterr.Pos
	typ    *ir.NamedType
}

// NewNamedType returns a new node representing an exported type.
func NewNamedType(errF fmterr.FileSet, typ *ir.NamedType) *NamedType {
	return &NamedType{typ: typ}
}

// Flatten returns the named type in a slice of elements.
func (n *NamedType) Flatten() ([]Element, error) {
	return []Element{n}, nil
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (n *NamedType) Unflatten(handles *Unflattener) (values.Value, error) {
	return nil, fmterr.Internal(errors.Errorf("%T does not support converting device handles into GX values", n), "")
}

// ErrPos returns the error formatter for the position of the token representing the node in the graph.
func (n *NamedType) ErrPos() fmterr.Pos {
	return n.errFmt
}

// Type of a package.
func (n *NamedType) Type() ir.Type {
	return n.typ
}

// String returns a string representation of the node.
func (n *NamedType) String() string {
	return "package"
}
