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

package engine

import (
	"math/big"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

// ScalarNumber is an element representing a number without a type yet.
type ScalarNumber interface {
	ir.Element
	ir.WithExpr

	// Float value of the number.
	Float() *big.Float

	// UnaryOp applies a unary operator on x.
	UnaryOp(env ir.SourceFile, expr *ir.UnaryExpr) (ScalarNumber, error)

	// BinaryOp applies a binary operator to x and y.
	BinaryOp(env ir.SourceFile, expr *ir.BinaryExpr, y ScalarNumber) (ScalarNumber, error)

	// CmpOp compares x to y.
	// Returns a nil element if the expression is not a comparison expression.
	CmpOp(env ir.SourceFile, expr *ir.BinaryExpr, y ScalarNumber) BoolConstant

	// Cast an element into a given data type.
	Cast(expr *ir.NumberCastExpr) Constant

	// BuildIR returns the IR expression of the element.
	BuildIR() ir.Expr

	// String representation of the number.
	String() string
}

// Constant is an element representing a value known at compile time and
// stored on the host by the backend.
type Constant interface {
	ir.Element
	ir.WithExpr
	ir.StringSourcer
	ir.StringShorter

	// BuildNode the constant into a given graph.
	BuildNode(ops.Graph, irkind.Kind) (ops.Node, error)

	// AxisLengths returns the lengths of the axes of the constant.
	AxisLengths() ([]int, error)
}

// AtomConstant is an atomic constant.
type AtomConstant interface {
	Constant
	// Atom returns the Go value of the scalar.
	Atom(irkind.Kind) (any, error)
}

// BoolConstant is a boolean constant.
type BoolConstant interface {
	AtomConstant
	Bool() bool
}

// ScalarConstant is a number constant
type ScalarConstant interface {
	AtomConstant
	Number() ScalarNumber
}
