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

	"github.com/gx-org/gx/build/ir"
)

type returnStmt struct {
	ext     ir.ReturnStmt
	fn      *funcDecl
	results []exprNode
}

func processReturnStmt(owner owner, fn *funcDecl, ret *ast.ReturnStmt) (*returnStmt, bool) {
	n := &returnStmt{
		ext: ir.ReturnStmt{
			Src: ret,
		},
		fn:      fn,
		results: make([]exprNode, len(ret.Results)),
	}
	ok := true
	for i, stmt := range ret.Results {
		expr, okE := processExpr(owner, stmt)
		n.results[i] = expr
		ok = ok && okE
	}
	return n, ok
}

// source is the position of the return statement in the code.
func (n *returnStmt) source() ast.Node {
	return n.ext.Source()
}

func (n *returnStmt) buildStmt() ir.Stmt {
	n.ext.Results = make([]ir.Expr, len(n.results))
	for i, expr := range n.results {
		n.ext.Results[i] = expr.buildExpr()
	}
	return &n.ext
}

func (n *returnStmt) resolveType(scope *scopeBlock) bool {
	if len(n.results) == 0 {
		// Naked return: nothing to check.
		return true
	}
	// Resolve all the expressions composing the return.
	ok := true
	resultTypes := make([]typeNode, len(n.results))
	for i, expr := range n.results {
		var exprOk bool
		resultTypes[i], exprOk = expr.resolveType(scope)
		ok = exprOk && ok
	}
	var isTuple = false
	if len(resultTypes) == 1 {
		// Expand the results if a function call is returned.
		var tuple *tupleType
		tuple, isTuple = resultTypes[0].(*tupleType)
		if isTuple {
			resultTypes = tuple.types()
		}
	}
	want := n.fn.funcType.resultTypes()
	// Compare the number of values being returned to the function results.
	if len(resultTypes) > want.len() {
		scope.err().Appendf(n.source(), "too many return values")
		ok = false
	}
	if len(resultTypes) < want.len() {
		scope.err().Appendf(n.source(), "not enough return values")
		ok = false
	}
	if !ok {
		return false
	}
	resultPos := n.results[0]
	// Check if the types being returned are assignable to the types declared by the signature.
	for i, wantI := range want.types() {
		gotI := resultTypes[i]
		if ir.IsNumber(gotI.kind()) {
			var retOk bool
			n.results[i], gotI, retOk = castNumber(scope, n.results[i], wantI.irType())
			if !retOk {
				ok = false
				continue
			}
		}
		if gotI.kind() == ir.InvalidKind || wantI.kind() == ir.InvalidKind {
			ok = false
			continue
		}
		posI := resultPos
		if !isTuple {
			posI = n.results[i]
		}
		var okI bool
		var err error
		want.fields[i].group.typ, okI, err = assignableTo(scope, posI, gotI, wantI)
		if err != nil {
			scope.err().AppendAt(n.results[i].source(), err)
			ok = false
			continue
		}
		if !okI {
			ok = false
			scope.err().Appendf(posI.source(), "cannot use %s as %s in return statement", gotI.String(), wantI.String())
			continue
		}
	}
	return ok
}
