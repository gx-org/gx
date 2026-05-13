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

package interp

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// Tuple value grouping multiple values together.
type Tuple struct {
	elements []ir.Element
	typ      *ir.TupleType
}

var (
	_ elements.Slicer = (*Tuple)(nil)
	_ ir.Element      = (*Tuple)(nil)
	_ ir.TupleElement = (*Tuple)(nil)
	_ ir.Canonical    = (*Tuple)(nil)
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

// Expr converts a tuple of the form (Expr, error) to an expression.
func (n *Tuple) Expr(ev ir.Evaluator, src ast.Expr) (_ ir.Expr, cpErr ir.CompEvalError, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot convert tuple %s to an IR expression: %w", n.Type().ReferString(nil), err)
		}
	}()
	if len(n.elements) != 2 {
		return nil, nil, errors.Errorf("expect 2 elements but got %d", len(n.elements))
	}
	expr, cpErr, err := ir.ToExpr(ev, src, n.elements[0])
	if cpErr != nil || err != nil {
		return expr, cpErr, err
	}
	cpErr, err = ev.ToCompEvalError(src, n.elements[1])
	return expr, cpErr, err
}

// TupleElements returns the elements stored in the tuple.
func (n *Tuple) TupleElements() []ir.Element {
	return n.elements
}

// SliceAt of the tuple.
func (n *Tuple) SliceAt(expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return elements.SliceVals(expr, index, n.elements)
}

// Slice is not implemented for tuples.
func (n *Tuple) Slice(expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return n, errors.Errorf("not implemented for %T", n)
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
