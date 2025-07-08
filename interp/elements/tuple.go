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
	"github.com/gx-org/gx/internal/interp/flatten"
)

// Tuple value grouping multiple values together.
type Tuple struct {
	elements []ir.Element
	typ      *ir.TupleType
}

var (
	_ Slicer  = (*Tuple)(nil)
	_ Element = (*Tuple)(nil)
)

// NewTuple returns a tuple to store the result of a function returning more than one value.
func NewTuple(values []ir.Element) *Tuple {
	return &Tuple{
		elements: values,
	}
}

// Flatten the tuple and all its elements.
func (n *Tuple) Flatten() ([]ir.Element, error) {
	return flatten.Flatten(n.elements...)
}

// Elements returns the elements stored in the tuple.
func (n *Tuple) Elements() []ir.Element {
	return n.elements
}

// Slice of the tuple.
func (n *Tuple) Slice(ctx ir.Evaluator, expr *ir.IndexExpr, index NumericalElement) (ir.Element, error) {
	return slice(ctx, expr, index, n.elements)
}

// Unflatten creates a GX value from the next handles available in the parser.
func (n *Tuple) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return nil, fmterr.Internal(errors.Errorf("%T does not support converting device handles into GX values", n))
}

// Type of the element.
func (n *Tuple) Type() ir.Type {
	if n.typ != nil {
		return n.typ
	}
	n.typ = &ir.TupleType{
		Types: make([]ir.Type, len(n.elements)),
	}
	for i, el := range n.elements {
		n.typ.Types[i] = el.Type()
	}
	return n.typ
}

func (n *Tuple) String() string {
	els := make([]string, len(n.elements))
	for i, el := range n.elements {
		els[i] = fmt.Sprint(el)
	}
	return fmt.Sprintf("(%s)", strings.Join(els, ", "))
}
