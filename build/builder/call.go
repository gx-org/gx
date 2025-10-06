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
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/fun"
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
		return ext, rscope.Err().Appendf(n.src, "missing argument in conversion to %s", ext.Typ.String())
	}
	if len(args) > 1 {
		return ext, rscope.Err().Appendf(n.src, "too many arguments in conversion to %s", ext.Typ.String())
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

func checkNumArgs(rscope resolveScope, fn ir.Func, fType *ir.FuncType, numArgs int) bool {
	numParams := fType.Params.Len()
	if numArgs > numParams {
		return rscope.Err().Appendf(fn.Source(), "too many arguments in call to %s", fn.ShortString())
	}
	if numArgs < numParams {
		return rscope.Err().Appendf(fn.Source(), "not enough arguments in call to %s", fn.ShortString())
	}
	return true
}

func (n *callExpr) buildFuncType(rscope resolveScope, callee ir.Func, args []ir.AssignableExpr) (*ir.FuncType, bool) {
	fType := callee.FuncType()
	if fType != nil {
		return fType, true
	}
	var impl ir.FuncImpl
	switch fT := callee.(type) {
	case *ir.FuncBuiltin:
		impl = fT.Impl
	case *ir.FuncKeyword:
		impl = fT.Impl
	default:
		return nil, rscope.Err().AppendInternalf(callee.Source(), "missing function type but function %s:%T is not a builtin function", callee.ShortString(), callee)
	}
	compEval, ok := rscope.compEval()
	if !ok {
		return nil, false
	}
	fType, err := impl.BuildFuncType(compEval, &ir.CallExpr{
		Src:    n.src,
		Callee: &ir.FuncValExpr{F: callee},
		Args:   args,
	})
	if err != nil {
		return nil, rscope.Err().AppendAt(callee.Source(), err)
	}
	return fType, true
}

func (n *callExpr) buildCallee(rscope resolveScope, expr ir.AssignableExpr, fn ir.Func) ([]ir.AssignableExpr, *ir.FuncType, bool) {
	args, argsOk := n.buildArgs(rscope)
	if !argsOk {
		return nil, nil, false
	}
	fType, ok := n.buildFuncType(rscope, fn, args)
	if !ok {
		return nil, nil, false
	}
	if numArgsOk := checkNumArgs(rscope, fn, fType, len(args)); !numArgsOk {
		return nil, nil, false
	}
	args, callee, callOk := buildFuncForCall(rscope, &ir.FuncValExpr{
		X: expr,
		F: fn,
		T: fType,
	}, args)
	return args, callee.T, callOk
}

func (n *callExpr) buildFuncCallExpr(rscope resolveScope, expr ir.AssignableExpr, fn ir.Func) (ir.Expr, bool) {
	extCallee := &ir.FuncValExpr{
		X: expr,
		F: fn,
	}
	extCall := &ir.CallExpr{Src: n.src, Callee: extCallee}
	var ok bool
	extCall.Args, extCallee.T, ok = n.buildCallee(rscope, expr, fn)
	if !ok {
		return invalidExpr(), false
	}
	return extCall, true
}

func (n *callExpr) buildMacroCallExpr(rscope resolveScope, compEval *compileEvaluator, expr ir.AssignableExpr, astBuilder cpevelements.FuncASTBuilder) (ir.Expr, bool) {
	synDecl, synScope, ok := buildSyntheticFuncSig(rscope.fileScope(), astBuilder, nil)
	if !ok {
		return invalidExpr(), false
	}
	if _, ok = synScope.buildBody(synDecl, compEval); !ok {
		return invalidExpr(), false
	}
	extCallee := &ir.MacroCallExpr{
		X: expr,
		F: synDecl,
	}
	extCall := &ir.CallExpr{Src: n.src, Callee: extCallee}
	extCall.Args, extCallee.T, ok = n.buildCallee(rscope, expr, synDecl)
	if !ok {
		return invalidExpr(), false
	}
	return extCall, true
}

func (n *callExpr) buildCallExpr(rscope resolveScope, callee ir.AssignableExpr) (ir.Expr, bool) {
	compEval, ok := rscope.compEval()
	if !ok {
		return invalidExpr(), ok
	}
	el, err := compEval.fitp.EvalExpr(callee)
	if err != nil {
		return invalidExpr(), rscope.Err().AppendAt(callee.Source(), err)
	}
	switch elT := el.(type) {
	case *cpevelements.Macro:
		return n.buildFuncCallExpr(rscope, callee, elT.Func())
	case ir.Type:
		return n.buildTypeCast(rscope, callee, elT)
	case fun.Func:
		return n.buildFuncCallExpr(rscope, callee, elT.Func())
	case cpevelements.FuncASTBuilder:
		return n.buildMacroCallExpr(rscope, compEval, callee, elT)
	case cpevelements.FuncAnnotator:
		return invalidExpr(), rscope.Err().Appendf(callee.Source(), "invalid use of annotation macro %s", elT.From().ShortString())
	case *fun.NamedType:
		return n.buildTypeCast(rscope, callee, elT.Type())
	case *ir.TypeValExpr:
		return n.buildTypeCast(rscope, callee, elT.Store())
	default:
		return invalidExpr(), rscope.Err().AppendInternalf(callee.Source(), "expression %s evaluated to unsupported element of type %T. Scope:\n%s\nCompEval:\n%s", callee.String(), elT, rscope.String(), compEval.String())
	}
}

func (n *callExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	callee, calleeOk := buildAExpr(rscope, n.callee)
	if !calleeOk {
		return nil, false
	}
	return n.buildCallExpr(rscope, callee)
}
