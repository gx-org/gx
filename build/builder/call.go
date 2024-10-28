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

// callExpr represents a function call.
type callExpr struct {
	ext ir.Expr
	src *ast.CallExpr

	args []exprNode

	callee exprNode
	cast   *castExpr
}

var _ exprNode = (*callExpr)(nil)

func processCallExpr(owner owner, expr *ast.CallExpr) (exprNode, bool) {
	callee, ok := processExpr(owner, expr.Fun)
	if typCast := toTypeCast(callee); typCast != nil {
		return processCastExpr(owner, expr, typCast)
	}
	n := &callExpr{
		src:    expr,
		callee: callee,
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

func (n *callExpr) resolveArgs(scope scoper, ext *ir.CallExpr) ([]typeNode, bool) {
	ok := true
	ext.Args = make([]ir.Expr, len(n.args))
	args := make([]typeNode, len(n.args))
	for i, arg := range n.args {
		var argOk bool
		args[i], argOk = arg.resolveType(scope)
		ok = ok && argOk
		ext.Args[i] = arg.buildExpr()
	}
	return args, ok
}

func (n *callExpr) resolveType(scope scoper) (typeNode, bool) {
	ext := &ir.CallExpr{
		Src: n.src,
	}
	n.ext = ext

	// Resolve the type of the function being called.
	calleeTp, ok := n.callee.resolveType(scope)
	if !ok {
		return invalid, false
	}

	// Resolve the arguments passed to the function.
	argsType, ok := n.resolveArgs(scope, ext)
	if !ok {
		return invalid, false
	}
	var callType typeNode
	if calleeTp.kind() == ir.FuncKind {
		callType, ok = n.resolveFunctionCall(scope, ext, calleeTp, argsType)
	} else {
		callType, ok = n.resolveCast(scope, ext, calleeTp, argsType)
	}
	ext.Func = n.callee.buildExpr()
	return callType, ok
}

func (n *callExpr) resolveCast(scope scoper, ext *ir.CallExpr, calleeTp typeNode, argsType []typeNode) (typeNode, bool) {
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
	fnType, ok := resolveGenericCallType(scope, n.callee.String(), n.source(), calleeTp, scope.evalFetcher(), ext)
	if !ok {
		return invalid, false
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
	wantArgs := fnType.params.fields()
	ext.Args = make([]ir.Expr, len(n.args))
	for i, arg := range n.args {
		argOk := true
		argType := argsType[i]
		if _, argOk = typeNodeOk(argType); !argOk {
			continue
		}
		if argType.kind() == ir.NumberKind {
			arg, argType, argOk = buildNumberNode(scope, arg, wantArgs[i].typ().buildType())
			ok = argOk && ok
			n.args[i] = arg
		}
		if !ok {
			return invalid, false
		}
		_, assignable := assignableToAt(scope, arg, argType, wantArgs[i].typ())
		if !assignable {
			ok = false
		}
		ext.Args[i] = arg.buildExpr()
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
