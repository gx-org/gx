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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

// BackendNode is a state element owning a node in the backend graph.
type BackendNode struct {
	nod  *ops.OutputNode
	expr elements.ExprAt
}

var (
	_ elements.Slicer            = (*BackendNode)(nil)
	_ elements.Copier            = (*BackendNode)(nil)
	_ elements.Materialiser      = (*BackendNode)(nil)
	_ evaluator.NumericalElement = (*BackendNode)(nil)
)

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
func (ev *Evaluator) elementFromTuple(src elements.ExprAt, tpl ops.Tuple, shps []*shape.Shape) (*elements.Tuple, error) {
	elts := make([]ir.Element, tpl.Size())
	for i := range tpl.Size() {
		node, err := tpl.Element(i)
		if err != nil {
			return nil, err
		}
		elts[i], err = ElementFromNode(src.ToExprAt(), &ops.OutputNode{
			Node:  node,
			Shape: shps[i],
		})
		if err != nil {
			return nil, err
		}
	}
	return elements.NewTuple(elts), nil
}

// BinaryOp applies a binary operator to x and y.
func (n *BackendNode) BinaryOp(ctx ir.Evaluator, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	ao := opsFromContext(ctx.(elements.Evaluator))
	xNode, xShape, err := NodeFromElement(ctx, x)
	if err != nil {
		return nil, err
	}
	yNode, yShape, err := NodeFromElement(ctx, y)
	if err != nil {
		return nil, err
	}
	binaryNode, err := ao.Graph().Core().Binary(expr.Src, xNode, yNode)
	if err != nil {
		return nil, err
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
	return ElementFromNode(
		elements.NewExprAt(ctx.File(), expr),
		&ops.OutputNode{
			Node:  binaryNode,
			Shape: targetShape,
		})
}

// UnaryOp applies a unary operator on x.
func (n *BackendNode) UnaryOp(ctx ir.Evaluator, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	ao := opsFromContext(ctx.(elements.Evaluator))
	unaryNode, err := ao.Graph().Core().Unary(expr.Src, n.nod.Node)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		elements.NewExprAt(ctx.File(), expr),
		&ops.OutputNode{
			Node:  unaryNode,
			Shape: n.nod.Shape,
		})
}

// Cast an element into a given data type.
func (n *BackendNode) Cast(ctx ir.Evaluator, expr ir.AssignableExpr, target ir.Type) (evaluator.NumericalElement, error) {
	ao := opsFromContext(ctx.(elements.Evaluator))
	targetKind := target.Kind().DType()
	casted, err := ao.Graph().Core().Cast(n.nod.Node, targetKind)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		elements.NewExprAt(ctx.File(), expr),
		&ops.OutputNode{
			Node: casted,
			Shape: &shape.Shape{
				DType:       targetKind,
				AxisLengths: n.nod.Shape.AxisLengths,
			},
		})
}

// Reshape an element into a given shape.
func (n *BackendNode) Reshape(ctx ir.Evaluator, expr ir.AssignableExpr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	axes := make([]int, len(axisLengths))
	for i, el := range axisLengths {
		var err error
		axes[i], err = elements.ConstantIntFromElement(el)
		if err != nil {
			return nil, err
		}
	}
	ao := opsFromContext(ctx.(elements.Evaluator))
	reshaped, err := ao.Graph().Core().Reshape(n.nod.Node, axes)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		elements.NewExprAt(ctx.File(), expr),
		&ops.OutputNode{
			Node: reshaped,
			Shape: &shape.Shape{
				DType:       n.nod.Shape.DType,
				AxisLengths: axes,
			},
		})
}

// Slice of the value on the first axis given an index.
func (n *BackendNode) Slice(ctx elements.Evaluator, expr *ir.IndexExpr, index evaluator.NumericalElement) (ir.Element, error) {
	return n.SliceArray(ctx, expr, index)
}

// SliceArray of the value on the first axis given an index.
func (n *BackendNode) SliceArray(ctx elements.Evaluator, expr ir.AssignableExpr, index evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	i, err := elements.ConstantIntFromElement(index)
	if err != nil {
		return nil, err
	}
	sliceNode, err := opsFromContext(ctx).Graph().Core().Slice(n.nod.Node, i)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(elements.NewExprAt(ctx.File(), expr), &ops.OutputNode{
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
func (n *BackendNode) Materialise(elements.ArrayMaterialiser) (elements.Node, error) {
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

// NodeFromElement converts an element into a graph node.
// We unpack the value of *ops.OutputNode to prevent
// from changing the output of Materialise accidentally.
// Returns an error if the element is not a numerical element.
func NodeFromElement(ctx ir.Evaluator, el ir.Element) (ops.Node, *shape.Shape, error) {
	materialiser, ok := el.(elements.Materialiser)
	if !ok {
		return nil, nil, errors.Errorf("cannot convert %T to a backend node graph: does not implement Materialiser", el)
	}
	ev := ctx.(evaluator.Context).Evaluator().(*Evaluator)
	outEl, err := materialiser.Materialise(ev.Materialiser())
	if err != nil {
		return nil, nil, err
	}
	outNode := outEl.OutNode()
	return outNode.Node, outNode.Shape, nil
}

// ElementFromNode returns an element representing a node in the backend graph.
func ElementFromNode(expr elements.ExprAt, node *ops.OutputNode) (*BackendNode, error) {
	if err := checkShape(node); err != nil {
		return nil, err
	}
	if _, err := ir.ToArrayTypeGivenShape(expr.Node().Type(), node.Shape); err != nil {
		return nil, errors.Errorf("cannot create a backend node: %v", err)
	}
	return &BackendNode{
		expr: expr,
		nod:  node,
	}, nil
}

// ElementsFromNode returns a slice of element from a graph node.
func ElementsFromNode(expr elements.ExprAt, node *ops.OutputNode) ([]ir.Element, error) {
	el, err := ElementFromNode(expr, node)
	if err != nil {
		return nil, err
	}
	return []ir.Element{el}, nil
}

// extractGraphNodes extracts all the graph nodes from an element and its children.
// It first flattens the element, then extracts all the BackendNodes (that is, any elements
// representing a node in the backend graph).
func extractGraphNodes(els []ir.Element) ([]ir.AssignableExpr, []*ops.OutputNode, error) {
	flatten, err := flatten.Flatten(els...)
	if err != nil {
		return nil, nil, err
	}
	var graphNodes []*ops.OutputNode
	var exprs []ir.AssignableExpr
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

// MaterialiseAll materialises a slice of elements into a slice of output graph nodes.
func MaterialiseAll(ctx ir.Evaluator, els []ir.Element) ([]*ops.OutputNode, error) {
	nodes := make([]*ops.OutputNode, len(els))
	for i, el := range els {
		node, shape, err := NodeFromElement(ctx, el)
		if err != nil {
			return nil, err
		}
		nodes[i] = &ops.OutputNode{
			Node:  node,
			Shape: shape,
		}
	}
	return nodes, nil
}
