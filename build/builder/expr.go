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

package builder

import (
	"go/ast"
)

func processExpr(owner owner, expr ast.Expr) (exprNode, bool) {
	switch exprT := expr.(type) {
	case *ast.Ident:
		return processIdentExpr(exprT), true
	case *ast.BasicLit:
		return processBasicLit(owner, exprT)
	case *ast.ArrayType:
		return processCast(owner, exprT)
	case *ast.CompositeLit:
		return processCompositeLit(owner, exprT)
	case *ast.CallExpr:
		return processCallExpr(owner, exprT)
	case *ast.ParenExpr:
		return processParenExpr(owner, exprT)
	case *ast.UnaryExpr:
		return processUnaryExpr(owner, exprT)
	case *ast.BinaryExpr:
		return processBinaryExpr(owner, exprT)
	case *ast.SelectorExpr:
		return processSelectorReference(owner, exprT)
	case *ast.IndexExpr:
		return processIndexExpr(owner, exprT)
	case *ast.FuncLit:
		return processFuncLit(owner, exprT)
	default:
		owner.err().Appendf(expr, "expression of type %T not supported", expr)
	}
	return nil, false
}

// processTypeExpr processes an expr in the context of defining a type.
func processTypeExpr(owner owner, expr ast.Node) (typeNode, bool) {
	switch exprT := expr.(type) {
	case *ast.Ident:
		return processIdentTypeExpr(owner, exprT)
	case *ast.ArrayType:
		return processArraySliceType(owner, exprT)
	case *ast.StructType:
		return processStructType(owner, exprT)
	case *ast.SelectorExpr:
		return processTypeSelectorReference(owner, exprT)
	default:
		owner.err().Appendf(expr, "expression of type %T not supported", expr)
	}
	return nil, false
}

func processCompositeLit(owner owner, expr *ast.CompositeLit) (exprNode, bool) {
	if expr.Type == nil {
		owner.err().Appendf(expr, "untyped composite literal")
		return nil, false
	}
	switch exprT := expr.Type.(type) {
	case *ast.ArrayType:
		return processArraySliceTypeWithValue(owner, expr, exprT)
	case *ast.CompositeLit:
		return processCompositeLit(owner, exprT)
	}
	return processCompositeLitStruct(owner, expr, expr.Type)
}
