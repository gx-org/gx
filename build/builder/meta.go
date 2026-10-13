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
	"go/token"
	"io"
	"runtime/debug"
	"strings"

	"github.com/pkg/errors"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/src/gx/ast"
	"github.com/gx-org/gx/src/gx/format"
	"github.com/gx-org/gx/src/gx/parser"
)

type coreSyntheticFunc struct {
	bFile *file
	src   *ast.FuncDecl
}

func (f *coreSyntheticFunc) buildBody(fnScope fnResolveScope, fn *irFunc) bool {
	mScope := fnScope.(iSynthResolveScope)
	compEval, ok := fnScope.compEval()
	if !ok {
		return false
	}
	return mScope.buildBody(fn.irFunc.(*ir.FuncDecl), compEval)
}

func (f *coreSyntheticFunc) file() *file {
	return f.bFile
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

func (f *coreSyntheticFunc) buildAnnotations(*fileResolveScope, *irFunc) bool {
	return true
}

type (
	synthResolveScope struct {
		fnResolveScope
		fnBuilder ir.FuncASTBuilder
	}

	iSynthResolveScope interface {
		buildBody(*ir.FuncDecl, *compileEvaluator) bool
	}
)

var unknownIdent = &ast.Ident{Name: "_"}

func formatFunction(fset *token.FileSet, fnBuilder ir.FuncASTBuilder, from *ir.File, orig *ast.FuncDecl) (src string, decl *ast.FuncDecl, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(string(debug.Stack()))
		}
	}()
	w := &strings.Builder{}
	w.WriteString("package _\n")
	srcDecl := *orig
	if srcDecl.Name == nil {
		srcDecl.Name = unknownIdent
	}
	err = format.Node(w, token.NewFileSet(), &srcDecl)
	src = w.String()
	if err != nil { // No error: we are done here.
		fmtMsg := fmt.Sprintf("Formatted:\n%s\nError:\n%v", src, err)
		astW := &strings.Builder{}
		errAST := astVisitBody(srcDecl.Body, astW)
		astMsg := fmt.Sprintf("%s\nVisited:\n%s\nError:\n%v", fmtMsg, astW.String(), errAST)
		err = fmt.Errorf("cannot format synthetic body:\n%s", astMsg)
		return
	}
	fileName := fmt.Sprintf("macro %s", fnBuilder.SourceString(from))
	astFile, err := parser.ParseFile(fset, fileName, w.String(), parser.ParseComments)
	if err != nil {
		err = errors.Errorf("cannot parse the following source code: %v\n%s", err, src)
		return
	}
	decl = astFile.Decls[0].(*ast.FuncDecl)
	decl.Name = orig.Name
	return
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

func buildIRBody(mScope *synthResolveScope, ext *ir.FuncDecl, compEval *compileEvaluator, src string, body *ast.BlockStmt) (ok bool) {
	fScope := mScope.fileScope()
	pkgPScope := fScope.pkgResolveScope.pkgProcScope
	pScope := pkgPScope.newFilePScope(fScope.bFile())
	pScope.Err().Push(fmterr.PrefixWith("cannot compile synthetic body from source:\n%s\n", gxfmt.Number(src)))
	defer pScope.Err().Pop()
	bBody, ok := processBlockStmt(pScope, body)
	if !ok {
		return false
	}
	irBlock, ok := buildFuncBody(mScope.fnResolveScope, bBody)
	if !ok {
		return false
	}
	ext.Body = irBlock
	return true
}

func (m *synthResolveScope) buildBody(fn *ir.FuncDecl, compEval *compileEvaluator) bool {
	// First, build the AST body of the synthetic function.
	var ok bool
	fn.Src.Body, ok = m.fnBuilder.BuildBody(compEval, fn)
	if !ok {
		return false
	}
	if fn.Src.Body == nil {
		return true
	}
	// Format the body to build line numbers.
	var src string
	var err error
	fScope := m.fileScope()
	src, fn.Src, err = formatFunction(fScope.pkg().fset, m.fnBuilder, fScope.irFile(), fn.Src)
	if err != nil {
		return m.Err().Append(err)
	}
	// Build the IR representation of the function body.
	return buildIRBody(m, fn, compEval, src, fn.Src.Body)
}

