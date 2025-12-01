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

	"github.com/gx-org/gx/build/ir"
)

// Value being represented.
type Value int

const (
	// Any is a value that can be anything.
	Any = iota
	// Zero is a value equals to 0.
	Zero
	// One is a value equals to 1.
	One
)

// Expr is an expression with the type of value it represents.
type Expr struct {
	Value Value
	Expr  ast.Expr
}

// New returns an expression for any value given an IR source node.
func New(node ir.SourceNode) *Expr {
	return &Expr{Expr: node.Source().(ast.Expr)}
}

func addCast(expr ast.Expr, typ ir.Type) ast.Expr {
	return &ast.CallExpr{
		Fun:  typ.Source().(ast.Expr),
		Args: []ast.Expr{expr},
	}
}

func addCastIfRequired(expr ast.Expr, typ ir.Type) ast.Expr {
	basic, isBasic := expr.(*ast.BasicLit)
	if !isBasic {
		return expr
	}
	if basic.Kind != token.INT && basic.Kind != token.FLOAT {
		return expr
	}
	return addCast(expr, typ)
}

// AddCastIfRequired returns a new expression with a cast when the expression is a basic literal.
func (r *Expr) AddCastIfRequired(typ ir.Type) *Expr {
	return &Expr{
		Value: r.Value,
		Expr:  addCastIfRequired(r.Expr, typ),
	}
}

// Print the expression (only used for debugging).
func (r *Expr) Print() {
	ast.Print(token.NewFileSet(), r.Expr)
}

// Add builds an add expression, simplifying if possible.
func Add(xs ...*Expr) *Expr {
	if len(xs) == 1 {
		return xs[0]
	}
	left := xs[0]
	right := Add(xs[1:]...)
	if left.Value == Zero {
		return right
	}
	if right.Value == Zero {
		return left
	}
	return &Expr{
		Expr: &ast.BinaryExpr{
			Op: token.ADD,
			X:  left.Expr,
			Y:  right.Expr,
		},
	}
}

// Mul builds a multiplication expression, simplifying if possible.
func Mul(x, y *Expr) *Expr {
	// Multiplication by 0.
	if x.Value == Zero {
		return x
	}
	if y.Value == Zero {
		return y
	}
	// Multiplication by 1.
	if x.Value == One {
		return y
	}
	if y.Value == One {
		return x
	}
	// All other cases.
	return &Expr{
		Expr: &ast.BinaryExpr{
			Op: token.MUL,
			X:  x.Expr,
			Y:  y.Expr,
		},
	}
}

// UnarySub builds an unary subtraction expression, simplifying if possible.
func UnarySub(x *Expr) *Expr {
	if x.Value == Zero {
		return x
	}
	return &Expr{
		Expr: &ast.UnaryExpr{
			Op: token.SUB,
			X:  x.Expr,
		},
	}
}

// Sub builds a subtraction expression, simplifying if possible.
func Sub(x, y *Expr) *Expr {
	// Subtraction by 0.
	if y.Value == Zero {
		return x
	}
	// Subtraction from 0.
	if x.Value == Zero {
		return &Expr{
			Expr: &ast.UnaryExpr{
				Op: token.SUB,
				X:  y.Expr,
			},
		}
	}
	return &Expr{
		Expr: &ast.BinaryExpr{
			Op: token.SUB,
			X:  x.Expr,
			Y:  y.Expr,
		},
	}
}

// Quo builds a quotient expression, simplifying if possible.
func Quo(x, y *Expr) *Expr {
	if y.Value == One {
		return x
	}
	return &Expr{
		Expr: &ast.BinaryExpr{
			Op: token.QUO,
			X:  x.Expr,
			Y:  y.Expr,
		},
	}
}

var zero = &Expr{
	Value: Zero,
	Expr:  &ast.BasicLit{Value: "0", Kind: token.INT},
}

// ZeroExpr returns an expression representing zero.
func ZeroExpr() *Expr {
	return zero
}

var one = &Expr{
	Value: One,
	Expr:  &ast.BasicLit{Value: "1", Kind: token.INT},
}

// OneExpr returns an expression representing one.
func OneExpr(node ast.Node) *Expr {
	return one
}
