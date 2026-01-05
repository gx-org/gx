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
	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
)

// BackendNode is a state element owning a node in the backend graph.
type BackendNode struct {
	ev   *Evaluator
	nod  *ops.OutputNode
	expr elements.ExprAt
}

var (
	_ elements.Slicer                 = (*BackendNode)(nil)
	_ elements.Copier                 = (*BackendNode)(nil)
	_ materialise.ElementMaterialiser = (*BackendNode)(nil)
	_ evaluator.NumericalElement      = (*BackendNode)(nil)
)

// NewBackendNode returns an element representing a node in the backend graph.
func NewBackendNode(ev *Evaluator, expr elements.ExprAt, node *ops.OutputNode) (*BackendNode, error) {
	if err := checkShape(node); err != nil {
		return nil, err
	}
	if _, err := ir.ToArrayTypeGivenShape(expr.Node().Type(), node.Shape); err != nil {
		return nil, errors.Errorf("cannot create a backend node: %v", err)
	}
	return &BackendNode{
		ev:   ev,
		expr: expr,
		nod:  node,
	}, nil
}

// hasShape is a node in the graph with a shape inferred by the backend.
type hasShape interface {
	// BackendShape returns the shape inferred by the backend.
	BackendShape() *shape.Shape
}

func checkShape(node *ops.OutputNode) error {
	want := node.Shape
	nodeWithShape, ok := node.Node.(hasShape)
	if !ok {
		return nil
	}
	got := nodeWithShape.BackendShape()
	if got.DType != want.DType {
		return errors.Errorf("backend returned a buffer with a %s data type but GX expects a %s data type", got.DType, want.DType)
	}
	if got.Size() != want.Size() {
		return errors.Errorf("backend returned a buffer with axis lengths %v but GX expects %v", got.AxisLengths, want.AxisLengths)
	}
	return nil
}

// elementFromTuple converts a backend tuple to a Tuple element.
func (ev *Evaluator) elementFromTuple(src elements.ExprAt, tpl ops.Tuple, shps []*shape.Shape) (*interp.Tuple, error) {
	elts := make([]ir.Element, tpl.Size())
	for i := range tpl.Size() {
		node, err := tpl.Element(i)
		if err != nil {
			return nil, err
		}
		elts[i], err = NewBackendNode(ev, src.ToExprAt(), &ops.OutputNode{
			Node:  node,
			Shape: shps[i],
		})
		if err != nil {
			return nil, err
		}
	}
	return interp.NewTuple(elts), nil
}

// BinaryOp applies a binary operator to x and y.
func (n *BackendNode) BinaryOp(env evaluator.Env, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	ao := n.ev.ArrayOps()
	xNode, xShape, err := materialise.Element(n.ev.ao, x)
	if err != nil {
		return nil, err
	}
	yNode, yShape, err := materialise.Element(n.ev.ao, y)
	if err != nil {
		return nil, err
	}
	binaryNode, err := ao.Graph().Core().Binary(expr.Src, xNode, yNode)
	if err != nil {
		return nil, fmterr.Errorf(env.File().FileSet(), expr.Src, "cannot create binary operation for %v%s%v: %v", x, expr.Src.Op, y, err)
	}
	targetShape := &shape.Shape{
		DType:       xShape.DType,
		AxisLengths: xShape.AxisLengths,
	}
	if ir.IsBoolOp(expr.Src.Op) {
		targetShape.DType = dtype.Bool
	}
	if len(yShape.AxisLengths) > 0 {
		targetShape.AxisLengths = yShape.AxisLengths
	}
	return NewBackendNode(
		n.ev,
		elements.NewExprAt(env.File(), expr),
		&ops.OutputNode{
			Node:  binaryNode,
			Shape: targetShape,
		})
}

// UnaryOp applies a unary operator on x.
func (n *BackendNode) UnaryOp(env evaluator.Env, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	ao := n.ev.ArrayOps()
	unaryNode, err := ao.Graph().Core().Unary(expr.Src, n.nod.Node)
	if err != nil {
		return nil, err
	}
	return NewBackendNode(
		n.ev,
		elements.NewExprAt(env.File(), expr),
		&ops.OutputNode{
			Node:  unaryNode,
			Shape: n.nod.Shape,
		})
}

