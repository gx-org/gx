package grapheval

import (
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type arrayOps struct {
	ev    *Evaluator
	graph graph.Graph
}

// Graph where nodes are being added to.
func (ao *arrayOps) Graph() graph.Graph {
	return ao.graph
}

func computeEinsumAxisLengths(ref *ir.EinsumExpr, xShape, yShape *shape.Shape, node graph.Node) []int {
	// TODO(degris): hack to postpone computing einstein sum in the interpreter.
	// We rely on PJRT for the moment.
	return node.(interface{ PJRTDims() []int }).PJRTDims()
}

// SubGraph returns a new graph builder.
func (ao *arrayOps) SubGraph(name string) (elements.ArrayOps, error) {
	sub, err := ao.graph.Core().NewSubgraph(name)
	if err != nil {
		return nil, err
	}
	return &arrayOps{graph: sub}, nil
}

// Einsum calls an einstein sum on x and y given the expression in ref.
func (ao *arrayOps) Einsum(ref elements.NodeFile[*ir.EinsumExpr], x, y elements.NumericalElement) (elements.NumericalElement, error) {
	xNode, xShape, err := NodeFromElement(ao, x)
	if err != nil {
		return nil, err
	}
	yNode, yShape, err := NodeFromElement(ao, y)
	if err != nil {
		return nil, err
	}
	dotNode, err := ao.graph.Core().NewDotGeneral(xNode, yNode, ref.Node().BatchAxes, ref.Node().ReduceAxes)
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       xShape.DType,
		AxisLengths: computeEinsumAxisLengths(ref.Node(), xShape, yShape, dotNode),
	}
	return ElementFromNode(
		ref.ToExprAt(),
		&graph.OutputNode{
			Node:  dotNode,
			Shape: targetShape,
		})
}

// Reshape an element into a given shape.
func (ao *arrayOps) Reshape(expr elements.ExprAt, x elements.NumericalElement, axisLengths []elements.NumericalElement) (elements.NumericalElement, error) {
	axes := make([]int, len(axisLengths))
	for i, el := range axisLengths {
		var err error
		axes[i], err = elements.ConstantIntFromElement(el)
		if err != nil {
			return nil, err
		}
	}
	xNode, xShape, err := NodeFromElement(ao, x)
	if err != nil {
		return nil, err
	}
	reshaped, err := ao.graph.Core().NewReshape(xNode, axes)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		expr.ToExprAt(),
		&graph.OutputNode{
			Node: reshaped,
			Shape: &shape.Shape{
				DType:       xShape.DType,
				AxisLengths: axes,
			},
		})
}

// Concat concatenates scalars elements into an array with one axis.
func (ao *arrayOps) Concat(expr elements.ExprAt, xs []elements.NumericalElement) (elements.NumericalElement, error) {
	nodes := make([]graph.Node, len(xs))
	var dtype dtype.DataType
	for i, x := range xs {
		iNode, iShape, err := NodeFromElement(ao, x)
		if err != nil {
			return nil, err
		}
		dtype = iShape.DType
		// Reshape scalars to 1-element array to work with Concat.
		iNodeArray, err := ao.graph.Core().NewReshape(iNode, []int{1})
		if err != nil {
			return nil, err
		}
		nodes[i] = iNodeArray
	}
	array1d, err := ao.graph.Core().NewConcat(0, nodes)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		expr.ToExprAt(),
		&graph.OutputNode{
			Node: array1d,
			Shape: &shape.Shape{
				DType:       dtype,
				AxisLengths: []int{len(xs)},
			},
		})
}

// Set a slice in an array.
func (ao *arrayOps) Set(call elements.NodeFile[*ir.CallExpr], x, updates, index elements.Element) (elements.Element, error) {
	nodes, err := MaterialiseAll(ao, []elements.Element{x, updates, index})
	if err != nil {
		return nil, err
	}
	setNode, err := ao.graph.Core().NewSet(nodes[0].Node, nodes[1].Node, nodes[2].Node)
	if err != nil {
		return nil, err
	}
	return ElementFromNode(
		call.ToExprAt(),
		&graph.OutputNode{
			Node:  setNode,
			Shape: nodes[0].Shape,
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

// ElementFromArray returns an element from an array GX value.
func (ao *arrayOps) ElementFromArray(expr elements.ExprAt, val values.Array) (elements.Node, error) {
	return newValueElement(ao.ev, expr, val)
}
