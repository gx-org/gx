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

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// InputElements is the receiver and arguments with which the function was called.
type InputElements struct {
	// Values are the initial input GX values passed to the function call
	// before they were encapsulated in elements for the interpreter.
	Values InputValues

	// Receiver on which the function call was done.
	// Can be nil.
	Receiver Element

	// Args returns list of arguments passed to the interpreter at call time.
	Args []Element
}

// InputValues are GX values passed to the function call.
type InputValues struct {
	// Receiver on which the function call was done.
	// Can be nil.
	Receiver values.Value

	// Args returns list of arguments passed to the interpreter at call time.
	Args []values.Value
}

type (
	// Element in the state.
	Element interface {
		// Flatten the element, that is:
		// returns itself if the element is atomic,
		// returns its components if the element is a composite.
		Flatten() ([]Element, error)

		// Unflatten creates a GX value from the next handles available in the Unflattener.
		Unflatten(handles *Unflattener) (values.Value, error)

		// Kind returns the kind of the element being stored.
		Kind() ir.Kind
	}

	// Copier is an interface implemented by nodes that need to be copied when passed to a function.
	Copier interface {
		Element
		Copy() Copier
	}

	// Selector selects a field given its index.
	Selector interface {
		Select(SelectAt) (Element, error)
	}

	// Slicer is a state element that can be sliced.
	Slicer interface {
		Slice(ctx FileContext, expr ir.AssignableExpr, index NumericalElement) (Element, error)
	}

	// ArraySlicer is a state element with an array that can be sliced.
	ArraySlicer interface {
		NumericalElement
		SliceArray(ctx FileContext, expr ir.AssignableExpr, index NumericalElement) (NumericalElement, error)
		Type() ir.Type
	}
)

type (
	// NodeFile is an expression with the file in which it is declared.
	NodeFile[T ir.Node] struct {
		file *ir.File
		node T
	}

	// NodeAt is a generic GX node.
	NodeAt = NodeFile[ir.Node]

	// ExprAt is a generic GX expression.
	ExprAt = NodeFile[ir.AssignableExpr]

	// CallAt is a function call GX expression.
	CallAt = NodeFile[*ir.CallExpr]

	// FieldAt is a typed field at a given position.
	FieldAt = NodeFile[*ir.Field]

	// SelectAt is a typed field at a given position.
	SelectAt = NodeFile[*ir.SelectorExpr]

	// ValueAt is a generic GX expression.
	ValueAt = NodeFile[ir.Value]

	// StorageAt is a generic GX expression.
	StorageAt = NodeFile[ir.Storage]
)

// NewNodeAt returns a new expression at a given position.
func NewNodeAt[T ir.Node](file *ir.File, expr T) NodeFile[T] {
	return NodeFile[T]{file: file, node: expr}
}

// NewValueAt returns a new expression at a given position.
func NewValueAt(file *ir.File, expr ir.Value) ValueAt {
	return NewNodeAt(file, expr)
}

// NewExprAt returns a new expression at a given position.
func NewExprAt(file *ir.File, expr ir.AssignableExpr) ExprAt {
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

// ExprSrc returns the source expression.
func (ea NodeFile[T]) ExprSrc() ast.Expr {
	var node any = ea.node
	src := node.(ir.SourceNode).Source()
	if src == nil {
		return nil
	}
	return src.(ast.Expr)
}

// NodeFile returns a general node.
func (ea NodeFile[T]) NodeFile() NodeFile[ir.Node] {
	return NodeFile[ir.Node]{file: ea.file, node: ea.node}
}

// ToNodeAt converts a type position into a generic node position.
func (ea NodeFile[T]) ToNodeAt() NodeAt {
	return NewNodeAt[ir.Node](ea.file, ea.node)
}

// ToExprAt converts a type position into a generic node position.
func (ea NodeFile[T]) ToExprAt() ExprAt {
	node := any(ea.node)
	return NewNodeAt(ea.file, node.(ir.AssignableExpr))
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
	var node ir.Node = ea.node
	return fmt.Sprintf("%s%s",
		fmterr.PosString(ea.file.FileSet(), node.(ir.SourceNode).Source().Pos()),
		gxfmt.String(ea.node),
	)
}

// AxesFromElement returns a shape from a state element.
// An error is returned if a concrete shape cannot be returned.
func AxesFromElement(el Element) ([]int, error) {
	dimElements, err := el.Flatten()
	if err != nil {
		return nil, err
	}
	dimensions := make([]int, len(dimElements))
	for i, dimElement := range dimElements {
		var err error
		dimScalarI, err := ConstantIntFromElement(dimElement)
		if err != nil {
			return nil, err
		}
		dimensions[i] = dimScalarI
	}
	return dimensions, nil
}

// ShapeFromElement returns the shape of a numerical element.
func ShapeFromElement(node Element) (*shape.Shape, error) {
	numerical, ok := node.(NumericalElement)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to a numerical element", node)
	}
	return numerical.Shape(), nil
}

