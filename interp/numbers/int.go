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
	"go/ast"
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
	val  *big.Int
	expr ir.Expr
	typ  ir.Type
}

var (
	_ Number       = (*Int)(nil)
	_ ir.Canonical = (*Int)(nil)
)

// NewInt returns a new element Int number element.
func NewInt(env evaluator.Env, expr ir.Expr, val *big.Int) (*Int, error) {
	typ, cpErr, err := env.ToConcrete(expr.Expr(), expr.Type())
	return NewIntForType(expr, val, typ), ir.UnifyErr(cpErr, err)
}

// NewIntForType returns a new element Int number element for a given type.
func NewIntForType(expr ir.Expr, val *big.Int, typ ir.Type) *Int {
	return &Int{
		val:  val,
		expr: expr,
		typ:  typ,
	}
}

// UnaryOp applies a unary operator on x.
func (n *Int) UnaryOp(env evaluator.Env, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	switch expr.Src.Op {
	case token.ADD:
		return n, nil
	case token.SUB:
		return &Int{
			val:  new(big.Int).Neg(n.val),
			expr: expr,
			typ:  n.typ,
		}, nil
	default:
		return nil, fmterr.Errorf(env.File().FileSet(), expr.Src, "number int unary operator %s not implemented", expr.Src.Op)
	}
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
	return &Float{val: n.Float(), expr: n.expr, typ: n.typ}
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
	typ, cpErr, err := env.ToConcrete(expr.Src, expr.Typ)
	return &Int{
		val:  val,
		expr: expr,
		typ:  typ,
	}, ir.UnifyErr(cpErr, err)
}

// Cast an element into a given data type.
func (n *Int) Cast(env evaluator.Env, expr ir.Expr, target ir.Type) (evaluator.NumericalElement, error) {
	typ, cpErr, err := env.ToConcrete(expr.Expr(), target)
	return &Int{
		val:  n.val,
		expr: expr,
		typ:  typ,
	}, ir.UnifyErr(cpErr, err)
}

// Reshape the number into an array.
func (n *Int) Reshape(env evaluator.Env, expr ir.Expr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return cpevelements.NewReshape(env, expr, n, axisLengths)
}

// Shape of the value represented by the element.
func (n *Int) Shape() *shape.Shape {
	return numberShape
}

// Expr returns the expression representing the integer.
func (n *Int) Expr(ir.Evaluator, ast.Expr) (ir.Expr, ir.CompEvalError, error) {
	return n.expr, nil, nil
}

// Type of the element.
func (n *Int) Type() ir.Type {
	return n.typ
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

// Copy returns the receiver.
func (n *Int) Copy() elements.Copier {
	return n
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *Int) NumericalConstant() (*values.HostArray, error) {
	return values.AtomNumberInt(n.val, n.typ)
}

// Unflatten creates a GX value from the next handles available in the parser.
func (n *Int) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(n.typ)
}

// Materialise the value into a node in the backend graph.
func (n *Int) Materialise(ao materialise.Materialiser) (materialise.Node, error) {
	val, err := n.NumericalConstant()
	if err != nil {
		return nil, err
	}
	return ao.NodeFromArray(val, n.typ)
}

// ShortString returns a short string representation of the value.
func (n *Int) ShortString() string {
	return n.SourceString(nil)
}

// SourceString returns the GX source code to represent the float.
func (n *Int) SourceString(from *ir.File) string {
	val := n.expr.SourceString(from)
	if n.typ.Kind() == irkind.NumberInt {
		return val
	}
	return fmt.Sprintf("%s(%s)", n.Type().ReferString(from), val)
}
