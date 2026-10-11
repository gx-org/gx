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
	"path/filepath"
	"strconv"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/src/gx/ast"
)

// ----------------------------------------------------------------------------
// Import statement processing.

type importDecl struct {
	ext ir.ImportDecl

	pkg *basePackage
}

func processImportSpec(pscope procScope, spec *ast.ImportSpec) bool {
	ext := &ir.ImportDecl{Src: spec}
	var err error
	ext.Path, err = strconv.Unquote(ext.Src.Path.Value)
	if err != nil {
		return pscope.Err().Appendf(ext.Src.Path, "malformed path: %s", ext.Src.Path.Value)
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
	return pscope.file().declareImports(pscope, ident, ext)
}

func processImportDecl(pscope procScope, decl *ast.GenDecl) bool {
	ok := true
	for _, spec := range decl.Specs {
		importOk := processImportSpec(pscope, spec.(*ast.ImportSpec))
		ok = importOk && ok
	}
	return ok
}

type importedPackage struct {
	pkg  *ir.Package
	defs map[string]ir.Storage
}

func importPackage(pkgScope *pkgResolveScope, decl *ir.ImportDecl) (*importedPackage, bool) {
	bPackage, err := pkgScope.pkg().builder().loadPath(decl.Path)
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
		pkg: irPackage,
	}
	if irPackage.Decls == nil && irPackage.Runtime == nil {
		return imp, pkgScope.Err().Appendf(decl.Node(), "package %s has no declarations and no runtime", irPackage.Path())
	}
	if irPackage.Decls == nil {
		return imp, ok
	}
	imp.defs = make(map[string]ir.Storage)
	for fun := range irPackage.ExportedFuncs() {
		imp.defs[fun.Name()] = fun
	}
	for _, typ := range irPackage.ExportedTypes() {
		imp.defs[typ.Name()] = typ
	}
	for _, cst := range irPackage.ExportedConsts() {
		imp.defs[cst.VName.Name] = cst
	}
	return imp, ok
}

func (ip *importedPackage) resolve(scope resolveScope, sel *ir.SelectorExpr) (ir.Storage, bool) {
	name := sel.Src.Sel
	if ip.pkg.Runtime != nil {
		return &ir.LocalVarStorage{
			Src: sel.Src.Sel,
			Typ: ir.UnknownType(),
		}, true
	}
	if !ir.IsExported(name.Name) {
		return invalidStore(), scope.Err().Appendf(name, "%s is not exported", name.Name)
	}
	return ip.defs[name.Name], true
}
