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
	"go/token"
	"strings"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir/generics"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
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
		if i == len(src.Args)-1 && src.Ellipsis != token.NoPos {
			// Process the last argument differently if we have an ellipsis in the function call.
			n.args[i], argOk = processExprWithEllipsis(pscope, arg)
		} else {
			n.args[i], argOk = processExpr(pscope, arg)
		}
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
		args[i], argOk = buildExpr(scope, arg)
		ok = ok && argOk
	}
	return args, ok
}

func (n *callExpr) buildTypeCast(rscope resolveScope, callee ir.Expr, store ir.Storage) (ir.Expr, bool) {
	typRef := ir.TypeFromStorage(callee, store)
	if typRef == nil {
		return typeError(rscope, callee)
	}
	ext := &ir.CastExpr{Src: n.src, Typ: typRef.Val()}
	args, argsOk := n.buildArgs(rscope)
	if !argsOk {
		return ext, false
	}
	if len(args) == 0 {
		return ext, rscope.Err().Appendf(n.src, "missing argument in conversion to %s", ext.Typ.ReferString(rscope.fileScope().irFile()))
	}
	if len(args) > 1 {
		return ext, rscope.Err().Appendf(n.src, "too many arguments in conversion to %s", ext.Typ.ReferString(rscope.fileScope().irFile()))
	}
	var ok bool
	ext.X = args[0]
	ext.X, ok = castNilAndNumber(rscope, ext.X, ext.Typ)
	if !ok {
		return ext, false
	}
	srcType := ext.X.Type()
	convertOk := convertToAt(rscope, n.src, srcType, ext.Typ)
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
		return rscope.Err().Appendf(fn.Node(), "too many arguments in call to %s (expected %d, found %d)", fn.SourceString(rscope.fileScope().irFile()), numParams, numArgs)
	}
	if numArgs < minNumParams {
		return rscope.Err().Appendf(fn.Node(), "not enough arguments in call to %s (expected %d, found %d)", fn.SourceString(rscope.fileScope().irFile()), numParams, numArgs)
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
		return nil, rscope.Err().AppendInternalf(callee.Node(), "missing function type but function %s:%T is not a builtin function", callee.SourceString(rscope.fileScope().irFile()), callee)
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

func (n *callExpr) buildFuncCallExpr(rscope resolveScope, expr *ir.FuncValExpr) (*ir.FuncCallExpr, bool) {
	ext := &ir.FuncCallExpr{Src: n.src}
	var ok bool
	ext.Args, ok = n.buildArgs(rscope)
	if !ok {
		return ext, false
	}
	if expr.FuncType() == nil {
		var ok bool
		expr, ok = n.completeFuncType(rscope, expr, ext.Args)
		if !ok {
			return ext, false
		}
	}
	if numArgsOk := checkNumArgs(rscope, expr, len(ext.Args)); !numArgsOk {
		return ext, false
	}
	ext.Args, ext.Callee, ok = buildFuncForCall(rscope, expr, ext.Args)
	return ext, ok
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
		return fmt.Sprintf("error while calling macro %s:\n", callExpr.SourceString(rscope.fileScope().irFile()))
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
	switch elT := ir.BareValue(el).(type) {
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
	case *ir.MacroKeyword:
		return n.callMacroKeyword(rscope, elT)
	case ir.TupleElement:
		return invalidExpr(), rscope.Err().Appendf(callee.Node(), "multiple values %s in single-value context", callee.SourceString(rscope.fileScope().irFile()))
	default:
		return invalidExpr(), rscope.Err().Appendf(callee.Node(), "%s not callable", callee.SourceString(rscope.fileScope().irFile()))
	}
}

func (n *callExpr) callMacroKeyword(rscope resolveScope, callee *ir.MacroKeyword) (ir.Expr, bool) {
	if len(n.args) != 1 {
		return invalidExpr(), rscope.Err().Appendf(n.src, "expect 1 argument, got %d", len(n.args))
	}
	args, ok := n.buildArgs(rscope)
	if !ok {
		return invalidExpr(), false
	}
	cpev, ok := rscope.compEval()
	if !ok {
		return invalidExpr(), false
	}
	x, err := callee.BuildSynthetic(cpev, args[0])
	if err != nil {
		return x, rscope.Err().AppendAt(n.src, err)
	}
	return x, true
}

func (n *callExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	callee, calleeOk := buildCoreExpr(rscope, n.callee)
	if !calleeOk {
		return nil, false
	}
	return n.buildCallExpr(rscope, callee)
}

type atSource struct {
	fmterr.ErrAppender
	src ast.Expr
}

func (s atSource) Source() ast.Expr {
	return s.src
}

func checkArgsForCall(rscope resolveScope, ce *compileEvaluator, fExpr *ir.FuncValExpr, args []ir.Expr) ([]ir.Expr, bool) {
	ok := true
	ftype := fExpr.FuncType()
	out := make([]ir.Expr, len(args))
	numParams := ftype.Params.Len()
	for i, arg := range args {
		param, isVarArg := ftype.ArgIndexToParamField(i)
		target := param.Type()
		if isVarArg {
			var targetOk bool
			target, targetOk = ftype.VarArgs.IndexForVarArgs(&atSource{
				ErrAppender: rscope,
				src:         fExpr.Expr(),
			}, i-(numParams-1))
			ok = ok && targetOk
		}
		argType := arg.Type()
		if unpack, isUnpack := arg.(*ir.UnpackExpr); isUnpack {
			argType = unpack.EltTyp
		}
		if irkind.IsNumber(argType.Kind()) {
			var argOk bool
			arg, argOk = castNilAndNumber(rscope, arg, target)
			argType = arg.Type()
			if !argOk {
				ok = argOk
			}
		}
		assignable, err := ir.AssignableTo(ce, argType, target)
		if err != nil {
			return args, ce.Err().AppendAt(arg.Node(), err)
		}
		if !assignable {
			from := ce.File()
			ok = ce.Err().Appendf(arg.Node(), "cannot use type %s as %s in argument to %s", argType.ReferString(from), target.ReferString(from), fExpr.SourceString(from))
		}
		out[i] = arg
	}
	return out, ok
}

func evalGenericValues(ce *compileEvaluator, ftype *ir.FuncType) (map[string]ir.Element, bool) {
	out := make(map[string]ir.Element)
	for _, tParam := range ftype.TypeParams.Fields() {
		if !ir.ValidIdent(tParam.Name) {
			continue
		}
		name := tParam.Name.Name
		file := ce.File()
		storage := tParam.Storage()
		var el ir.Element
		if ir.IsNonTypeGeneric(tParam.Type()) {
			storeAt := elements.NewNodeAt[ir.Storage](file, storage)
			el = cpevelements.NewProxy(storeAt)
		} else {
			genType := ir.NewGenericTypeParam(storage.Field)
			el = cpevelements.NewStoredValue(file, genType, genType)
		}
		out[name] = el
	}
	ok := true
	for _, genVal := range ftype.GenericValues {
		if genVal == nil {
			continue
		}
		field := genVal.Generic().OrigField()
		if !ir.ValidIdent(field.Name) {
			continue
		}
		var err error
		out[field.Name.Name], err = ce.EvalExpr(genVal.Value())
		if err != nil {
			ok = ce.Err().AppendAt(genVal.Value().Node(), err)
		}
	}
	return out, ok
}

func instantiateFType(ce *compileEvaluator, fExpr *ir.FuncValExpr, funcFile *ir.File) (*ir.FuncType, bool) {
	ftype := fExpr.FuncType()
	genVals, ok := evalGenericValues(ce, ftype)
	if !ok {
		return ftype, false
	}
	sigCE, ok := ce.sub(funcFile, genVals)
	if !ok {
		return ftype, false
	}
	return generics.Instantiate(sigCE, fExpr, ftype.GenericValues)
}

func buildFuncForCall(rscope resolveScope, fExpr *ir.FuncValExpr, args []ir.Expr) ([]ir.Expr, *ir.FuncValExpr, bool) {
	if rscope.requireCompevalCall() && !fExpr.Func().FuncType().CompEval {
		return args, fExpr, rscope.Err().Appendf(fExpr.Node(), "expect a compeval function, function %s is not", fExpr.Func().ShortString())
	}
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return args, fExpr, false
	}
	var ok bool
	fExpr, ok = generics.Infer(compEval, fExpr, args)
	if !ok {
		return args, fExpr, false
	}
	typeParams := fExpr.FuncType().TypeParams.Fields()
	if len(typeParams) > 0 {
		names := make([]string, len(typeParams))
		for i, field := range typeParams {
			names[i] = field.Name.Name
		}
		return args, fExpr, rscope.Err().Appendf(fExpr.Node(), "in call to %s, cannot infer type %s", fExpr.Func().ShortString(), strings.Join(names, ","))
	}
	fTypeInst, instOk := instantiateFType(compEval, fExpr, fExpr.Func().File())
	fExprInst := fExpr.NewFType(fTypeInst)
	var argsOk bool
	args, argsOk = checkArgsForCall(rscope, compEval, fExprInst, args)
	return args, fExprInst, instOk && argsOk
}
