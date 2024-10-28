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

package state

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

type (
	// AtomicLiteral is a constant atom.
	AtomicLiteral interface {
		ElementWithConstant
		ToArray(ExprAt, []int) (Element, error)
	}

	// AtomicLiteralT is an atomic value.
	AtomicLiteralT[T dtype.GoDataType] struct {
		baseLiteral[T]
		value T
	}

	// ArrayLiteralT is an atomic value.
	ArrayLiteralT[T dtype.GoDataType] struct {
		baseLiteral[T]
		values    []T
		arrayType *ir.ArrayType
	}

	baseLiteral[T dtype.GoDataType] struct {
		state    *State
		typ      ir.Type
		shape    *shape.Shape
		array    *values.HostArray
		bckNodes []*graph.OutputNode
	}
)

var (
	_ AtomicLiteral  = (*AtomicLiteralT[float32])(nil)
	_ backendElement = (*AtomicLiteralT[float32])(nil)

	_ ElementWithConstant = (*ArrayLiteralT[float32])(nil)
	_ backendElement      = (*ArrayLiteralT[float32])(nil)
)

// NewAtomicLiteral returns a new atomic value.
func NewAtomicLiteral[T dtype.GoDataType](s *State, typ ir.Type, val T) (*AtomicLiteralT[T], error) {
	underlying := ir.Underlying(typ)
	atomicType, ok := underlying.(*ir.AtomicType)
	if !ok {
		return nil, errors.Errorf("type %s is not an atomic type", typ.String())
	}
	return &AtomicLiteralT[T]{
		baseLiteral: baseLiteral[T]{
			state: s,
			shape: &shape.Shape{
				DType: atomicType.Kind().DType(),
			},
			typ: typ,
		},
		value: val,
	}, nil
}

// NewArrayLiteral returns a new atomic value.
func NewArrayLiteral[T dtype.GoDataType](s *State, typ ir.Type, vals []T, axlens []int) (*ArrayLiteralT[T], error) {
	underlying := ir.Underlying(typ)
	arrayType, ok := underlying.(*ir.ArrayType)
	if !ok {
		return nil, errors.Errorf("type %s is not an array type", typ.String())
	}
	return &ArrayLiteralT[T]{
		baseLiteral: baseLiteral[T]{
			state: s,
			shape: &shape.Shape{
				DType:       arrayType.DataType().Kind().DType(),
				AxisLengths: axlens,
			},
			typ: typ,
		},
		values:    vals,
		arrayType: arrayType,
	}, nil
}

// State returns the state owning the element.
func (n *baseLiteral[T]) State() *State {
	return n.state
}

// Shape of the value represented by the element.
func (n *baseLiteral[T]) Shape() *shape.Shape {
	return n.shape
}

func (n *baseLiteral[T]) hostArray(vals []T, dims ...int) *values.HostArray {
	buffer := kernels.ToBuffer(vals, n.Shape())
	n.array = values.NewHostArray(n.typ, buffer)
	return n.array
}

func (n *baseLiteral[T]) nodes(cst *values.HostArray) ([]*graph.OutputNode, error) {
	if len(n.bckNodes) > 0 {
		return n.bckNodes, nil
	}
	cstNode, err := n.state.backendGraph.Core().NewConstant(cst.Buffer())
	if err != nil {
		return nil, err
	}
	n.bckNodes = []*graph.OutputNode{&graph.OutputNode{
		Node:  cstNode,
		Shape: cst.Shape(),
	}}
	return n.bckNodes, nil
}

func (n *baseLiteral[T]) valueFromHandle(handles *handleParser) (values.Value, error) {
	return values.NewDeviceArray(n.typ, handles.next()), nil
}

// ToExpr returns the value as a GX IR expression.
func (n *AtomicLiteralT[T]) ToExpr() *HostArrayExpr {
	return &HostArrayExpr{
		Typ: n.typ,
		Val: n.NumericalConstant(),
	}
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *AtomicLiteralT[T]) NumericalConstant() *values.HostArray {
	if n.array != nil {
		return n.array
	}
	return n.hostArray([]T{n.value})
}

// ToArray converts the atomic value into an array of a given shape.
func (n *AtomicLiteralT[T]) ToArray(expr ExprAt, axisLengths []int) (Element, error) {
	nodes, err := n.nodes()
	if err != nil {
		return nil, err
	}
	node := nodes[0]
	op, err := n.state.backendGraph.Core().NewReshape(node.Node, axisLengths)
	if err != nil {
		return nil, err
	}
	return n.state.ElementFromNode(expr, op, &shape.Shape{
		DType:       node.Shape.DType,
		AxisLengths: axisLengths,
	})
}

func (n *AtomicLiteralT[T]) nodes() ([]*graph.OutputNode, error) {
	return n.baseLiteral.nodes(n.NumericalConstant())
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *ArrayLiteralT[T]) NumericalConstant() *values.HostArray {
	if n.array != nil {
		return n.array
	}
	return n.hostArray(n.values)
}

func (n *ArrayLiteralT[T]) nodes() ([]*graph.OutputNode, error) {
	return n.baseLiteral.nodes(n.NumericalConstant())
}

// Slice of the value on the first axis given an index.
func (n *ArrayLiteralT[T]) Slice(expr ExprAt, i int) (Element, error) {
	x, err := n.nodes()
	if err != nil {
		return nil, err
	}
	sliceNode, err := n.state.backendGraph.Core().NewSlice(x[0].Node, i)
	if err != nil {
		return nil, err
	}
	sh := n.Shape()
	return n.state.ElementFromNode(expr, sliceNode, &shape.Shape{
		DType:       sh.DType,
		AxisLengths: sh.AxisLengths[1:],
	})
}

// ToExpr returns the value as a GX IR expression.
func (n *ArrayLiteralT[T]) ToExpr() *HostArrayExpr {
	return &HostArrayExpr{
		Typ: n.typ,
		Val: n.NumericalConstant(),
	}
}
