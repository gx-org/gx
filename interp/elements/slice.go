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
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/hostio"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/algexpr"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/src/gx/ast"
)

type (
	// ArraySlicer is a state element with an array that can be sliced.
	ArraySlicer interface {
		engine.NumericalElement
		SliceArray(expr ir.Expr, index engine.NumericalElement) (engine.NumericalElement, error)
		Type() ir.Type
	}

	// Unpacker unpacks the elements it contains into a tuple.
	Unpacker interface {
		Unpack(ev ir.Evaluator) (ir.Element, error)
	}

	// ISlice is an interface implemented by all slice elements.
	ISlice interface {
		ir.Element
		ir.WithExpr
		ir.WithLength
		cmp.Canonical
		engine.Slicer
		engine.Slice
		Unpacker
	}
)

// Slice element storing a slice of elements.
type Slice struct {
	typ    ir.Type
	values []ir.Element
}

var (
	_ ISlice                 = (*Slice)(nil)
	_ ir.FixedSlice          = (*Slice)(nil)
	_ coreiface.WithElements = (*Slice)(nil)
	_ engine.Copier          = (*Slice)(nil)
	_ hostio.Value           = (*Slice)(nil)
)

// NewSlice returns a slice from a slice of elements.
func NewSlice(typ ir.Type, elements []ir.Element) (*Slice, error) {
	var err error
	if typ.Kind() != irkind.Slice {
		err = fmterr.Internal(errors.Errorf("cannot create a slice of type %s", typ.ReferString(nil)))
	}
	n := &Slice{typ: typ}
	if err != nil {
		return n, err
	}
	for _, el := range elements {
		n.AppendInPlace(el)
	}
	return n, nil
}

// Flatten returns the elements of the slice.
func (n *Slice) Flatten() ([]ir.Element, error) {
	return flatten.Flatten(n.values...)
}

// Slice returns a slice of a slice.
func (n *Slice) Slice(_ *engine.Env, expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
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
	i, err := IntFromElement(high)
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
	i, err := IntFromElement(low)
	if err != nil {
		return -1, err
	}
	if i < 0 || i >= len(n.values) {
		return -1, errors.Errorf("lower slice bound out of range [%d:%d]", i, len(n.values))
	}
	return i, nil
}

// SliceAt returns the element at a given position in the slice.
func (n *Slice) SliceAt(_ *engine.Env, expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
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
func (n *Slice) Append(_ *ir.FuncCallExpr, args []ir.Element) engine.Slice {
	values := append([]ir.Element{}, n.values...)
	r := &Slice{typ: n.typ, values: values}
	for _, el := range args {
		r.AppendInPlace(el)
	}
	return r
}

// AppendInPlace appends an element to the slice without creating a copy of the slice.
func (n *Slice) AppendInPlace(el ir.Element) {
	n.values = append(n.values, unpack(el)...)
}

// Unflatten consumes the next handles to return a GX value.
func (n *Slice) Unflatten(handles *flatten.Parser) (hostio.Value, error) {
	return handles.ParseComposite(flatten.ParseCompositeOf(newSliceFromValues), n.typ, n.values)
}

// newSliceFromValues returns a slice from a slice of GX values.
func newSliceFromValues(typ ir.Type, vals []hostio.Value) (*Slice, error) {
	return NewSlice(typ, hostio.ToElements(vals))
}

// gxValues returns the elements of the slice as GX values.
// An error is returned if an element is not a GX value.
func (n *Slice) gxValues() ([]hostio.Value, error) {
	vals := make([]hostio.Value, len(n.values))
	for i, el := range n.values {
		var err error
		if vals[i], err = cast.To[hostio.Value](el); err != nil {
			return nil, err
		}
	}
	return vals, nil
}

// ToHost transfers all the elements of the slice to the host.
func (n *Slice) ToHost(alloc kernels.Allocator) (hostio.Value, error) {
	vals, err := n.gxValues()
	if err != nil {
		return nil, err
	}
	hostVals, err := hostio.ToHost(alloc, vals)
	if err != nil {
		return nil, err
	}
	return &Slice{typ: n.typ, values: hostio.ToElements(hostVals)}, nil
}

// Finalize releases the resources allocated for the slice elements.
func (n *Slice) Finalize() error {
	vals, err := n.gxValues()
	if err != nil {
		return err
	}
	return hostio.Finalize(vals...)
}

// Expr returns the IR expression representing the slice.
func (n *Slice) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	ext := &ir.SliceLitExpr{
		Typ:  n.typ,
		Elts: make([]ir.Expr, n.Len()),
	}
	for i, el := range n.values {
		var err error
		ext.Elts[i], err = ir.ToSingleExpr(ev, src, el)
		if err != nil {
			return nil, err
		}
	}
	return []ir.Expr{ext}, nil
}

