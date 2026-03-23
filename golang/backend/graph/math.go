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

// Abs returns the absolute value of x.
func (g *Graph) Abs(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Abs)
}

// Ceil returns the ceiling of x.
func (g *Graph) Ceil(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Ceil)
}

// Cos returns the computation for cosine.
func (g *Graph) Cos(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Cos)
}

// Erf returns the error function of x.
func (g *Graph) Erf(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Erf)
}

// Exp returns the computation for the exponential.
func (g *Graph) Exp(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Exp)
}

// Expm1 returns Exp(x)-1.
func (g *Graph) Expm1(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Expm1)
}

// Floor returns the floor of x.
func (g *Graph) Floor(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Floor)
}

// Log returns the natural logarithm of x.
func (g *Graph) Log(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Log)
}

// Log1p returns log(1+x).
func (g *Graph) Log1p(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Log1p)
}

// Logistic returns 1/(1+exp(-x)).
func (g *Graph) Logistic(x ops.Node) (ops.Node, error) {
	return nil, errors.Errorf("Logistic not implemented in the Go backend")
}

// Round returns the nearest integer of x.
func (g *Graph) Round(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Round)
}

// Rsqrt returns 1/sqrt(x).
func (g *Graph) Rsqrt(x ops.Node) (ops.Node, error) {
	return nil, errors.Errorf("Rsqrt not implemented in the Go backend")
}

// Sign returns the sign of x.
func (g *Graph) Sign(x ops.Node) (ops.Node, error) {
	return nil, errors.Errorf("Sign not implemented in the Go backend")
}

// Sin returns a node computing the sine.
func (g *Graph) Sin(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Sin)
}

// Sqrt returns sqrt(x).
func (g *Graph) Sqrt(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Sqrt)
}

// Tanh returns a node computing the hyperbolic tangent.
func (g *Graph) Tanh(x ops.Node) (ops.Node, error) {
	return newMathUnaryNode(g, x, math.Tanh)
}
