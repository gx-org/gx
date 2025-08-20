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
	"go/scanner"
	"go/token"
	"reflect"
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

const annotationPrefix = "gx@="

type astErrorNode struct {
	doc *ast.Comment
	err *scanner.Error
}

var _ ast.Node = (*astErrorNode)(nil)

func (n astErrorNode) Pos() token.Pos {
	return n.doc.Pos() + token.Pos(n.err.Pos.Column) + token.Pos(len(annotationPrefix)) - 1
}

func (n astErrorNode) End() token.Pos {
	return n.doc.End()
}

func processScannerError(pscope procScope, doc *ast.Comment, errScanner error) bool {
	errList, ok := errScanner.(scanner.ErrorList)
	if !ok {
		// Unknown error: we build a new error at the comment position
		// that includes the error type and its message.
		return pscope.Err().Appendf(doc, "%T:%s", errScanner, errScanner.Error())
	}
	for _, err := range errList {
		pscope.Err().Appendf(astErrorNode{doc: doc, err: err}, "%s", err.Msg)
	}
	return false
}

func processFuncAnnotations(pscope procScope, src *ast.FuncDecl, fn function) (function, bool) {
	if src.Doc == nil {
		return fn, true
	}
	for _, doc := range src.Doc.List {
		text := trimCommentPrefix(doc)
		if !strings.HasPrefix(text, annotationPrefix) {
			continue
		}
		text = strings.TrimPrefix(text, annotationPrefix)
		annFN, ok := processFuncAnnotation(pscope, src, fn, doc, text)
		if !ok {
			return fn, false
		}
		fn = annFN
	}
	return fn, true
}

type atFuncExpr struct {
	ann     *assignFuncAnnotation
	irFun   ir.PkgFunc
	fnScope iFuncResolveScope
}

func (at *atFuncExpr) source() ast.Node {
	return at.ann.fn.source()
}

func (at *atFuncExpr) buildExpr(rScope resolveScope) (ir.Expr, bool) {
	return &ir.ValueRef{
		Stor: at.irFun,
	}, true
}

func (at *atFuncExpr) String() string {
	return "@"
}

type assignFuncAnnotation struct {
	coreSyntheticFunc
	fn        function
	macroCall *callExpr
}

func processFuncAnnotation(pscope procScope, src *ast.FuncDecl, fn function, comment *ast.Comment, annotSrc string) (function, bool) {
	f := &assignFuncAnnotation{
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

func (f *assignFuncAnnotation) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
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
	atExpr := &atFuncExpr{
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
	under *atFuncExpr
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
