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

	"github.com/gx-org/gx/build/builder/irb"
	"github.com/gx-org/gx/build/ir"
)

type file struct {
	pkg     *basePackage
	src     *ast.File
	name    string
	imports []*ir.ImportDecl
}

var _ irb.Node[*pkgResolveScope] = (*file)(nil)

func newFile(pkg *basePackage, name string, src *ast.File) *file {
	f := &file{
		name: name,
		src:  src,
		pkg:  pkg,
	}
	pkg.files.Store(f.name, f)
	return f
}

func processFile(pkgctx *pkgProcScope, name string, src *ast.File) bool {
	f := newFile(pkgctx.pkg(), name, src)
	if err := f.pkg.setOrCheckName(src.Name); err != nil {
		pkgctx.Err().Appendf(src.Name, "%q is an invalid package name: %v", src.Name, err)
	}
	return f.processDecls(pkgctx.newFilePScope(f), src.Decls)
}

func (f *file) processDecls(pscope procScope, decls []ast.Decl) bool {
	ok := true
	for _, decl := range decls {
		var declOk bool
		switch declT := decl.(type) {
		case *ast.GenDecl:
			declOk = f.processGenDecl(pscope, declT)
		case *ast.FuncDecl:
			declOk = f.processFunc(pscope, declT)
		default:
			declOk = pscope.Err().Appendf(decl, "declarator %T not supported", declT)
		}
		ok = ok && declOk
	}
	return ok
}

func (f *file) declareImports(pscope procScope, id *ast.Ident, decl *ir.ImportDecl) bool {
	f.imports = append(f.imports, decl)
	return true
}

func (f *file) processGenDecl(pscope procScope, gen *ast.GenDecl) bool {
	switch gen.Tok {
	case token.TYPE:
		return processTypeDecl(pscope, gen)
	case token.IMPORT:
		return processImportDecl(pscope, gen)
	case token.VAR:
		return processVarDecl(pscope, gen)
	case token.CONST:
		return processConstDecl(pscope, gen)
	default:
		return pscope.Err().Appendf(gen, "generic declaration %s not supported", gen.Tok)
	}
}

func (f *file) file() *file {
	return f
}

func (f *file) Build(irb irBuilder) (ir.Node, bool) {
	return &ir.File{
		Src:     f.src,
		Imports: f.imports,
		Package: irb.Pkg(),
	}, true
}