type syntheticFunc struct {
	coreSyntheticFunc
	underFun  ir.PkgFunc
	fnBuilder ir.FuncASTBuilder
}

func (f *syntheticFunc) buildSignature(pkgScope *pkgResolveScope) (ir.Expr, ir.Func, fnResolveScope, bool) {
	fScope, ok := pkgScope.fileScope(f.bFile)
	if !ok {
		return nil, nil, nil, false
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
		return fScope.Err().Appendf(f.src, "%s requires a %s type receiver", f.src.Name.Name, fOutputRecv.Type().ReferString(fScope.irFile()))
	}
	if fInputRecv != nil && fOutputRecv == nil {
		return fScope.Err().Appendf(f.src, "%s requires no receiver", f.src.Name.Name)
	}
	if ok := equalToAt(fScope, f.src.Recv, fOutputRecv.Type(), fInputRecv.Type()); !ok {
		return fScope.Err().Appendf(f.src, "cannot assign %s.%s to %s.%s", fOutputRecv.Type().NameDef().Name, fOutputName, fInputRecv.Type().NameDef().Name, fInputName)
	}
	return true
}

func buildSyntheticFuncSig(parentScope *fileResolveScope, srcloc ast.Node, astBuilder ir.FuncASTBuilder, target ir.PkgFunc) (ir.Expr, *ir.FuncDecl, *synthResolveScope, bool) {
	macroFile, recvExpr, synDecl, ok := astBuilder.BuildDecl(target)
	if !ok {
		return nil, nil, nil, false
	}
	pMacroScope, ok := newMacroProcScope(parentScope, srcloc, macroFile)
	if !ok {
		return nil, nil, nil, false
	}
	src, synDecl, err := formatFunction(pMacroScope.pkg().fset, astBuilder, parentScope.irFile(), synDecl)
	if err != nil {
		return nil, nil, nil, parentScope.Err().Append(err)
	}
	pMacroScope.Err().Push(fmterr.PrefixWith("cannot compile synthetic signature:\n%s\n", src))
	defer pMacroScope.Err().Pop()
	if !ok {
		return nil, nil, nil, false
	}
	ft, ok := processFuncType(pMacroScope.filePScope(), synDecl.Type, synDecl.Recv, ir.FuncNature{})
	if !ok {
		return nil, nil, nil, false
	}
	fScope, ok := pMacroScope.newResolveScope()
	if !ok {
		return nil, nil, nil, false
	}
	fType, fnScope, ok := ft.buildFuncType(fScope)
	if !ok {
		return nil, nil, nil, false
	}
	return recvExpr,
		&ir.FuncDecl{
			FFile: fScope.irFile(),
			Src:   synDecl,
			FType: fType,
		},
		&synthResolveScope{
			fnResolveScope: fnScope,
			fnBuilder:      astBuilder,
		}, true
}

func (f *syntheticFunc) buildSignatureFScope(fScope *fileResolveScope) (ir.Expr, ir.PkgFunc, *synthResolveScope, bool) {
	recvExpr, synDecl, synScope, ok := buildSyntheticFuncSig(fScope, f.src, f.fnBuilder, f.underFun)
	if !ok {
		return nil, nil, nil, false
	}
	synDecl.Src.Name = f.src.Name
	if ok := f.checkSyntheticSignature(fScope, f.underFun.Name(), f.underFun.FuncType(), synDecl.Src.Name.Name, synDecl.FType); !ok {
		return nil, nil, nil, false
	}
	return recvExpr, synDecl, synScope, ok
}
