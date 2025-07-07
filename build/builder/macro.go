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

func (bFile *file) processIRMacroFunc(scope procScope, src *ast.FuncDecl, comment *ast.Comment) bool {
	fDecl, fDeclOk := newFuncDecl(scope, src, false)
	fn := &funcMacro{
		funcDecl: fDecl,
	}
	returnOk := fn.funcDecl.checkReturnValue(scope)
	_, declareOk := scope.decls().registerFunc(fn)
	return fDeclOk && returnOk && declareOk
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

func (f *funcMacro) buildBody(fScope iFuncResolveScope, extF ir.Func) ([]*cpevelements.SyntheticFuncDecl, bool) {
	return nil, true
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
