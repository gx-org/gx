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
	"go/token"
	"reflect"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/src/gx/ast"
	"github.com/gx-org/gx/stdlib/math/grad/gradann"
	"github.com/gx-org/gx/stdlib/math/grad/special"
	"github.com/gx-org/gx/stdlib/math/grad/wrt"
)

type annFunc struct {
	node[ir.PkgFunc]
	ann *gradann.Annotation
}

func (p *processor) processFuncWithAnn(ann *gradann.Annotation) (stmt, bool) {
	fn, ok := p.fn.(ir.PkgFunc)
	if !ok {
		return nil, p.fetcher.Err().Appendf(p.fn.Node(), "cannot cast %T to %s", p.fn, reflect.TypeFor[ir.PkgFunc]().Name())
	}
	return &annFunc{
		node: newNode(p, fn),
		ann:  ann,
	}, true
}

type annFuncReturnStmt struct {
	*annFunc
	grad ir.PkgFunc
	args []ast.Expr
}

func (n *annFunc) buildVJPFunction(outCore *astOutCore, gradParam wrt.WRT, grad ir.PkgFunc, args []ast.Expr) (*ast.FuncLit, bool) {
	// For each result of the function, builds a VJP for all parameters.
	// Return the forward results, and all the VJPs.
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, len(n.graph.nResults.names))}
	for i, res := range n.graph.nResults.names {
		ret.Results[i] = special.Mul(
			special.New(&ast.Ident{Name: res}),
			special.New(&ast.CallExpr{
				Fun:  n.buildForwardCallee(&ast.Ident{Name: grad.Name()}),
				Args: args,
			}),
		).AST()
	}
	return &ast.FuncLit{
		Type: gradParam.FuncType(),
		Body: &ast.BlockStmt{List: []ast.Stmt{ret}},
	}, true
}

func (n *annFunc) build(astmts *astOut) bool {
	ann := gradann.Get(astmts.graph.Func())
	if vjp := ann.VJP(); vjp != nil {
		return n.forwardCall(astmts, vjp)
	}
	return n.buildBody(astmts)
}

func (n *annFunc) forwardCall(astmts *astOut, vjp ir.PkgFunc) bool {
	ftype := n.graph.fn.FuncType()
	params := ftype.Params.Fields()
	args := make([]ast.Expr, len(params))
	for i, field := range params {
		args[i] = field.Storage().NameDef()
	}
	var callee ast.Expr = vjp.NameDef()
	genFields := ftype.GenParams.Fields.Fields()
	for _, field := range genFields {
		callee = &ast.IndexExpr{
			X:     callee,
			Index: &ast.Ident{Name: field.Name.Name},
		}
	}
	astmts.append(&ast.ReturnStmt{Results: []ast.Expr{
		&ast.CallExpr{
			Fun:  callee,
			Args: args,
		},
	}})
	return true
}

func (n *annFunc) buildForwardCallee(callee ast.Expr) ast.Expr {
	for _, genParam := range n.irnode.FuncType().GenParams.Fields.Fields() {
		callee = &ast.IndexExpr{
			X:     callee,
			Index: genParam.Name,
		}
	}
	return callee
}

func (n *annFunc) buildBody(astmts *astOut) bool {
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
			Fun:  n.buildForwardCallee(n.irnode.NameDef()),
			Args: args,
		}},
	})
	// Generate forward statements for the expressions in the statement.
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, nVals)}
	for i, expr := range idents {
		ret.Results[i] = expr
	}

	// Build a backward function for each function parameter.
	names := make([]ast.Expr, len(n.graph.wrts))
	for i, gradParam := range n.graph.wrts.Arrays() {
		vjpFuncLit, ok := n.buildVJPFunction(astmts.astOutCore, gradParam, n.ann.Partial(i), args)
		if !ok {
			return false
		}
		root := "selfVJPFunc"
		if len(names) > 1 {
			root += "WRT" + wrt.ToName(gradParam.Name())
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
