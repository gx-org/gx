// Copyright 2025 Google LLC
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

package cpevelements

import (
	"fmt"
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/compeval/cpevops"
	"github.com/gx-org/gx/internal/interp/proxies"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

type proxy struct {
	canonical.AtomStringImpl
	src elements.StorageAt
}

var (
	_ cpevops.Element       = (*proxy)(nil)
	_ ir.StorageElement     = (*proxy)(nil)
	_ elements.WithAxes     = (*proxy)(nil)
	_ ir.Canonical          = (*proxy)(nil)
	_ elements.Slicer       = (*proxy)(nil)
	_ engine.Slice          = (*proxy)(nil)
	_ elements.Selector     = (*proxy)(nil)
	_ ir.WithStore          = (*proxy)(nil)
	_ elements.WithElements = (*proxy)(nil)
	_ proxies.Proxy         = (*proxy)(nil)
	_ elements.Unpacker     = (*proxy)(nil)
	_ elements.EvalShaper   = (*proxy)(nil)
)

// NewProxy returns a new variable element given a GX variable name.
func NewProxy(src elements.StorageAt) ir.Element {
	return newProxy(src)
}

func newProxy(src elements.StorageAt) *proxy {
	return &proxy{src: src}
}

// UnaryOp applies a unary operator on x.
func (a *proxy) UnaryOp(env engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	return cpevops.NewUnary(env, expr, a)
}

// BinaryOp applies a binary operator to x and y.
func (a *proxy) BinaryOp(env engine.Env, expr *ir.BinaryExpr, x, y engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewBinary(env, expr, x, y)
}

// Cast an element into a given data type.
func (a *proxy) Cast(env engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return cpevops.NewCast(env, expr, a, target)
}

// Reshape the variable into a different shape.
func (a *proxy) Reshape(env engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewReshape(env, expr, a, axisLengths)
}

// Append elements to the slice.
func (a *proxy) Append(*ir.FuncCallExpr, []ir.Element) engine.Slice {
	stor := &ir.LocalVarStorage{Typ: a.Type()}
	storage := elements.NewNodeAt[ir.Storage](a.src.File(), stor)
	return &proxy{src: storage}
}

// Shape of the value represented by the element.
func (a *proxy) EvalShape() (*shape.Shape, error) {
	return &shape.Shape{}, nil
}

// Select a member.
func (a *proxy) Select(expr *ir.SelectorExpr) (ir.Element, error) {
	method, field := expr.Select(a.Type())
	if method == nil && field == nil {
		return nil, errors.Errorf("%s is an invalid member", expr.SourceString(nil))
	}
	return NewRuntimeValue(a.src.File(), expr)
}

// Store returns the storage represented by this variable.
func (a *proxy) Store() ir.Storage {
	return a.src.Node()
}

// Type of the element.
func (a *proxy) Type() ir.Type {
	return a.src.Node().Type()
}

// Unpack the proxy value.
func (a *proxy) Unpack(ev ir.TypeCmp) (ir.Element, error) {
	return a, nil
}

// Axes returns the axes of the value as a slice element.
func (a *proxy) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	return cpevops.AxesFromType(ev, a.src.Node().Type())
}

// SliceAt computes a slice from the variable.
func (a *proxy) SliceAt(expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return NewRuntimeValue(a.src.File(), expr)
}

func (a *proxy) Slice(expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return NewRuntimeValue(a.src.File(), expr)
}

// Compare to another element.
func (a *proxy) Compare(x canonical.Comparable) (bool, error) {
	other, ok := x.(*proxy)
	if !ok {
		return false, nil
	}
	return a.src.Node().Same(other.src.Node()), nil
}

// Expr returns the IR expression represented by the variable.
func (a *proxy) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{&ir.Ident{
		Src:  a.src.Node().NameDef(),
		Stor: a.src.Node(),
	}}, nil
}

func (a *proxy) Elements() []ir.Element {
	return []ir.Element{a}
}

func (a *proxy) CanonicalExpr() canonical.Canonical {
	return a
}

func (a *proxy) IsProxy() bool {
	return true
}

func (a *proxy) ShortString() string {
	return a.SourceString(nil)
}

func (a *proxy) SourceString(from *ir.File) string {
	return a.src.Node().NameDef().Name
}

func (a *proxy) String() string {
	namePrefix := "_"
	nameDef := a.src.Node().NameDef()
	if nameDef != nil {
		namePrefix = nameDef.Name
	}
	return fmt.Sprintf("(proxy:%s %s)", namePrefix, a.Type().ReferString(nil))
}
