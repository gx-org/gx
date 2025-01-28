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
)

// Slice element storing a slice of elements.
type Slice struct {
	expr   ExprAt
	values []Element
	slicer Slicer
}

var _ Slicer = (*Slice)(nil)

// NewSlice returns a new slice where elements are constructed on demand.
func NewSlice(expr ExprAt, selector Slicer, numFields int) *Slice {
	return &Slice{
		expr:   expr,
		values: make([]Element, numFields),
		slicer: selector,
	}
}

// ToSlice returns a slice from a slice of elements.
func ToSlice(expr ExprAt, elements []Element) *Slice {
	return &Slice{
		expr:   expr,
		values: elements,
	}
}

// Flatten returns the elements of the slice.
func (n *Slice) Flatten() ([]Element, error) {
	return flattenAll(n.values)
}

// Slice of the tuple.
func (n *Slice) Slice(expr ExprAt, i int) (Element, error) {
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

// Unflatten consumes the next handles to return a GX value.
func (n *Slice) Unflatten(handles *Unflattener) (values.Value, error) {
	return handles.ParseComposite(ParseCompositeOf(values.NewSlice), n.expr.Node().Type(), n.values)
}

// Elements stored in the slice.
func (n *Slice) Elements() []Element {
	return n.values
}
