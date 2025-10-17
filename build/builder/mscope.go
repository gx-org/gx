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
	"go/token"

	"github.com/gx-org/gx/build/ir"
)

type macroProcScope struct {
	*pkgProcScope
	file   *file
	rscope *pkgResolveScope
}

func copyImportDecls(tgt *ast.File, src *ast.File) {
	for _, decl := range src.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if genDecl.Tok != token.IMPORT {
			continue
		}
		tgt.Decls = append(tgt.Decls, genDecl)
	}
}

func ephemeralPkgScope(fscope *fileResolveScope, src ast.Node, pkgPath string) (*pkgResolveScope, bool) {
	pkg, err := fscope.pkg().bld.importPath(pkgPath)
	if err != nil {
		return nil, fscope.Err().AppendInternalf(src, "cannot import %s: %v", pkgPath, err)
	}
	base := pkg.base()
	pscope := &pkgProcScope{
		pkgScope: pkgScope{
			bpkg: base,
			errs: fscope.Err().Errors().NewAppender(base.fset),
		},
		dcls: base.last.builderState.dcls,
	}
	return &pkgResolveScope{
		pkgProcScope: newPackageProcScope(false, pkg.base(), pscope.Err().Errors()),
		state:        pkg.base().last.builderState,
	}, true
}

func newMacroProcScope(pScope *fileResolveScope, src ast.Node, macroFile *ir.File) (*macroProcScope, bool) {
	currentPackagePath := pScope.irBuilder().Pkg().FullName()
	pkgPath := macroFile.Package.FullName()
	var rscope *pkgResolveScope
	if currentPackagePath == pkgPath {
		rscope = pScope.pkgResolveScope
	} else {
		var ok bool
		rscope, ok = ephemeralPkgScope(pScope, src, pkgPath)
		if !ok {
			return nil, false
		}
	}
	pscope := rscope.pkgProcScope
	base := pscope.pkg().base()
	astFile := &ast.File{
		Name:    macroFile.Src.Name,
		Imports: macroFile.Src.Imports,
	}
	copyImportDecls(astFile, macroFile.Src)
	bFile, ok := processFile(pscope, base.macroRoot.Next().String(), astFile)
	if !ok {
		return nil, false
	}
	return &macroProcScope{
		pkgProcScope: pscope,
		file:         bFile,
		rscope:       rscope,
	}, true
}

func (m *macroProcScope) filePScope() *fileProcScope {
	return m.newFilePScope(m.file)
}

func (m *macroProcScope) newResolveScope() (*fileResolveScope, bool) {
	return m.rscope.newFileRScope(m.file)
}
