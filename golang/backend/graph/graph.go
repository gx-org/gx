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

// Package graph implements a computational graph in GX.
package graph

import (
	"fmt"
	"go/token"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes/bfloat16"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/notimplemented"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/golang/backend/kernels"
	goplatform "github.com/gx-org/gx/golang/backend/platform"
)

// Graph of the Go native backend.
type Graph struct {
	notimplemented.Function
	plat     *goplatform.Platform
	builder  compute.Builder
	parent   compute.Function
	funcName string
	args     []*argument
	returned bool
	outputs  []compute.Value
}

var _ compute.Function = (*Graph)(nil)

func errNotImplemented(op compute.OpType) error {
	return errors.Errorf("%s not implemented in the Go backend", op)
}

// New graph running operations implemented in Go.
func New(plat *goplatform.Platform, builder compute.Builder, funcName string) *Graph {
	return &Graph{
		Function: notimplemented.Function{ErrFn: errNotImplemented},
		plat:     plat,
		builder:  builder,
		funcName: funcName,
	}
}

// Platform owning the graph.
func (g *Graph) Platform() *goplatform.Platform {
	return g.plat
}

// Name of the function.
func (g *Graph) Name() string {
	return g.funcName
}

// Builder returns the builder of which this function is part of.
func (g *Graph) Builder() compute.Builder {
	return g.builder
}

// Parent returns the parent function of the current function.
func (g *Graph) Parent() compute.Function {
	return g.parent
}

// Closure returns a new local function within this function.
func (g *Graph) Closure() (compute.Function, error) {
	return &Graph{
		Function: notimplemented.Function{ErrFn: errNotImplemented},
		plat:     g.plat,
		builder:  g.builder,
		parent:   g,
		funcName: g.funcName + ".closure",
	}, nil
}

// Return marks the outputs of this function.
func (g *Graph) Return(outputs []compute.Value, shardings []*compute.ShardingSpec) error {
	if g.returned {
		return errors.Errorf("Return() already called for function %q", g.funcName)
	}
	g.outputs = outputs
	g.returned = true
	return nil
}

// Shape returns the shape of the given Value.
func (g *Graph) Shape(v compute.Value) (shapes.Shape, error) {
	n, ok := v.(execNode)
	if !ok {
		return shapes.Invalid(), errors.Errorf("invalid value type %T", v)
	}
	return n.shape(), nil
}

type node struct {
	g  *Graph
	k  kernels.Factory
	sh shapes.Shape
}

var _ compute.Value = (*node)(nil)

func (g *Graph) node(shape shapes.Shape, k kernels.Factory) node {
	return node{g: g, sh: shape, k: k}
}

func (n *node) shape() shapes.Shape {
	return n.sh
}

func (n *node) BackendShape() shapes.Shape {
	return n.sh
}

func (n *node) kernelFactory() kernels.Factory {
	return n.k
}

type constant struct {
	node
	value kernels.Array
}

var _ execNode = (*constant)(nil)

// Constant creates a constant in the function with the given flat values
// and the shape defined by the dimensions.
func (g *Graph) Constant(flat any, dims ...int) (compute.Value, error) {
	var arr kernels.Array
	switch flatT := flat.(type) {
	case []bool:
		arr = kernels.ToBoolArray(flatT, dims)
	case []bfloat16.BFloat16:
		arr = kernels.ToBfloat16Array(flatT, dims)
	case []float32:
		arr = kernels.ToFloatArray(flatT, dims)
	case []float64:
		arr = kernels.ToFloatArray(flatT, dims)
	case []int:
		arr = kernels.ToIntegerArray(flatT, dims)
	case []int32:
		arr = kernels.ToIntegerArray(flatT, dims)
	case []int64:
		arr = kernels.ToIntegerArray(flatT, dims)
	case []uint32:
		arr = kernels.ToIntegerArray(flatT, dims)
	case []uint64:
		arr = kernels.ToIntegerArray(flatT, dims)
	default:
		return nil, errors.Errorf("cannot create a constant literal for %T: not implemented", flatT)
	}
	return g.constant(arr), nil
}

func (g *Graph) constant(array kernels.Array) compute.Value {
	return &constant{
		node:  g.node(array.Shape(), array.Factory()),
		value: array,
	}
}

func (n *constant) exec(exec *executor) (kernels.Array, error) {
	return n.value, nil
}

func (n *constant) String() string {
	return n.value.String()
}

func toString(x any) string {
	s, ok := x.(fmt.Stringer)
	if ok {
		return s.String()
	}
	return fmt.Sprintf("%T", x)
}

type binary struct {
	node
	x, y   execNode
	kernel kernels.Binary
}

var _ execNode = (*binary)(nil)

// Add returns a node adding two nodes.
func (g *Graph) Add(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.ADD, x.(execNode), y.(execNode))
}

