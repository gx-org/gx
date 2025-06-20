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
	src *ast.CallExpr

	args []exprNode

	callee exprNode
}

var _ exprNode = (*callExpr)(nil)

func processCallExpr(pscope procScope, src *ast.CallExpr) (exprNode, bool) {
	n := &callExpr{src: src}
	var calleeOk bool
	n.callee, calleeOk = processExpr(pscope, src.Fun)
	n.args = make([]exprNode, len(src.Args))
	argsOk := true
	for i, arg := range src.Args {
		var argOk bool
		n.args[i], argOk = processExpr(pscope, arg)
		argsOk = argsOk && argOk
	}
	return n, calleeOk && argsOk
}

// pos of the call in the code.
func (n *callExpr) source() ast.Node {
	return n.src
}

func (n *callExpr) String() string {
	return n.callee.String()
}

func (n *callExpr) buildArgs(scope resolveScope) ([]ir.AssignableExpr, bool) {
	ok := true
	args := make([]ir.AssignableExpr, len(n.args))
	for i, arg := range n.args {
		var argOk bool
		args[i], argOk = buildAExpr(scope, arg)
		ok = ok && argOk
	}
	return args, ok
}

func (n *callExpr) buildTypeCast(rscope resolveScope, callee ir.AssignableExpr, store ir.Storage) (ir.Expr, bool) {
	typRef, ok := typeFromStorage(rscope, callee, store)
	if !ok {
		return nil, false
	}
	dst := typRef.Typ
	ext := &ir.CastExpr{Src: n.src, Typ: dst}
	args, argsOk := n.buildArgs(rscope)
	if !argsOk {
		return ext, false
	}
	if len(args) == 0 {
		return ext, rscope.err().Appendf(n.src, "missing argument in conversion to %s", ext.Typ.String())
	}
	if len(args) > 1 {
		return ext, rscope.err().Appendf(n.src, "too many arguments in conversion to %s", ext.Typ.String())
	}
	ext.X = args[0]
	if ir.IsNumber(ext.X.Type().Kind()) {
		ext.X, ok = castNumber(rscope, ext.X, ext.Typ)
		targetArrayType := ir.ToArrayType(ext.Typ)
		if targetArrayType != nil && !targetArrayType.Rank().IsAtomic() {
			ext.X = &ir.CastExpr{
				Src: n.src,
				Typ: ext.Typ,
				X:   ext.X,
			}
		}
	}
	if !ok {
		return ext, false
	}
	convertOk := convertToAt(rscope, n.src, ext.X.Type(), ext.Typ)
	return ext, convertOk
}

func checkNumArgs(rscope resolveScope, callee *ir.FuncValExpr, numArgs int) bool {
	numParams := callee.T.Params.Len()
	if numArgs > numParams {
		return rscope.err().Appendf(callee.Source(), "too many arguments in call to %s", callee.Name())
	}
	if numArgs < numParams {
		return rscope.err().Appendf(callee.Source(), "not enough arguments in call to %s", callee.Name())
	}
	return true
}

func buildMissingFuncType(rscope resolveScope, callee *ir.FuncValExpr, call *ir.CallExpr) (*ir.FuncValExpr, bool) {
	if callee.T != nil {
		return callee, true
	}
	builtin, isBuiltin := callee.F.(*ir.FuncBuiltin)
	if !isBuiltin {
		return callee, rscope.err().AppendInternalf(callee.Source(), "missing function but function %s:%T is not a builtin function", callee.Name(), callee.T)
	}
	compEval, ok := rscope.compEval()
	if !ok {
		return callee, false
	}
	ext := *callee
	var err error
	ext.T, err = builtin.Impl.BuildFuncType(compEval, call)
	if err != nil {
		return callee, rscope.err().AppendAt(callee.Source(), err)
	}
	return &ext, true
}

func (n *callExpr) buildFunctionCall(rscope resolveScope, callee *ir.FuncValExpr) (ir.Expr, bool) {
	ext := &ir.CallExpr{Src: n.src, Callee: callee}
	var argsOk bool
	ext.Args, argsOk = n.buildArgs(rscope)
	if !argsOk {
		return ext, false
	}
	callee, ok := buildMissingFuncType(rscope, callee, ext)
	if !ok {
		return ext, false
	}
	if numArgsOk := checkNumArgs(rscope, callee, len(ext.Args)); !numArgsOk {
		return ext, false
	}
	var callOk bool
	ext.Callee, ext.Args, callOk = buildFuncForCall(rscope, callee, ext.Args)
	return ext, callOk
}

func (n *callExpr) buildCallExpr(rscope resolveScope, callee ir.AssignableExpr) (ir.Expr, bool) {
	store, ok := storageFromExpr(rscope, callee)
	if !ok {
		return nil, false
	}
	if store.Type().Kind() == ir.MetaTypeKind {
		return n.buildTypeCast(rscope, callee, store)
	}
	val, ok := valueFromStorage(rscope, callee, store)
	if !ok {
		return nil, false
	}
	funcRef, ok := val.(*ir.FuncValExpr)
	if !ok {
		return nil, rscope.err().Appendf(n.callee.source(), "cannot call non-function %s (type %s)", callee.String(), callee.Type().String())
	}
	return n.buildFunctionCall(rscope, funcRef)
}

func (n *callExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	callee, calleeOk := buildAExpr(rscope, n.callee)
	if !calleeOk {
		return nil, false
	}
	return n.buildCallExpr(rscope, callee)
}
