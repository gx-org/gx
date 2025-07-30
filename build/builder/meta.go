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
	"go/token"
	"io"
	"runtime/debug"
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

func buildIRBody(mScope *synthResolveScope, extF ir.Func, compEval *compileEvaluator, body *ast.BlockStmt) (ok bool) {
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

type coreSyntheticFunc struct {
	bFile *file
	src   *ast.FuncDecl
}

func (f *coreSyntheticFunc) buildBody(fnScope iFuncResolveScope, fn *irFunc) ([]*irFunc, bool) {
	mScope := fnScope.(iSynthResolveScope)
	compEval, ok := fnScope.compEval()
	if !ok {
		return nil, false
	}
	return mScope.buildBody(fn, compEval)
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

type (
	synthResolveScope struct {
		iFuncResolveScope
		fnBuilder *cpevelements.SyntheticFunc
	}

	iSynthResolveScope interface {
		buildBody(*irFunc, *compileEvaluator) ([]*irFunc, bool)
	}
)

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

func (m *synthResolveScope) errorFor(body *ast.BlockStmt) error {
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

func (m *synthResolveScope) buildBody(fn *irFunc, compEval *compileEvaluator) ([]*irFunc, bool) {
	// First, build the AST body of the synthetic function.
	astBody, auxs, ok := m.fnBuilder.Builder().BuildBody(compEval)
	if !ok {
		return nil, false
	}
	// Register all the auxiliary functions, so that they are available to the builder.
	pkgScope := m.fileScope().pkgResolveScope
	irAuxs, ok := pkgScope.dcls.registerAuxFuncs(m, auxs)
	if !ok {
		return nil, false
	}
	// Build the IR representation of the function body.
	if ok := buildIRBody(m, fn.irFunc, compEval, astBody); !ok {
		return nil, false
	}
	return irAuxs, m.fnBuilder.Builder().BuildAnnotations(compEval, fn.irFunc)
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
		return fScope.err().Appendf(f.src, "%s requires a %s type receiver", f.src.Name.Name, fSynthRecv.Type().String())
	}
	if fSrcRecv != nil && fSynthRecv == nil {
		return fScope.err().Appendf(f.src, "%s requires no receiver", f.src.Name.Name)
	}
	if ok := equalToAt(fScope, f.src.Recv, fSrcRecv.Type(), fSynthRecv.Type()); !ok {
		return fScope.err().Appendf(f.src, "cannot assign %s.%s to %s.%s", fSynthRecv.Type().NameDef().Name, fSynth.Src.Name.Name, fSrcRecv.Type().NameDef().Name, fSrc.Src.Name.Name)
	}
	return true
}

func (f *syntheticFunc) buildSignatureFScope(fScope *fileResolveScope) (ir.Func, *synthResolveScope, bool) {
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
	return fDecl, &synthResolveScope{
		iFuncResolveScope: fnScope,
		fnBuilder:         f.fnBuilder,
	}, ok
}
