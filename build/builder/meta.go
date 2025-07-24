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
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
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

type coreSyntheticFunc struct {
	bFile *file
	src   *ast.FuncDecl
}

func (f *coreSyntheticFunc) buildIRBody(mScope *macroResolveScope, extF ir.Func, compEval *compileEvaluator, body *ast.BlockStmt) (ok bool) {
	defer func() {
		if !ok {
			mScope.err().Append(mScope.errorFor(body))
		}
	}()
	fScope := mScope.fileScope()
	pkgPScope := fScope.pkgResolveScope.pkgProcScope
	pScope := pkgPScope.newScope(fScope.bFile())
	bBody, ok := processBlockStmt(pScope, body)
	if !ok {
		return false
	}
	irBlock, ok := bBody.buildBlockStmt(mScope.iFuncResolveScope)
	if !ok {
		return false
	}
	ext := extF.(*ir.FuncDecl)
	ext.Body = irBlock
	return ok
}

func (f *coreSyntheticFunc) buildBody(fnScope iFuncResolveScope, fn *irFunc) ([]*irFunc, bool) {
	mScope := fnScope.(*macroResolveScope)
	compEval, ok := fnScope.compEval()
	if !ok {
		return nil, false
	}
	// First, build the AST body of the synthetic function.
	astBody, auxs, ok := mScope.fnBuilder.Builder().BuildBody(compEval)
	if !ok {
		return nil, false
	}
	// Register all the auxiliary functions, so that they are available to the builder.
	pkgScope := fnScope.fileScope().pkgResolveScope
	irAuxs, ok := pkgScope.dcls.registerAuxFuncs(mScope, auxs)
	if !ok {
		return nil, false
	}
	// Build the IR representation of the function body.
	return irAuxs, f.buildIRBody(mScope, fn.irFunc, compEval, astBody)
}

func (f *coreSyntheticFunc) resolveOrder() int {
	// Synthetic functions need to be resolved last.
	return 1
}
func (f *coreSyntheticFunc) source() ast.Node {
	return f.src
}

func (f *coreSyntheticFunc) name() *ast.Ident {
	return f.src.Name
}

func (f *coreSyntheticFunc) isMethod() bool {
	return false
}

func (f *coreSyntheticFunc) compEval() bool {
	return false
}

type assignFuncDirective struct {
	coreSyntheticFunc
	macro *callExpr
}

func (bFile *file) processSyntheticFunc(pscope procScope, src *ast.FuncDecl, comment *ast.Comment) bool {
	f := &assignFuncDirective{
		coreSyntheticFunc: coreSyntheticFunc{
			bFile: pscope.file(),
			src:   src,
		},
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

func (f *assignFuncDirective) processFuncAssignDirective(pscope procScope, doc *ast.Comment) bool {
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

func (f *assignFuncDirective) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
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
	macro, err := compEval.fitp.EvalExpr(fCallExpr)
	if err != nil {
		return nil, nil, fScope.err().AppendAt(f.macro.source(), err)
	}
	sFunc, ok := macro.(*cpevelements.SyntheticFunc)
	if !ok {
		return nil, nil, fScope.err().AppendInternalf(f.macro.source(), "cannot convert %T to %s", macro, reflect.TypeFor[*cpevelements.SyntheticFunc]())
	}
	return (&syntheticFunc{
		coreSyntheticFunc: f.coreSyntheticFunc,
		fnBuilder:         sFunc,
	}).buildSignatureFScope(fScope)
}

type macroResolveScope struct {
	iFuncResolveScope
	fnBuilder *cpevelements.SyntheticFunc
}

func formatBody(body *ast.BlockStmt, w io.Writer) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(string(debug.Stack()))
		}
	}()
	fset := token.NewFileSet()
	return format.Node(w, fset, body)
}

func astVisitBody(body *ast.BlockStmt, w io.Writer) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(string(debug.Stack()))
		}
	}()
	fset := token.NewFileSet()
	return ast.Fprint(w, fset, body, nil)
}

func (m *macroResolveScope) errorFor(body *ast.BlockStmt) error {
	const coreError = "cannot generate IR from synthetic function body:\n%s"
	fmtW := &strings.Builder{}
	errFmt := formatBody(body, fmtW)
	if errFmt == nil {
		return fmt.Errorf(coreError, fmtW.String())
	}
	fmtMsg := fmt.Sprintf("Formatted:\n%s\nError:\n%v", fmtW.String(), errFmt)
	astW := &strings.Builder{}
	errAST := astVisitBody(body, astW)
	astMsg := fmt.Sprintf("%s\nVisited:\n%s\nError:\n%v", fmtMsg, astW.String(), errAST)
	return fmt.Errorf(coreError, astMsg)
}

func buildFuncTypeFromAST(fScope *fileResolveScope, src *ast.FuncDecl) (*ir.FuncDecl, *funcResolveScope, bool) {
	pkgPScope := fScope.pkgProcScope
	pScope := pkgPScope.newScope(fScope.bFile())
	ft, ok := processFuncType(pScope, src.Type, src.Recv, false)
	if !ok {
		return nil, nil, false
	}
	fType, fnScope, ok := ft.buildFuncType(fScope)
	if !ok {
		return nil, nil, false
	}
	return &ir.FuncDecl{Src: src, FFile: fScope.irFile(), FType: fType}, fnScope, true
}

type syntheticFunc struct {
	coreSyntheticFunc
	fnBuilder *cpevelements.SyntheticFunc
}

func (f *syntheticFunc) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
	fScope, ok := pkgScope.newFileScope(f.bFile)
	if !ok {
		return nil, nil, false
	}
	return f.buildSignatureFScope(fScope)
}

func (f *syntheticFunc) checkSyntheticSignature(fScope *fileResolveScope, fSynth *ir.FuncDecl) bool {
	fSrc, _, ok := buildFuncTypeFromAST(fScope, f.src)
	if !ok {
		return false
	}
	fSrcRecv := fSrc.FType.ReceiverField()
	fSynthRecv := fSynth.FType.ReceiverField()
	if fSrcRecv == nil && fSynthRecv == nil {
		return true
	}
	if fSrcRecv == nil && fSynthRecv != nil {
		return fScope.err().Appendf(f.src, "%s requires %s receiver", f.src.Name.Name, fSynthRecv.Type().String())
	}
	if fSrcRecv != nil && fSynthRecv == nil {
		return fScope.err().Appendf(f.src, "%s requires no receiver", f.src.Name.Name)
	}
	if ok := equalToAt(fScope, f.src.Recv, fSrcRecv.Type(), fSynthRecv.Type()); !ok {
		return fScope.err().Appendf(f.src, "cannot assign %s.%s to %s.%s", fSynthRecv.Type().NameDef().Name, fSynth.Src.Name.Name, fSrcRecv.Type().NameDef().Name, fSrc.Src.Name.Name)
	}
	return true
}

func (f *syntheticFunc) buildSignatureFScope(fScope *fileResolveScope) (ir.Func, iFuncResolveScope, bool) {
	astFDecl, err := f.fnBuilder.Builder().BuildType()
	if err != nil {
		return nil, nil, fScope.err().AppendAt(f.src, err)
	}
	astFDecl.Name = f.src.Name
	fDecl, fnScope, ok := buildFuncTypeFromAST(fScope, astFDecl)
	if !ok {
		return nil, nil, false
	}
	if ok := f.checkSyntheticSignature(fScope, fDecl); !ok {
		return nil, nil, false
	}
	return fDecl, &macroResolveScope{
		iFuncResolveScope: fnScope,
		fnBuilder:         f.fnBuilder,
	}, ok
}
