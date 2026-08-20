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

// elFloat is a GX number.
type elFloat struct {
	fmtexpr.AtomStringImpl
	val *big.Float
}

func newFloat(f *big.Float) *elFloat {
	return &elFloat{val: f}
}

// NewFloatNumber returns a new element Float number element.
func NewFloatNumber(x *ir.NumberFloat) engine.ScalarNumber {
	return newFloat(x.Val)
}

// NewFloatIR returns the IR for a float.
func NewFloatIR(val float64, typ ir.Type) (*ir.NumberFloat, *ir.NumberCastExpr) {
	elF := newFloat(big.NewFloat(val))
	irF := elF.buildIR()
	return irF, &ir.NumberCastExpr{X: irF, Typ: typ}
}

func newFloatFrom(val float64, typ ir.Type) engine.Constant {
	nb, x := NewFloatIR(val, typ)
	return NewFloatNumber(nb).Cast(x)
}

// CmpOp compares x to y.
// Returns a nil element if the operator is not a comparison operator.
func (n *elFloat) CmpOp(env ir.SourceFile, expr *ir.BinaryExpr, y engine.ScalarNumber) engine.BoolConstant {
	return compare(expr.Src.Op, n.Float(), y.Float())
}

// UnaryOp applies a unary operator on x.
func (n *elFloat) UnaryOp(env ir.SourceFile, expr *ir.UnaryExpr) (engine.ScalarNumber, error) {
	switch expr.Src.Op {
	case token.ADD:
		return n, nil
	case token.SUB:
		return newFloat(new(big.Float).Neg(n.val)), nil
	default:
		return nil, fmterr.InternalAt(env.File().FileSet(), expr.Src, "unary operator %s for %T not implemented", expr.Src.Op, n)
	}
}

func notSupported(env ir.SourceFile, expr *ir.BinaryExpr, x, y engine.ScalarNumber) (engine.ScalarNumber, error) {
	return nil, fmterr.InternalAt(env.File().FileSet(), expr.Src, "%T%s%T not supported", x, expr.Src.Op, y)
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (n *elFloat) BinaryOp(env ir.SourceFile, expr *ir.BinaryExpr, y engine.ScalarNumber) (engine.ScalarNumber, error) {
	return binaryFloat(env, expr, n.val, y.Float())
}

func binaryFloat(env ir.SourceFile, expr *ir.BinaryExpr, x *big.Float, y *big.Float) (engine.ScalarNumber, error) {
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
		return nil, fmterr.Errorf(env.File().FileSet(), expr.Src, "number float binary operator %s not implemented", expr.Src.Op)
	}
	return newFloat(val), nil
}

// Cast an element into a given data type.
func (n *elFloat) Cast(expr *ir.NumberCastExpr) engine.Constant {
	return constants.NewScalar(expr.Type(), n)
}

// Type of the element.
func (n *elFloat) Type() ir.Type {
	return ir.NumberFloatType()
}

// Float value of the number.
func (n *elFloat) Float() *big.Float {
	return n.val
}

// BuildIR returns the IR expression for the float number.
func (n *elFloat) BuildIR() ir.Expr {
	return n.buildIR()
}

func (n *elFloat) buildIR() *ir.NumberFloat {
	return &ir.NumberFloat{
		Src: &ast.BasicLit{
			Kind:  token.FLOAT,
			Value: n.val.String(),
		},
		Val: n.val,
	}
}

// Expr returns the expression representing the integer.
func (n *elFloat) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{n.BuildIR()}, nil
}

// ShortString returns a short string representation of the value.
func (n *elFloat) ShortString(from *ir.File) string {
	return n.Float().String()
}

// SourceString returns the GX source code to represent the float.
func (n *elFloat) SourceString(from *ir.File) string {
	return n.ShortString(from)
}

func (n *elFloat) String() string {
	return n.val.String()
}
