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
	"github.com/gx-org/gx/build/ir/irkind"
)

type returnStmt struct {
	src     *ast.ReturnStmt
	results []exprNode
}

var _ stmtNode = (*returnStmt)(nil)

func processReturnStmt(pscope procScope, src *ast.ReturnStmt) (*returnStmt, bool) {
	n := &returnStmt{
		src:     src,
		results: make([]exprNode, len(src.Results)),
	}
	ok := true
	for i, stmt := range src.Results {
		expr, exprOk := processExpr(pscope, stmt)
		n.results[i] = expr
		ok = ok && exprOk
	}
	return n, ok
}

// source is the position of the return statement in the code.
func (n *returnStmt) source() ast.Node {
	return n.src
}

func hasNamedResult(fields []*ir.Field) bool {
	for _, field := range fields {
		if field.Name == nil {
			return false
		}
	}
	return true
}

func allTypes[T ir.Value](list []T) []ir.Type {
	types := make([]ir.Type, len(list))
	for i, val := range list {
		types[i] = val.Type()
	}
	return types
}

func resultTypes(exprs []ir.Expr) ([]ir.Type, bool) {
	if len(exprs) == 1 {
		if callExpr, ok := exprs[0].(*ir.FuncCallExpr); ok {
			return allTypes(callExpr.Callee.FuncType().Results.Fields()), true
		}
	}
	return allTypes(exprs), false
}

func returnAs(rscope resolveScope, pos ast.Node, src, dst ir.Type) bool {
	assignable, err := assignableTo(rscope, src, dst)
	if err != nil {
		return rscope.Err().AppendAt(pos, err)
	}
	if !assignable {
		return rscope.Err().Appendf(pos, "cannot use %s as %s value in return statement", src.String(), dst.String())
	}
	return true
}

func (n *returnStmt) castNumber(scope fnResolveScope, expr ir.Expr, want ir.Type) (ir.Expr, bool) {
	ret, ok := castNumber(scope, expr, want)
	if !ok {
		return expr, false
	}
	arrayWant := ir.Underlying(want).(ir.ArrayType)
	if arrayWant.Rank().IsAtomic() {
		return ret, true
	}
	return &ir.CastExpr{
		Src: &ast.CallExpr{
			Fun:  arrayWant.Node().(ast.Expr),
			Args: []ast.Expr{expr.Node().(ast.Expr)},
		},
		X:   ret,
		Typ: arrayWant,
	}, true
}

func (n *returnStmt) buildStmt(scope fnResolveScope) (ir.Stmt, bool, bool) {
	ext := &ir.ReturnStmt{Src: n.src}
	fType := scope.funcType()
	if fType == nil {
		return ext, true, scope.Err().AppendInternalf(n.src, "return statement without a function context")
	}
	wants := fType.Results.Fields()
	if hasNamedResult(wants) && len(n.results) == 0 {
		// Naked return with named results: nothing else to check.
		return ext, true, true
	}
	// Resolve all the expressions composing the return.
	ok := true
	ext.Results = make([]ir.Expr, len(n.results))
	for i, expr := range n.results {
		var exprOk bool
		ext.Results[i], exprOk = expr.buildExpr(scope)
		ok = exprOk && ok
	}
	if !ok {
		return ext, true, false
	}
	rTypes, isTuple := resultTypes(ext.Results)
	// Compare the number of values being returned to the function results.
	if len(rTypes) > len(wants) {
		ok = scope.Err().Appendf(n.source(), "too many return values")
	}
	if len(rTypes) < len(wants) {
		ok = scope.Err().Appendf(n.source(), "not enough return values")
	}
	if !ok {
		return ext, true, false
	}
	resultPos := ext.Results[0].Node()
	// Check if the types being returned are assignable to the types declared by the signature.
	for i, wantI := range wants {
		retOk := true
		if irkind.IsNumber(rTypes[i].Kind()) {
			ext.Results[i], retOk = n.castNumber(scope, ext.Results[i], wantI.Type())
			rTypes[i] = ext.Results[i].Type()
		}
		if !retOk {
			continue
		}
		gotType := rTypes[i]
		wantType := wantI.Group.Type.Typ
		if gotType.Kind() == irkind.Invalid || wantType.Kind() == irkind.Invalid {
			ok = false
			continue
		}
		posI := resultPos
		if !isTuple {
			posI = ext.Results[i].Node()
		}
		okI := returnAs(scope, posI, gotType, wantType)
		ok = ok && okI
	}
	return ext, true, ok
}