// Cast an element into a given data type.
func (n *BackendNode) Cast(env evaluator.Env, expr ir.Expr, target ir.Type) (evaluator.NumericalElement, error) {
	ao := n.ev.ArrayOps()
	targetKind := target.Kind().DType()
	casted, err := ao.Graph().Core().Cast(n.nod.Node, targetKind)
	if err != nil {
		return nil, err
	}
	return NewBackendNode(
		n.ev,
		elements.NewExprAt(env.File(), expr),
		&ops.OutputNode{
			Node: casted,
			Shape: &shape.Shape{
				DType:       targetKind,
				AxisLengths: n.nod.Shape.AxisLengths,
			},
		})
}

// Reshape an element into a given shape.
func (n *BackendNode) Reshape(env evaluator.Env, expr ir.Expr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	axes := make([]int, len(axisLengths))
	for i, el := range axisLengths {
		var err error
		axes[i], err = elements.ConstantIntFromElement(el)
		if err != nil {
			return nil, err
		}
	}
	ao := n.ev.ArrayOps()
	reshaped, err := ao.Graph().Core().Reshape(n.nod.Node, axes)
	if err != nil {
		return nil, err
	}
	return NewBackendNode(
		n.ev,
		elements.NewExprAt(env.File(), expr),
		&ops.OutputNode{
			Node: reshaped,
			Shape: &shape.Shape{
				DType:       n.nod.Shape.DType,
				AxisLengths: axes,
			},
		})
}

// Slice of the value on the first axis given an index.
func (n *BackendNode) Slice(expr *ir.IndexExpr, index evaluator.NumericalElement) (ir.Element, error) {
	return n.SliceArray(expr, index)
}

// SliceArray of the value on the first axis given an index.
func (n *BackendNode) SliceArray(expr ir.Expr, index evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	i, err := elements.ConstantIntFromElement(index)
	if err != nil {
		return nil, err
	}
	sliceNode, err := n.ev.ao.Graph().Core().Slice(n.nod.Node, i)
	if err != nil {
		return nil, err
	}
	return NewBackendNode(n.ev, elements.NewExprAt(n.expr.File(), expr), &ops.OutputNode{
		Node: sliceNode,
		Shape: &shape.Shape{
			DType:       n.Shape().DType,
			AxisLengths: n.Shape().AxisLengths[1:],
		},
	})
}

// Unflatten consumes the next handles to return a GX value.
func (n *BackendNode) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(n.expr.Node().Type())
}

// Copy the node by returning itself.
func (n *BackendNode) Copy() elements.Copier {
	return n
}

// Shape returns the shape of the element.
func (n *BackendNode) Shape() *shape.Shape {
	return n.nod.Shape
}

// Type of the element.
func (n *BackendNode) Type() ir.Type {
	return n.expr.Node().Type()
}

// Materialise returns itself.
func (n *BackendNode) Materialise(materialise.Materialiser) (materialise.Node, error) {
	return n, nil
}

// OutNode returns the graph node.
func (n *BackendNode) OutNode() *ops.OutputNode {
	return n.nod
}

// String representation of the backend node.
func (n *BackendNode) String() string {
	return n.nod.String()
}

// extractGraphNodes extracts all the graph nodes from an element and its children.
// It first flattens the element, then extracts all the BackendNodes (that is, any elements
// representing a node in the backend graph).
func extractGraphNodes(els []ir.Element) ([]ir.Expr, []*ops.OutputNode, error) {
	flatten, err := flatten.Flatten(els...)
	if err != nil {
		return nil, nil, err
	}
	var graphNodes []*ops.OutputNode
	var exprs []ir.Expr
	for _, elt := range flatten {
		node, ok := elt.(*BackendNode)
		if !ok {
			continue
		}
		graphNodes = append(graphNodes, node.nod)
		exprs = append(exprs, node.expr.Node())
	}
	return exprs, graphNodes, nil
}
