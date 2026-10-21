// Copyright 2026 Google LLC
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

package compeval

import (
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/compeval/cpevops"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates"
	"github.com/gx-org/gx/internal/interp/csteager"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/src/gx/ast"
)

// constant checks the arguments of operators.
// It returns a surrogate value if one of the argument is unknown.
type constant struct {
	ao  engine.Engine
	cst engine.Constant
}

var (
	_ engine.EvalShaper   = (*constant)(nil)
	_ cmp.Canonical       = (*constant)(nil)
	_ togo.CompileValue   = (*constant)(nil)
	_ engine.Element      = (*constant)(nil)
	_ flatten.Unflattener = (*constant)(nil)
)

func newConstant(ao engine.Engine, cst engine.Constant) engine.ConstantElement {
	return &constant{ao: ao, cst: cst}
}

func (c *constant) Constant() engine.Constant {
	return c.cst
}

func (c *constant) Type() ir.Type {
	return c.cst.Type()
}

// UnaryOp applies a unary operator on x.
func (c *constant) UnaryOp(env *engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	res, err := csteager.Unary(env, expr, c.cst)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return cpevops.NewUnary(env, expr, c)
	}
	return newConstant(c.ao, res), err
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (c *constant) BinaryOp(env *engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (engine.NumericalElement, error) {
	res, err := csteager.Binary(env, expr, c.cst, y)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return cpevops.NewBinaryFrom(env, expr, c, y)
	}
	return newConstant(c.ao, res), err
}

// Cast an element into a given data type.
func (c *constant) Cast(env *engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	res, err := csteager.Cast(env, expr, target, c.cst)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return cpevops.NewCast(env, expr, c, target), nil
	}
	return newConstant(c.ao, res), err
}

func (c *constant) toArraySurrogate(expr ir.Expr) (engine.NumericalElement, error) {
	path, err := storepath.NewUnique(c)
	if err != nil {
		return nil, err
	}
	return surrogates.NewArrayFrom(path, expr.Type())
}

// Reshape an element.
func (c *constant) Reshape(env *engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return c.toArraySurrogate(expr)
}

func (c *constant) SliceAt(env *engine.Env, expr *ir.IndexExpr, index engine.NumericalElement) (_ ir.Element, err error) {
	return c.toArraySurrogate(expr)
}

func (c *constant) Slice(env *engine.Env, expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return c.toArraySurrogate(expr)
}

// AlgExpr returns an algebraic expression.
func (c *constant) AlgExpr(eva ir.Evaluator) (cmp.Expr, error) {
	return cmp.ToAlgExpr(eva, c.cst)
}

func (c *constant) EvalShape() (shapes.Shape, error) {
	return cpevops.EvalShape(c.Type())
}

// Axes returns the axes of the value as a slice element.
func (c *constant) Axes(ev ir.Evaluator) (ir.FixedSlice, error) {
	return cpevops.AxesFromType(ev, c.Type())
}

func (c *constant) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	return c.cst.Expr(ev, src)
}

// GoValue of the underlying element.
func (c *constant) GoValue() (any, error) {
	return togo.CpValue(c.cst)
}

func (c *constant) toHostArray() (*hostio.HostArray, error) {
	val, err := c.GoValue()
	if err != nil {
		return nil, err
	}
	return hostio.FromAtom(val, c.Type())
}

// Unflatten returns the same constant.
func (c *constant) Unflatten(handles *flatten.Parser) (hostio.Value, error) {
	return c.toHostArray()
}

// Materialise returns the element with all its values from the graph.
func (c *constant) Materialise(mat materialise.Materialiser) (materialise.Node, error) {
	hostArray, err := c.toHostArray()
	if err != nil {
		return nil, err
	}
	return mat.NodeFromArray(hostArray)
}

func (c *constant) ShortString(from *ir.File) string {
	return c.cst.ShortString(from)
}

func (c *constant) SourceString(from *ir.File) string {
	return c.cst.SourceString(from)
}

func (c *constant) String() string {
	return c.cst.SourceString(nil)
}
