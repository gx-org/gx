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
	"reflect"

	"github.com/gx-org/gx/build/ir"
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

func (f *funcMeta) buildBody(iFuncResolveScope, *irFunc) bool {
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

func (f *funcMeta) buildSignature(pkgScope *pkgResolveScope) (ir.MetaCore, iFuncResolveScope, bool) {
	fScope, scopeOk := pkgScope.newFileRScope(f.bFile)
	if !scopeOk {
		return ir.MetaCore{}, nil, false
	}
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

func (f *funcMacro) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
	core, fnScope, ok := f.funcMeta.buildSignature(pkgScope)
	return &ir.Macro{MetaCore: core}, fnScope, ok
}

type funcAnnotator struct {
	funcMeta
}

func (bFile *file) processAnnotatorFunc(scope procScope, src *ast.FuncDecl, comment *ast.Comment) (function, bool) {
	fDecl, declOk := newFuncDecl(scope, src, false)
	fn := &funcAnnotator{funcMeta{funcDecl: fDecl}}
	return fn, declOk
}

func (f *funcAnnotator) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
	core, fnScope, ok := f.funcMeta.buildSignature(pkgScope)
	return &ir.Annotator{MetaCore: core}, fnScope, ok
}

type irExpr struct {
	expr ir.Expr
}

func (e *irExpr) source() ast.Node {
	return e.expr.Source()
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

func evalMetaCallee[T funcWithIR](rscope resolveScope, compEval *compileEvaluator, macroCall *callExpr) (*ir.CallExpr, T, bool) {
	var zero T
	callee, calleeOk := buildAExpr(rscope, macroCall.callee)
	if !calleeOk {
		return nil, zero, false
	}
	el, err := compEval.fitp.EvalExpr(callee)
	if err != nil {
		return nil, zero, rscope.Err().AppendAt(callee.Source(), err)
	}
	elT, ok := el.(T)
	if !ok {
		return nil, zero, rscope.Err().Appendf(callee.Source(), "cannot use %s as %s", callee.String(), reflect.TypeFor[T]())
	}
	call, ok := macroCall.buildFuncCallExpr(rscope, callee, elT.Func())
	return call, elT, ok
}

func evalMacroCall(compEval *compileEvaluator, call *ir.CallExpr) (ir.MacroElement, bool) {
	el, err := compEval.fitp.EvalExpr(call)
	if err != nil {
		return nil, compEval.Err().AppendAt(call.Source(), err)
	}
	macroEl, ok := el.(ir.MacroElement)
	if !ok {
		return nil, compEval.Err().AppendInternalf(call.Source(), "unexpected element type %s returned by macro %s", el.Type().String(), call.Callee.Func().ShortString())
	}
	return macroEl, true
}