// ConstantScalarFromElement returns a scalar on a host given an element.
func ConstantScalarFromElement[T dtype.GoDataType](el Element) (val T, err error) {
	var hostArray *values.HostArray
	hostArray = ConstantFromElement(el)
	if hostArray == nil {
		err = errors.Errorf("state element %T does not store a constant numerical value", el)
		return
	}
	return values.ToAtom[T](hostArray)
}

// ConstantIntFromElement returns a scalar on a host given an element.
func ConstantIntFromElement(el Element) (val int, err error) {
	var hostArray *values.HostArray
	hostArray = ConstantFromElement(el)
	if hostArray == nil {
		err = errors.Errorf("state element %T does not store a constant numerical value", el)
		return
	}
	return toGoInt(hostArray)
}

func toGoInt(val *values.HostArray) (int, error) {
	valT := val.Shape().DType
	switch valT {
	case dtype.Int32:
		i32, err := values.ToAtom[int32](val)
		if err != nil {
			return 0, err
		}
		return int(i32), nil
	case dtype.Int64:
		i64, err := values.ToAtom[int64](val)
		if err != nil {
			return 0, err
		}
		return int(i64), nil
	default:
		return -1, errors.Errorf("cannot cast type %s to int", valT.String())
	}
}

// ConstantFromElement returns the host value represented by an element.
// The function returns (nil, nil) if the element does not host a numerical value.
func ConstantFromElement(el Element) *values.HostArray {
	numerical, ok := el.(ElementWithConstant)
	if !ok {
		return nil
	}
	return numerical.NumericalConstant()
}

// HostValueFromContext returns a host value from the function call.
func HostValueFromContext(ci *InputValues, el Element) (*values.HostArray, error) {
	withValue, ok := el.(ElementWithArrayFromContext)
	if !ok {
		return nil, errors.Errorf("state element %T does not support returning a value given a context", el)
	}
	array, err := withValue.ArrayFromContext(ci)

	if err != nil {
		return nil, err
	}
	return array.ToHostArray(kernels.Allocator())
}

// PackageVarSetElement is an option to set a package variable to an element.
type PackageVarSetElement struct {
	// Pck is the package owning the variable.
	Pkg string
	// Index of the variable in the package definition.
	Var string
	// Value of the static variable for the compiler.
	Value Element
}

// Package for which the option has been built.
func (p PackageVarSetElement) Package() string {
	return p.Pkg
}

type (
	// Receiver of a function.
	Receiver struct {
		Ident   *ast.Ident
		Element *NamedType
	}

	// Func is an element owning a callable function.
	Func interface {
		Element
		Func() ir.Func
		Recv() *Receiver
		Call(ctx FileContext, call *ir.CallExpr, args []Element) ([]Element, error)
	}

	// NewFunc creates function elements from function IRs.
	NewFunc func(ir.Func, *Receiver) Func
)

// NewReceiver returns a new receiver given a function definition and the element representing the receiver.
func NewReceiver(el *NamedType, fn ir.Func) *Receiver {
	if el == nil {
		return nil
	}
	names := fn.FuncType().Receiver.Src.List[0].Names
	var name *ast.Ident
	if len(names) > 0 {
		name = names[0]
	}
	return &Receiver{
		Ident:   name,
		Element: el,
	}
}

// Underlying returns the underlying element.
func Underlying(val Element) Element {
	named, ok := val.(*NamedType)
	if !ok {
		return val
	}
	return Underlying(named.under)
}

// FuncDeclFromElement extracts a function declaration from an element.
func FuncDeclFromElement(el Element) (*ir.FuncDecl, error) {
	fEl, ok := el.(Func)
	if !ok {
		return nil, errors.Errorf("cannot convert element %T to a function", el)
	}
	fun := fEl.Func()
	fDecl, ok := fun.(*ir.FuncDecl)
	if !ok {
		return nil, errors.Errorf("%s is not a GX user function", fun.Name())
	}
	return fDecl, nil
}

// StringFromElement returns the string value stored in a element.
func StringFromElement(el Element) (string, error) {
	sEl, ok := el.(*String)
	if !ok {
		return "", errors.Errorf("cannot convert element %T is not a string literal", el)
	}
	return sEl.StringValue().String(), nil
}