// Sub returns a node subtracting y from x.
func (g *Graph) Sub(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.SUB, x.(execNode), y.(execNode))
}

// Mul returns a node multiplying two nodes.
func (g *Graph) Mul(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.MUL, x.(execNode), y.(execNode))
}

// Div returns a node dividing x by y.
func (g *Graph) Div(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.QUO, x.(execNode), y.(execNode))
}

// Rem returns a node computing remainder of x divided by y.
func (g *Graph) Rem(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.REM, x.(execNode), y.(execNode))
}

// Equal returns a boolean node checking if x == y.
func (g *Graph) Equal(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.EQL, x.(execNode), y.(execNode))
}

// NotEqual returns a boolean node checking if x != y.
func (g *Graph) NotEqual(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.NEQ, x.(execNode), y.(execNode))
}

// LessThan returns a boolean node checking if x < y.
func (g *Graph) LessThan(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.LSS, x.(execNode), y.(execNode))
}

// LessOrEqual returns a boolean node checking if x <= y.
func (g *Graph) LessOrEqual(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.LEQ, x.(execNode), y.(execNode))
}

// GreaterThan returns a boolean node checking if x > y.
func (g *Graph) GreaterThan(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.GTR, x.(execNode), y.(execNode))
}

// GreaterOrEqual returns a boolean node checking if x >= y.
func (g *Graph) GreaterOrEqual(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.GEQ, x.(execNode), y.(execNode))
}

// ShiftLeft returns a node shifting x left by y.
func (g *Graph) ShiftLeft(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.SHL, x.(execNode), y.(execNode))
}

// ShiftRightArithmetic returns a node shifting lhs right by rhs, preserving the sign bit.
func (g *Graph) ShiftRightArithmetic(lhs, rhs compute.Value) (compute.Value, error) {
	return g.newBinary(token.SHR, lhs.(execNode), rhs.(execNode))
}

// ShiftRightLogical returns a node shifting lhs right by rhs, ignoring the sign bit.
func (g *Graph) ShiftRightLogical(lhs, rhs compute.Value) (compute.Value, error) {
	return g.newBinary(token.SHR, lhs.(execNode), rhs.(execNode))
}

// BitwiseAnd returns a node computing bitwise AND of x and y.
func (g *Graph) BitwiseAnd(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.AND, x.(execNode), y.(execNode))
}

// BitwiseOr returns a node computing bitwise OR of x and y.
func (g *Graph) BitwiseOr(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.OR, x.(execNode), y.(execNode))
}

// BitwiseXor returns a node computing bitwise XOR of x and y.
func (g *Graph) BitwiseXor(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.XOR, x.(execNode), y.(execNode))
}

// LogicalAnd returns a node computing logical AND of x and y.
func (g *Graph) LogicalAnd(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.LAND, x.(execNode), y.(execNode))
}

// LogicalOr returns a node computing logical OR of x and y.
func (g *Graph) LogicalOr(x, y compute.Value) (compute.Value, error) {
	return g.newBinary(token.LOR, x.(execNode), y.(execNode))
}

func (g *Graph) newBinary(op token.Token, x, y execNode) (compute.Value, error) {
	xShape := x.shape()
	yShape := y.shape()
	kernel, shape, err := x.kernelFactory().BinaryOp(op, xShape, yShape)
	if err != nil {
		return nil, err
	}
	return &binary{
		node:   g.node(shape, x.kernelFactory()),
		x:      x,
		y:      y,
		kernel: kernel,
	}, nil
}

func (n binary) exec(exec *executor) (kernels.Array, error) {
	x, err := n.x.exec(exec)
	if err != nil {
		return nil, err
	}
	y, err := n.y.exec(exec)
	if err != nil {
		return nil, err
	}
	return n.kernel(x, y)
}

func (n binary) String() string {
	return fmt.Sprintf("%s(%s,%s)", toString(n.kernel), toString(n.x), toString(n.y))
}

type unary struct {
	node
	x      execNode
	kernel kernels.Unary
}

var _ execNode = (*unary)(nil)

func (n unary) exec(exec *executor) (kernels.Array, error) {
	x, err := n.x.exec(exec)
	if err != nil {
		return nil, err
	}
	return n.kernel(x)
}

// LogicalNot returns a node computing the logical not of x.
func (g *Graph) LogicalNot(x compute.Value) (compute.Value, error) {
	return g.newUnary(token.NOT, x.(execNode))
}

// Neg returns a node computing the negation of x.
func (g *Graph) Neg(x compute.Value) (compute.Value, error) {
	return g.newUnary(token.SUB, x.(execNode))
}

func (g *Graph) newUnary(op token.Token, x execNode) (compute.Value, error) {
	xShape := x.shape()
	kernel, shape, err := x.kernelFactory().UnaryOp(op, xShape)
	if err != nil {
		return nil, err
	}
	return &unary{
		node:   g.node(shape, x.kernelFactory()),
		x:      x,
		kernel: kernel,
	}, nil
}

