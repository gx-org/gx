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
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/gx/build/ir"
)

// Tuple value grouping multiple values together.
type Tuple struct {
	state    *State
	file     *ir.File
	stmt     ir.SourceNode
	elements []Element
}

var (
	_ backendElement = (*Tuple)(nil)
	_ Element        = (*Tuple)(nil)
)

// Tuple returns a tuple to store the result of a function returning more than one value.
func (g *State) Tuple(file *ir.File, stmt ir.SourceNode, values []Element) *Tuple {
	return &Tuple{
		state:    g,
		file:     file,
		stmt:     stmt,
		elements: values,
	}
}

// State owning the element.
func (n *Tuple) State() *State {
	return n.state
}

func (n *Tuple) nodes() ([]*graph.OutputNode, error) {
	return OutputsFromElements(n.elements)
}

// Unpack returns the elements stored in the tuple.
func (n *Tuple) Unpack() []Element {
	return n.elements
}
