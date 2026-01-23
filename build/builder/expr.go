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

func processExpr(pscope procScope, expr ast.Expr) (exprNode, bool) {
	switch exprT := expr.(type) {
	case *ast.Ident:
		return pscope.processIdentExpr(exprT)
	case *ast.BasicLit:
		return processBasicLit(pscope, exprT)
	case *ast.ArrayType:
		typScope := defaultTypeProcScope(pscope)
		return processArrayType(typScope, exprT)
	case *ast.CompositeLit:
		return processCompositeLit(pscope, exprT)
	case *ast.CallExpr:
		return processCallExpr(pscope, exprT)
	case *ast.ParenExpr:
		return processParenExpr(pscope, exprT)
	case *ast.UnaryExpr:
		return processUnaryExpr(pscope, exprT)
	case *ast.BinaryExpr:
		return processBinaryExpr(pscope, exprT)
	case *ast.SelectorExpr:
		return processSelectorExpr(pscope, exprT)
	case *ast.IndexExpr:
		return processIndexExpr(pscope, exprT)
	case *ast.IndexListExpr:
		return processIndexListExpr(pscope, exprT)
	case *ast.FuncLit:
		return processFuncLit(pscope, exprT)
	case *ast.TypeAssertExpr:
		return processTypeAssertExpr(pscope, exprT)
	case *ast.StarExpr:
		return nil, pscope.Err().Appendf(exprT, "star expression *<expr> not supported")
	}
	return nil, pscope.Err().Appendf(expr, "expression of type %T not supported", expr)
}

// processTypeExpr processes an expr in the context of defining a type.
func processTypeExpr(pscope typeProcScope, expr ast.Node) (typeExprNode, bool) {
	switch exprT := expr.(type) {
	case *ast.Ident:
		return processIdent(pscope, exprT)
	case *ast.ArrayType:
		return processArraySliceType(pscope, exprT)
	case *ast.StructType:
		return processStructType(pscope, exprT)
	case *ast.SelectorExpr:
		return processSelectorExpr(pscope, exprT)
	case *ast.InterfaceType:
		return processInterfaceType(pscope, exprT)
	case *ast.FuncType:
		return processFuncType(pscope, exprT, nil, false)
	case *ast.Ellipsis:
		return processVarArgsType(pscope, exprT)
	default:
		pscope.Err().Appendf(expr, "type expression %T not supported", expr)
	}
	return nil, false
}
