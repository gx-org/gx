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

// Package elements provides generic elements, independent of the evaluator, for the interpreter.
package elements

import (
	"go/token"

	"github.com/gx-org/gx/build/ir"
)

// Element in the state.
type Element interface {
	// Flatten the element, that is:
	// returns itself if the element is atomic,
	// returns its components if the element is a composite.
	Flatten() ([]Element, error)
}

type (
	// NodeFile is an expression with the file in which it is declared.
	NodeFile[T ir.Node] struct {
		file *ir.File
		node T
	}

	// NodeAt is a generic GX node.
	NodeAt = NodeFile[ir.Node]

	// ExprAt is a generic GX expression.
	ExprAt = NodeFile[ir.Expr]

	// CallAt is a function call GX expression.
	CallAt = NodeFile[*ir.CallExpr]

	// FieldAt is a typed field at a given position.
	FieldAt = NodeFile[*ir.Field]
)

// NewNodeAt returns a new expression at a given position.
func NewNodeAt[T ir.Node](file *ir.File, expr T) NodeFile[T] {
	return NodeFile[T]{file: file, node: expr}
}

// NewExprAt returns a new expression at a given position.
func NewExprAt[T ir.Expr](file *ir.File, expr T) ExprAt {
	return ExprAt{file: file, node: expr}
}

// FSet returns the fileset of the expression.
func (ea NodeFile[T]) FSet() *token.FileSet {
	return ea.file.Package.FSet
}

// Node returns the expression.
func (ea NodeFile[T]) Node() T {
	return ea.node
}

// NodeFile returns a general node.
func (ea NodeFile[T]) NodeFile() NodeFile[ir.Node] {
	return NodeFile[ir.Node]{file: ea.file, node: ea.node}
}

// ToExprAt converts a type position into a generic node position.
func (ea NodeFile[T]) ToExprAt() ExprAt {
	node := any(ea.node)
	return NewNodeAt[ir.Expr](ea.file, node.(ir.Expr))
}

// File returns the file in which the expression is declared.
func (ea NodeFile[T]) File() *ir.File {
	return ea.file
}
