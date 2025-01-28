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

func unpackIndexedExpr(n ast.Node) (ast.Expr, []ast.Expr) {
	switch e := n.(type) {
	case *ast.IndexExpr:
		return e.X, []ast.Expr{e.Index}
	case *ast.IndexListExpr:
		return e.X, e.Indices
	}
	return nil, nil
}

// callExpr represents a function call.
type callExpr struct {
	ext ir.Expr
	src *ast.CallExpr

	typeArgs []typeNode
	args     []exprNode

	callee exprNode
	cast   *castExpr
}

var _ exprNode = (*callExpr)(nil)

func processCallExpr(owner owner, expr *ast.CallExpr) (exprNode, bool) {
	fn, typeArgs := unpackIndexedExpr(expr.Fun)
	if fn == nil {
		fn = expr.Fun
	}

	callee, ok := processExpr(owner, fn)
	if typCast := toTypeCast(callee); typCast != nil {
		if len(typeArgs) > 0 {
			// For completeness sake, warn if type arguments are somehow ignored here.
			owner.err().Appendf(expr, "cast may not have type arguments")
		}
		return processCastExpr(owner, expr, typCast)
	}
	n := &callExpr{
		src:    expr,
		callee: callee,
	}

	n.typeArgs = make([]typeNode, len(typeArgs))
	for i, expr := range typeArgs {
		ident, identOk := expr.(*ast.Ident)
		if !identOk {
			owner.err().Appendf(expr, "expected identifier as type argument")
			ok = false
			continue
		}
		argI, okI := processIdentTypeExpr(owner, ident)
		n.typeArgs[i] = argI
		ok = ok && okI
	}
	n.args = make([]exprNode, len(expr.Args))
	for i, expr := range expr.Args {
		argI, okI := processExpr(owner, expr)
		n.args[i] = argI
		ok = ok && okI
	}
	return n, ok
}

func (n *callExpr) expr() ast.Expr {
	return n.src
}

// pos of the call in the code.
func (n *callExpr) source() ast.Node {
	return n.src
}

func (n *callExpr) buildExpr() ir.Expr {
	return n.ext
}

func (n *callExpr) String() string {
	return n.callee.String()
}

func (n *callExpr) resolveArgs(scope scoper) ([]ir.Expr, []typeNode, bool) {
	ok := true
	argsExpr := make([]ir.Expr, len(n.args))
	argsType := make([]typeNode, len(n.args))
	for i, arg := range n.args {
		var argOk bool
		argsType[i], argOk = arg.resolveType(scope)
		ok = ok && argOk
		argsExpr[i] = arg.buildExpr()
	}
	return argsExpr, argsType, ok
}

func (n *callExpr) resolveType(scope scoper) (typeNode, bool) {
	// Resolve the type of the function being called.
	calleeTp, ok := n.callee.resolveType(scope)
	if !ok {
		return invalid, false
	}
	if calleeTp.kind() != ir.FuncKind {
		return n.resolveCast(scope)
	}

	ext := &ir.CallExpr{
		Src: n.src,
	}
	n.ext = ext

	// Resolve the arguments passed to the function.
	var argsType []typeNode
	ext.Args, argsType, ok = n.resolveArgs(scope)
	if !ok {
		return invalid, false
	}

	callType, ok := n.resolveFunctionCall(scope, ext, calleeTp, argsType)
	ext.Func = n.callee.buildExpr()
	return callType, ok
}

func (n *callExpr) resolveCast(scope scoper) (typeNode, bool) {
	typeCast, ok := processCast(scope, n.callee.expr())
	if !ok {
		return invalid, false
	}
	n.cast, ok = processCastExpr(scope, n.src, typeCast)
	if !ok {
		return invalid, false
	}
	typ, ok := n.cast.resolveType(scope)
	if !ok {
		return invalid, false
	}
	n.ext = n.cast.buildExpr()
	return typeNodeOk(typ)
}

func (n *callExpr) resolveFunctionCall(scope scoper, ext *ir.CallExpr, calleeTp typeNode, argsType []typeNode) (typeNode, bool) {
	var ok bool
	fnType, ok := resolveGenericCallType(scope, calleeTp, scope.evalFetcher(), n)
	if !ok {
		return invalid, false
	}

	if fnType.params.isGeneric() {
		// When invoking a function with generic types (or generic shape parameters), create a new scope
		// here so that the instantiated types are isolated from the caller.
		// TODO: Pass in a valid *funcDecl to scopeFunc().
		scope = scope.fileScope().scopeFunc(nil, scope.namespace().newChild())
	}

	// Check that the number of arguments matches with what the function expects.
	numFuncArgs := fnType.params.numFields()
	if len(n.args) > numFuncArgs {
		scope.err().Appendf(n.source(), "too many arguments in call to %s", n.callee.String())
		return invalid, false
	}
	if len(n.args) < numFuncArgs {
		scope.err().Appendf(n.source(), "not enough arguments in call to %s", n.callee.String())
		return invalid, false
	}

	// Check that the arguments match what the function expects.
	ext.Args = make([]ir.Expr, len(n.args))
	for i, wantArg := range fnType.params.fields() {
		argOk := true
		argType := argsType[i]
		if _, argOk = typeNodeOk(argType); !argOk {
			continue
		}
		arg := n.args[i]
		if ir.IsNumber(argType.kind()) {
			arg, argType, argOk = castNumber(scope, arg, wantArg.typ().irType())
			ok = argOk && ok
			n.args[i] = arg
		}
		if !ok {
			return invalid, false
		}
		if wantArg.typ().isGeneric() {
			// Re-resolve generic parameter types, since they may depend on previous arguments. For
			// example: when calling func ([___X]int32, [___X]int32), both arguments must have the same
			// shape.
			wantArg.group.typ, argOk = resolveType(scope, wantArg.group, wantArg.typ())
			ok = ok && argOk
		}
		newType, assignable := assignableToAt(scope, arg, argType, wantArg.typ())
		if !assignable {
			ok = false
		}
		wantArg.group.typ = newType
		ext.Args[i] = arg.buildExpr()
	}

	for _, result := range fnType.results.fields() {
		// Resolve any generic types in the result, since they may depend on the arguments.
		if !result.typ().isGeneric() {
			continue
		}
		var resultOk bool
		result.group.typ, resultOk = resolveType(scope, result.group, result.typ())
		ok = ok && resultOk
	}

	ext.FuncType = fnType.buildFuncType()
	fnResult := fnType.resultTypes()
	var typ typeNode
	switch fnResult.len() {
	case 0:
		typ = void
	case 1:
		typ = fnResult.elt(0).typ()
	default:
		typ = fnResult
	}
	return typ, ok
}
