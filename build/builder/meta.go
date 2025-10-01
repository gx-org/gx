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
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"runtime/debug"
	"strings"

	"github.com/pkg/errors"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

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

func (f *coreSyntheticFunc) fnSource() *ast.FuncDecl {
	return f.src
}

func (f *coreSyntheticFunc) isMethod() bool {
	return false
}

func (f *coreSyntheticFunc) compEval() bool {
	return false
}

func (f *coreSyntheticFunc) buildAnnotations(fScope iFuncResolveScope, extF *irFunc) bool {
	return true
}

type (
	synthResolveScope struct {
		iFuncResolveScope
		fnBuilder cpevelements.FuncASTBuilder
	}

	iSynthResolveScope interface {
		buildBody(*irFunc, *compileEvaluator) ([]*irFunc, bool)
	}
)

func formatBody(fType *ir.FuncType, body *ast.BlockStmt, w *strings.Builder) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(string(debug.Stack()))
		}
	}()
	fset := token.NewFileSet()
	err = format.Node(w, fset, &ast.FuncDecl{
		Type: fType.Src,
		Name: &ast.Ident{Name: "_"},
		Body: body,
	})
	if err == nil { // No error: we are done here.
		return
	}
	fmtMsg := fmt.Sprintf("Formatted:\n%s\nError:\n%v", w.String(), err)
	astW := &strings.Builder{}
	errAST := astVisitBody(body, astW)
	astMsg := fmt.Sprintf("%s\nVisited:\n%s\nError:\n%v", fmtMsg, astW.String(), errAST)
	return fmt.Errorf("cannot format synthetic body:\n%s", astMsg)
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

func buildIRBody(mScope *synthResolveScope, extF ir.Func, compEval *compileEvaluator, src string, body *ast.BlockStmt) (ok bool) {
	defer func() {
		if !ok {
			mScope.Err().Append(errors.Errorf("cannot compile synthetic body from source:\n%s", gxfmt.Number(src)))
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

func (m *synthResolveScope) buildBody(fn *irFunc, compEval *compileEvaluator) ([]*irFunc, bool) {
	// First, build the AST body of the synthetic function.
	astBody, auxs, ok := m.fnBuilder.BuildBody(compEval, fn.irFunc)
	if !ok {
		return nil, false
	}
	if astBody == nil {
		return nil, true
	}
	// Register all the auxiliary functions, so that they are available to the builder.
	pkgScope := m.fileScope().pkgResolveScope
	irAuxs, ok := pkgScope.dcls.registerAuxFuncs(m, auxs)
	if !ok {
		return nil, false
	}
	// Format the body to build line numbers.
	src := &strings.Builder{}
	src.WriteString("package _\n")
	err := formatBody(m.funcType(), astBody, src)
	if err != nil {
		return nil, m.Err().Append(err)
	}
	fs := token.NewFileSet()
	astFile, err := parser.ParseFile(fs, "", src.String(), parser.ParseComments)
	if err != nil {
		return nil, m.Err().Append(errors.Errorf("cannot parse formatted file: %v\nSource:\n%s", err, src.String()))
	}
	astBody = astFile.Decls[0].(*ast.FuncDecl).Body
	// Build the IR representation of the function body.
	return irAuxs, buildIRBody(m, fn.irFunc, compEval, src.String(), astBody)
}

type syntheticFunc struct {
	coreSyntheticFunc
	underFun  ir.PkgFunc
	fnBuilder cpevelements.FuncASTBuilder
}

func (f *syntheticFunc) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
	fScope, ok := pkgScope.newFileScope(f.bFile)
	if !ok {
		return nil, nil, false
	}
	return f.buildSignatureFScope(fScope)
}

func (f *syntheticFunc) checkSyntheticSignature(fScope *fileResolveScope, fInputName string, fInputType *ir.FuncType, fOutputName string, fOutputType *ir.FuncType) bool {
	fOutputRecv := fOutputType.ReceiverField()
	fInputRecv := fInputType.ReceiverField()
	if fOutputRecv == nil && fInputRecv == nil {
		return true
	}
	if fInputRecv == nil && fOutputRecv != nil {
		return fScope.Err().Appendf(f.src, "%s requires a %s type receiver", f.src.Name.Name, fOutputRecv.Type().String())
	}
	if fInputRecv != nil && fOutputRecv == nil {
		return fScope.Err().Appendf(f.src, "%s requires no receiver", f.src.Name.Name)
	}
	if ok := equalToAt(fScope, f.src.Recv, fOutputRecv.Type(), fInputRecv.Type()); !ok {
		return fScope.Err().Appendf(f.src, "cannot assign %s.%s to %s.%s", fOutputRecv.Type().NameDef().Name, fOutputName, fInputRecv.Type().NameDef().Name, fInputName)
	}
	return true
}

func (f *syntheticFunc) buildSignatureFScope(fScope *fileResolveScope) (ir.PkgFunc, *synthResolveScope, bool) {
	synDecl, ok := f.fnBuilder.BuildDecl(f.underFun)
	if !ok {
		return nil, nil, false
	}
	synDecl.Name = f.src.Name
	pkgPScope := fScope.pkgProcScope
	pScope := pkgPScope.newScope(fScope.bFile())
	ft, ok := processFuncType(pScope, synDecl.Type, synDecl.Recv, false)
	if !ok {
		return nil, nil, false
	}
	fnSyntheticType, fnScope, ok := ft.buildFuncType(fScope)
	if !ok {
		return nil, nil, false
	}
	if ok := f.checkSyntheticSignature(fScope, f.underFun.Name(), f.underFun.FuncType(), synDecl.Name.Name, fnSyntheticType); !ok {
		return nil, nil, false
	}
	return &ir.FuncDecl{
			FFile: fScope.irFile(),
			Src:   synDecl,
			FType: fnSyntheticType,
		}, &synthResolveScope{
			iFuncResolveScope: fnScope,
			fnBuilder:         f.fnBuilder,
		}, ok
}
