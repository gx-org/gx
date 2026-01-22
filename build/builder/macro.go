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

package builder

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

type funcMeta struct {
	*funcDecl
}

func (f *funcMeta) source() ast.Node {
	return f.src
}

func (f *funcMeta) name() *ast.Ident {
	return f.src.Name
}

func (f *funcMeta) buildBody(fnResolveScope, *irFunc) bool {
	return true
}

func (f *funcMeta) receiver() *fieldList {
	return nil
}

func (f *funcMeta) compEval() bool {
	return true
}

func (f *funcMeta) resolveOrder() int {
	// Macro needs to be resolved first before any other functions.
	return -1
}

func (f *funcMeta) buildSignature(fScope *fileResolveScope) (ir.MetaCore, fnResolveScope, bool) {
	ext := ir.MetaCore{
		Src:   f.src,
		FFile: fScope.irFile(),
	}
	var ok bool
	var fnScope *funcResolveScope
	ext.FType, fnScope, ok = f.fType.buildFuncType(fScope)
	return ext, fnScope, ok
}

type funcMacro struct {
	funcMeta
}

func (bFile *file) processIRMacroFunc(scope procScope, src *ast.FuncDecl, comment *ast.Comment) (function, bool) {
	fDecl, declOk := newFuncDecl(scope, src, false)
	fn := &funcMacro{funcMeta{funcDecl: fDecl}}
	return fn, declOk
}

func (f *funcMacro) buildSignature(fScope *fileResolveScope) (ir.Func, fnResolveScope, bool) {
	core, fnScope, ok := f.funcMeta.buildSignature(fScope)
	return &ir.Macro{MetaCore: core}, fnScope, ok
}

type funcAnnotatorBuilder struct {
	funcMeta
}

func (bFile *file) processAnnotatorFunc(scope procScope, src *ast.FuncDecl, comment *ast.Comment) (function, bool) {
	fDecl, declOk := newFuncDecl(scope, src, false)
	fn := &funcAnnotatorBuilder{funcMeta{funcDecl: fDecl}}
	return fn, declOk
}

func (f *funcAnnotatorBuilder) buildSignature(fScope *fileResolveScope) (ir.Func, fnResolveScope, bool) {
	core, fnScope, ok := f.funcMeta.buildSignature(fScope)
	return &ir.AnnotatorFunc{MetaCore: core}, fnScope, ok
}

type fieldAnnotatorBuilder struct {
	funcMeta
}

func (bFile *file) processAnnotatorField(scope procScope, src *ast.FuncDecl, comment *ast.Comment) (function, bool) {
	fDecl, declOk := newFuncDecl(scope, src, false)
	fn := &fieldAnnotatorBuilder{funcMeta{funcDecl: fDecl}}
	return fn, declOk
}

func (f *fieldAnnotatorBuilder) buildSignature(fScope *fileResolveScope) (ir.Func, fnResolveScope, bool) {
	core, fnScope, ok := f.funcMeta.buildSignature(fScope)
	return &ir.AnnotatorField{MetaCore: core}, fnScope, ok
}

type irExpr struct {
	expr ir.Expr
}

func (e *irExpr) source() ast.Node {
	return e.expr.Node()
}

func (e *irExpr) buildExpr(rScope resolveScope) (ir.Expr, bool) {
	return e.expr, true
}

func (e *irExpr) String() string {
	return "irexpr: " + e.expr.String()
}

type funcWithIR interface {
	Func() ir.Func
}

func evalFuncAnnotator(rscope resolveScope, compEval *compileEvaluator, macroCall *callExpr) (*ir.FuncCallExpr, *cpevelements.FuncAnnotator, bool) {
	return evalMetaCallee[*cpevelements.FuncAnnotator](rscope, compEval, macroCall, "to annotate a function")
}

func evalFieldAnnotator(rscope resolveScope, compEval *compileEvaluator, macroCall *callExpr) (*ir.FuncCallExpr, *cpevelements.FieldAnnotator, bool) {
	return evalMetaCallee[*cpevelements.FieldAnnotator](rscope, compEval, macroCall, "to annotate the field of a structure")
}

func evalMacro(rscope resolveScope, compEval *compileEvaluator, macroCall *callExpr) (*ir.FuncCallExpr, *cpevelements.Macro, bool) {
	return evalMetaCallee[*cpevelements.Macro](rscope, compEval, macroCall, "as a macro")
}

func evalMetaCallee[T funcWithIR](rscope resolveScope, compEval *compileEvaluator, macroCall *callExpr, target string) (*ir.FuncCallExpr, T, bool) {
	var zero T
	callee, calleeOk := buildAExpr(rscope, macroCall.callee)
	if !calleeOk {
		return nil, zero, false
	}
	el, err := compEval.fitp.EvalExpr(callee)
	if err != nil {
		return nil, zero, rscope.Err().AppendAt(callee.Node(), err)
	}
	elT, ok := el.(T)
	if !ok {
		return nil, zero, rscope.Err().Appendf(callee.Node(), "cannot use %s %s", callee.String(), target)
	}
	call, ok := macroCall.buildFuncCallExpr(rscope, ir.NewFuncValExpr(
		callee,
		elT.Func(),
	))
	return call, elT, ok
}

func evalCallArgs(rscope resolveScope, compEval *compileEvaluator, call *ir.FuncCallExpr) ([]ir.Element, bool) {
	els := make([]ir.Element, len(call.Args))
	ok := true
	for i, arg := range call.Args {
		var err error
		els[i], err = compEval.EvalExpr(arg)
		if err != nil {
			ok = rscope.Err().AppendAt(arg.Node(), err)
		}
	}
	return els, ok
}

func evalMacroCall(compEval *compileEvaluator, call *ir.FuncCallExpr) (ir.MacroElement, bool) {
	el, err := compEval.fitp.EvalExpr(call)
	if err != nil {
		return nil, compEval.Err().AppendAt(call.Node(), err)
	}
	macroEl, ok := el.(ir.MacroElement)
	if !ok {
		return nil, compEval.Err().AppendInternalf(call.Node(), "unexpected element type %s returned by macro %s", el.Type().String(), call.Callee.Func().ShortString())
	}
	return macroEl, true
}
