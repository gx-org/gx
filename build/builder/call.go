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
	"reflect"

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

func checkNumArgs(rscope resolveScope, callee *ir.FuncValExpr, numArgs int) bool {
	numParams := callee.T.Params.Len()
	if numArgs > numParams {
		return rscope.Err().Appendf(callee.Source(), "too many arguments in call to %s", callee.ShortString())
	}
	if numArgs < numParams {
		return rscope.Err().Appendf(callee.Source(), "not enough arguments in call to %s", callee.ShortString())
	}
	return true
}

func buildMissingFuncType(rscope resolveScope, callee *ir.FuncValExpr, call *ir.CallExpr) (*ir.FuncValExpr, bool) {
	if callee.T != nil {
		return callee, true
	}
	var impl ir.FuncImpl
	switch fT := callee.F.(type) {
	case *ir.FuncBuiltin:
		impl = fT.Impl
	case *ir.FuncKeyword:
		impl = fT.Impl
	default:
		return callee, rscope.Err().AppendInternalf(callee.Source(), "missing function but function %s:%T is not a builtin function", callee.ShortString(), callee.T)
	}
	compEval, ok := rscope.compEval()
	if !ok {
		return callee, false
	}
	ext := *callee
	var err error
	ext.T, err = impl.BuildFuncType(compEval, call)
	if err != nil {
		return callee, rscope.Err().AppendAt(callee.Source(), err)
	}
	return &ext, true
}

func (n *callExpr) buildFunctionCall(rscope resolveScope, callee *ir.FuncValExpr) (*ir.CallExpr, bool) {
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

func buildFuncValExpr(rscope resolveScope, callee ir.AssignableExpr, stor ir.Storage, val ir.AssignableExpr) (*ir.FuncValExpr, bool) {
	fType, isFunc := ir.Underlying(val.Type()).(*ir.FuncType)
	if !isFunc {
		return nil, rscope.Err().Appendf(callee.Source(), "cannot call non-function %s (type %s)", callee.String(), callee.Type().String())
	}
	fn, isFunc := val.(ir.Func)
	if !isFunc {
		return nil, rscope.Err().AppendInternalf(callee.Source(), "%T has function type %s but does not implement %s", val, fType.String(), reflect.TypeFor[ir.Func]().Name())
	}
	return &ir.FuncValExpr{
		X: callee,
		F: fn,
		T: fType,
	}, true
}

func (n *callExpr) buildMacroCall(rscope resolveScope, compEval *compileEvaluator, callee ir.AssignableExpr, astBuilder cpevelements.FuncASTBuilder) (ir.Expr, bool) {
	src, ok := astBuilder.BuildFuncLit(compEval)
	if !ok {
		return invalidExpr, false
	}
	fn, ok := processFuncLit(rscope.fileScope().processScope(), src)
	if !ok {
		return invalidExpr, false
	}
	fnIR, ok := fn.buildFuncLit(rscope)
	if !ok {
		return invalidExpr, false
	}
	return n.buildFunctionCall(rscope, &ir.FuncValExpr{
		X: callee,
		F: fnIR,
		T: fnIR.FuncType(),
	})
}

func (n *callExpr) buildCallExpr(rscope resolveScope, callee ir.AssignableExpr) (ir.Expr, bool) {
	compEval, ok := rscope.compEval()
	if !ok {
		return invalidExpr, ok
	}
	el, err := compEval.fitp.EvalExpr(callee)
	if err != nil {
		return invalidExpr, rscope.Err().AppendAt(callee.Source(), err)
	}
	switch elT := el.(type) {
	case *cpevelements.Macro:
		return n.buildFunctionCall(rscope, &ir.FuncValExpr{
			X: callee,
			F: elT.Func(),
			T: elT.Func().FuncType(),
		})
	case ir.Type:
		return n.buildTypeCast(rscope, callee, elT)
	case fun.Func:
		return n.buildFunctionCall(rscope, &ir.FuncValExpr{
			X: callee,
			F: elT.Func(),
			T: elT.Func().FuncType(),
		})
	case cpevelements.FuncASTBuilder:
		return n.buildMacroCall(rscope, compEval, callee, elT)
	case cpevelements.FuncAnnotator:
		return invalidExpr, rscope.Err().Appendf(callee.Source(), "invalid use of annotation macro %s", elT.From().IR().ShortString())
	case *fun.NamedType:
		return n.buildTypeCast(rscope, callee, elT.Type())
	case *ir.TypeValExpr:
		return n.buildTypeCast(rscope, callee, elT.Store())
	default:
		return invalidExpr, rscope.Err().AppendInternalf(callee.Source(), "expression %s evaluated to unsupported element of type %T. Scope:\n%s\nCompEval:\n%s", callee.String(), elT, rscope.String(), compEval.String())
	}
}

func (n *callExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	callee, calleeOk := buildAExpr(rscope, n.callee)
	if !calleeOk {
		return nil, false
	}
	return n.buildCallExpr(rscope, callee)
}
