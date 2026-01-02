// Copyright 2025 Google LLC
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

// Package special represents values in a gradient expression that can be simplified.
package special

import (
	"go/ast"
	"go/token"
	"math/big"
	"strconv"

	"github.com/gx-org/gx/build/ir"
)

// Value being represented.
type Value int

const (
	// anyV is a value that can be anything.
	anyV = iota
	// zeroV is a value equals to 0.
	zeroV
	// oneV is a value equals to 1.
	oneV
)

// Expr is an expression with the type of value it represents.
type Expr struct {
	value Value
	expr  ast.Expr
}

// New returns a new expression from its AST.
func New(expr ast.Expr) *Expr {
	return &Expr{expr: expr}
}

var (
	bigZeroInt   = big.NewInt(0)
	bigOneInt    = big.NewInt(1)
	bigZeroFloat = big.NewFloat(0)
	bigOneFloat  = big.NewFloat(1)
)

// NewFromIR returns an expression for any value given an IR source node.
func NewFromIR(node ir.SourceNode) *Expr {
	switch nodeT := node.(type) {
	case *ir.NumberInt:
		if nodeT.Val.Cmp(bigZeroInt) == 0 {
			return zero
		}
		if nodeT.Val.Cmp(bigOneInt) == 0 {
			return one
		}
	case *ir.NumberFloat:
		if nodeT.Val.Cmp(bigZeroFloat) == 0 {
			return zero
		}
		if nodeT.Val.Cmp(bigOneFloat) == 0 {
			return one
		}
	}
	return New(node.Source().(ast.Expr))
}

// AST of the expression.
func (r *Expr) AST() ast.Expr {
	return r.expr
}

// Index returns an index expression given a index number.
func (r *Expr) Index(i int) *Expr {
	return New(&ast.IndexExpr{
		X: r.AST(),
		Index: &ast.BasicLit{
			Kind:  token.INT,
			Value: strconv.Itoa(i),
		},
	})
}

// IsAny returns true if the value can be anything.
func (r *Expr) IsAny() bool {
	return r.value == anyV
}

// IsOne returns true if the value is one.
func (r *Expr) IsOne() bool {
	return r.value == oneV
}

// IsZero returns true if the value is zero.
func (r *Expr) IsZero() bool {
	return r.value == zeroV
}

func addCast(expr ast.Expr, typ ir.Type) ast.Expr {
	return &ast.CallExpr{
		Fun:  typ.Source().(ast.Expr),
		Args: []ast.Expr{expr},
	}
}

// CastIfRequired casts an expression if required.
func CastIfRequired(expr ast.Expr, typ ir.Type) ast.Expr {
	basic, isBasic := expr.(*ast.BasicLit)
	if !isBasic {
		return expr
	}
	if basic.Kind != token.INT && basic.Kind != token.FLOAT {
		return expr
	}
	return addCast(expr, typ)
}

// CastIfRequired returns a new expression with a cast when the expression is a basic literal.
func (r *Expr) CastIfRequired(typ ir.Type) *Expr {
	return &Expr{
		value: r.value,
		expr:  CastIfRequired(r.expr, typ),
	}
}

// RemoveParen removes the top parenthesis if required.
func (r *Expr) RemoveParen() *Expr {
	parent, ok := r.expr.(*ast.ParenExpr)
	if !ok {
		return r
	}
	return (&Expr{expr: parent.X}).RemoveParen()
}

// Print the expression (only used for debugging).
func (r *Expr) Print() {
	ast.Print(token.NewFileSet(), r.expr)
}

// Add builds an add expression, simplifying if possible.
func Add(xs ...*Expr) *Expr {
	if len(xs) == 1 {
		return xs[0]
	}
	left := xs[0]
	right := Add(xs[1:]...)
	if left.value == zeroV {
		return right
	}
	if right.value == zeroV {
		return left
	}
	return &Expr{
		expr: &ast.BinaryExpr{
			Op: token.ADD,
			X:  left.expr,
			Y:  right.expr,
		},
	}
}

// Mul builds a multiplication expression, simplifying if possible.
func Mul(x, y *Expr) *Expr {
	// Multiplication by 0.
	if x.value == zeroV {
		return x
	}
	if y.value == zeroV {
		return y
	}
	// Multiplication by 1.
	if x.value == oneV {
		return y
	}
	if y.value == oneV {
		return x
	}
	// All other cases.
	return &Expr{
		expr: &ast.BinaryExpr{
			Op: token.MUL,
			X:  x.expr,
			Y:  y.expr,
		},
	}
}

// UnarySub builds an unary subtraction expression, simplifying if possible.
func UnarySub(x *Expr) *Expr {
	if x.value == zeroV {
		return x
	}
	return &Expr{
		expr: &ast.UnaryExpr{
			Op: token.SUB,
			X:  x.expr,
		},
	}
}

// Sub builds a subtraction expression, simplifying if possible.
func Sub(x, y *Expr) *Expr {
	// Subtraction by 0.
	if y.value == zeroV {
		return x
	}
	// Subtraction from 0.
	if x.value == zeroV {
		return &Expr{
			expr: &ast.UnaryExpr{
				Op: token.SUB,
				X:  y.expr,
			},
		}
	}
	return &Expr{
		expr: &ast.BinaryExpr{
			Op: token.SUB,
			X:  x.expr,
			Y:  y.expr,
		},
	}
}

// Quo builds a quotient expression, simplifying if possible.
func Quo(x, y *Expr) *Expr {
	if y.value == oneV {
		return x
	}
	return &Expr{
		expr: &ast.BinaryExpr{
			Op: token.QUO,
			X:  x.expr,
			Y:  y.expr,
		},
	}
}

var zero = &Expr{
	value: zeroV,
	expr:  &ast.BasicLit{Value: "0", Kind: token.INT},
}

// ZeroExpr returns an expression representing zero.
func ZeroExpr() *Expr {
	return zero
}

var one = &Expr{
	value: oneV,
	expr:  &ast.BasicLit{Value: "1", Kind: token.INT},
}

// OneExpr returns an expression representing one.
func OneExpr() *Expr {
	return one
}

// Paren returns an expression inside parenthesis if necessary.
func Paren(x *Expr) *Expr {
	if x.value != anyV {
		return x
	}
	switch x.expr.(type) {
	case *ast.Ident:
		return x
	case *ast.ParenExpr:
		return x
	}
	return &Expr{
		expr: &ast.ParenExpr{X: x.expr},
	}
}
