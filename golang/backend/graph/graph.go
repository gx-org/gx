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
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Graph of the Go native backend.
type Graph struct {
	plat     platform.Platform
	funcName string
}

var _ graph.Graph = (*Graph)(nil)

// New graph running operations implemented in Go.
func New(plat platform.Platform, funcName string) *Graph {
	return &Graph{
		plat:     plat,
		funcName: funcName,
	}
}

// Platform owning the graph.
func (g *Graph) Platform() platform.Platform {
	return g.plat
}

// Core returns the builder to build core operations.
func (g *Graph) Core() graph.CoreBuilder {
	return g
}

// Graph returns the graph in which the nodes are created into.
func (g *Graph) Graph() graph.Graph {
	return g
}

type node struct {
	g  *Graph
	k  kernels.Factory
	sh *shape.Shape
}

var _ graph.Node = (*node)(nil)

func (g *Graph) node(shape *shape.Shape, k kernels.Factory) node {
	return node{g: g, sh: shape, k: k}
}

func (n *node) shape() *shape.Shape {
	return n.sh
}

func (n *node) Graph() graph.Graph {
	return n.g
}

func (n *node) BackendShape() *shape.Shape {
	return n.sh
}

func (n *node) Platform() platform.Platform {
	return n.g.Platform()
}

func (n *node) kernelFactory() kernels.Factory {
	return n.k
}

type constant struct {
	node
	value kernels.Array
}

var _ execNode = (*constant)(nil)

// NewConstant returns a node representing a numerical constant value in the graph.
func (g *Graph) NewConstant(value platform.HostBuffer) (graph.Node, error) {
	array := value.(*kernels.Buffer).KernelValue()
	return &constant{
		node:  g.node(value.Shape(), array.Factory()),
		value: array.(kernels.Array),
	}, nil
}

func (n *constant) exec(exec *executor) (kernels.Array, error) {
	return n.value, nil
}

type binary struct {
	node
	x, y   execNode
	kernel kernels.Binary
}

var _ execNode = (*binary)(nil)

// NewBinary returns a node applying a binary operator between two nodes.
func (g *Graph) NewBinary(op *ast.BinaryExpr, x, y graph.Node) (graph.Node, error) {
	return g.newBinary(op, x.(execNode), y.(execNode))
}

func (g *Graph) newBinary(op *ast.BinaryExpr, x, y execNode) (graph.Node, error) {
	xShape := x.shape()
	yShape := y.shape()
	kernel, shape, err := x.kernelFactory().BinaryOp(op.Op, xShape, yShape)
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

// NewUnary returns a node applying a unary operator to a node.
func (g *Graph) NewUnary(op *ast.UnaryExpr, x graph.Node) (graph.Node, error) {
	return g.newUnary(op, x.(execNode))
}

func (g *Graph) newUnary(op *ast.UnaryExpr, x execNode) (graph.Node, error) {
	xShape := x.shape()
	kernel, shape, err := x.kernelFactory().UnaryOp(op.Op, xShape)
	if err != nil {
		return nil, err
	}
	return &unary{
		node:   g.node(shape, x.kernelFactory()),
		x:      x,
		kernel: kernel,
	}, nil
}

// NewCast returns a cast/convert operator node.
func (g *Graph) NewCast(x graph.Node, dt dtype.DataType) (graph.Node, error) {
	return g.newCast(x.(execNode), dt)
}

func (g *Graph) newCast(x execNode, dt dtype.DataType) (graph.Node, error) {
	xShape := x.shape()
	kernel, shape, factory, err := x.kernelFactory().Cast(dt, xShape.AxisLengths)
	if err != nil {
		return nil, err
	}
	return &unary{
		node:   g.node(shape, factory),
		x:      x,
		kernel: kernel,
	}, nil
}

// NewReshape returns a reshape operator node.
func (g *Graph) NewReshape(x graph.Node, axisLengths []int) (graph.Node, error) {
	return g.newReshape(x.(execNode), axisLengths)
}

func (g *Graph) newReshape(x execNode, axisLengths []int) (graph.Node, error) {
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

// NewConcat concatenates multiple arrays into a single array.
func (g *Graph) NewConcat(axis int, nodes []graph.Node) (graph.Node, error) {
	eNodes := make([]execNode, len(nodes))
	for i, n := range nodes {
		eNodes[i] = n.(execNode)
	}
	return g.newConcat(axis, eNodes)
}

func (g *Graph) newConcat(axis int, nodes []execNode) (graph.Node, error) {
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

// NewDotGeneral returns a general dot operator node.
func (g *Graph) NewDotGeneral(x, y graph.Node, batchAxes, reduceAxes [2][]int) (graph.Node, error) {
	return nil, errors.Errorf("not implemented")
}

// NewSet returns a node to set a slice in an array.
func (g *Graph) NewSet(x, updates, index graph.Node) (graph.Node, error) {
	return nil, errors.Errorf("not implemented")
}

// NewSlice returns a slice on a node.
func (g *Graph) NewSlice(x graph.Node, index int) (graph.Node, error) {
	return nil, errors.Errorf("not implemented")
}
