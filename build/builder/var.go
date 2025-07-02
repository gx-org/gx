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

type varSpec struct {
	src   *ast.ValueSpec
	bFile *file
	exprs []*varExpr
	typ   typeExprNode
}

var _ parentNodeBuilder = (*varSpec)(nil)

func processVarDecl(pscope procScope, decl *ast.GenDecl) bool {
	ok := true
	for _, spec := range decl.Specs {
		ok = processVarSpec(pscope.axisLengthScope(), spec.(*ast.ValueSpec)) && ok
	}
	return ok
}

func processVarSpec(pscope procScope, src *ast.ValueSpec) bool {
	if len(src.Values) > 0 {
		pscope.err().Appendf(src, "cannot assign a value to a static variable")
	}
	spec := &varSpec{
		src:   src,
		bFile: pscope.file(),
	}
	var typeOk bool
	if src.Type != nil {
		spec.typ, typeOk = processTypeExpr(pscope, src.Type)
	} else {
		typeOk = pscope.err().Appendf(src, "static variable has no type")
	}
	exprsOk := true
	spec.exprs = make([]*varExpr, len(src.Names))
	for i, name := range src.Names {
		var exprOk bool
		spec.exprs[i], exprOk = spec.processVarExpr(pscope, name, nil)
		exprsOk = exprsOk && exprOk
	}
	return typeOk && exprsOk && pscope.decls().registerStaticVar(spec)
}

func (spec *varSpec) buildSpecNode(rscope resolveScope) (*ir.VarSpec, bool) {
	ext := &ir.VarSpec{Src: spec.src}
	if spec.typ == nil {
		return ext, false
	}
	typeRef, ok := spec.typ.buildTypeExpr(rscope)
	if !ok {
		return ext, false
	}
	return &ir.VarSpec{
		Src:   spec.src,
		TypeV: typeRef.Typ,
	}, true
}

func (spec *varSpec) buildParentNode(irb *irBuilder, decls *ir.Declarations) (ir.Node, bool) {
	fscope, scopeOk := irb.pkgScope.newFileScope(spec.bFile, nil)
	ext, specOk := spec.buildSpecNode(fscope)
	var fileOk bool
	ext.FFile, fileOk = buildParentNode[*ir.File](irb, decls, spec.bFile)
	decls.Vars = append(decls.Vars, ext)
	return ext, scopeOk && specOk && fileOk
}

func (spec *varSpec) file() *file {
	return spec.bFile
}

type varExpr struct {
	spec *varSpec
	name *ast.Ident
}

func (spec *varSpec) processVarExpr(pscope procScope, name *ast.Ident, value ast.Expr) (*varExpr, bool) {
	return &varExpr{spec: spec, name: name}, true
}

func (vr *varExpr) build(pkgScope *pkgResolveScope) (*ir.VarExpr, bool) {
	fscope, scopeOk := pkgScope.newFileScope(vr.spec.bFile, nil)
	spec, specOk := vr.spec.buildSpecNode(fscope)
	return &ir.VarExpr{Decl: spec, VName: vr.name}, specOk && scopeOk
}

func (vr *varExpr) pNode() processNode {
	return newProcessNode[*varExpr](token.VAR, vr.name, vr)
}

func varDeclarator(spec *varSpec) declarator {
	return func(irb *irBuilder, decls *ir.Declarations, dNode *declNode) bool {
		irSpec, ok := buildParentNode[*ir.VarSpec](irb, decls, spec)
		if !ok {
			return false
		}
		expr := dNode.ir.(*ir.VarExpr)
		expr.Decl = irSpec
		irSpec.Exprs = append(irSpec.Exprs, expr)
		return true
	}
}

func (vr *varExpr) String() string {
	return fmt.Sprintf("var %s", vr.name.Name)
}
