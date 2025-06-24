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

// Float is a GX number.
type Float struct {
	expr elements.ExprAt
	val  *big.Float
}

var (
	_ Number               = (*Float)(nil)
	_ canonical.Simplifier = (*Float)(nil)
)

// NewFloat returns a new element Float number element.
func NewFloat(expr elements.ExprAt, val *big.Float) *Float {
	return &Float{expr: expr, val: val}
}

// UnaryOp applies a unary operator on x.
func (n *Float) UnaryOp(ctx elements.FileContext, expr *ir.UnaryExpr) (elements.NumericalElement, error) {
	var val *big.Float
	switch expr.Src.Op {
	case token.ADD:
		return n, nil
	case token.SUB:
		val = new(big.Float).Neg(n.val)
	default:
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Src, "number int unary operator %s not implemented", expr.Src.Op)
	}
	return NewFloat(elements.NewExprAt(ctx.File(), expr), val), nil
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (n *Float) BinaryOp(ctx elements.FileContext, expr *ir.BinaryExpr, x, y elements.NumericalElement) (elements.NumericalElement, error) {
	switch yT := y.(type) {
	case *Float:
		return binaryFloat(ctx, expr, n.val, yT.val)
	case *Int:
		return binaryFloat(ctx, expr, n.val, yT.Float())
	}
	return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Src, "number int operator not implemented for %T%s%T", x, expr.Src.Op, y)
}

func binaryFloat(ctx elements.FileContext, expr *ir.BinaryExpr, x, y *big.Float) (elements.NumericalElement, error) {
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
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Src, "number int binary operator %s not implemented", expr.Src.Op)
	}
	return NewFloat(elements.NewExprAt(ctx.File(), expr), val), nil
}

// Cast an element into a given data type.
func (n *Float) Cast(ctx elements.FileContext, expr ir.AssignableExpr, target ir.Type) (elements.NumericalElement, error) {
	val, err := values.AtomNumberFloat(n.val, target)
	if err != nil {
		return nil, err
	}
	return ctx.(evaluator.Context).Evaluation().Evaluator().ElementFromAtom(elements.NewExprAt(ctx.File(), expr), val)
}

// Reshape the number into an array.
func (n *Float) Reshape(ctx elements.FileContext, expr ir.AssignableExpr, axisLengths []elements.NumericalElement) (elements.NumericalElement, error) {
	val, err := values.AtomNumberFloat(n.val, expr.Type())
	if err != nil {
		return nil, err
	}
	return ctx.(evaluator.Context).Evaluation().Evaluator().ElementFromAtom(elements.NewExprAt(ctx.File(), expr), val)
}

// Shape of the value represented by the element.
func (n *Float) Shape() *shape.Shape {
	return numberShape
}

// Flatten returns the number in a slice of elements.
func (n *Float) Flatten() ([]elements.Element, error) {
	return []elements.Element{n}, nil
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (n *Float) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return nil, fmterr.Internal(errors.Errorf("%T does not support converting device handles into GX values", n))
}

// Kind of the element.
func (n *Float) Kind() ir.Kind {
	return ir.NumberFloatKind
}

// Float value of the number.
func (n *Float) Float() *big.Float {
	return n.val
}

// Compare with another number.
func (n *Float) Compare(x canonical.Comparable) bool {
	switch xT := x.(type) {
	case *Float:
		return n.val.Cmp(xT.val) == 0
	case *Int:
		return n.val.Cmp(xT.Float()) == 0
	}
	// Because the compiler cast numbers to concrete types,
	// numbers should only be compared to other numbers.
	// Always return false if that is not the case.
	return false
}

// CanonicalExpr returns the canonical expression used for comparison.
func (n *Float) CanonicalExpr() canonical.Canonical {
	return n
}

// Simplify returns the expression simplified.
func (n *Float) Simplify() canonical.Simplifier {
	return n
}

// String return the float literal.
func (n *Float) String() string {
	return fmt.Sprint(n.expr.Node())
}
