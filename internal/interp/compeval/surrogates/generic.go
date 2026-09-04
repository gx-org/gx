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

package surrogates

import (
	"go/ast"

	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevops"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

type genericType struct {
	path storepath.Path
	*ir.GenericTypeParam
}

var _ elements.Generic = (*genericType)(nil)

func newGenericType(path storepath.Path, typ *ir.GenericTypeParam) Element {
	return &genericType{
		path:             path,
		GenericTypeParam: typ,
	}
}

func (g *genericType) Type() ir.Type {
	return g.GenericTypeParam
}

func (g *genericType) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{g.path.Expr()}, nil
}

// UnaryOp applies a unary operator on x.
func (g *genericType) UnaryOp(env *engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	return cpevops.NewUnary(env, expr, g)
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (g *genericType) BinaryOp(env *engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewBinaryFrom(env, expr, g, y)
}

// Cast an element into a given data type.
func (g *genericType) Cast(env *engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return cpevops.NewCast(env, expr, g, target), nil
}

// SliceAt returns an element of the array given an index.
func (g *genericType) SliceAt(env *engine.Env, expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return cpevops.NewIndex(env, expr, g, index)
}

// Slice the array given a low and high value.
func (g *genericType) Slice(env *engine.Env, expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return cpevops.NewSlice(env, expr, g, low, high)
}

// Reshape an element.
func (g *genericType) Reshape(env *engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewReshape(env, expr, g, axisLengths)
}

// Axes of the result of the cast.
func (g *genericType) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	return cpevops.AxesFromType(ev, g.GenericTypeParam.Type())
}

func (g *genericType) EvalShape() (*shape.Shape, error) {
	return cpevops.EvalShape(g.GenericTypeParam.Type())
}

func (g *genericType) ShortString(from *ir.File) string {
	return g.SourceString(from)
}

func (g *genericType) SourceString(from *ir.File) string {
	return g.path.Expr().SourceString(from)
}
