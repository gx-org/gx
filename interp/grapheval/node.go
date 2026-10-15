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
	"fmt"
	"go/token"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/api/hostio"
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
		out() compute.Value
	}

	// BackendNode is a state element owning a node in the backend graph.
	BackendNode struct {
		ao   *Ops
		node compute.Value
		typ  ir.Type
	}
)

var (
	_ elements.Slicer                 = (*BackendNode)(nil)
	_ engine.Copier                   = (*BackendNode)(nil)
	_ materialise.ElementMaterialiser = (*BackendNode)(nil)
	_ materialise.Node                = (*BackendNode)(nil)
	_ engine.NumericalElement         = (*BackendNode)(nil)
	_ ir.WithLength                   = (*BackendNode)(nil)
	_ graphNode                       = (*BackendNode)(nil)
	_ elements.EvalShaper             = (*BackendNode)(nil)
)

// NewBackendNode returns an element representing a node in the backend graph.
func NewBackendNode(ev *Ops, typ ir.Type, node compute.Value) (*BackendNode, error) {
	if err := checkIsConcrete(typ); err != nil {
		return nil, err
	}
	return &BackendNode{
		ao:   ev,
		typ:  typ,
		node: node,
	}, nil
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

func (n *BackendNode) out() compute.Value {
	return n.node
}

// BinaryOp applies a binary operator to x and y.
func (n *BackendNode) BinaryOp(env *engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (_ engine.NumericalElement, err error) {
	xNode, xShape, err := materialise.ElementWithShape(n.ao, n)
	if err != nil {
		return nil, fmt.Errorf("cannot materialise %s in %s: %w", expr.X.SourceString(nil), expr.SourceString(nil), err)
	}
	yNode, err := materialise.Element(n.ao, y)
	if err != nil {
		return nil, fmt.Errorf("cannot materialise %s in %s: %w", expr.Y.SourceString(nil), expr.SourceString(nil), err)
	}
	var binaryNode compute.Value
	switch expr.Src.Op {
	case token.ADD:
		binaryNode, err = n.ao.Graph().Add(xNode, yNode)
	case token.SUB:
		binaryNode, err = n.ao.Graph().Sub(xNode, yNode)
	case token.MUL:
		binaryNode, err = n.ao.Graph().Mul(xNode, yNode)
	case token.QUO:
		binaryNode, err = n.ao.Graph().Div(xNode, yNode)
	case token.REM:
		binaryNode, err = n.ao.Graph().Rem(xNode, yNode)
	case token.EQL:
		binaryNode, err = n.ao.Graph().Equal(xNode, yNode)
	case token.NEQ:
		binaryNode, err = n.ao.Graph().NotEqual(xNode, yNode)
	case token.LSS:
		binaryNode, err = n.ao.Graph().LessThan(xNode, yNode)
	case token.LEQ:
		binaryNode, err = n.ao.Graph().LessOrEqual(xNode, yNode)
	case token.GTR:
		binaryNode, err = n.ao.Graph().GreaterThan(xNode, yNode)
	case token.GEQ:
		binaryNode, err = n.ao.Graph().GreaterOrEqual(xNode, yNode)
	case token.SHL:
		binaryNode, err = n.ao.Graph().ShiftLeft(xNode, yNode)
	case token.SHR:
		// We copy Go's behavior: "shift operators implement arithmetic shifts if the left operand is a signed integer and logical shifts if it is an unsigned integer."
		if xShape.DType.IsUnsigned() {
			binaryNode, err = n.ao.Graph().ShiftRightLogical(xNode, yNode)
		} else {
			binaryNode, err = n.ao.Graph().ShiftRightArithmetic(xNode, yNode)
		}
	case token.AND:
		binaryNode, err = n.ao.Graph().BitwiseAnd(xNode, yNode)
	case token.OR:
		binaryNode, err = n.ao.Graph().BitwiseOr(xNode, yNode)
	case token.XOR:
		binaryNode, err = n.ao.Graph().BitwiseXor(xNode, yNode)
	case token.LAND:
		binaryNode, err = n.ao.Graph().LogicalAnd(xNode, yNode)
	case token.LOR:
		binaryNode, err = n.ao.Graph().LogicalOr(xNode, yNode)
	default:
		return nil, fmterr.InternalAt(env.File().FileSet(), expr.Src, "binary operator %s for %T not implemented", expr.Src.Op, n)
	}
	if err != nil {
		return nil, fmterr.Errorf(env.File().FileSet(), expr.Src, "cannot create binary operation for %v%s%v: %v", n, expr.Src.Op, y, err)
	}
	typ, err := concrete.Concrete(env.ExprEval(), expr.Typ)
	if err != nil {
		return nil, fmterr.Error(env.File().FileSet(), expr.Src, err)
	}
	return NewBackendNode(n.ao, typ, binaryNode)
}

// UnaryOp applies a unary operator on x.
func (n *BackendNode) UnaryOp(env *engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	if expr.Src.Op == token.ADD {
		return n, nil
	}
	var unaryNode compute.Value
	var err error
	switch expr.Src.Op {
	case token.SUB:
		unaryNode, err = n.ao.Graph().Neg(n.node)
	case token.NOT:
		unaryNode, err = n.ao.Graph().LogicalNot(n.node)
	default:
		return nil, fmterr.InternalAt(env.File().FileSet(), expr.Src, "unary operator %s for %T not implemented", expr.Src.Op, n)
	}
	if err != nil {
		return nil, err
	}
	typ, err := concrete.Concrete(env.ExprEval(), expr.Type())
	if err != nil {
		return nil, fmterr.Error(env.File().FileSet(), expr.Src, err)
	}
	return NewBackendNode(n.ao, typ, unaryNode)
}

// Cast an element into a given data type.
func (n *BackendNode) Cast(env *engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	targetKind := target.Kind().DType()
	casted, err := n.ao.Graph().ConvertDType(n.node, targetKind)
	if err != nil {
		return nil, err
	}
	typ, err := concrete.Concrete(env.ExprEval(), expr.Type())
	if err != nil {
		return nil, fmterr.Error(env.File().FileSet(), expr.Node(), err)
	}
	return NewBackendNode(n.ao, typ, casted)
}

// Reshape an element into a given shape.
func (n *BackendNode) Reshape(env *engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	axes := make([]int, len(axisLengths))
	for i, el := range axisLengths {
		var err error
		axes[i], err = elements.IntFromElement(el)
		if err != nil {
			return nil, err
		}
	}
	reshaped, err := n.ao.Graph().Reshape(n.node, axes...)
	if err != nil {
		return nil, err
	}
	return NewBackendNode(n.ao, expr.Type(), reshaped)
}

// SliceAt of the value on the first axis given an index.
func (n *BackendNode) SliceAt(env *engine.Env, expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return n.SliceArray(expr, index)
}

// Slice returns a node slicing the array.
func (n *BackendNode) Slice(env *engine.Env, expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return nil, errors.Errorf("not implemented for %T", n)
}
func (n *BackendNode) sliceArrayFromConstant(expr ir.Expr, index engine.ConstantElement) (engine.NumericalElement, error) {
	i, err := elements.IntFromElement(index)
	if err != nil {
		return nil, err
	}
	shape, err := n.EvalShape()
	if err != nil {
		return nil, err
	}
	rank := len(shape.Dimensions)
	starts := make([]int, rank)
	limits := make([]int, rank)
	strides := make([]int, rank)
	for axis, axisSize := range shape.Dimensions {
		starts[axis] = 0
		limits[axis] = axisSize
		strides[axis] = 1
	}
	starts[0] = i
	limits[0] = i + 1
	sliced, err := n.ao.Graph().Slice(n.node, starts, limits, strides)
	if err != nil {
		return nil, err
	}
	sliceNode, err := n.ao.Graph().Reshape(sliced, shape.Dimensions[1:]...)
	if err != nil {
		return nil, err
	}
	return NewBackendNode(n.ao, expr.Type(), sliceNode)
}

func (n *BackendNode) sliceArrayFromNode(expr ir.Expr, index graphNode) (engine.NumericalElement, error) {
	shape, err := n.EvalShape()
	if err != nil {
		return nil, err
	}
	remaining := shape.Dimensions[1:]

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
	sliceNode, err := n.ao.Graph().Gather(n.node, indexNode, 0, offsetAxes, []int{0}, []int{0}, sliceSizes, false)
	if err != nil {
		return nil, err
	}
	return NewBackendNode(n.ao, expr.Type(), sliceNode)
}

// SliceArray of the value on the first axis given an index.
func (n *BackendNode) SliceArray(expr ir.Expr, index engine.NumericalElement) (engine.NumericalElement, error) {
	switch indexT := index.(type) {
	case engine.ConstantElement:
		return n.sliceArrayFromConstant(expr, indexT)
	case graphNode:
		return n.sliceArrayFromNode(expr, indexT)
	default:
		return nil, errors.Errorf("cannot use %T as an array index: not supported", indexT)
	}
}

func shapeLength(sh shapes.Shape) (int, error) {
	if len(sh.Dimensions) == 0 {
		return 1, nil
	}
	return sh.Dimensions[0], nil
}

// Length returns the evaluation of the len built-in.
func (n *BackendNode) Length(ev ir.Evaluator) (int, error) {
	sh, err := n.EvalShape()
	if err != nil {
		return 0, err
	}
	return shapeLength(sh)
}

// Unflatten consumes the next handles to return a GX value.
func (n *BackendNode) Unflatten(handles *flatten.Parser) (hostio.Value, error) {
	return handles.ParseArray(n.typ)
}

// Copy the node by returning itself.
func (n *BackendNode) Copy() engine.Copier {
	return n
}

// EvalShape returns the shape of the element.
func (n *BackendNode) EvalShape() (shapes.Shape, error) {
	return n.ao.graph.Builder().OpShape(n.node)
}

// Type of the element.
func (n *BackendNode) Type() ir.Type {
	return n.typ
}

// Materialise returns itself.
func (n *BackendNode) Materialise(materialise.Materialiser) (materialise.Node, error) {
	return n, nil
}

// Value returns the graph node.
func (n *BackendNode) Value() compute.Value {
	return n.node
}

// String representation of the backend node.
func (n *BackendNode) String() string {
	sh, err := n.EvalShape()
	if err != nil {
		return fmt.Sprintf("invalid(%v): %v", err, n.node)
	}
	return fmt.Sprintf("%s: %v", sh.String(), n.node)
}
