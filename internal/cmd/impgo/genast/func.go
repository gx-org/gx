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

import "go/ast"

// FuncDecl declares a function.
func (f *File) FuncDecl(name string, tp *ast.FuncType, body *Block) *ast.FuncDecl {
	if tp == nil {
		tp = &ast.FuncType{
			Params:  &ast.FieldList{},
			Results: &ast.FieldList{},
		}
	}
	var bodyBlock *ast.BlockStmt
	if body != nil {
		bodyBlock = body.Block()
	}
	decl := &ast.FuncDecl{
		Name: Ident(name).X,
		Type: tp,
		Body: bodyBlock,
	}
	f.Src.Decls = append(f.Src.Decls, decl)
	return decl
}

// CallExpr returns a call expression.
func CallExpr(callee ast.Expr, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  callee,
		Args: args,
	}
}

// CallStmt returns a statement calling a function.
func CallStmt(callee ast.Expr, args ...ast.Expr) ast.Stmt {
	return &ast.ExprStmt{X: CallExpr(callee, args...)}
}
