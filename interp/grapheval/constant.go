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

package grapheval

import (
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/internal/interp/csteager"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
)

// constant is a GX value represented as a node in the graph.
type constant struct {
	*BackendNode
	cst  engine.Constant
	ctyp ir.Type
}

var (
	_ engine.ConstantElement          = (*constant)(nil)
	_ elements.ArraySlicer            = (*constant)(nil)
	_ elements.Slicer                 = (*constant)(nil)
	_ materialise.ElementMaterialiser = (*constant)(nil)
	_ materialise.Node                = (*constant)(nil)
	_ engine.Copier                   = (*constant)(nil)
	_ elements.WithAxes               = (*constant)(nil)
	_ ir.WithLength                   = (*constant)(nil)
)

func newConstant(ctx ir.Evaluator, ev *Evaluator, cst engine.Constant) (*constant, error) {
	ctyp, err := concrete.Concrete(ctx, cst.Type())
	if err != nil {
		return nil, err
	}
	cArray := ir.ToArrayType(ctyp)
	if cArray == nil {
		return nil, fmterr.Internalf("cannot get array type from %s", ctyp.ReferString(ctx.File()))
	}
	kind := cArray.DataType().Kind()
	cstNode, err := cst.BuildNode(ev.ao.graph, kind)
	if err != nil {
		return nil, err
	}
	axlens, err := cst.AxisLengths()
	if err != nil {
		return nil, err
	}
	shape := &shape.Shape{
		DType:       kind.DType(),
		AxisLengths: axlens,
	}
	node, err := NewBackendNode(ev, ctyp, &ops.OutputNode{
		Node:  cstNode,
		Shape: shape,
	})
	if err != nil {
		return nil, err
	}
	return &constant{
		BackendNode: node,
		cst:         cst,
		ctyp:        ctyp,
	}, nil
}

func (n *constant) Constant() engine.Constant {
	return n.cst
}

// Unflatten creates a GX value from the next handles available in the parser.
func (n *constant) Unflatten(handles *flatten.Parser) (hostio.Value, error) {
	return hostio.NewDeviceArray(n.ctyp, handles.Next())
}

// Copy the graph node by returning itself.
func (n *constant) Copy() engine.Copier {
	return n
}

func (n *constant) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	axlens, err := n.cst.AxisLengths()
	if err != nil {
		return nil, err
	}
	return n.ev.axesFromShape(ev, axlens)
}

// UnaryOp applies an unary operator to x.
func (n *constant) UnaryOp(env *engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	res, err := csteager.Unary(env, expr, n.cst)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return n.BackendNode.UnaryOp(env, expr)
	}
	return newConstant(env.ExprEval(), n.ev, res)
}

// BinaryOp applies a binary operator to x and y.
func (n *constant) BinaryOp(env *engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (engine.NumericalElement, error) {
	res, err := csteager.Binary(env, expr, n.cst, y)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return n.BackendNode.BinaryOp(env, expr, y)
	}
	return newConstant(env.ExprEval(), n.ev, res)
}

// BinaryOp applies a binary operator to x and y.
func (n *constant) Cast(env *engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	res, err := csteager.Cast(env, expr, target, n.cst)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return n.BackendNode.Cast(env, expr, target)
	}
	return newConstant(env.ExprEval(), n.ev, res)
}

// Length returns the value corresponding to calling the built-in len.
func (n *constant) Length(ev ir.Evaluator) (int, error) {
	return n.Shape().OuterAxisLength(), nil
}

func (n *constant) Type() ir.Type {
	return n.ctyp
}

func (n *constant) Kind() irkind.Kind {
	return n.cst.Type().Kind()
}

func (n *constant) String() string {
	return n.cst.SourceString(nil)
}

// Materialise returns itself.
func (n *constant) Materialise(materialise.Materialiser) (materialise.Node, error) {
	return n, nil
}
