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
	"go/token"

	"github.com/gx-org/gx/base/iter"
	"github.com/gx-org/gx/build/ir"
)

type fileNamespace struct {
	file *file
	nameToIdent
	imports []*importDecl
}

var _ namespace = (*fileNamespace)(nil)

func (ns *fileNamespace) newChild() *blockNamespace {
	return newNamespace(ns)
}

func (ns *fileNamespace) fetch(name string) *identNode {
	return fetch(ns.nameToIdent.fetch, ns.file.pkg.ns, name)
}

type file struct {
	pkg  *basePackage
	repr ir.File
	name string
	ns   *fileNamespace

	importOk bool

	types  []*namedType
	vars   []*varDecl
	consts []*constDecl

	// builtins are functions declared in a GX package and for
	// which the implementation is provided by a backend.
	builtins []*funcBuiltin
	// funcs are functions declared and for which is going to have
	// a body using the IR.
	funcs []*funcDecl
	// macros are functions called by the compiler to construct other
	// GX functions.
	macros []*funcMacro

	// methods stores pointers to function that have not been assigned
	// to their named type yet.
	methods []*funcDecl
}

func newFile(pkg *basePackage, name string) *file {
	f := &file{
		name: name,
		repr: ir.File{
			Package: &pkg.repr,
			Src:     &ast.File{},
		},
		ns: &fileNamespace{
			nameToIdent: make(map[string]*identNode),
		},
		pkg:      pkg,
		importOk: true,
	}
	f.ns.file = f
	return f
}

func processFile(pkgScope *scopePackage, name string, src *ast.File) *file {
	f := newFile(pkgScope.pkg(), name)
	f.repr.Src = src
	if err := f.pkg.setOrCheckName(src.Name); err != nil {
		pkgScope.err().Appendf(src.Name, "%q is an invalid package name: %v", src.Name, err)
	}
	f.processDecls(pkgScope.scopeFile(f), src.Decls)
	return f
}

func (f *file) processDecls(scope *scopeFile, decls []ast.Decl) {
	for _, decl := range decls {
		switch declT := decl.(type) {
		case *ast.GenDecl:
			f.processGenDecl(scope, declT)
		case *ast.FuncDecl:
			f.processFunc(scope, declT)
		default:
			scope.err().Appendf(decl, "declarator %T not supported", declT)
		}
	}
}

func (f *file) declareImports(block *scopeFile, ident *ast.Ident, ref *packageRef) bool {
	if prev := f.ns.fetch(ident.Name); prev != nil {
		appendRedeclaredError(block.err(), ident, prev)
		return false
	}
	f.ns.assign(newIdent(ident, ref))
	f.ns.imports = append(f.ns.imports, ref.decl)
	return true
}

func (f *file) processGenDecl(scope *scopeFile, gen *ast.GenDecl) {
	switch gen.Tok {
	case token.TYPE:
		processTypeDecl(scope, gen)
	case token.IMPORT:
		ok := processImportDecl(scope, gen)
		f.importOk = f.importOk && ok
	case token.VAR:
		processVarDecl(scope, gen)
	case token.CONST:
		processConstDecl(scope, gen)
	default:
		scope.err().Appendf(gen, "generic declaration %s not supported", gen.Tok)
	}
	return
}

func (f *file) resolveAll(pkgScope *scopePackage) {
	scope := pkgScope.scopeFile(f)
	// Resolve declared types.
	for _, tp := range f.types {
		resolveType(scope, tp, tp)
	}
	// Resolve constants.
	for _, cst := range f.consts {
		cst.resolveType(scope)
	}
	// Resolve static variables.
	for _, vr := range f.vars {
		vr.resolveType(scope)
	}
	// Resolve declared functions type.
	for fn := range iter.All(f.methods, f.funcs) {
		fn.resolve(scope)
	}
}

func (f *file) buildIR(pkg *ir.Package) {
	pkg.Files[f.name] = &f.repr
	f.repr.Imports = make([]*ir.ImportDecl, len(f.ns.imports))
	for i, imprt := range f.ns.imports {
		f.repr.Imports[i] = &imprt.ext
	}
}

func (f *file) declareFuncDecl(scope *scopeFile, decl *funcDecl) bool {
	if decl.receiver() != nil {
		f.methods = append(f.methods, decl)
		return true
	}
	f.funcs = append(f.funcs, decl)
	return f.declarePackageFunc(scope, decl)
}

func (f *file) declareFuncBuiltin(scope *scopeFile, decl *funcBuiltin) bool {
	f.builtins = append(f.builtins, decl)
	return f.declarePackageFunc(scope, decl)
}

func (f *file) declareFuncMacro(scope *scopeFile, decl *funcMacro) bool {
	f.macros = append(f.macros, decl)
	return f.declarePackageFunc(scope, decl)
}

func (f *file) declarePackageFunc(scope *scopeFile, fn function) (ok bool) {
	fb := funcBuilder{fn: fn}
	name := fn.name()
	if !f.pkg.checkIfDefined(scope, name) {
		return false
	}
	f.pkg.ns.assign(newIdentExpr(name, fn), fb)
	return true
}

func (f *file) declareType(block *scopeFile, ident *ast.Ident, tp *namedType) bool {
	tp.repr.File = &f.repr
	if !f.pkg.checkIfDefined(block, ident) {
		return false
	}
	f.pkg.ns.assign(newIdent(ident, tp), tp)
	f.types = append(f.types, tp)
	return true
}

func (f *file) declareStaticVar(block *scopeFile, ident *ast.Ident, decl *varDecl) bool {
	if !f.pkg.checkIfDefined(block, ident) {
		return false
	}
	f.pkg.ns.assign(&identNode{
		ident: ident,
		typeF: func(scope scoper) (typeNode, bool) {
			return decl.resolveType(scope)
		},
	}, decl)
	f.vars = append(f.vars, decl)
	return true
}
