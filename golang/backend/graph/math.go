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

package graph

import (
	"math"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Math returns the builder for the math package.
func (g *Graph) Math() ops.MathBuilder {
	return g
}

func mathFactory(x ops.Node) (execNode, kernels.MathFactory, error) {
	xT := x.(execNode)
	fac := xT.kernelFactory().Math()
	if fac == nil {
		return nil, nil, errors.Errorf("native math factory not implemented for %T", x)
	}
	return xT, fac, nil
}

type mathFunc = func(kernels.MathFactory) kernels.Unary

func newMathUnaryNode(g *Graph, x ops.Node, f func(float64) float64) (ops.Node, error) {
	xT, mth, err := mathFactory(x)
	if err != nil {
		return nil, err
	}
	return &unary{
		node:   g.node(xT.shape(), xT.kernelFactory()),
		x:      xT,
		kernel: mth.Kernelize(f),
	}, nil
}

// Cos returns a node computing the cosine.
func (g *Graph) Cos(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Cos)
}

// Sin returns a node computing the sine.
func (g *Graph) Sin(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Sin)
}

// Tanh returns a node computing the hyperbolic tangent.
func (g *Graph) Tanh(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Tanh)
}
