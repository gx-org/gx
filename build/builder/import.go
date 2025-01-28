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
	"strings"

	"github.com/gx-org/gx/build/ir"
)

// ----------------------------------------------------------------------------
// Import statement processing.

type importDecl struct {
	ext ir.ImportDecl

	pkg *basePackage
}

func processImportDecl(block *scopeFile, decl *ast.GenDecl) bool {
	ok := true
	for _, spec := range decl.Specs {
		imp := spec.(*ast.ImportSpec)
		path := imp.Path.Value
		path = strings.TrimPrefix(path, `"`)
		path = strings.TrimSuffix(path, `"`)
		imported, err := block.file().pkg.builder().importPath(path)
		if err != nil {
			block.err().Append(err)
			ok = false
			continue
		}
		name := imp.Name
		if name == nil {
			name = &ast.Ident{
				NamePos: imp.Path.Pos(),
				Name:    filepath.Base(path),
			}
		}
		imp.Name = name
		decl := &importDecl{
			ext: ir.ImportDecl{
				Src:     imp,
				Package: imported.IR(),
			},
			pkg: imported.base(),
		}
		importOk := block.file().declareImports(block, name, &packageRef{
			ext: ir.PackageRef{
				Src:  name,
				Decl: &decl.ext,
			},
			decl: decl,
		})
		ok = importOk && ok
	}
	return ok
}

// packageRef is a reference to a package.
type packageRef struct {
	ext  ir.PackageRef
	decl *importDecl
}

var (
	_ selector = (*packageRef)(nil)
	_ typeNode = (*packageRef)(nil)
)

func (n *packageRef) buildSelectNode(scope scoper, sel *ast.SelectorExpr) selectNode {
	pkgIdentNode := n.decl.pkg.base().ns.fetch(sel.Sel.Name)
	if pkgIdentNode == nil {
		scope.err().Appendf(sel, "undefined: %s.%s", n.ext.Src.Name, sel.Sel.Name)
		return nil
	}
	typ, ok := pkgIdentNode.typeF(scope)
	if !ok {
		return nil
	}
	switch exprT := pkgIdentNode.expr.(type) {
	case function:
		return buildPackageFuncSelectorExpr(sel, n, exprT)
	case *constExpr:
		return buildPackageConstSelectorExpr(sel, n, exprT)
	}
	switch tpT := typ.(type) {
	case *builtinType[*ir.NamedType]:
		return buildPackageTypeSelectorExpr(sel, n, tpT)
	default:
		scope.err().Appendf(sel, "%T package selector not supported", tpT)
		return nil
	}
}

func (n *packageRef) irType() ir.Type {
	return &n.ext
}

func (n *packageRef) convertibleTo(scope scoper, typ typeNode) (bool, []*ir.ValueRef, error) {
	return false, nil, nil
}

func (n *packageRef) kind() ir.Kind {
	return packageKind
}

func (n *packageRef) isGeneric() bool {
	return false
}

func (n *packageRef) String() string {
	return n.decl.ext.Package.String()
}

func (n *packageRef) resolveType(scope scoper) (typeNode, bool) {
	return n, true
}

// packageMethodSelectorExpr references a function on an imported package.
// The function needs to be exported.
type packageFuncSelectorExpr struct {
	ext ir.PackageFuncSelectorExpr
	ref *packageRef

	fn  function
	typ typeNode
}

func buildPackageFuncSelectorExpr(expr *ast.SelectorExpr, ref *packageRef, fn function) selectNode {
	return &packageFuncSelectorExpr{
		ext: ir.PackageFuncSelectorExpr{
			Src:     expr,
			Package: &ref.ext,
		},
		fn:  fn,
		ref: ref,
	}
}

func (n *packageFuncSelectorExpr) buildExpr(exprNode) ir.Expr {
	n.ext.Typ = n.typ.irType()
	n.ext.Func = n.fn.irFunc()
	return &n.ext
}

func (n *packageFuncSelectorExpr) resolveType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	var ok bool
	n.typ, ok = n.fn.resolveType(scope)
	return n.typ, ok
}

// packageTypeSelectorExpr references a type on an imported package.
type packageTypeSelectorExpr struct {
	ext ir.PackageTypeSelector
	ref *packageRef

	*builtinType[*ir.NamedType]
}

func buildPackageTypeSelectorExpr(expr *ast.SelectorExpr, ref *packageRef, typ *builtinType[*ir.NamedType]) selectNode {
	return &packageTypeSelectorExpr{
		ext: ir.PackageTypeSelector{
			Src:     expr,
			Package: &ref.ext,
			Typ:     typ.ext,
		},
		ref:         ref,
		builtinType: typ,
	}
}

func (n *packageTypeSelectorExpr) buildExpr(exprNode) ir.Expr {
	return &n.ext
}

func (n *packageTypeSelectorExpr) resolveType(scope scoper) (typeNode, bool) {
	return typeNodeOk(n.builtinType)
}

// packageConstSelectorExpr references a type on an imported package.
type packageConstSelectorExpr struct {
	ext ir.PackageConstSelectorExpr
	ref *packageRef

	cst *constExpr
}

func buildPackageConstSelectorExpr(expr *ast.SelectorExpr, ref *packageRef, cst *constExpr) selectNode {
	return &packageConstSelectorExpr{
		ext: ir.PackageConstSelectorExpr{
			Src:     expr,
			Package: &ref.ext,
			Const:   cst.ext,
			X:       cst.ext.Value,
		},
		ref: ref,
		cst: cst,
	}
}

func (n *packageConstSelectorExpr) buildExpr(exprNode) ir.Expr {
	return &n.ext
}

func (n *packageConstSelectorExpr) resolveType(scope scoper) (typeNode, bool) {
	return n.cst.resolveType(scope)
}
