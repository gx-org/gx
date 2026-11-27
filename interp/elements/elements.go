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

// Package elements provides generic elements, independent of the evaluator, for the interpreter.
package elements

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/constants"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/interp/engine"
)

// Map transforms a collection of element into a different type.
func Map[T any](f func(ir.Element) (T, error), el ir.Element) ([]T, error) {
	return coreiface.Map(f, el)
}

// ToWithElements returns the string value stored in a element.
func ToWithElements(el ir.Element) (coreiface.WithElements, error) {
	return coreiface.ToWithElements(el)
}

// IntFromElement converts an element to an int.
func IntFromElement(el ir.Element) (int, error) {
	return constants.Convert(constants.CInt, el)
}

// Int64FromElement converts an element to a bool.
func Int64FromElement(el ir.Element) (int64, error) {
	return constants.Convert(constants.CInt64, el)
}

// AxesFromElement returns a shape from a state element.
// An error is returned if a concrete shape cannot be returned.
func AxesFromElement(el ir.Element) ([]int, error) {
	slice, isSlice := el.(*Slice)
	if !isSlice {
		return nil, errors.Errorf("cannot convert %T to %s", el, reflect.TypeFor[*Slice]().String())
	}
	dimensions := make([]int, slice.Len())
	for i, dimElement := range slice.Elements() {
		var err error
		dimScalarI, err := IntFromElement(dimElement)
		if err != nil {
			return nil, err
		}
		dimensions[i] = dimScalarI
	}
	return dimensions, nil
}

// PackageVarSetElement is an option to set a package variable to an element.
type PackageVarSetElement struct {
	// Pck is the package owning the variable.
	Pkg string
	// Index of the variable in the package definition.
	Var string
	// Value of the static variable for the compiler.
	Value ir.Element
}

// Package for which the option has been built.
func (p PackageVarSetElement) Package() string {
	return p.Pkg
}

// String representation of the option.
func (p PackageVarSetElement) String() string {
	return fmt.Sprintf("%s.%s=%T:%v", p.Pkg, p.Var, p.Value, p.Value)
}

// SliceVals slices a slice of elements.
func SliceVals(expr ir.Expr, index engine.NumericalElement, vals []ir.Element) (ir.Element, error) {
	i, err := IntFromElement(index)
	if err != nil {
		return nil, err
	}
	if i < 0 || i >= len(vals) {
		return nil, errors.Errorf("invalid argument: index %d out of bounds [0:%d]", i, len(vals))
	}
	return vals[i], nil
}

// EvalRank evaluates an expression to build the rank of an array.
func EvalRank(ev ir.Evaluator, expr ir.Expr) (ir.ArrayRank, []cmp.Canonical, error) {
	rankVal, err := ev.EvalExpr(expr)
	if err != nil {
		return nil, nil, err
	}
	slice, err := cast.To[*Slice](rankVal)
	if err != nil {
		return nil, nil, fmterr.InternalAt(ev.File().FileSet(), expr.Node(), "cannot evaluate rank: %v", err)
	}
	axes := make([]ir.AxisLengths, slice.Len())
	cans := make([]cmp.Canonical, slice.Len())
	for i, el := range slice.Elements() {
		ex, ok := el.(cmp.Canonical)
		if !ok {
			return nil, nil, fmterr.InternalAt(ev.File().FileSet(), expr.Node(), "cannot build an axis expression from element %T: not supported", el)
		}
		irExpr, err := ir.ToSingleExpr(ev, expr.Expr(), ex)
		if err != nil {
			return nil, nil, err
		}
		axes[i] = &ir.AxisExpr{
			X: irExpr,
		}
		cans[i] = el.(cmp.Canonical)
	}
	return &ir.Rank{Ax: axes}, cans, nil
}

// ShapeFromElement returns the shape of a numerical element.
func ShapeFromElement(el ir.Element) (shapes.Shape, error) {
	shaper, err := coreiface.Cast[engine.EvalShaper](el)
	if err != nil {
		return shapes.Invalid(), err
	}
	return shaper.EvalShape()
}

// ToNumericalElement converts an element into a numerical element.
func ToNumericalElement(el ir.Element) (engine.NumericalElement, error) {
	return coreiface.Cast[engine.NumericalElement](el)
}

// Finalize the element.
// Named types, structures, and slices are walked to finalize their content.
// Other elements are finalized if they implement Finalizer.
func Finalize(el ir.Element) error {
	return Walk(func(el ir.Element) error {
		f, ok := el.(coreiface.Finalizer)
		if !ok {
			return nil
		}
		return f.Finalize()
	}, el)
}
