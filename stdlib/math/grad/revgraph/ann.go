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

package revgraph

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/math/grad/setann"
	"github.com/gx-org/gx/stdlib/math/grad/special"
)

type annFunc struct {
	node[ir.PkgFunc]
	ann *setann.Annotation
}

func (p *processor) processFuncWithAnn(ann *setann.Annotation) (stmt, bool) {
	fn, ok := p.fn.(ir.PkgFunc)
	if !ok {
		return nil, p.fetcher.Err().Appendf(p.fn.Node(), "cannot cast %T to %s", p.fn, reflect.TypeFor[ir.PkgFunc]().Name())
	}
	return &annFunc{
		node: newNode(p, fn),
		ann:  ann,
	}, true
}

func (n *annFunc) buildVJPFunctionWRTFromAnn(astmts *astOut, grad ir.PkgFunc, param VJPParam, args []ast.Expr) (*ast.FuncLit, bool) {
	backwarder := astmts.newASTOutWRT(param.wrt)
	// For each result of the function, builds a VJP for all parameters.
	// Return the forward results, and all the VJPs.
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, len(n.graph.nResults.names))}
	for i, res := range n.graph.nResults.names {
		ret.Results[i] = special.Mul(
			special.New(&ast.Ident{Name: res}),
			special.New(&ast.CallExpr{
				Fun: n.graph.funcNameWithTypeParamsExpr(
					&ast.Ident{Name: grad.Name()},
				),
				Args: args,
			}),
		).AST()
	}
	var body []ast.Stmt
	body = append(body, backwarder.stmts...)
	body = append(body, ret)
	return &ast.FuncLit{
		Type: param.vjpFType,
		Body: &ast.BlockStmt{List: body},
	}, true
}

func (n *annFunc) build(astmts *astOut) bool {
	// Build the arguments to call the forward functions.
	args := make([]ast.Expr, len(n.graph.nParams.names))
	for i, fieldName := range n.graph.nParams.names {
		args[i] = &ast.Ident{Name: fieldName}
	}

	// Call the original function to get the forward values.
	fType := n.graph.fn.FuncType()
	nVals := fType.Results.Len()
	idents := astmts.buildIdents(n.id, nVals, "")
	astmts.append(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: toExprs(idents),
		Rhs: []ast.Expr{&ast.CallExpr{
			Fun: n.graph.funcNameWithTypeParamsExpr(
				&ast.Ident{Name: n.irnode.Name()},
			),
			Args: args,
		}},
	})

	// Generate forward statements for the expressions in the statement.
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, nVals)}
	for i, expr := range idents {
		ret.Results[i] = expr
	}

	// Build a backward function for each function parameter.
	names := make([]ast.Expr, len(n.graph.params))
	for i, param := range n.graph.params {
		vjpFuncLit, ok := n.buildVJPFunctionWRTFromAnn(astmts, n.ann.Partials[i], param, args)
		if !ok {
			return false
		}
		root := "selfVJPFunc"
		if len(names) > 1 {
			root += "WRT" + param.wrt.name()
		}
		vjpFuncName := n.graph.unames.Name(root)
		astmts.append(&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: []ast.Expr{&ast.Ident{Name: vjpFuncName}},
			Rhs: []ast.Expr{vjpFuncLit},
		})
		ret.Results = append(ret.Results, &ast.Ident{Name: vjpFuncName})
	}
	astmts.append(ret)
	return true
}
