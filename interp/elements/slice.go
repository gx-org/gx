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
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/evaluator"
)

type (

	// Slicer is a state element that can be sliced.
	Slicer interface {
		Slice(expr *ir.IndexExpr, index evaluator.NumericalElement) (ir.Element, error)
	}

	// ArraySlicer is a state element with an array that can be sliced.
	ArraySlicer interface {
		evaluator.NumericalElement
		SliceArray(expr ir.Expr, index evaluator.NumericalElement) (evaluator.NumericalElement, error)
		Type() ir.Type
	}

	// WithAxes is an element able to return its axes as a slice of element.
	WithAxes interface {
		Axes(ev ir.Evaluator) (*Slice, error)
	}

	// FixedSlice is a slice where the number of elements is known.
	FixedSlice interface {
		ir.Element
		Elements() []ir.Element
		Len() int
	}
)

// Slice element storing a slice of elements.
type Slice struct {
	typ    ir.Type
	values []ir.Element
}

var (
	_ Slicer              = (*Slice)(nil)
	_ FixedSlice          = (*Slice)(nil)
	_ ir.Element          = (*Slice)(nil)
	_ ir.Canonical        = (*Slice)(nil)
	_ canonical.Canonical = (*Slice)(nil)
)

// NewSlice returns a slice from a slice of elements.
func NewSlice(typ ir.Type, elements []ir.Element) *Slice {
	return &Slice{
		typ:    typ,
		values: elements,
	}
}

// Flatten returns the elements of the slice.
func (n *Slice) Flatten() ([]ir.Element, error) {
	return flatten.Flatten(n.values...)
}

// Slice of the tuple.
func (n *Slice) Slice(expr *ir.IndexExpr, index evaluator.NumericalElement) (ir.Element, error) {
	return SliceVals(expr, index, n.values)
}

// Type of the slice.
func (n *Slice) Type() ir.Type {
	return n.typ
}

// Kind of the element.
func (*Slice) Kind() irkind.Kind {
	return irkind.Slice
}

// Unflatten consumes the next handles to return a GX value.
func (n *Slice) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseComposite(flatten.ParseCompositeOf(values.NewSlice), n.typ, n.values)
}

// Expr returns the IR expression representing the slice.
func (n *Slice) Expr() (ir.Expr, error) {
	ext := &ir.SliceLitExpr{
		Typ:  n.typ,
		Elts: make([]ir.Expr, n.Len()),
	}
	for i, el := range n.values {
		irEl, ok := el.(ir.Canonical)
		if !ok {
			return nil, errors.Errorf("cannot build an IR expression from %T", el)
		}
		var err error
		ext.Elts[i], err = irEl.Expr()
		if err != nil {
			return nil, err
		}
	}
	return ext, nil
}

// Elements stored in the slice.
func (n *Slice) Elements() []ir.Element {
	return n.values
}

// Len returns the number of elements in the slice.
func (n *Slice) Len() int {
	return len(n.values)
}

// Compare the slice to another canonical value.
func (n *Slice) Compare(x canonical.Comparable) (bool, error) {
	other, ok := x.(*Slice)
	if !ok {
		return false, nil
	}
	if len(n.values) != len(other.values) {
		return false, nil
	}
	for i, vi := range n.values {
		xComp, xOk := vi.(canonical.Canonical)
		yComp, yOk := other.values[i].(canonical.Canonical)
		if !xOk || !yOk {
			return false, nil
		}
		cmp, err := xComp.Compare(yComp)
		if err != nil {
			return false, err
		}
		if !cmp {
			return false, nil
		}
	}
	return true, nil
}

// ShortString returns a string representation of the slice.
func (n *Slice) ShortString() string {
	return n.String()
}

// String returns a string representation of the slice.
func (n *Slice) String() string {
	return gxfmt.String(n.values)
}
