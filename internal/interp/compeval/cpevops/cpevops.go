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

// Package cpevops provides core operators for numerical elements.
package cpevops

import (
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// Element returned after an evaluation at compeval.
type Element interface {
	engine.NumericalElement
	ir.WithExpr
	ir.StringShorter
	ir.StringSourcer
	elements.WithAxes
	elements.EvalShaper
	elements.Slicer
}

// AxesFromType returns a slice element of axis lengths from an array type.
func AxesFromType(ev ir.Evaluator, typ ir.Type) (*elements.Slice, error) {
	aTyp, ok := typ.(ir.ArrayType)
	if !ok {
		return nil, nil
	}
	rank := aTyp.Rank()
	axes := rank.Axes()
	elts := make([]ir.Element, len(axes))
	for i, ax := range axes {
		var err error
		elts[i], err = ev.EvalExpr(ax.AsExpr())
		if err != nil {
			return nil, err
		}
	}
	return elements.NewSlice(ir.IntSliceType(), elts)
}

// EvalShape evaluates a shape given for a given array type.
// Returns a nil shape if the shape is generic.
func EvalShape(typ ir.Type) (*shape.Shape, error) {
	atyp, err := cast.To[ir.ArrayType](typ)
	if err != nil {
		return nil, err
	}
	if atyp.Rank().IsAtomic() {
		return &shape.Shape{
			DType: atyp.DataType().Kind().DType(),
		}, nil
	}
	return nil, nil
}

type core struct {
	el Element
}

func (c *core) UnaryOp(env engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	return NewUnary(env, expr, c.el)
}

func (c *core) BinaryOp(env engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (engine.NumericalElement, error) {
	return NewBinaryFrom(env, expr, c.el, y)
}

func (c *core) Cast(env engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return NewCast(env, expr, c.el, target), nil
}

func (c *core) Reshape(env engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return NewReshape(env, expr, c.el, axisLengths)
}

func (c *core) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	return AxesFromType(ev, c.el.Type())
}

func (c *core) EvalShape() (*shape.Shape, error) {
	return EvalShape(c.el.Type())
}

func (c *core) SliceAt(env engine.Env, expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return NewIndex(env, expr, c.el, index)
}

func (c *core) Slice(env engine.Env, expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return NewSlice(env, expr, c.el, low, high)
}
