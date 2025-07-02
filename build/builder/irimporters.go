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

func pNodeFromIR[T ir.Storage](tok token.Token, name string, node T, decl declarator) processNode {
	pNode := newProcessNode[T](
		tok,
		&ast.Ident{Name: name},
		node)
	pNode.decl = pNode.setDeclNode(node, decl)
	return pNode
}

func importNamedTypes(pkgScope *pkgProcScope, bFile *file, types []*ir.NamedType) bool {
	ok := true
	for _, typ := range types {
		if pNode, exist := pkgScope.decls().declarations.Load(typ.Name()); exist {
			// The type has already been imported. Only add the methods.
			namedTyp := pNode.ir().(*ir.NamedType)
			namedTyp.Methods = append(namedTyp.Methods, typ.Methods...)
			continue
		}
		namedTyp := *typ
		pNode := pNodeFromIR(token.TYPE, namedTyp.Name(), &namedTyp, namedTypeDeclarator(bFile))
		ok = pkgScope.decls().declarePackageName(pNode) && ok
	}
	return ok
}

func importFuncs(pkgScope *pkgProcScope, bFile *file, funcs []ir.PkgFunc) bool {
	ok := true
	for _, fun := range funcs {
		pNode := pNodeFromIR(token.FUNC, fun.Name(), fun, funcDeclarator(bFile, nil))
		ok = pkgScope.decls().declarePackageName(pNode) && ok
	}
	return ok
}

type importedConstDecl struct {
	bFile *file
	decl  *ir.ConstSpec
}

func (b *importedConstDecl) buildParentNode(irb *irBuilder, decls *ir.Declarations) (ir.Node, bool) {
	ext := *b.decl
	var ok bool
	ext.FFile, ok = buildParentNode[*ir.File](irb, decls, b.bFile)
	decls.Consts = append(decls.Consts, &ext)
	return &ext, ok
}

func (b *importedConstDecl) file() *file {
	return b.bFile
}

func importConstDecls(pkgScope *pkgProcScope, file *file, cstDecls []*ir.ConstSpec) bool {
	ok := true
	for _, cstDecl := range cstDecls {
		impDecl := &importedConstDecl{bFile: file, decl: &ir.ConstSpec{
			Src:  cstDecl.Src,
			Type: cstDecl.Type,
		}}
		declarator := constDeclarator(impDecl)
		for _, expr := range cstDecl.Exprs {
			pNode := pNodeFromIR(token.CONST, expr.VName.Name, expr, declarator)
			ok = pkgScope.decls().declarePackageName(pNode) && ok
		}
	}
	return ok
}
