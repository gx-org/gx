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

package tracer

import (
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/state"
)

type (
	// Context provides the context in which an operator is applied.
	// TODO(degris): remove this interface and use Context defined in interp
	// once *state.State and *interp.context have been fully abstracted.
	Context interface {
		// ExprAt returns an expression located in a file.
		ExprAt(expr ir.Expr) elements.ExprAt

		// BuildGraph builds a function in the given graph within a new Context.
		BuildGraph(fn ir.Func, g graph.Graph, args []state.Element) (state.Element, error)
	}

	// Evaluator evaluates GX operations by adding the corresponding
	// node in the backend graph.
	Evaluator struct {
		state *state.State
	}
)

// NewEvaluator returns a new evaluator given a state.
func NewEvaluator(state *state.State) *Evaluator {
	return &Evaluator{state: state}
}

// BinaryOp applies a binary operator to x and y.
func (ev *Evaluator) BinaryOp(ctx Context, expr *ir.BinaryExpr, x, y state.Element) (state.Element, error) {
	xNode, xShape, err := state.NodeFromElement(x)
	if err != nil {
		return nil, err
	}
	yNode, yShape, err := state.NodeFromElement(y)
	if err != nil {
		return nil, err
	}
	binaryNode, err := ev.state.BackendGraph().Core().NewBinary(expr.Src, xNode, yNode)
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
	return ev.state.ElementFromNode(ctx.ExprAt(expr), &graph.OutputNode{
		Node:  binaryNode,
		Shape: targetShape,
	})
}

// UnaryOp applies a unary operator on x.
func (ev *Evaluator) UnaryOp(ctx Context, expr *ir.UnaryExpr, x state.Element) (state.Element, error) {
	xNode, xShape, err := state.NodeFromElement(x)
	if err != nil {
		return nil, err
	}
	unaryNode, err := ev.state.BackendGraph().Core().NewUnary(expr.Src, xNode)
	if err != nil {
		return nil, err
	}
	return ev.state.ElementFromNode(ctx.ExprAt(expr), &graph.OutputNode{
		Node:  unaryNode,
		Shape: xShape,
	})
}

func unpackOutputs(outputs []*graph.OutputNode) (nodes []graph.Node, shapes []*shape.Shape) {
	nodes, shapes = make([]graph.Node, len(outputs)), make([]*shape.Shape, len(outputs))
	for i, output := range outputs {
		nodes[i] = output.Node
		shapes[i] = output.Shape
	}
	return
}

// CallFuncLit calls a function literal.
func (ev *Evaluator) CallFuncLit(ctx Context, ref *ir.FuncLit, args []state.Element) (state.Element, error) {
	core := ev.state.BackendGraph().Core()
	subgraph, err := core.NewSubgraph(ref.Name())
	if err != nil {
		return nil, err
	}
	output, err := ctx.BuildGraph(ref, subgraph, args)
	if err != nil {
		return nil, err
	}

	outputGraphNodes, err := state.ExtractGraphNodes(output)
	if err != nil {
		return nil, err
	}
	if len(outputGraphNodes) == 0 {
		// The sub-function does not need the graph.
		// Returns its output element.
		return output, nil
	}
	subresultNodes, shapes := unpackOutputs(outputGraphNodes)
	// If the function has multiple return values, we alter the graph so that it returns a tuple of
	// values, and we also transparently unpack the tuple below.
	graphSingleOutput := outputGraphNodes[0]
	if len(outputGraphNodes) > 1 {
		tpl, err := subgraph.Core().NewTuple(subresultNodes)
		if err != nil {
			return nil, err
		}
		graphSingleOutput = &graph.OutputNode{Node: tpl}
	}
	result, err := core.NewCall(graph.Subgraph{Graph: subgraph, Result: *graphSingleOutput})
	if err != nil {
		return nil, err
	}

	if resultTpl, ok := result.(graph.Tuple); ok {
		return ev.state.ElementFromTuple(ctx.ExprAt(ref), resultTpl, shapes)
	}
	return ev.state.ElementFromNode(ctx.ExprAt(ref), &graph.OutputNode{
		Node:  result,
		Shape: shapes[0],
	})
}

func computeEinsumAxisLengths(ref *ir.EinsumExpr, xShape, yShape *shape.Shape, node graph.Node) []int {
	// TODO(degris): hack to postpone computing einstein sum in the interpreter.
	// We rely on PJRT for the moment.
	return node.(interface{ PJRTDims() []int }).PJRTDims()
}

