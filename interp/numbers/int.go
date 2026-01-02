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

// Int is a GX number.
type Int struct {
	canonical.AtomStringImpl
	expr elements.ExprAt
	val  *big.Int

	concrete ir.Type // Concrete type in the interpreter.
}

var (
	_ Number       = (*Int)(nil)
	_ ir.Canonical = (*Int)(nil)
)

// NewInt returns a new element Int number element.
func NewInt(expr elements.ExprAt, val *big.Int) Number {
	return &Int{
		expr:     expr,
		val:      val,
		concrete: toConcrete(nil, expr.Node().Type()),
	}
}

// UnaryOp applies a unary operator on x.
func (n *Int) UnaryOp(env evaluator.Env, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	var val *big.Int
	switch expr.Src.Op {
	case token.ADD:
		return n, nil
	case token.SUB:
		val = new(big.Int).Neg(n.val)
	default:
		return nil, fmterr.Errorf(env.File().FileSet(), expr.Src, "number int unary operator %s not implemented", expr.Src.Op)
	}
	return &Int{
		expr:     elements.NewExprAt(env.File(), expr),
		val:      val,
		concrete: n.concrete,
	}, nil
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (n *Int) BinaryOp(env evaluator.Env, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	switch yT := y.(type) {
	case *Float:
		return binaryFloat(env, expr, n.toFloat(), yT)
	case *Int:
		return binaryInt(env, expr, n, yT)
	}
	return cpevelements.NewBinary(env, expr, x, y)
}

func (n *Int) toFloat() *Float {
	return &Float{
		expr:     n.expr,
		val:      n.Float(),
		concrete: n.concrete,
	}
}

// Float value of the integer.
func (n *Int) Float() *big.Float {
	return new(big.Float).SetInt(n.val)
}

func binaryInt(env evaluator.Env, expr *ir.BinaryExpr, xInt, yInt *Int) (evaluator.NumericalElement, error) {
	x, y := xInt.val, yInt.val
	var val *big.Int
	switch expr.Src.Op {
	case token.ADD:
		val = new(big.Int).Add(x, y)
	case token.SUB:
		val = new(big.Int).Sub(x, y)
	case token.MUL:
		val = new(big.Int).Mul(x, y)
	case token.QUO:
		val = new(big.Int).Div(x, y)
	case token.REM:
		val = new(big.Int).Rem(x, y)
	case token.SHL:
		val = new(big.Int).Lsh(x, uint(y.Uint64()))
	case token.SHR:
		val = new(big.Int).Rsh(x, uint(y.Uint64()))
	case token.AND:
		val = new(big.Int).And(x, y)
	case token.OR:
		val = new(big.Int).Or(x, y)
	case token.XOR:
		val = new(big.Int).Xor(x, y)
	default:
		return cpevelements.NewBinary(env, expr, xInt, yInt)
	}
	return &Int{
		expr:     elements.NewExprAt(env.File(), expr),
		val:      val,
		concrete: toConcrete(expr.Type(), xInt.concrete, yInt.concrete),
	}, nil
}

// Cast an element into a given data type.
func (n *Int) Cast(env evaluator.Env, expr ir.AssignableExpr, target ir.Type) (evaluator.NumericalElement, error) {
	return &Int{
		expr:     elements.NewExprAt(env.File(), expr),
		val:      n.val,
		concrete: toConcrete(target, n.concrete),
	}, nil
}

// Reshape the number into an array.
func (n *Int) Reshape(env evaluator.Env, expr ir.AssignableExpr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return cpevelements.NewReshape(env, expr, n, axisLengths)
}

// Shape of the value represented by the element.
func (n *Int) Shape() *shape.Shape {
	return numberShape
}

// Type of the element.
func (n *Int) Type() ir.Type {
	return n.expr.Node().Type()
}

// Compare with another number.
func (n *Int) Compare(x canonical.Comparable) (bool, error) {
	switch xT := x.(type) {
	case *Float:
		return n.Float().Cmp(xT.val) == 0, nil
	case *Int:
		return n.val.Cmp(xT.val) == 0, nil
	case elements.ElementWithConstant:
		val, err := xT.NumericalConstant()
		if err != nil {
			return false, err
		}
		if val == nil {
			return false, nil
		}
		if !val.Shape().IsAtomic() {
			return false, nil
		}
		other, err := val.ToAtom()
		if err != nil {
			return false, nil
		}
		var otherI *big.Int
		switch otherT := other.(type) {
		case int32:
			otherI = big.NewInt(int64(otherT))
		case int64:
			otherI = big.NewInt(otherT)
		default:
			return false, nil
		}
		return n.val.Cmp(otherI) == 0, nil
	}
	// Because the compiler cast numbers to concrete types,
	// numbers should only be compared to other numbers.
	// Always return false if that is not the case.
	return false, nil
}

// CanonicalExpr returns the canonical expression used for comparison.
func (n *Int) CanonicalExpr() canonical.Canonical {
	return n
}

// Expr returns an IR expression representing the integer value.
func (n *Int) Expr() (ir.AssignableExpr, error) {
	return n.expr.Node(), nil
}

// Copy returns the receiver.
func (n *Int) Copy() elements.Copier {
	return n
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *Int) NumericalConstant() (*values.HostArray, error) {
	if n.concrete == nil {
		return nil, fmterr.Internalf(n.expr.File().FileSet(), n.expr.Source(), "number %s:%s has no concrete type", n.expr.String(), n.expr.Node().Type().String())
	}
	return values.AtomNumberInt(n.val, n.concrete)
}

// Unflatten creates a GX value from the next handles available in the parser.
func (n *Int) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(n.expr.Node().Type())
}

// Materialise the value into a node in the backend graph.
func (n *Int) Materialise(ao materialise.Materialiser) (materialise.Node, error) {
	val, err := n.NumericalConstant()
	if err != nil {
		return nil, err
	}
	return ao.NodeFromArray(n.expr.File(), n.expr.Node(), val)
}

// String return the float literal.
func (n *Int) String() string {
	val := n.expr.Node().String()
	if n.Type().Kind() == irkind.NumberInt {
		return val
	}
	return fmt.Sprintf("%s(%s)", n.Type(), val)
}
