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
	"go/ast"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/interp/engine"
)

type (

	// Slicer is a state element that can be sliced.
	Slicer interface {
		SliceAt(expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error)
		Slice(expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error)
	}

	// ArraySlicer is a state element with an array that can be sliced.
	ArraySlicer interface {
		engine.NumericalElement
		SliceArray(expr ir.Expr, index engine.NumericalElement) (engine.NumericalElement, error)
		Type() ir.Type
	}

	// WithAxes is an element able to return its axes as a slice of element.
	WithAxes interface {
		Axes(ev ir.Evaluator) (*Slice, error)
	}

	// WithElements is an element able to returns the elements it contains.
	WithElements interface {
		Elements() []ir.Element
	}

	// Unpacker unpacks the elements it contains into a tuple.
	Unpacker interface {
		Unpack(ev ir.TypeCmp) (ir.Element, error)
	}
)

// Slice element storing a slice of elements.
type Slice struct {
	typ    ir.Type
	values []ir.Element
}

var (
	_ Slicer              = (*Slice)(nil)
	_ ir.FixedSlice       = (*Slice)(nil)
	_ ir.Element          = (*Slice)(nil)
	_ ir.Canonical        = (*Slice)(nil)
	_ canonical.Canonical = (*Slice)(nil)
	_ WithElements        = (*Slice)(nil)
	_ engine.Copier       = (*Slice)(nil)
	_ engine.Slice        = (*Slice)(nil)
)

// NewSlice returns a slice from a slice of elements.
func NewSlice(typ ir.Type, elements []ir.Element) (*Slice, error) {
	var err error
	if typ.Kind() != irkind.Slice {
		err = fmterr.Internal(errors.Errorf("cannot create a slice of type %s", typ.ReferString(nil)))
	}
	return &Slice{
		typ:    typ,
		values: elements,
	}, err
}

// Flatten returns the elements of the slice.
func (n *Slice) Flatten() ([]ir.Element, error) {
	return flatten.Flatten(n.values...)
}

// Slice returns a slice of a slice.
func (n *Slice) Slice(expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	lowIndex, err := n.lowBound(low)
	if err != nil {
		return nil, err
	}
	highIndex, err := n.highBound(high)
	if err != nil {
		return nil, err
	}
	return NewSlice(n.typ, n.values[lowIndex:highIndex])
}

func (n *Slice) highBound(high engine.NumericalElement) (int, error) {
	if high == nil {
		return len(n.values), nil
	}
	i, err := ConstantIntFromElement(high)
	if err != nil {
		return -1, err
	}
	if i < 0 || i > len(n.values) {
		return -1, errors.Errorf("higher slice bound out of range [:%d] with %d elements", i, len(n.values))
	}
	return i, nil
}

func (n *Slice) lowBound(low engine.NumericalElement) (int, error) {
	if low == nil {
		return 0, nil
	}
	i, err := ConstantIntFromElement(low)
	if err != nil {
		return -1, err
	}
	if i < 0 || i >= len(n.values) {
		return -1, errors.Errorf("lower slice bound out of range [%d:%d]", i, len(n.values))
	}
	return i, nil
}

// SliceAt returns the element at a given position in the slice.
func (n *Slice) SliceAt(expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
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

// Append elements to a copy of the slice.
func (n *Slice) Append(args []ir.Element) engine.Slice {
	values := append([]ir.Element{}, n.values...)
	values = append(values, args...)
	return &Slice{typ: n.typ, values: values}
}

// AppendInPlace appends an element to the slice without creating a copy of the slice.
func (n *Slice) AppendInPlace(el ir.Element) {
	n.values = append(n.values, el)
}

// Unflatten consumes the next handles to return a GX value.
func (n *Slice) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseComposite(flatten.ParseCompositeOf(values.NewSlice), n.typ, n.values)
}

// Expr returns the IR expression representing the slice.
func (n *Slice) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
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
		ext.Elts[i], err = ir.ToSingleExpr(ev, src, irEl)
		if err != nil {
			return nil, err
		}
	}
	return []ir.Expr{ext}, nil
}

// Elements stored in the slice.
func (n *Slice) Elements() []ir.Element {
	return n.values
}

// Unpack the values of the slice into a tuple.
func (n *Slice) Unpack(ev ir.TypeCmp) (ir.Element, error) {
	return TupleFromElements(n.values)
}

// Copy the slice.
func (n *Slice) Copy() engine.Copier {
	values := make([]ir.Element, len(n.values))
	for i, val := range n.values {
		values[i] = engine.Copy(val)
	}
	return &Slice{typ: n.typ, values: values}
}

// Length returns the value corresponding to calling the built-in len.
func (n *Slice) Length(ev ir.Evaluator) (int, error) {
	return n.Len(), nil
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

// GoValue of the underlying element.
func (n *Slice) GoValue() (any, error) {
	return MapSlice(togo.Value, n.values)
}

// ShortString returns a string representation of the slice.
func (n *Slice) ShortString() string {
	return n.String()
}

// String returns a string representation of the slice.
func (n *Slice) String() string {
	return gxfmt.String(n.values)
}

// ToWithElements returns the string value stored in a element.
func ToWithElements(el ir.Element) (WithElements, error) {
	under := Underlying(el)
	sl, ok := under.(WithElements)
	if !ok {
		return nil, errors.Errorf("cannot convert element %T (underlying: %T) to %s", el, under, reflect.TypeFor[WithElements]().String())
	}
	return sl, nil
}

// SliceFromElement returns a slice if the element is one, an error otherwise.
func SliceFromElement(el ir.Element) (*Slice, error) {
	under := Underlying(el)
	sl, ok := under.(*Slice)
	if !ok {
		return nil, errors.Errorf("cannot convert element %T (underlying: %T) to %s", el, under, reflect.TypeFor[WithElements]().String())
	}
	return sl, nil
}
