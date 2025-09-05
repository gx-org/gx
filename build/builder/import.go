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
	"path/filepath"
	"strconv"

	"github.com/gx-org/gx/build/ir"
)

// ----------------------------------------------------------------------------
// Import statement processing.

type importDecl struct {
	ext ir.ImportDecl

	pkg *basePackage
}

func processImportDecl(pscope procScope, decl *ast.GenDecl) bool {
	ok := true
	for _, spec := range decl.Specs {
		ext := &ir.ImportDecl{Src: spec.(*ast.ImportSpec)}
		var err error
		ext.Path, err = strconv.Unquote(ext.Src.Path.Value)
		if err != nil {
			ok = pscope.Err().Appendf(ext.Src.Path, "malformed path: %s", ext.Src.Path.Value)
		}
		ident := ext.Src.Name
		if ident == nil {
			// The import does not define a name for the package.
			// Use the last element of the import path.
			ident = &ast.Ident{
				NamePos: ext.Src.Path.ValuePos,
				Name:    filepath.Base(ext.Path),
			}
		}
		importOk := pscope.file().declareImports(pscope, ident, ext)
		ok = importOk && ok
	}
	return ok
}

type importedPackage struct {
	pkg   *ir.Package
	names map[string]ir.Storage
}

func importPackage(pkgScope *pkgResolveScope, decl *ir.ImportDecl) (*importedPackage, bool) {
	bPackage, err := pkgScope.pkg().builder().importPath(decl.Path)
	ok := true
	var irPackage *ir.Package
	if err != nil {
		ok = pkgScope.Err().Append(err)
		irPackage = &ir.Package{Decls: &ir.Declarations{}}
	} else {
		irPackage = bPackage.IR()
	}
	decl.Package = irPackage
	imp := &importedPackage{
		pkg:   irPackage,
		names: make(map[string]ir.Storage),
	}
	for fun := range irPackage.ExportedFuncs() {
		imp.names[fun.Name()] = fun
	}
	for _, typ := range irPackage.ExportedTypes() {
		imp.names[typ.Name()] = typ
	}
	for _, cst := range irPackage.ExportedConsts() {
		imp.names[cst.VName.Name] = cst
	}
	return imp, ok
}
