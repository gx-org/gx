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
	"go/ast"
	"go/token"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/constants"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/interp/engine"
)

// InputElements is the receiver and arguments with which the function was called.
type InputElements struct {
	// Values are the initial input GX values passed to the function call
	// before they were encapsulated in elements for the interpreter.
	Values values.FuncInputs

	// Receiver on which the function call was done.
	// Can be nil.
	Receiver ir.Element

	// Args returns list of arguments passed to the interpreter at call time.
	Args []ir.Element
}

type (
	// NodeFile is an expression with the file in which it is declared.
	NodeFile[T ir.IR] struct {
		file *ir.File
		node T
	}

	// NodeAt is a generic GX node.
	NodeAt = NodeFile[ir.IR]

	// ExprAt is a generic GX expression.
	ExprAt = NodeFile[ir.Expr]

	// CallAt is a function call GX expression.
	CallAt = NodeFile[*ir.FuncCallExpr]

	// FieldAt is a typed field at a given position.
	FieldAt = NodeFile[*ir.Field]

	// SelectAt is a typed field at a given position.
	SelectAt = NodeFile[*ir.SelectorExpr]

	// ValueAt is a generic GX expression.
	ValueAt = NodeFile[ir.Value]

	// StorageAt is a generic GX expression.
	StorageAt = NodeFile[ir.Storage]
)

// Map transforms a collection of element into a different type.
func Map[T any](f func(ir.Element) (T, error), el ir.Element) ([]T, error) {
	return coreiface.Map(f, el)
}

// ToWithElements returns the string value stored in a element.
func ToWithElements(el ir.Element) (coreiface.WithElements, error) {
	return coreiface.ToWithElements(el)
}

// NewNodeAt returns a new expression at a given position.
func NewNodeAt[T ir.IR](file *ir.File, expr T) NodeFile[T] {
	return NodeFile[T]{file: file, node: expr}
}

// NewValueAt returns a new expression at a given position.
func NewValueAt(file *ir.File, expr ir.Value) ValueAt {
	return NewNodeAt(file, expr)
}

// NewExprAt returns a new expression at a given position.
func NewExprAt(file *ir.File, expr ir.Expr) ExprAt {
	return NewNodeAt(file, expr)
}

// FSet returns the fileset of the expression.
func (ea NodeFile[T]) FSet() *token.FileSet {
	return ea.file.Package.FSet
}

// Node returns the expression.
func (ea NodeFile[T]) Node() T {
	return ea.node
}

// Source of the node.
func (ea NodeFile[T]) Source() ast.Node {
	var node ir.IR = ea.node
	return node.(ir.Node).Node()
}

// ExprSrc returns the source expression.
func (ea NodeFile[T]) ExprSrc() ast.Expr {
	var node any = ea.node
	return node.(ir.Expr).Expr()
}

// NodeFile returns a general node.
func (ea NodeFile[T]) NodeFile() NodeFile[ir.IR] {
	return NodeFile[ir.IR]{file: ea.file, node: ea.node}
}

// ToNodeAt converts a type position into a generic node position.
func (ea NodeFile[T]) ToNodeAt() NodeAt {
	return NewNodeAt[ir.IR](ea.file, ea.node)
}

// ToExprAt converts a type position into a generic node position.
func (ea NodeFile[T]) ToExprAt() ExprAt {
	node := any(ea.node)
	return NewNodeAt(ea.file, node.(ir.Expr))
}

// ToValueAt converts a type position into a generic node position.
func (ea NodeFile[T]) ToValueAt() ValueAt {
	node := any(ea.node)
	return NewNodeAt[ir.Value](ea.file, node.(ir.Value))
}

// File returns the file in which the expression is declared.
func (ea NodeFile[T]) File() *ir.File {
	return ea.file
}

// String representation of the node in the source code.
func (ea NodeFile[T]) String() string {
	var node ir.IR = ea.node
	return fmt.Sprintf("%s%s",
		fmterr.At(ea.file.FileSet(), node.(ir.Node).Node()).String(),
		gxfmt.String(ea.node),
	)
}

// BoolFromElement converts an element to a bool.
func BoolFromElement(el ir.Element) (bool, error) {
	return constants.Convert(constants.CBool, el)
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
func ShapeFromElement(el ir.Element) (*shape.Shape, error) {
	shaper, err := coreiface.Cast[EvalShaper](el)
	if err != nil {
		return nil, err
	}
	return shaper.EvalShape()
}

// ToNumericalElement converts an element into a numerical element.
func ToNumericalElement(el ir.Element) (engine.NumericalElement, error) {
	return coreiface.Cast[engine.NumericalElement](el)
}

// StringFrom converts an element into fmt.Stringer.
func StringFrom(el ir.Element) (string, error) {
	stringer, err := cast.To[fmt.Stringer](el)
	if err != nil {
		return "", err
	}
	return stringer.String(), nil
}
