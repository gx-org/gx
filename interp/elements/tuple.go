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
	"go/ast"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/engine"
)

// Tuple value grouping multiple values together.
type Tuple struct {
	typ      *ir.TupleType
	elements []ir.Element
}

var (
	_ Slicer          = (*Tuple)(nil)
	_ ir.Element      = (*Tuple)(nil)
	_ ir.TupleElement = (*Tuple)(nil)
	_ ir.Canonical    = (*Tuple)(nil)
)

// TupleFromElements creates a new tuple from a slice of elements.
// The type of the tuple is inferred from the elements.
// Such tuple can be of any size (for example, a slice can unpack into a tuple with no element).
func TupleFromElements(elts []ir.Element) (*Tuple, error) {
	types := make([]ir.Type, len(elts))
	for i, val := range elts {
		types[i] = val.Type()
	}
	return &Tuple{
		typ:      &ir.TupleType{Types: types},
		elements: elts,
	}, nil
}

// NewTuple returns a tuple to store the result of a function returning more than one value.
func NewTuple(ftyp *ir.FuncType, values []ir.Element) (*Tuple, error) {
	resultFields := ftyp.Results.Fields()
	tupleType := &ir.TupleType{
		Types: make([]ir.Type, len(resultFields)),
	}
	for i, field := range resultFields {
		tupleType.Types[i] = field.Type()
	}
	n := &Tuple{typ: tupleType, elements: values}
	if len(n.typ.Types) <= 1 {
		return n, fmterr.Internal(errors.Errorf("tuple of less than one element"))
	}
	if len(n.typ.Types) != len(n.elements) {
		return n, fmterr.Internal(errors.Errorf("got tuple of %d elements but type %s composed of %d types", len(values), n.typ.ReferString(nil), len(n.typ.Types)))
	}
	return n, nil
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
func (n *Tuple) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	var exprs []ir.Expr
	for _, el := range n.elements {
		elExprs, err := ir.ToExpr(ev, src, el)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, elExprs...)
	}
	return exprs, nil
}

// TupleElements returns the elements stored in the tuple.
func (n *Tuple) TupleElements() []ir.Element {
	return n.elements
}

// SliceAt of the tuple.
func (n *Tuple) SliceAt(expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return SliceVals(expr, index, n.elements)
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

// UnpackError unpacks an error and its value.
func (n *Tuple) UnpackError(ev ir.TypeCmp) (ir.Element, ir.Element, error) {
	lastType := n.typ.Types[len(n.typ.Types)-1]
	if ok, err := lastType.AssignableTo(ev, ir.ErrorType()); !ok || err != nil {
		return n, nil, err
	}
	last := n.elements[len(n.elements)-1]
	withoutLast := n.elements[:len(n.elements)-1]
	if len(withoutLast) == 1 {
		return withoutLast[0], last, nil
	}
	return &Tuple{
		typ: &ir.TupleType{
			Types: n.typ.Types[:len(n.elements)-1],
		},
		elements: n.elements[:len(n.elements)-1],
	}, last, nil
}

func (n *Tuple) String() string {
	els := make([]string, len(n.elements))
	for i, el := range n.elements {
		els[i] = fmt.Sprint(el)
	}
	return fmt.Sprintf("(%s)", strings.Join(els, ", "))
}
