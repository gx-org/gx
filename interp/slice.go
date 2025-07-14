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
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

// Slice element storing a slice of elements.
type Slice struct {
	typ    ir.Type
	values []ir.Element
}

var (
	_ Slicer              = (*Slice)(nil)
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

func slice(fitp *FileScope, expr ir.AssignableExpr, index evaluator.NumericalElement, vals []ir.Element) (ir.Element, error) {
	i, err := elements.ConstantIntFromElement(index)
	if err != nil {
		return nil, err
	}
	if i < 0 || i >= len(vals) {
		return nil, errors.Errorf("invalid argument: index %d out of bounds [0:%d]", i, len(vals))
	}
	return vals[i], nil
}

// Slice of the tuple.
func (n *Slice) Slice(fitp *FileScope, expr *ir.IndexExpr, index evaluator.NumericalElement) (ir.Element, error) {
	return slice(fitp, expr, index, n.values)
}

// Type of the slice.
func (n *Slice) Type() ir.Type {
	return n.typ
}

// Kind of the element.
func (*Slice) Kind() ir.Kind {
	return ir.SliceKind
}

// Unflatten consumes the next handles to return a GX value.
func (n *Slice) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseComposite(flatten.ParseCompositeOf(values.NewSlice), n.typ, n.values)
}

// Expr returns the IR expression representing the slice.
func (n *Slice) Expr() (ir.AssignableExpr, error) {
	ext := &ir.SliceLitExpr{
		Typ:  n.typ,
		Elts: make([]ir.AssignableExpr, n.Len()),
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
func (n *Slice) Compare(x canonical.Comparable) bool {
	other, ok := x.(*Slice)
	if !ok {
		return false
	}
	if len(n.values) != len(other.values) {
		return false
	}
	for i, vi := range n.values {
		xComp, xOk := vi.(canonical.Canonical)
		yComp, yOk := other.values[i].(canonical.Canonical)
		if !xOk || !yOk {
			return false
		}
		if !xComp.Compare(yComp) {
			return false
		}
	}
	return true
}

// String returns a string representation of the slice.
func (n *Slice) String() string {
	return gxfmt.String(n.values)
}
