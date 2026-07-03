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
	"github.com/gx-org/backend/dtypes"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
)

type (
	graphNode interface {
		out() ops.Node
	}

	// BackendNode is a state element owning a node in the backend graph.
	BackendNode struct {
		ev  *Evaluator
		nod *ops.OutputNode
		typ ir.Type
	}
)

var (
	_ elements.Slicer                 = (*BackendNode)(nil)
	_ engine.Copier                   = (*BackendNode)(nil)
	_ materialise.ElementMaterialiser = (*BackendNode)(nil)
	_ engine.NumericalElement         = (*BackendNode)(nil)
	_ ir.WithLength                   = (*BackendNode)(nil)
	_ graphNode                       = (*BackendNode)(nil)
)

// NewBackendNode returns an element representing a node in the backend graph.
func NewBackendNode(ev *Evaluator, typ ir.Type, node *ops.OutputNode) (*BackendNode, error) {
	if err := checkShape(node); err != nil {
		return nil, err
	}
	if err := checkIsConcrete(typ); err != nil {
		return nil, err
	}
	if _, err := ir.ToArrayTypeGivenShape(nil, typ, node.Shape); err != nil {
		return nil, errors.Errorf("cannot create a backend node: %v", err)
	}
	return &BackendNode{
		ev:  ev,
		typ: typ,
		nod: node,
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
	gotDType := got.DType
	if gotDType == dtypes.Int64 && want.DType == dtypes.Int {
		gotDType = want.DType
	}
	if gotDType != want.DType {
		return errors.Errorf("backend returned a buffer with a %s data type but GX expects a %s data type", got.DType, want.DType)
	}
	if got.Size() != want.Size() {
		return errors.Errorf("backend returned a buffer with axis lengths %v but GX expects %v", got.AxisLengths, want.AxisLengths)
	}
	return nil
}

func checkIsConcrete(typ ir.Type) error {
	arrayType, isArray := ir.Underlying(typ).(ir.ArrayType)
	if !isArray {
		return errors.Errorf("%T is not an array type", typ)
	}
	dtype := arrayType.DataType()
	dtypeKind := dtype.Kind()
	if !irkind.IsConcrete(dtypeKind) {
		return errors.Errorf("array type %s has not been resolved", typ.ReferString(nil))
	}
	return nil
}

// elementFromTuple converts a backend tuple to a Tuple element.
func (ev *Evaluator) elementFromTuple(types []ir.Type, nodeTuple ops.Tuple, shps []*shape.Shape) (*elements.Tuple, error) {
	elts := make([]ir.Element, nodeTuple.Size())
	for i := range nodeTuple.Size() {
		node, err := nodeTuple.Element(i)
		if err != nil {
			return nil, err
		}
		elts[i], err = NewBackendNode(ev, types[i], &ops.OutputNode{
			Node:  node,
			Shape: shps[i],
		})
		if err != nil {
			return nil, err
		}
	}
	return elements.TupleFromElements(elts)
}

func (n *BackendNode) out() ops.Node {
	return n.nod.Node
}

// BinaryOp applies a binary operator to x and y.
func (n *BackendNode) BinaryOp(env engine.Env, expr *ir.BinaryExpr, x, y engine.NumericalElement) (engine.NumericalElement, error) {
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
		targetShape.DType = dtypes.Bool
	}
	if len(yShape.AxisLengths) > 0 {
		targetShape.AxisLengths = yShape.AxisLengths
	}
	typ, err := concrete.Concrete(env.ExprEval(), expr.Src, expr.Typ)
	if err != nil {
		return nil, err
	}
	return NewBackendNode(
		n.ev,
		typ,
		&ops.OutputNode{
			Node:  binaryNode,
			Shape: targetShape,
		})
}

// UnaryOp applies a unary operator on x.
func (n *BackendNode) UnaryOp(env engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	ao := n.ev.ArrayOps()
	unaryNode, err := ao.Graph().Core().Unary(expr.Src, n.nod.Node)
	if err != nil {
		return nil, err
	}
	typ, err := concrete.Concrete(env.ExprEval(), expr.Src, expr.Type())
	if err != nil {
		return nil, err
	}
	return NewBackendNode(
		n.ev,
		typ,
		&ops.OutputNode{
			Node:  unaryNode,
			Shape: n.nod.Shape,
		})
}

