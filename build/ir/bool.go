// Copyright 2026 Google LLC
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

package ir

import "go/ast"

// BoolValue represents a bool value literal.
type BoolValue struct {
	Src *ast.Ident
	Val bool
}

var _ Expr = (*BoolValue)(nil)

func (*BoolValue) node() {}

// Node returns the AST node.
func (b *BoolValue) Node() ast.Node {
	return b.Src
}

// Expr returns the AST expression.
func (b *BoolValue) Expr() ast.Expr {
	return b.Src
}

// Type of the expression.
func (b *BoolValue) Type() Type {
	return BoolType()
}

// SourceString returns the GX source code.
func (b *BoolValue) SourceString(from *File) string {
	return b.Src.Name
}

var falseValue = BoolValue{
	Src: &ast.Ident{Name: "false"},
	Val: false,
}

// False returns an atomic value equal to false.
func False() *BoolValue {
	return &falseValue
}

var trueValue = BoolValue{
	Src: &ast.Ident{Name: "true"},
	Val: true,
}

// True returns an atomic value equal to true.
func True() *BoolValue {
	return &trueValue
}
