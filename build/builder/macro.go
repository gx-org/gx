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

type funcMacro struct {
	*funcDecl
}

func (bFile *file) processIRMacroFunc(scope procScope, src *ast.FuncDecl, comment *ast.Comment) (function, bool) {
	fDecl, declOk := newFuncDecl(scope, src, false)
	fn := &funcMacro{funcDecl: fDecl}
	return fn, declOk
}

func (f *funcMacro) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
	fScope, scopeOk := pkgScope.newFileScope(f.bFile)
	if !scopeOk {
		return nil, nil, false
	}
	ext := &ir.Macro{
		Src:   f.src,
		FFile: fScope.irFile(),
	}
	var ok bool
	var fnScope *funcResolveScope
	ext.FType, fnScope, ok = f.fType.buildFuncType(fScope)
	return ext, fnScope, ok
}

func (f *funcMacro) source() ast.Node {
	return f.src
}

func (f *funcMacro) name() *ast.Ident {
	return f.src.Name
}

func (f *funcMacro) buildBody(iFuncResolveScope, *irFunc) bool {
	return true
}

func (f *funcMacro) receiver() *fieldList {
	return nil
}

func (f *funcMacro) compEval() bool {
	return true
}

func (f *funcMacro) resolveOrder() int {
	// Macro needs to be resolved first before any other functions.
	return -1
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

func evalMacroCallee(rscope resolveScope, compEval *compileEvaluator, macroCall *callExpr) (*ir.CallExpr, bool) {
	callee, calleeOk := buildAExpr(rscope, macroCall.callee)
	if !calleeOk {
		return nil, false
	}
	el, err := compEval.fitp.EvalExpr(callee)
	if err != nil {
		return nil, rscope.Err().AppendAt(callee.Source(), err)
	}
	elT, ok := el.(*cpevelements.Macro)
	if !ok {
		return nil, rscope.Err().Appendf(callee.Source(), "cannot use %s as macro", callee.String())
	}
	return macroCall.buildFuncCallExpr(rscope, callee, elT.IR())
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
