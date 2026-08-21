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
	"go/ast"
	"go/token"
	"math/big"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/constants"
	"github.com/gx-org/gx/internal/interp/fmtexpr"
	"github.com/gx-org/gx/interp/engine"
)

// elInt is a GX number.
type elInt struct {
	fmtexpr.AtomStringImpl
	val *big.Int
}

func newInt(f *big.Int) *elInt {
	return &elInt{val: f}
}

// NewIntNumber returns a new element Int number element.
func NewIntNumber(x *ir.NumberInt) engine.ScalarNumber {
	return newInt(x.Val)
}

func newIntFrom(val int64, typ ir.Type) engine.Constant {
	elI := newInt(big.NewInt(val))
	irI := elI.BuildIR()
	return elI.Cast(&ir.NumberCastExpr{X: irI, Typ: typ})
}

// UnaryOp applies a unary operator on x.
func (n *elInt) UnaryOp(env ir.SourceFile, expr *ir.UnaryExpr) (engine.ScalarNumber, error) {
	switch expr.Src.Op {
	case token.ADD:
		return n, nil
	case token.SUB:
		return newInt(new(big.Int).Neg(n.val)), nil
	default:
		return nil, fmterr.InternalAt(env.File().FileSet(), expr.Src, "unary operator %s for %T not implemented", expr.Src.Op, n)
	}
}

// CmpOp compares x to y.
// Returns a nil element if the operator is not a comparison operator.
func (n *elInt) CmpOp(env ir.SourceFile, expr *ir.BinaryExpr, y engine.ScalarNumber) engine.BoolConstant {
	return compare(expr.Src.Op, n.Float(), y.Float())
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (n *elInt) BinaryOp(env ir.SourceFile, expr *ir.BinaryExpr, yEl engine.ScalarNumber) (engine.ScalarNumber, error) {
	var y *big.Int
	switch yT := yEl.(type) {
	case *elFloat:
		return binaryFloat(env, expr, n.Float(), yT.val)
	case *elInt:
		y = yT.val
	default:
		return notSupported(env, expr, n, yEl)
	}
	x := n.val
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
		return notSupported(env, expr, n, yEl)
	}
	return newInt(val), nil
}

// Float returns the integer value as a big float.
func (n *elInt) Float() *big.Float {
	return (&big.Float{}).SetInt(n.val)
}

// Cast an element into a given data type.
func (n *elInt) Cast(expr *ir.NumberCastExpr) engine.Constant {
	return constants.NewScalar(expr.Type(), n)
}

// Expr returns the expression representing the integer.
func (n *elInt) BuildIR() ir.Expr {
	return &ir.NumberInt{
		Src: &ast.BasicLit{
			Kind:  token.INT,
			Value: n.val.String(),
		},
		Val: n.val,
	}
}

// Expr returns the expression representing the integer.
func (n *elInt) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{n.BuildIR()}, nil
}

// Type of the element.
func (n *elInt) Type() ir.Type {
	return ir.NumberIntType()
}

// ShortString returns a short string representation for the integer.
func (n *elInt) ShortString(from *ir.File) string {
	return n.val.String()
}

// SourceString returns the GX source code to represent the float.
func (n *elInt) SourceString(from *ir.File) string {
	return n.ShortString(from)
}

func (n *elInt) String() string {
	return n.val.String()
}
