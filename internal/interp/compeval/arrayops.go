package compeval

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
)

type compArrayOps struct{}

var hostArrayOps elements.ArrayOps = &compArrayOps{}

// Graph returns the graph to which new nodes are being added.
func (compArrayOps) Graph() graph.Graph {
	return nil
}

// SubGraph returns a new graph builder.
func (compArrayOps) SubGraph(name string) (elements.ArrayOps, error) {
	return nil, errors.Errorf("not implemented")
}

// Einsum calls an einstein sum on x and y given the expression in ref.
func (compArrayOps) Einsum(expr elements.NodeFile[*ir.EinsumExpr], x, y elements.NumericalElement) (elements.NumericalElement, error) {
	return cpevelements.NewArray(expr.ToExprAt(), expr.Node().Type().(ir.ArrayType)), nil
}

// Reshape an element into a given shape.
func (compArrayOps) Reshape(expr elements.ExprAt, x elements.NumericalElement, axisLengths []elements.NumericalElement) (elements.NumericalElement, error) {
	return cpevelements.NewArray(expr.ToExprAt(), expr.Node().Type().(ir.ArrayType)), nil
}

// Concat concatenates scalars elements into an array with one axis.
func (compArrayOps) Concat(expr elements.ExprAt, xs []elements.NumericalElement) (elements.NumericalElement, error) {
	return cpevelements.NewArray(expr.ToExprAt(), expr.Node().Type().(ir.ArrayType)), nil
}

// Set a slice in an array.
func (compArrayOps) Set(call elements.NodeFile[*ir.CallExpr], x, updates, index elements.Element) (elements.Element, error) {
	return cpevelements.NewArray(call.ToExprAt(), call.Node().Type().(ir.ArrayType)), nil
}

// ElementFromArray returns an element from an array GX value.
func (compArrayOps) ElementFromArray(expr elements.ExprAt, val values.Array) (elements.Node, error) {
	return cpevelements.NewArray(expr, val.Type().(ir.ArrayType)).(elements.Node), nil
}
