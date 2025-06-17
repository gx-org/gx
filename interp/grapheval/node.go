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
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

// BackendNode is a state element owning a node in the backend graph.
type BackendNode struct {
	nod  *graph.OutputNode
	expr elements.ExprAt
}

var (
	_ elements.Slicer           = (*BackendNode)(nil)
	_ elements.Materialiser     = (*BackendNode)(nil)
	_ elements.NumericalElement = (*BackendNode)(nil)
)

// hasShape is a node in the graph with a shape inferred by the backend.
type hasShape interface {
	// BackendShape returns the shape inferred by the backend.
	BackendShape() *shape.Shape
}

func checkShape(node *graph.OutputNode) error {
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
func (ev *Evaluator) elementFromTuple(src elements.ExprAt, tpl graph.Tuple, shps []*shape.Shape) (*elements.Tuple, error) {
	elts := make([]elements.Element, tpl.Size())
	for i := range tpl.Size() {
		node, err := tpl.Element(i)
		if err != nil {
			return nil, err
		}
		elts[i], err = ElementFromNode(src.ToExprAt(), &graph.OutputNode{
			Node:  node,
			Shape: shps[i],
		})
		if err != nil {
			return nil, err
		}
	}
	return elements.NewTuple(src.File(), src.Node(), elts), nil
}

// BinaryOp applies a binary operator to x and y.
func (n *BackendNode) BinaryOp(ctx elements.FileContext, expr *ir.BinaryExpr, x, y elements.NumericalElement) (elements.NumericalElement, error) {
	ao := evalFromContext(ctx).ArrayOps()
	xNode, xShape, err := NodeFromElement(ao, x)
	if err != nil {
		return nil, err
	}
	yNode, yShape, err := NodeFromElement(ao, y)
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
		&graph.OutputNode{
			Node:  binaryNode,
			Shape: targetShape,
		})
}

// UnaryOp applies a unary operator on x.
func (n *BackendNode) UnaryOp(ctx elements.FileContext, expr *ir.UnaryExpr) (elements.NumericalElement, error) {
	ao := evalFromContext(ctx).ArrayOps()
	unaryNode, err := ao.Graph().Core().Unary(expr.Src, n.nod.Node)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		elements.NewExprAt(ctx.File(), expr),
		&graph.OutputNode{
			Node:  unaryNode,
			Shape: n.nod.Shape,
		})
}

// Cast an element into a given data type.
func (n *BackendNode) Cast(ctx elements.FileContext, expr ir.AssignableExpr, target ir.Type) (elements.NumericalElement, error) {
	ao := evalFromContext(ctx).ArrayOps()
	targetKind := target.Kind().DType()
	casted, err := ao.Graph().Core().Cast(n.nod.Node, targetKind)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		elements.NewExprAt(ctx.File(), expr),
		&graph.OutputNode{
			Node: casted,
			Shape: &shape.Shape{
				DType:       targetKind,
				AxisLengths: n.nod.Shape.AxisLengths,
			},
		})
}

// Slice of the value on the first axis given an index.
func (n *BackendNode) Slice(ctx elements.FileContext, expr ir.AssignableExpr, index elements.NumericalElement) (elements.Element, error) {
	return n.SliceArray(ctx, expr, index)
}

// SliceArray of the value on the first axis given an index.
func (n *BackendNode) SliceArray(ctx elements.FileContext, expr ir.AssignableExpr, index elements.NumericalElement) (elements.NumericalElement, error) {
	i, err := elements.ConstantIntFromElement(index)
	if err != nil {
		return nil, err
	}
	sliceNode, err := evalFromContext(ctx).ArrayOps().Graph().Core().Slice(n.nod.Node, i)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(elements.NewExprAt(ctx.File(), expr), &graph.OutputNode{
		Node: sliceNode,
		Shape: &shape.Shape{
			DType:       n.Shape().DType,
			AxisLengths: n.Shape().AxisLengths[1:],
		},
	})
}

// Flatten returns the element in a slice.
func (n *BackendNode) Flatten() ([]elements.Element, error) {
	return []elements.Element{n}, nil
}

// Unflatten consumes the next handles to return a GX value.
func (n *BackendNode) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return handles.ParseArray(n.expr.ToExprAt())
}

// Shape returns the shape of the element.
func (n *BackendNode) Shape() *shape.Shape {
	return n.nod.Shape
}

// Kind of the element.
func (*BackendNode) Kind() ir.Kind {
	return ir.ArrayKind
}

// Materialise returns itself.
func (n *BackendNode) Materialise(elements.ArrayOps) (elements.Node, error) {
	return n, nil
}

// OutNode returns the graph node.
func (n *BackendNode) OutNode() *graph.OutputNode {
	return n.nod
}

// String representation of the backend node.
func (n *BackendNode) String() string {
	return n.nod.String()
}

// NodeFromElement converts an element into a graph node.
// We unpack the value of *graph.OutputNode to prevent
// from changing the output of Materialise accidentally.
// Returns an error if the element is not a numerical element.
func NodeFromElement(ao elements.ArrayOps, el elements.Element) (graph.Node, *shape.Shape, error) {
	materialiser, ok := el.(elements.Materialiser)
	if !ok {
		return nil, nil, errors.Errorf("cannot convert %T to a backend node graph: does not implement Materialiser", el)
	}
	outEl, err := materialiser.Materialise(ao)
	if err != nil {
		return nil, nil, err
	}
	outNode := outEl.OutNode()
	return outNode.Node, outNode.Shape, nil
}

// ElementFromNode returns an element representing a node in the backend graph.
func ElementFromNode(expr elements.ExprAt, node *graph.OutputNode) (*BackendNode, error) {
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
func ElementsFromNode(expr elements.ExprAt, node *graph.OutputNode) ([]elements.Element, error) {
	el, err := ElementFromNode(expr, node)
	if err != nil {
		return nil, err
	}
	return []elements.Element{el}, nil
}

// extractGraphNodes extracts all the graph nodes from an element and its children.
// It first flattens the element, then extracts all the BackendNodes (that is, any elements
// representing a node in the backend graph).
func extractGraphNodes(els []elements.Element) ([]ir.AssignableExpr, []*graph.OutputNode, error) {
	flatten, err := elements.Flatten(els...)
	if err != nil {
		return nil, nil, err
	}
	var graphNodes []*graph.OutputNode
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
func MaterialiseAll(ao elements.ArrayOps, els []elements.Element) ([]*graph.OutputNode, error) {
	nodes := make([]*graph.OutputNode, len(els))
	for i, el := range els {
		node, shape, err := NodeFromElement(ao, el)
		if err != nil {
			return nil, err
		}
		nodes[i] = &graph.OutputNode{
			Node:  node,
			Shape: shape,
		}
	}
	return nodes, nil
}