// Einsum calls an einstein sum on x and y given the expression in ref.
func (ev *Evaluator) Einsum(ctx Context, ref *ir.EinsumExpr, x, y state.Element) (state.Element, error) {
	xNode, xShape, err := state.NodeFromElement(x)
	if err != nil {
		return nil, err
	}
	yNode, yShape, err := state.NodeFromElement(y)
	if err != nil {
		return nil, err
	}
	dotNode, err := ev.state.BackendGraph().Core().NewDotGeneral(xNode, yNode, ref.BatchAxes, ref.ReduceAxes)
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       xShape.DType,
		AxisLengths: computeEinsumAxisLengths(ref, xShape, yShape, dotNode),
	}
	return ev.state.ElementFromNode(ctx.ExprAt(ref), &graph.OutputNode{
		Node:  dotNode,
		Shape: targetShape,
	})
}

// Reshape an element into a given shape.
func (ev *Evaluator) Reshape(ctx Context, expr ir.Expr, x state.Element, axisLengths []int) (state.Element, error) {
	xNode, xShape, err := state.NodeFromElement(x)
	if err != nil {
		return nil, err
	}
	reshaped, err := ev.state.BackendGraph().Core().NewReshape(xNode, axisLengths)
	if err != nil {
		return nil, err
	}
	return ev.state.ElementFromNode(ctx.ExprAt(expr), &graph.OutputNode{
		Node: reshaped,
		Shape: &shape.Shape{
			DType:       xShape.DType,
			AxisLengths: axisLengths,
		},
	})
}

// Cast an element into a given data type.
func (ev *Evaluator) Cast(ctx Context, expr ir.Expr, x state.Element, targetType dtype.DataType) (state.Element, error) {
	xNode, xShape, err := state.NodeFromElement(x)
	if err != nil {
		return nil, err
	}
	casted, err := ev.state.BackendGraph().Core().NewCast(xNode, targetType)
	if err != nil {
		return nil, err
	}
	return ev.state.ElementFromNode(ctx.ExprAt(expr), &graph.OutputNode{
		Node: casted,
		Shape: &shape.Shape{
			DType:       targetType,
			AxisLengths: xShape.AxisLengths,
		},
	})
}

// Concat concatenates scalars elements into an array with one axis.
func (ev *Evaluator) Concat(ctx Context, expr ir.Expr, xs []state.Element) (state.Element, error) {
	nodes := make([]graph.Node, len(xs))
	var dtype dtype.DataType
	for i, x := range xs {
		iNode, iShape, err := state.NodeFromElement(x)
		if err != nil {
			return nil, err
		}
		dtype = iShape.DType
		// Reshape scalars to 1-element array to work with Concat.
		iNodeArray, err := ev.state.BackendGraph().Core().NewReshape(iNode, []int{1})
		if err != nil {
			return nil, err
		}
		nodes[i] = iNodeArray
	}
	array1d, err := ev.state.BackendGraph().Core().NewConcat(0, nodes)
	if err != nil {
		return nil, err
	}
	return ev.state.ElementFromNode(ctx.ExprAt(expr), &graph.OutputNode{
		Node: array1d,
		Shape: &shape.Shape{
			DType:       dtype,
			AxisLengths: []int{len(xs)},
		},
	})
}

// Set a slice in an array.
func (ev *Evaluator) Set(ctx Context, call *ir.CallExpr, x, updates, index state.Element) (state.Element, error) {
	nodes, err := state.MaterialiseAll([]state.Element{x, updates, index})
	if err != nil {
		return nil, err
	}
	setNode, err := ev.state.BackendGraph().Core().NewSet(nodes[0].Node, nodes[1].Node, nodes[2].Node)
	if err != nil {
		return nil, err
	}
	return ev.state.ElementFromNode(ctx.ExprAt(call), &graph.OutputNode{
		Node:  setNode,
		Shape: nodes[0].Shape,
	})
}

// ElementFromValue returns an element from a GX value.
func (ev *Evaluator) ElementFromValue(ctx Context, expr ir.Expr, val *values.HostArray) (state.NumericalElement, error) {
	return ev.state.NewValueLiteral(ctx.ExprAt(expr), val)
}
