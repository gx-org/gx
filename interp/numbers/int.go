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

	"github.com/pkg/errors"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

// Int is a GX number.
type Int struct {
	expr elements.ExprAt
	val  *big.Int
}

var _ Number = (*Int)(nil)

// NewInt returns a new element Int number element.
func NewInt(expr elements.ExprAt, val *big.Int) Number {
	return &Int{expr: expr, val: val}
}

// UnaryOp applies a unary operator on x.
func (n *Int) UnaryOp(ctx ir.Evaluator, expr *ir.UnaryExpr) (elements.NumericalElement, error) {
	var val *big.Int
	switch expr.Src.Op {
	case token.ADD:
		return n, nil
	case token.SUB:
		val = new(big.Int).Neg(n.val)
	default:
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Src, "number int unary operator %s not implemented", expr.Src.Op)
	}
	return NewInt(elements.NewExprAt(ctx.File(), expr), val), nil
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (n *Int) BinaryOp(ctx ir.Evaluator, expr *ir.BinaryExpr, x, y elements.NumericalElement) (elements.NumericalElement, error) {
	switch yT := y.(type) {
	case *Float:
		return binaryFloat(ctx, expr, n.Float(), yT.val)
	case *Int:
		return binaryInt(ctx, expr, n.val, yT.val)
	}
	return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Src, "number int operator not implemented for %T%s%T", x, expr.Src.Op, y)
}

// Float value of the integer.
func (n *Int) Float() *big.Float {
	return new(big.Float).SetInt(n.val)
}

func binaryInt(ctx ir.Evaluator, expr *ir.BinaryExpr, x, y *big.Int) (elements.NumericalElement, error) {
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
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Src, "number int binary operator %s not implemented", expr.Src.Op)
	}
	return NewInt(elements.NewExprAt(ctx.File(), expr), val), nil
}

// Cast an element into a given data type.
func (n *Int) Cast(ctx ir.Evaluator, expr ir.AssignableExpr, target ir.Type) (elements.NumericalElement, error) {
	val, err := values.AtomNumberInt(n.val, target)
	if err != nil {
		return nil, err
	}
	return ctx.(evaluator.Context).Evaluator().ElementFromAtom(elements.NewExprAt(ctx.File(), expr), val)
}

// Reshape the number into an array.
func (n *Int) Reshape(ctx ir.Evaluator, expr ir.AssignableExpr, axisLengths []elements.NumericalElement) (elements.NumericalElement, error) {
	val, err := values.AtomNumberInt(n.val, expr.Type())
	if err != nil {
		return nil, err
	}
	return ctx.(evaluator.Context).Evaluator().ElementFromAtom(elements.NewExprAt(ctx.File(), expr), val)
}

// Shape of the value represented by the element.
func (n *Int) Shape() *shape.Shape {
	return numberShape
}

// Flatten returns the number in a slice of elements.
func (n *Int) Flatten() ([]elements.Element, error) {
	return []elements.Element{n}, nil
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (n *Int) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return nil, fmterr.Internal(errors.Errorf("%T does not support converting device handles into GX values", n))
}

// Type of the element.
func (n *Int) Type() ir.Type {
	return n.expr.Node().Type()
}

// Compare with another number.
func (n *Int) Compare(x canonical.Comparable) bool {
	switch xT := x.(type) {
	case *Float:
		return n.Float().Cmp(xT.val) == 0
	case *Int:
		return n.val.Cmp(xT.val) == 0
	case elements.ElementWithConstant:
		val := xT.NumericalConstant()
		if !val.Shape().IsAtomic() {
			return false
		}
		other, err := val.ToAtom()
		if err != nil {
			return false
		}
		var otherI *big.Int
		switch otherT := other.(type) {
		case int32:
			otherI = big.NewInt(int64(otherT))
		case int64:
			otherI = big.NewInt(otherT)
		default:
			return false
		}
		return n.val.Cmp(otherI) == 0
	}
	// Because the compiler cast numbers to concrete types,
	// numbers should only be compared to other numbers.
	// Always return false if that is not the case.
	return false
}

// CanonicalExpr returns the canonical expression used for comparison.
func (n *Int) CanonicalExpr() canonical.Canonical {
	return n
}

// String return the float literal.
func (n *Int) String() string {
	return fmt.Sprint(n.expr.Node())
}
