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

package numbers

import (
	"fmt"
	"go/token"
	"math/big"

	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/materialise"
)

// Float is a GX number.
type Float struct {
	canonical.AtomStringImpl
	expr elements.ExprAt
	val  *big.Float

	concrete ir.Type // Concrete type in the interpreter.
}

var (
	_ Number               = (*Float)(nil)
	_ canonical.Simplifier = (*Float)(nil)
)

// NewFloat returns a new element Float number element.
func NewFloat(expr elements.ExprAt, val *big.Float) *Float {
	return &Float{
		expr:     expr,
		val:      val,
		concrete: toConcrete(nil, expr.Node().Type()),
	}
}

// UnaryOp applies a unary operator on x.
func (n *Float) UnaryOp(env evaluator.Env, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	var val *big.Float
	switch expr.Src.Op {
	case token.ADD:
		return n, nil
	case token.SUB:
		val = new(big.Float).Neg(n.val)
	default:
		return nil, fmterr.Errorf(env.File().FileSet(), expr.Src, "number int unary operator %s not implemented", expr.Src.Op)
	}
	return &Float{
		expr:     elements.NewExprAt(env.File(), expr),
		val:      val,
		concrete: n.concrete,
	}, nil
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (n *Float) BinaryOp(env evaluator.Env, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	switch yT := y.(type) {
	case *Float:
		return binaryFloat(env, expr, n, yT)
	case *Int:
		return binaryFloat(env, expr, n, yT.toFloat())
	}
	return cpevelements.NewBinary(env, expr, x, y)
}

func binaryFloat(env evaluator.Env, expr *ir.BinaryExpr, xFloat, yFloat *Float) (evaluator.NumericalElement, error) {
	x, y := xFloat.val, yFloat.val
	var val *big.Float
	switch expr.Src.Op {
	case token.ADD:
		val = new(big.Float).Add(x, y)
	case token.SUB:
		val = new(big.Float).Sub(x, y)
	case token.MUL:
		val = new(big.Float).Mul(x, y)
	case token.QUO:
		val = new(big.Float).Quo(x, y)
	default:
		return nil, fmterr.Errorf(env.File().FileSet(), expr.Src, "number int binary operator %s not implemented", expr.Src.Op)
	}
	return &Float{
		expr:     elements.NewExprAt(env.File(), expr),
		val:      val,
		concrete: toConcrete(expr.Type(), xFloat.concrete, yFloat.concrete),
	}, nil
}

// Cast an element into a given data type.
func (n *Float) Cast(env evaluator.Env, expr ir.Expr, target ir.Type) (evaluator.NumericalElement, error) {
	return &Float{
		expr:     elements.NewExprAt(env.File(), expr),
		val:      n.val,
		concrete: toConcrete(target, n.concrete),
	}, nil
}

// Reshape the number into an array.
func (n *Float) Reshape(env evaluator.Env, expr ir.Expr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return cpevelements.NewReshape(env, expr, n, axisLengths)
}

// Shape of the value represented by the element.
func (n *Float) Shape() *shape.Shape {
	return numberShape
}

// Type of the element.
func (n *Float) Type() ir.Type {
	return n.expr.Node().Type()
}

// Float value of the number.
func (n *Float) Float() *big.Float {
	return n.val
}

// Compare with another number.
func (n *Float) Compare(x canonical.Comparable) (bool, error) {
	switch xT := x.(type) {
	case *Float:
		return n.val.Cmp(xT.val) == 0, nil
	case *Int:
		return n.val.Cmp(xT.Float()) == 0, nil
	}
	// Because the compiler cast numbers to concrete types,
	// numbers should only be compared to other numbers.
	// Always return false if that is not the case.
	return false, nil
}

// CanonicalExpr returns the canonical expression used for comparison.
func (n *Float) CanonicalExpr() canonical.Canonical {
	return n
}

// Simplify returns the expression simplified.
func (n *Float) Simplify() canonical.Simplifier {
	return n
}

// Copy returns the receiver.
func (n *Float) Copy() elements.Copier {
	return n
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *Float) NumericalConstant() (*values.HostArray, error) {
	if n.concrete == nil {
		return nil, fmterr.Internalf(n.expr.File().FileSet(), n.expr.Source(), "number %s:%s has no concrete type", n.expr.String(), n.expr.Node().Type().String())
	}
	return values.AtomNumberFloat(n.val, n.concrete)
}

// Unflatten creates a GX value from the next handles available in the parser.
func (n *Float) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(n.expr.Node().Type())
}

// Materialise the value into a node in the backend graph.
func (n *Float) Materialise(ao materialise.Materialiser) (materialise.Node, error) {
	val, err := n.NumericalConstant()
	if err != nil {
		return nil, err
	}
	return ao.NodeFromArray(n.expr.File(), n.expr.Node(), val)
}

// ShortString returns a short string representation of the value.
func (n *Float) ShortString() string {
	return n.expr.Node().String()
}

// String return the float literal.
func (n *Float) String() string {
	val := n.expr.Node().String()
	if n.Type().Kind() == irkind.NumberFloat {
		return val
	}
	return fmt.Sprintf("%s(%s)", n.Type(), val)
}
