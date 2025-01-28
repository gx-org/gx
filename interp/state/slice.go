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
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/interp/elements"
)

// Slice value.
type Slice struct {
	state  *State
	expr   elements.ExprAt
	values []Element
	slicer Slicer
}

var _ Slicer = (*Slice)(nil)

// Slice returns a new slice where elements are constructed on demand.
func (g *State) Slice(expr elements.ExprAt, selector Slicer, numFields int) *Slice {
	return &Slice{
		state:  g,
		expr:   expr,
		values: make([]Element, numFields),
		slicer: selector,
	}
}

// ToSlice returns a slice from a slice of elements.
func (g *State) ToSlice(expr elements.ExprAt, elements []Element) *Slice {
	return &Slice{
		state:  g,
		expr:   expr,
		values: elements,
	}
}

// Flatten returns the elements of the slice.
func (n *Slice) Flatten() ([]Element, error) {
	return flattenAll(n.values)
}

// Slice of the tuple.
func (n *Slice) Slice(expr elements.ExprAt, i int) (Element, error) {
	if i < 0 || i >= len(n.values) {
		return nil, errors.Errorf("invalid argument: index %d out of bounds [0:%d]", i, len(n.values))
	}
	if n.values[i] != nil {
		return n.values[i], nil
	}
	var err error
	n.values[i], err = n.slicer.Slice(expr, i)
	if err != nil {
		return nil, err
	}
	return n.values[i], nil
}

func (n *Slice) valueFromHandle(handles *handleParser) (values.Value, error) {
	return handles.parseComposite(parseCompositeOf(values.NewSlice), n.expr.Node().Type(), n.values)
}

// Elements stored in the slice.
func (n *Slice) Elements() []Element {
	return n.values
}

// State owning the element.
func (n *Slice) State() *State {
	return n.state
}