func (n unary) String() string {
	return fmt.Sprintf("%s(%s)", toString(n.kernel), toString(n.x))
}

// ConvertDType returns a cast/convert operator node.
func (g *Graph) ConvertDType(x compute.Value, dtype dtypes.DType) (compute.Value, error) {
	return g.newCast(x.(execNode), dtype)
}

func (g *Graph) newCast(x execNode, dt dtypes.DType) (compute.Value, error) {
	xShape := x.shape()
	kernel, shape, factory, err := x.kernelFactory().Cast(dt, xShape.Dimensions)
	if err != nil {
		return nil, err
	}
	return &unary{
		node:   g.node(shape, factory),
		x:      x,
		kernel: kernel,
	}, nil
}

// Reshape returns a reshape operator node.
func (g *Graph) Reshape(x compute.Value, dimensions ...int) (compute.Value, error) {
	return g.newReshape(x.(execNode), dimensions)
}

func (g *Graph) newReshape(x execNode, axisLengths []int) (compute.Value, error) {
	kernel, shap, err := x.kernelFactory().Reshape(x.shape(), axisLengths)
	if err != nil {
		return nil, err
	}
	return &unary{
		node:   g.node(shap, x.kernelFactory()),
		x:      x,
		kernel: kernel,
	}, nil
}

type nAry struct {
	node
	xs     []execNode
	kernel kernels.NAry
}

func (n nAry) exec(exec *executor) (kernels.Array, error) {
	arrays := make([]kernels.Array, len(n.xs))
	for i, x := range n.xs {
		var err error
		arrays[i], err = x.exec(exec)
		if err != nil {
			return nil, err
		}
	}
	return n.kernel(arrays)
}

// Concatenate concatenates multiple arrays into a single array.
func (g *Graph) Concatenate(axis int, operands ...compute.Value) (compute.Value, error) {
	eNodes := make([]execNode, len(operands))
	for i, n := range operands {
		eNodes[i] = n.(execNode)
	}
	return g.newConcat(axis, eNodes)
}

func (g *Graph) newConcat(axis int, nodes []execNode) (compute.Value, error) {
	if axis != 0 {
		return nil, errors.Errorf("axis != 0 not supported")
	}
	for _, node := range nodes {
		if node.shape().Size() != 1 {
			return nil, errors.Errorf("concatenating arrays not supported")
		}
	}
	x := nodes[0]
	kernel, shape, err := x.kernelFactory().Concat(x.shape().DType, len(nodes))
	if err != nil {
		return nil, err
	}
	return &nAry{
		node:   g.node(shape, x.kernelFactory()),
		xs:     nodes,
		kernel: kernel,
	}, nil
}

// DotGeneral returns a general dot operator node.
func (g *Graph) DotGeneral(lhs compute.Value, lhsContractingAxes, lhsBatchAxes []int, rhs compute.Value, rhsContractingAxes, rhsBatchAxes []int, config compute.DotGeneralConfig) (compute.Value, error) {
	return nil, errors.Errorf("not implemented")
}

// DynamicUpdateSlice updates a slice in an array.
func (g *Graph) DynamicUpdateSlice(operand, update compute.Value, startIndices []compute.Value) (compute.Value, error) {
	return nil, errors.Errorf("not implemented")
}

// Slice returns a slice on a node.
func (g *Graph) Slice(x compute.Value, starts, limits, strides []int) (compute.Value, error) {
	node := x.(execNode)
	xShape := node.shape()
	kernel, shape, err := node.kernelFactory().Slice(xShape, starts[0])
	if err != nil {
		return nil, err
	}
	return &unary{
		node:   g.node(shape, node.kernelFactory()),
		x:      node,
		kernel: kernel,
	}, nil
}

// Call invokes another function with the given inputs.
func (g *Graph) Call(f compute.Function, inputs ...compute.Value) ([]compute.Value, error) {
	return f.(*Graph).outputs, nil
}

// While returns a while loop node.
func (g *Graph) While(cond, body compute.Function, initialState ...compute.Value) ([]compute.Value, error) {
	return nil, errors.Errorf("not implemented")
}

// BroadcastInDim broadcasts data across a given set of axis.
func (g *Graph) BroadcastInDim(x compute.Value, shape shapes.Shape, broadcastAxes []int) (compute.Value, error) {
	node := x.(execNode)
	if node.shape().Size() > 1 {
		return nil, errors.Errorf("cannot broadcast shape %v: this backend only supports the broadcast of atomic shapes", shape)
	}
	kernel, shape, err := node.kernelFactory().BroadcastInDim(shape, broadcastAxes)
	if err != nil {
		return nil, err
	}
	return &unary{
		node:   g.node(shape, node.kernelFactory()),
		x:      node,
		kernel: kernel,
	}, nil
}
