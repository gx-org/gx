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
	"fmt"
	"go/ast"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp"
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

func (n *callExpr) buildArgs(scope resolveScope) ([]ir.Expr, bool) {
	ok := true
	args := make([]ir.Expr, len(n.args))
	for i, arg := range n.args {
		var argOk bool
		args[i], argOk = buildAExpr(scope, arg)
		ok = ok && argOk
	}
	return args, ok
}

func (n *callExpr) buildTypeCast(rscope resolveScope, callee ir.Expr, store ir.Storage) (ir.Expr, bool) {
	typRef, ok := typeFromStorage(rscope, callee, store)
	if !ok {
		return nil, false
	}
	dst := typRef.Val()
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
	if irkind.IsNumber(ext.X.Type().Kind()) {
		ext.X, ok = castNumber(rscope, ext.X, ext.Typ)
	}
	if !ok {
		return ext, false
	}
	convertOk := convertToAt(rscope, n.src, ext.X.Type(), ext.Typ)
	return ext, convertOk
}

func checkNumArgs(rscope resolveScope, fn *ir.FuncValExpr, numArgs int) bool {
	ftype := fn.FuncType()
	numParams := ftype.Params.Len()
	minNumParams, maxNumParams := numParams, numParams
	if ftype.VarArgs != nil {
		minNumParams = numParams - 1
		maxNumParams = -1
	}
	if maxNumParams >= 0 && numArgs > maxNumParams {
		return rscope.Err().Appendf(fn.Node(), "too many arguments in call to %s (expected %d, found %d)", fn.ShortString(), numParams, numArgs)
	}
	if numArgs < minNumParams {
		return rscope.Err().Appendf(fn.Node(), "not enough arguments in call to %s (expected %d, found %d)", fn.ShortString(), numParams, numArgs)
	}
	return true
}

func (n *callExpr) completeFuncType(rscope resolveScope, callee *ir.FuncValExpr, args []ir.Expr) (*ir.FuncValExpr, bool) {
	fType := callee.FuncType()
	if fType != nil {
		return callee, true
	}
	var impl ir.FuncImpl
	switch fT := callee.Func().(type) {
	case *ir.FuncBuiltin:
		impl = fT.Impl
	case *ir.FuncKeyword:
		impl = fT.Impl
	default:
		return nil, rscope.Err().AppendInternalf(callee.Node(), "missing function type but function %s:%T is not a builtin function", callee.ShortString(), callee)
	}
	compEval, ok := rscope.compEval()
	if !ok {
		return nil, false
	}
	fType, err := impl.BuildFuncType(compEval, &ir.FuncCallExpr{
		Src:    n.src,
		Callee: callee,
		Args:   args,
	})
	if err != nil {
		return nil, rscope.Err().AppendAt(n.src, err)
	}
	return callee.NewFType(fType), true
}

func (n *callExpr) buildCallee(rscope resolveScope, expr *ir.FuncValExpr) ([]ir.Expr, *ir.FuncValExpr, bool) {
	args, argsOk := n.buildArgs(rscope)
	if !argsOk {
		return nil, nil, false
	}
	if expr.FuncType() == nil {
		var ok bool
		expr, ok = n.completeFuncType(rscope, expr, args)
		if !ok {
			return nil, nil, false
		}
	}
	if numArgsOk := checkNumArgs(rscope, expr, len(args)); !numArgsOk {
		return nil, nil, false
	}
	args, callee, callOk := buildFuncForCall(rscope, expr, args)
	return args, callee, callOk
}

func (n *callExpr) buildFuncCallExpr(rscope resolveScope, expr *ir.FuncValExpr) (*ir.FuncCallExpr, bool) {
	extCall := &ir.FuncCallExpr{Src: n.src}
	var ok bool
	extCall.Args, extCall.Callee, ok = n.buildCallee(rscope, expr)
	if !ok {
		return nil, false
	}
	return extCall, true
}

func (n *callExpr) buildMacroCall(rscope resolveScope, compEval *compileEvaluator, expr ir.Expr, mac *ir.Macro) (ir.Expr, bool) {
	callExpr, ok := n.buildFuncCallExpr(rscope, ir.NewFuncValExpr(expr, mac))
	if !ok {
		return invalidExpr(), false
	}
	macroEl, ok := evalMacroCall(compEval, callExpr)
	if !ok {
		return invalidExpr(), false
	}
	fnBuilder, ok := macroEl.(ir.FuncASTBuilder)
	if !ok {
		return invalidExpr(), rscope.Err().Appendf(n.source(), "macro %s does not build functions", mac.ShortString())
	}
	rscope.Err().Push(fmterr.PosPrefixWith(rscope.fileScope().pkg().fset, callExpr.Src.Fun, func() string {
		return fmt.Sprintf("error while calling macro %s:\n", callExpr.String())
	}))
	defer rscope.Err().Pop()

	synDecl, synScope, ok := buildSyntheticFuncSig(rscope.fileScope(), expr.Node(), fnBuilder, nil)
	if !ok {
		return invalidExpr(), false
	}
	if ok = synScope.buildBody(synDecl, compEval); !ok {
		return invalidExpr(), false
	}
	return &ir.MacroCallExpr{
		X: callExpr,
		M: fnBuilder,
		F: synDecl,
	}, true
}

func (n *callExpr) buildCallExpr(rscope resolveScope, callee ir.Expr) (ir.Expr, bool) {
	switch calleeT := callee.(type) {
	case *ir.FuncValExpr:
		return n.buildFuncCallExpr(rscope, calleeT)
	}
	compEval, ok := rscope.compEval()
	if !ok {
		return invalidExpr(), ok
	}
	el, err := compEval.fitp.EvalExpr(callee)
	if err != nil {
		return invalidExpr(), rscope.Err().AppendAt(callee.Node(), err)
	}
	switch elT := el.(type) {
	case *cpevelements.Macro:
		return n.buildMacroCall(rscope, compEval, callee, elT.IR())
	case ir.Type:
		return n.buildTypeCast(rscope, callee, elT)
	case ir.FuncAnnotator:
		return invalidExpr(), rscope.Err().Appendf(callee.Node(), "annotator gx:@%s only valid in a function annotation context", elT.ShortString())
	case *fun.NamedType:
		return n.buildTypeCast(rscope, callee, elT.Type())
	case *ir.TypeValExpr:
		return n.buildTypeCast(rscope, callee, elT.Store())
	case ir.FuncElement:
		return n.buildCallExpr(rscope, ir.NewFuncValExpr(callee, elT.Func()))
	case *interp.Tuple:
		return invalidExpr(), rscope.Err().Appendf(callee.Node(), "multiple value %s in single-value context", callee.String())
	default:
		return invalidExpr(), rscope.Err().AppendInternalf(callee.Node(), "expression %s evaluated to element of type %T that is not callable. Scope:\n%s\nCompEval:\n%s", callee.String(), elT, rscope.String(), compEval.String())
	}
}

func (n *callExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	callee, calleeOk := buildAExpr(rscope, n.callee)
	if !calleeOk {
		return nil, false
	}
	return n.buildCallExpr(rscope, callee)
}