// Cast an element into a given data type.
func (n *BackendNode) Cast(env engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	ao := n.ev.ArrayOps()
	targetKind := target.Kind().DType()
	casted, err := ao.Graph().Core().Cast(n.nod.Node, targetKind)
	if err != nil {
		return nil, err
	}
	typ, err := concrete.Concrete(env.ExprEval(), expr.Expr(), expr.Type())
	if err != nil {
		return nil, err
	}
	return NewBackendNode(
		n.ev,
		typ,
		&ops.OutputNode{
			Node: casted,
			Shape: &shape.Shape{
				DType:       targetKind,
				AxisLengths: n.nod.Shape.AxisLengths,
			},
		})
}

// Reshape an element into a given shape.
func (n *BackendNode) Reshape(env engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
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
		expr.Type(),
		&ops.OutputNode{
			Node: reshaped,
			Shape: &shape.Shape{
				DType:       n.nod.Shape.DType,
				AxisLengths: axes,
			},
		})
}

// SliceAt of the value on the first axis given an index.
func (n *BackendNode) SliceAt(expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return n.SliceArray(expr, index)
}

// Slice returns a node slicing the array.
func (n *BackendNode) Slice(expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return nil, errors.Errorf("not implemented for %T", n)
}
func (n *BackendNode) sliceArrayFromConstant(expr ir.Expr, index engine.NumericalElement) (engine.NumericalElement, error) {
	i, err := elements.ConstantIntFromElement(index)
	if err != nil {
		return nil, err
	}
	sliceNode, err := n.ev.ao.Graph().Core().Slice(n.nod.Node, i)
	if err != nil {
		return nil, err
	}
	return NewBackendNode(n.ev, expr.Type(), &ops.OutputNode{
		Node: sliceNode,
		Shape: &shape.Shape{
			DType:       n.Shape().DType,
			AxisLengths: n.Shape().AxisLengths[1:],
		},
	})
}

func (n *BackendNode) sliceArrayFromNode(expr ir.Expr, index graphNode) (engine.NumericalElement, error) {
	remaining := n.Shape().AxisLengths[1:]

	indexNode := index.out()
	// XLA Gather parameters for indexing axis 0 of the operand:
	//   indexVectorAxis=0: dimension 0 of startIndices is the index vector.
	//   startIndexMap=[0]: the single index element maps to operand axis 0.
	//   collapsedSliceAxes=[0]: axis 0 is collapsed (removed) from the output.
	//   sliceSizes=[1]+remaining: take 1 element along axis 0, full size along others.
	//   offsetAxes=[0..len(remaining)-1]: window result axes appear first in output.
	// XLA requires len(startIndexMap) == startIndices.shape[indexVectorAxis].
	sliceSizes := append([]int{1}, remaining...)
	offsetAxes := make([]int, len(remaining))
	for i := range offsetAxes {
		offsetAxes[i] = i
	}
	sliceNode, err := n.ev.ao.Graph().Shape().Gather(n.nod.Node, indexNode, 0, offsetAxes, []int{0}, []int{0}, sliceSizes, false)
	if err != nil {
		return nil, err
	}
	return NewBackendNode(n.ev, expr.Type(), &ops.OutputNode{
		Node: sliceNode,
		Shape: &shape.Shape{
			DType:       n.Shape().DType,
			AxisLengths: remaining,
		},
	})
}

// SliceArray of the value on the first axis given an index.
func (n *BackendNode) SliceArray(expr ir.Expr, index engine.NumericalElement) (engine.NumericalElement, error) {
	switch indexT := index.(type) {
	case elements.ElementWithConstant:
		return n.sliceArrayFromConstant(expr, indexT)
	case graphNode:
		return n.sliceArrayFromNode(expr, indexT)
	default:
		return nil, errors.Errorf("cannot use %T as an array index: not supported", indexT)
	}
}

// Length returns the evaluation of the len built-in.
func (n *BackendNode) Length(ev ir.Evaluator) (int, error) {
	return n.Shape().OuterAxisLength(), nil
}

// Unflatten consumes the next handles to return a GX value.
func (n *BackendNode) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(n.typ)
}

// Copy the node by returning itself.
func (n *BackendNode) Copy() engine.Copier {
	return n
}

// Shape returns the shape of the element.
func (n *BackendNode) Shape() *shape.Shape {
	return n.nod.Shape
}

// Type of the element.
func (n *BackendNode) Type() ir.Type {
	return n.typ
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
