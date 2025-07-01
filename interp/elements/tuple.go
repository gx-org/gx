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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// Tuple value grouping multiple values together.
type Tuple struct {
	node     NodeAt
	elements []Element
}

var (
	_ Slicer  = (*Tuple)(nil)
	_ Element = (*Tuple)(nil)
)

// NewTuple returns a tuple to store the result of a function returning more than one value.
func NewTuple(file *ir.File, node ir.Node, values []Element) *Tuple {
	return &Tuple{
		node:     NewNodeAt(file, node),
		elements: values,
	}
}

// Flatten the tuple and all its elements.
func (n *Tuple) Flatten() ([]Element, error) {
	return Flatten(n.elements...)
}

// Elements returns the elements stored in the tuple.
func (n *Tuple) Elements() []Element {
	return n.elements
}

// Slice of the tuple.
func (n *Tuple) Slice(ctx ir.Evaluator, expr *ir.IndexExpr, index NumericalElement) (Element, error) {
	return slice(ctx, expr, index, n.elements)
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (n *Tuple) Unflatten(handles *Unflattener) (values.Value, error) {
	return nil, fmterr.Internal(errors.Errorf("%T does not support converting device handles into GX values", n))
}

// Kind of the element.
func (*Tuple) Kind() ir.Kind {
	return ir.TupleKind
}

func (n *Tuple) String() string {
	els := make([]string, len(n.elements))
	for i, el := range n.elements {
		els[i] = fmt.Sprint(el)
	}
	return fmt.Sprintf("(%s)", strings.Join(els, ", "))
}

// Flatten elements.
func Flatten(elts ...Element) ([]Element, error) {
	var flat []Element
	for _, elt := range elts {
		subs, err := elt.Flatten()
		if err != nil {
			return nil, err
		}
		flat = append(flat, subs...)
	}
	return flat, nil
}
