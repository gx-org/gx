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
	"go/ast"
	"go/token"

	"github.com/gx-org/gx/build/ir"
)

type constSpec struct {
	src          *ast.ValueSpec
	declaredType typeExprNode
	bFile        *file
	exprs        []*constExpr
}

var _ parentNodeBuilder = (*constSpec)(nil)

func processConstDecl(pscope procScope, decl *ast.GenDecl) bool {
	ok := true
	for _, spec := range decl.Specs {
		ok = processConstSpec(pscope, spec.(*ast.ValueSpec)) && ok
	}
	return ok
}

func processConstSpec(pscope procScope, src *ast.ValueSpec) bool {
	spec := &constSpec{
		src:   src,
		bFile: pscope.file(),
	}
	declaredTypeOk := true
	if src.Type != nil {
		spec.declaredType, declaredTypeOk = processTypeExpr(pscope, src.Type)
	}
	numOk := true
	if len(src.Values) > len(src.Names) {
		pscope.err().Appendf(src.Values[len(src.Names)], "extra init expr")
		numOk = false
	}
	names := src.Names
	if len(src.Values) < len(names) {
		name := names[len(src.Values)]
		pscope.err().Appendf(name, "missing init expr for %s", name.Name)
		numOk = false
		names = src.Names[:len(src.Values)]
	}
	exprsOk := true
	spec.exprs = make([]*constExpr, len(src.Names))
	for i, name := range names {
		var exprOk bool
		spec.exprs[i], exprOk = spec.processConstExpr(pscope, name, src.Values[i])
		exprsOk = exprsOk && exprOk
	}
	return declaredTypeOk && numOk && exprsOk && pscope.decls().registerConst(spec)
}

func (spec *constSpec) buildSpecNode(rscope resolveScope) (*ir.ConstDecl, bool) {
	ext := &ir.ConstDecl{Src: spec.src}
	ok := true
	if spec.declaredType != nil {
		ext.Type, ok = spec.declaredType.buildTypeExpr(rscope)
	}
	return ext, ok
}

func (spec *constSpec) buildParentNode(irb *irBuilder, decls *ir.Declarations) (ir.Node, bool) {
	fscope, ok := irb.pkgScope.newFileScope(spec.bFile, nil)
	ext, specOk := spec.buildSpecNode(fscope)
	var fileOk bool
	ext.FFile, fileOk = buildParentNode[*ir.File](irb, decls, spec.bFile)
	decls.Consts = append(decls.Consts, ext)
	return ext, ok && specOk && fileOk
}

func (spec *constSpec) file() *file {
	return spec.bFile
}

type constExpr struct {
	spec *constSpec

	name  *ast.Ident
	value exprNode
}

func (spec *constSpec) processConstExpr(pscope procScope, name *ast.Ident, value ast.Expr) (*constExpr, bool) {
	val, ok := processExpr(pscope, value)
	return &constExpr{
		spec:  spec,
		name:  name,
		value: val,
	}, ok
}

func (cst *constExpr) build(pkgScope *pkgResolveScope) (*ir.ConstExpr, bool) {
	rscope, scopeOk := newConstResolveScope(pkgScope, cst.spec.bFile)
	expr, exprOk := cst.buildExpr(rscope)
	return expr, scopeOk && exprOk
}

func (cst *constExpr) buildExpr(rscope resolveScope) (*ir.ConstExpr, bool) {
	spec, specOk := cst.spec.buildSpecNode(rscope)
	x, ok := buildAExpr(rscope, cst.value)
	expr := &ir.ConstExpr{
		Decl:  spec,
		VName: cst.name,
		Val:   x,
	}
	if !ok {
		return expr, false
	}
	if cst.spec.declaredType == nil {
		return expr, specOk
	}
	typeExpr, typeOk := cst.spec.declaredType.buildTypeExpr(rscope)
	if !typeOk {
		return expr, false
	}
	typ := typeExpr.Typ
	if ir.IsNumber(typ.Kind()) {
		expr.Val, ok = castNumber(rscope, x, typ)
	}
	typeOk = assignableToAt(rscope, expr.Val.Source(), expr.Val.Type(), typ)
	return expr, ok && specOk && typeOk
}

func (cst *constExpr) pNode() processNode {
	return newProcessNode[*constExpr](token.CONST, cst.name, cst)
}

func constDeclarator(constSpec parentNodeBuilder) declarator {
	return func(irb *irBuilder, decls *ir.Declarations, dNode *declNode) bool {
		irSpec, ok := buildParentNode[*ir.ConstDecl](irb, decls, constSpec)
		if !ok {
			return false
		}
		expr := dNode.ir.(*ir.ConstExpr)
		expr.Decl = irSpec
		irSpec.Exprs = append(irSpec.Exprs, expr)
		return true
	}
}

func (cst *constExpr) String() string {
	return fmt.Sprintf("const %s %s", cst.name.Name, cst.value.String())
}
