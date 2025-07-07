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
	"go/parser"
	"go/scanner"
	"go/token"
	"reflect"
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp"
)

type directive int

const (
	none directive = iota
	assign
	irmacro
	cpeval
)

const assignDirectivePrefix = "//gx:="

type directivePrefix struct {
	dir        directive
	prefixOnly bool
}

var directives = map[string]directivePrefix{
	assignDirectivePrefix: directivePrefix{dir: assign, prefixOnly: false},
	"//gx:irmacro":        directivePrefix{dir: irmacro, prefixOnly: true},
	"//gx:compeval":       directivePrefix{dir: cpeval, prefixOnly: true},
}

type astErrorNode struct {
	doc *ast.Comment
	err *scanner.Error
}

var _ ast.Node = (*astErrorNode)(nil)

func (n astErrorNode) Pos() token.Pos {
	return n.doc.Pos() + token.Pos(n.err.Pos.Column) + token.Pos(len(assignDirectivePrefix)) - 1
}

func (n astErrorNode) End() token.Pos {
	return n.doc.End()
}

func processScannerError(pscope procScope, doc *ast.Comment, errScanner error) bool {
	errList, ok := errScanner.(scanner.ErrorList)
	if !ok {
		// Unknown error: we build a new error at the comment position
		// that includes the error type and its message.
		return pscope.err().Appendf(doc, "%T:%s", errScanner, errScanner.Error())
	}
	for _, err := range errList {
		pscope.err().Appendf(astErrorNode{doc: doc, err: err}, "%s", err.Msg)
	}
	return false
}

func directiveFromComment(doc string) directive {
	for prefix, dir := range directives {
		if !strings.HasPrefix(doc, prefix) {
			continue
		}
		if !dir.prefixOnly {
			return dir.dir
		}
		doc = strings.TrimSpace(doc)
		if prefix == doc {
			return dir.dir
		}
	}
	return none
}

func processFuncDirective(pscope procScope, fn *ast.FuncDecl) (directive, *ast.Comment, bool) {
	if fn.Doc == nil {
		return none, nil, true
	}
	var dir directive
	var comment *ast.Comment
	for _, doc := range fn.Doc.List {
		docDir := directiveFromComment(doc.Text)
		if docDir == none {
			continue
		}
		if dir != none {
			return dir, doc, pscope.err().Appendf(doc, "a function can only have one GX directive")
		}
		dir = docDir
		comment = doc
	}
	return dir, comment, true
}

type syntheticFunc struct {
	bFile *file
	src   *ast.FuncDecl
	macro *callExpr
}

func (bFile *file) processSyntheticFunc(pscope procScope, src *ast.FuncDecl, comment *ast.Comment) bool {
	f := &syntheticFunc{
		bFile: pscope.file(),
		src:   src,
	}
	ok := true
	if _, regOk := pscope.decls().registerFunc(f); !regOk {
		ok = false
	}
	if !checkEmptyParamsResults(pscope, src, "assigned") {
		ok = false
	}
	if !ok {
		return false
	}
	return f.processFuncAssignDirective(pscope, comment)
}

func (f *syntheticFunc) processFuncAssignDirective(pscope procScope, doc *ast.Comment) bool {
	src := strings.TrimPrefix(doc.Text, assignDirectivePrefix)
	astExpr, err := parser.ParseExprFrom(pscope.pkgScope().pkg().fset, f.bFile.name+":"+f.name().Name, src, parser.SkipObjectResolution)
	if err != nil {
		return processScannerError(pscope, doc, err)
	}
	if f.macro != nil {
		return pscope.err().Appendf(doc, "a function can only have one GX directive")
	}
	expr, ok := processExpr(pscope, astExpr)
	if !ok {
		return false
	}
	f.macro, ok = expr.(*callExpr)
	if !ok {
		return pscope.err().Appendf(doc, "GX equal directive (gx:=) only accept function call expression")
	}
	return true
}

func (f *syntheticFunc) source() ast.Node {
	return f.src
}

func (f *syntheticFunc) name() *ast.Ident {
	return f.src.Name
}

func (f *syntheticFunc) isMethod() bool {
	return false
}

func (f *syntheticFunc) compEval() bool {
	return false
}

type macroResolveScope struct {
	iFuncResolveScope
	sFunc *cpevelements.SyntheticFunc
}

func (f *syntheticFunc) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
	fScope, ok := pkgScope.newFileScope(f.bFile)
	if !ok {
		return nil, nil, false
	}
	callExpr, ok := f.macro.buildExpr(fScope)
	if !ok {
		return nil, nil, false
	}
	fCallExpr, ok := callExpr.(*ir.CallExpr)
	if !ok {
		return nil, nil, fScope.err().Appendf(f.macro.source(), "expect a function")
	}
	if _, ok := fCallExpr.Callee.F.(*ir.Macro); !ok {
		return nil, nil, fScope.err().Appendf(f.macro.source(), "cannot use %s as a macro", fCallExpr.Callee.F.NameDef().Name)
	}
	compEval, compEvalOk := fScope.compEval()
	if !compEvalOk {
		return nil, nil, false
	}
	macro, err := interp.EvalExprInContext(compEval.ev, fCallExpr)
	if err != nil {
		return nil, nil, fScope.err().AppendAt(f.macro.source(), err)
	}
	ext := &ir.FuncDecl{Src: f.src, FFile: fScope.irFile()}
	sFunc, ok := macro.(*cpevelements.SyntheticFunc)
	if !ok {
		return nil, nil, fScope.err().AppendInternalf(f.macro.source(), "cannot convert %T to %s", macro, reflect.TypeFor[*cpevelements.SyntheticFunc]())
	}
	ext.FType, err = sFunc.Builder().BuildType()
	if err != nil {
		return ext, nil, fScope.err().AppendAt(f.macro.source(), err)
	}
	return ext, &macroResolveScope{
		iFuncResolveScope: newFuncScope(fScope, ext.FType),
		sFunc:             sFunc,
	}, ok
}

func (f *syntheticFunc) buildBody(fScope iFuncResolveScope, extF ir.Func) ([]*cpevelements.SyntheticFuncDecl, bool) {
	mScope := fScope.(*macroResolveScope)
	compEval, ok := fScope.compEval()
	if !ok {
		return nil, false
	}
	ext := extF.(*ir.FuncDecl)
	var aux []*cpevelements.SyntheticFuncDecl
	ext.Body, aux, ok = mScope.sFunc.Builder().BuildBody(compEval)
	return aux, ok
}

func (f *syntheticFunc) resolveOrder() int {
	// Synthetic functions need to be resolved last.
	return 1
}
