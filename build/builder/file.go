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

	"github.com/gx-org/gx/build/ir"
)

type file struct {
	pkg     *bPackage
	repr    ir.File
	name    string
	imports []*importDecl
	ns      *namespace

	importOk bool

	funcs   []function
	methods []function
	types   []*namedType
	vars    []*varDecl
	consts  []*constDecl
}

func newFile(pkg *bPackage, name string) *file {
	return &file{
		name: name,
		repr: ir.File{
			Package: &pkg.repr,
			Src:     &ast.File{},
		},
		pkg:      pkg,
		ns:       pkg.ns.newChild(),
		importOk: true,
	}
}

func processFile(pkgScope *scopePackage, name string, src *ast.File) *file {
	f := newFile(pkgScope.pkg(), name)
	f.repr.Src = src
	if err := f.pkg.setOrCheckName(src.Name); err != nil {
		pkgScope.err().Appendf(src.Name, "%q is an invalid package name: %v", src.Name, err)
	}
	scope := pkgScope.scopeFile(f)
	for _, decl := range f.repr.Src.Decls {
		switch declT := decl.(type) {
		case *ast.GenDecl:
			f.processGenDecl(scope, declT)
		case *ast.FuncDecl:
			f.processFunc(scope, declT)
		default:
			scope.err().Appendf(decl, "declarator %T not supported", declT)
		}
	}
	return f
}

func (f *file) declareImports(block *scopeFile, ident *ast.Ident, ref *packageRef) bool {
	if prev := f.ns.assign(ident, nil, ref); prev != nil {
		appendRedeclaredError(block.err(), ident, prev)
		return false
	}
	f.imports = append(f.imports, ref.decl)
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
	for _, tp := range f.types {
		resolveType(scope, tp, tp)
	}
	for _, cst := range f.consts {
		cst.resolveType(scope)
	}
	for _, vr := range f.vars {
		vr.resolveType(scope)
	}
	for _, method := range f.methods {
		fnDecl, ok := method.(*funcDecl)
		if !ok {
			continue
		}
		fnDecl.recResolveTypes(scope)
	}
	for _, fn := range f.funcs {
		fnDecl, ok := fn.(*funcDecl)
		if !ok {
			continue
		}
		fnDecl.recResolveTypes(scope)
	}
}

func (f *file) buildIRs() {
	pkg := &f.pkg.repr
	pkg.Files[f.name] = &f.repr
	f.repr.Imports = make([]*ir.ImportDecl, len(f.imports))
	for i, imprt := range f.imports {
		f.repr.Imports[i] = &imprt.ext
	}
	for _, fn := range f.funcs {
		pkg.Funcs = append(pkg.Funcs, fn.irFunc())
	}
	for _, vr := range f.vars {
		pkg.Vars = append(pkg.Vars, vr.buildStmt())
	}
	for _, cst := range f.consts {
		pkg.Consts = append(pkg.Consts, cst.buildStmt())
	}
	for _, method := range f.methods {
		method.irFunc()
	}
	for _, typ := range f.types {
		id := len(pkg.Types)
		irType := typ.buildNamedType()
		irType.ID = id
		pkg.Types = append(pkg.Types, irType)
	}
}

func (f *file) declareMethod(block *scopeFile, decl function) {
	f.methods = append(f.methods, decl)
}

func (f *file) declareFunc(block *scopeFile, decl function) (ok bool) {
	f.funcs = append(f.funcs, decl)
	name := decl.name()
	if prev := f.pkg.ns.assign(name, nil, decl.typeNode()); prev != nil {
		appendRedeclaredError(block.err(), name, prev)
		return false
	}
	return true
}

func (f *file) declareType(block *scopeFile, ident *ast.Ident, tp *namedType) bool {
	tp.repr.File = &f.repr
	if prev := f.pkg.ns.assign(ident, nil, tp); prev != nil {
		appendRedeclaredError(block.err(), ident, prev)
		return false
	}
	f.types = append(f.types, tp)
	return true
}

func (f *file) declareStaticVar(block *scopeFile, ident *ast.Ident, decl *varDecl) bool {
	resolver := func(scope scoper) (exprNode, typeNode, bool) {
		typ, ok := decl.resolveType(scope)
		return nil, typ, ok
	}
	if prev := f.pkg.ns.assignTypeF(ident.Name, ident, resolver); prev != nil {
		appendRedeclaredError(block.err(), ident, prev)
		return false
	}
	f.vars = append(f.vars, decl)
	return true
}
