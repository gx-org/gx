// Copyright 2025 Google LLC
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
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type arrayOps struct {
	ev    *Evaluator
	graph ops.Graph
}

var (
	_ elements.ArrayOps          = (*arrayOps)(nil)
	_ elements.ArrayMaterialiser = (*arrayOps)(nil)
)

// Graph where nodes are being added to.
func (ao *arrayOps) Graph() ops.Graph {
	return ao.graph
}

func computeEinsumAxisLengths(ref *ir.EinsumExpr, xShape, yShape *shape.Shape, node ops.Node) []int {
	// TODO(degris): hack to postpone computing einstein sum in the interpreter.
	// We rely on PJRT for the moment.
	return node.(interface{ PJRTDims() []int }).PJRTDims()
}

// SubGraph returns a new graph builder.
func (ao *arrayOps) SubGraph(name string) (elements.ArrayOps, error) {
	sub, err := ao.graph.Core().Subgraph(name)
	if err != nil {
		return nil, err
	}
	return &arrayOps{graph: sub}, nil
}

// Einsum calls an einstein sum on x and y given the expression in ref.
func (ao *arrayOps) Einsum(ctx ir.Evaluator, ref *ir.EinsumExpr, x, y elements.NumericalElement) (elements.NumericalElement, error) {
	xNode, xShape, err := NodeFromElement(ctx, x)
	if err != nil {
		return nil, err
	}
	yNode, yShape, err := NodeFromElement(ctx, y)
	if err != nil {
		return nil, err
	}
	dotNode, err := ao.graph.Core().DotGeneral(xNode, yNode, ref.BatchAxes, ref.ReduceAxes)
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       xShape.DType,
		AxisLengths: computeEinsumAxisLengths(ref, xShape, yShape, dotNode),
	}
	return ElementFromNode(
		elements.NewExprAt(ctx.File(), ref),
		&ops.OutputNode{
			Node:  dotNode,
			Shape: targetShape,
		})
}

func elementsToInt(els []elements.NumericalElement) ([]int, error) {
	axes := make([]int, len(els))
	for i, el := range els {
		var err error
		axes[i], err = elements.ConstantIntFromElement(el)
		if err != nil {
			return nil, err
		}
	}
	return axes, nil
}

// BroadcastInDim the data of an array across dimensions.
func (ao *arrayOps) BroadcastInDim(ctx ir.Evaluator, expr ir.AssignableExpr, x elements.NumericalElement, axisLengths []elements.NumericalElement) (elements.NumericalElement, error) {
	axes, err := elementsToInt(axisLengths)
	if err != nil {
		return nil, err
	}
	xNode, xShape, err := NodeFromElement(ctx, x)
	if err != nil {
		return nil, err
	}
	broadcastAxes := make([]int, 0)
	for i, xAxis := range xShape.AxisLengths {
		if xAxis > 1 {
			continue
		}
		broadcastAxes = append(broadcastAxes, i)
	}
	targetShape := &shape.Shape{
		DType:       xShape.DType,
		AxisLengths: axes,
	}
	reshaped, err := ao.graph.Core().BroadcastInDim(xNode, targetShape, broadcastAxes)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		elements.NewExprAt(ctx.File(), expr),
		&ops.OutputNode{
			Node:  reshaped,
			Shape: targetShape,
		})
}

// Concat concatenates scalars elements into an array with one axis.
func (ao *arrayOps) Concat(ctx ir.Evaluator, expr ir.AssignableExpr, xs []elements.NumericalElement) (elements.NumericalElement, error) {
	nodes := make([]ops.Node, len(xs))
	var dtype dtype.DataType
	for i, x := range xs {
		iNode, iShape, err := NodeFromElement(ctx, x)
		if err != nil {
			return nil, err
		}
		dtype = iShape.DType
		// Reshape scalars to 1-element array to work with Concat.
		iNodeArray, err := ao.graph.Core().Reshape(iNode, []int{1})
		if err != nil {
			return nil, err
		}
		nodes[i] = iNodeArray
	}
	array1d, err := ao.graph.Core().Concat(0, nodes)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		elements.NewExprAt(ctx.File(), expr),
		&ops.OutputNode{
			Node: array1d,
			Shape: &shape.Shape{
				DType:       dtype,
				AxisLengths: []int{len(xs)},
			},
		})
}

// Set a slice in an array.
func (ao *arrayOps) Set(ctx ir.Evaluator, expr *ir.CallExpr, x, updates, index ir.Element) (ir.Element, error) {
	nodes, err := MaterialiseAll(ctx, []ir.Element{x, updates, index})
	if err != nil {
		return nil, err
	}
	setNode, err := ao.graph.Core().Set(nodes[0].Node, nodes[1].Node, nodes[2].Node)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		elements.NewExprAt(ctx.File(), expr),
		&ops.OutputNode{
			Node:  setNode,
			Shape: nodes[0].Shape,
		})
}

func unpackOutputs(outputs []*ops.OutputNode) (nodes []ops.Node, shapes []*shape.Shape) {
	nodes, shapes = make([]ops.Node, len(outputs)), make([]*shape.Shape, len(outputs))
	for i, output := range outputs {
		nodes[i] = output.Node
		shapes[i] = output.Shape
	}
	return
}

// ElementFromArray returns an element from an array GX value.
func (ao *arrayOps) ElementFromArray(ctx ir.Evaluator, expr ir.AssignableExpr, val values.Array) (elements.NumericalElement, error) {
	return newValueElement(ao.ev, elements.NewExprAt(ctx.File(), expr), val)
}

// ElementFromArray returns an element from an array GX value.
func (ao *arrayOps) NodeFromArray(expr elements.ExprAt, val values.Array) (elements.Node, error) {
	return newValueElement(ao.ev, expr, val)
}
