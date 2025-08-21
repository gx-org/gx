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
	"go/parser"
	"reflect"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

const assignPrefix = "gx:="

type assignFuncExpr struct {
	ann     *assignFuncFromMacro
	irFun   ir.PkgFunc
	fnScope iFuncResolveScope
}

func (at *assignFuncExpr) source() ast.Node {
	return at.ann.fn.source()
}

func (at *assignFuncExpr) buildExpr(rScope resolveScope) (ir.Expr, bool) {
	return &ir.ValueRef{
		Stor: at.irFun,
	}, true
}

func (at *assignFuncExpr) String() string {
	return assignPrefix
}

type assignFuncFromMacro struct {
	coreSyntheticFunc
	fn        function
	macroCall *callExpr
}

func processFuncAssignment(pscope procScope, src *ast.FuncDecl, fn function, comment *ast.Comment, annotSrc string) (function, bool) {
	f := &assignFuncFromMacro{
		fn: fn,
		coreSyntheticFunc: coreSyntheticFunc{
			bFile: pscope.file(),
			src:   src,
		},
	}
	astExpr, err := parser.ParseExprFrom(pscope.pkgScope().pkg().fset, f.bFile.name+":"+fnName(f), annotSrc, parser.SkipObjectResolution)
	if err != nil {
		return nil, processScannerError(pscope, comment, err)
	}
	expr, ok := processExpr(pscope, astExpr)
	if !ok {
		return nil, false
	}
	f.macroCall, ok = expr.(*callExpr)
	if !ok {
		return nil, pscope.Err().Appendf(comment, "GX equal directive (gx@=) only accept function call expression")
	}
	return f, true
}

func (f *assignFuncFromMacro) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
	fScope, ok := pkgScope.newFileScope(f.bFile)
	if !ok {
		return nil, nil, false
	}
	// Build the signature of the underlying function.
	underFun, underScope, ok := f.fn.buildSignature(pkgScope)
	if !ok {
		return nil, nil, false
	}
	underPkgFun, ok := underFun.(ir.PkgFunc)
	if !ok {
		return nil, nil, fScope.Err().AppendInternalf(f.fn.source(), "%T not a package function", underFun)
	}
	atExpr := &assignFuncExpr{
		ann:     f,
		irFun:   underPkgFun,
		fnScope: underScope,
	}
	// Build the IR to call the macro.
	callExpr, ok := f.macroCall.injectArg(atExpr).buildExpr(fScope)
	if !ok {
		return nil, nil, false
	}
	fCallExpr, ok := callExpr.(*ir.CallExpr)
	if !ok {
		return nil, nil, fScope.Err().Appendf(f.macroCall.source(), "expect a function")
	}
	if _, ok := fCallExpr.Callee.F.(*ir.Macro); !ok {
		return nil, nil, fScope.Err().Appendf(f.macroCall.source(), "cannot use %s as a macro", fCallExpr.Callee.F.NameDef().Name)
	}
	// Evaluate the macro expression.
	compEval, compEvalOk := fScope.compEval()
	if !compEvalOk {
		return nil, nil, false
	}
	macro, err := compEval.fitp.EvalExpr(fCallExpr)
	if err != nil {
		return nil, nil, fScope.Err().AppendAt(f.macroCall.source(), err)
	}
	// Return the result as a synthetic function.
	sFunc, ok := macro.(*cpevelements.SyntheticFunc)
	if !ok {
		return nil, nil, fScope.Err().AppendInternalf(f.macroCall.source(), "cannot convert %T to %s", macro, reflect.TypeFor[*cpevelements.SyntheticFunc]())
	}
	synthFun, synthScope, ok := (&syntheticFunc{
		coreSyntheticFunc: f.coreSyntheticFunc,
		fnBuilder:         sFunc,
	}).buildSignatureFScope(fScope)
	if !ok {
		return nil, nil, false
	}
	return synthFun, &macroResolveScope{
		synthResolveScope: synthScope,
		under:             atExpr,
	}, true
}

type macroResolveScope struct {
	*synthResolveScope
	under *assignFuncExpr
}

func (m *macroResolveScope) buildBody(fn *irFunc, compEval *compileEvaluator) ([]*irFunc, bool) {
	underAuxs, ok := m.under.ann.fn.buildBody(m.under.fnScope, &irFunc{
		bFunc:     m.under.ann.fn,
		scopeFunc: m,
		irFunc:    m.under.irFun,
	})
	if !ok {
		return nil, false
	}
	for _, ann := range m.under.irFun.Annotations().Anns {
		fn.irFunc.Annotations().AppendAnn(ann)
	}
	auxs, ok := m.synthResolveScope.buildBody(fn, compEval)
	if !ok {
		return nil, false
	}
	return append(underAuxs, auxs...), ok
}
