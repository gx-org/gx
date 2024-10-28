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

package state

import (
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// NamedType references a type exported by an imported package.
type NamedType struct {
	state  *State
	errFmt fmterr.Pos
	typ    *ir.NamedType
}

// NamedType returns a new node representing an exported type.
func (g *State) NamedType(errF fmterr.FileSet, typ *ir.NamedType) *NamedType {
	return &NamedType{state: g, typ: typ}
}

// ErrPos returns the error formatter for the position of the token representing the node in the graph.
func (n *NamedType) ErrPos() fmterr.Pos {
	return n.errFmt
}

// Type of a package.
func (n *NamedType) Type() ir.Type {
	return n.typ
}

// State owning the element.
func (n *NamedType) State() *State {
	return n.state
}

// String returns a string representation of the node.
func (n *NamedType) String() string {
	return "package"
}