// AlgExpr returns an algebraic expression.
func (n *Slice) AlgExpr(eva ir.Evaluator) (cmp.Expr, error) {
	els := make([]cmp.Expr, len(n.values))
	for i, el := range n.values {
		var err error
		els[i], err = cmp.ToAlgExpr(eva, el)
		if err != nil {
			return nil, err
		}
	}
	return algexpr.NewSlice(n.Type(), els), nil
}

// Elements stored in the slice.
func (n *Slice) Elements() []ir.Element {
	return n.values
}

// Unpack the values of the slice into a tuple.
func (n *Slice) Unpack(ev ir.Evaluator) (ir.Element, error) {
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

// Set an element at a given index.
// Returns an error if the index is out of bounds.
func (n *Slice) Set(index ir.Element, val ir.Element) error {
	i, err := IntFromElement(index)
	if err != nil {
		return err
	}
	if i < 0 || i >= len(n.values) {
		return errors.Errorf("invalid argument: index %d out of bounds [0:%d]", i, len(n.values))
	}
	n.values[i] = val
	return nil
}

// Length returns the value corresponding to calling the built-in len.
func (n *Slice) Length(ev ir.Evaluator) (int, error) {
	l := 0
	for _, el := range n.Elements() {
		if el.Type().Kind() != irkind.Tuple {
			l++
			continue
		}
		tuple, err := cast.ToIfDef[*Tuple](el)
		if err != nil {
			return -1, err
		}
		l += len(tuple.Elements())
	}
	return l, nil
}

// Len returns the number of elements in the slice.
func (n *Slice) Len() int {
	return len(n.values)
}

// GoValue of the underlying element.
func (n *Slice) GoValue() (any, error) {
	return coreiface.MapSlice(togo.CpValue, n.values)
}

func (n *Slice) hasCompositeElements() bool {
	sliceType, ok := ir.Underlying(n.typ).(*ir.SliceType)
	if !ok {
		return false
	}
	elt, ok := sliceType.ElementType()
	if !ok {
		return false
	}
	switch elt.Kind() {
	case irkind.Array, irkind.Slice, irkind.Struct:
		return true
	default:
		return false
	}
}

// ShortString returns the values of the slice as a string without the type.
func (n *Slice) ShortString(from *ir.File) string {
	if len(n.values) == 0 {
		return "{}"
	}
	if n.hasCompositeElements() {
		var bld strings.Builder
		bld.WriteString("{\n")
		for _, el := range n.values {
			bld.WriteString(gxfmt.Indent(ir.ShortString(from, el)))
			bld.WriteString(",\n")
		}
		bld.WriteString("}")
		return bld.String()
	}
	ss := make([]string, len(n.values))
	for i, el := range n.values {
		ss[i] = ir.ShortString(from, el)
	}
	return "{" + strings.Join(ss, ", ") + "}"
}

// SourceString returns the GX source code of the implementation.
func (n *Slice) SourceString(from *ir.File) string {
	return n.typ.ReferString(from) + n.ShortString(from)
}

// String returns a string representation of the slice.
func (n *Slice) String() string {
	return n.SourceString(nil)
}

// SliceFromElement returns a slice if the element is one, an error otherwise.
func SliceFromElement(el ir.Element) (*Slice, error) {
	under, err := coreiface.Underlying(el)
	if err != nil {
		return nil, err
	}
	return cast.To[*Slice](under)
}

// Flatten replaces the slices in a list of values by their elements, recursively.
// An error is returned if an element of a slice is not a GX value.
func Flatten(vals []hostio.Value) ([]hostio.Value, error) {
	var flat []hostio.Value
	for _, val := range vals {
		slice, ok := val.(*Slice)
		if !ok {
			flat = append(flat, val)
			continue
		}
		sliceVals, err := slice.gxValues()
		if err != nil {
			return nil, err
		}
		flatSlice, err := Flatten(sliceVals)
		if err != nil {
			return nil, err
		}
		flat = append(flat, flatSlice...)
	}
	return flat, nil
}
