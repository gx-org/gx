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

package genast

import (
	"go/ast"
	"go/token"
)

// Block of statements.
type Block struct {
	stmts []ast.Stmt
}

// NewBlock from an existing list of statements.
func NewBlock(stmts ...ast.Stmt) *Block {
	return &Block{stmts: stmts}
}

// Add a statement to the block.
func (b *Block) Add(s ast.Stmt) *Block {
	b.stmts = append(b.stmts, s)
	return b
}

// Block of all the accumulated statements.
func (b *Block) Block() *ast.BlockStmt {
	return &ast.BlockStmt{List: b.stmts}
}

// Return statement.
func (b *Block) Return(exprs ...ast.Expr) *Block {
	return b.Add(&ast.ReturnStmt{Results: exprs})
}

// Assign an expression to a variable.
func (b *Block) Assign(name string, val ast.Expr) IdentExpr {
	id := Ident(name)
	b.Add(&ast.AssignStmt{
		Lhs: []ast.Expr{id.X},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{val},
	})
	return id
}

// UnpackAssign assigns a function call to multiple variables.
func (b *Block) UnpackAssign(names []IdentExpr, call *ast.CallExpr) *Block {
	return b.Add(&ast.AssignStmt{
		Lhs: AstExprs(names),
		Tok: token.DEFINE,
		Rhs: []ast.Expr{call},
	})
}
