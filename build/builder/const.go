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

	"github.com/gx-org/gx/build/builder/irb"
	"github.com/gx-org/gx/build/ir"
)

type constSpec struct {
	src          *ast.ValueSpec
	declaredType typeExprNode
	bFile        *file
	exprs        []*constExpr
}

var _ irb.Node[*pkgResolveScope] = (*constSpec)(nil)

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
		pscope.Err().Appendf(src.Values[len(src.Names)], "extra init expr")
		numOk = false
	}
	names := src.Names
	if len(src.Values) < len(names) {
		name := names[len(src.Values)]
		pscope.Err().Appendf(name, "missing init expr for %s", name.Name)
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

func (spec *constSpec) Build(irb irBuilder) (ir.Node, bool) {
	fScope, ok := irb.Scope().newFileScope(spec.bFile)
	if !ok {
		return nil, false
	}
	ext := &ir.ConstSpec{Src: spec.src, FFile: fScope.irFile()}
	typeOk := true
	if spec.declaredType != nil {
		ext.Type, typeOk = spec.declaredType.buildTypeExpr(fScope)
	}
	irb.Register(constDeclarator(ext))
	return ext, ok && typeOk
}

type (
	constExpr struct {
		spec *constSpec

		name  *ast.Ident
		value exprNode
	}

	iConstExpr interface {
		buildDeclaration(irBuilder) (*ir.ConstExpr, bool)
		buildExpression(irBuilder, *ir.ConstExpr) bool
	}
)

var _ irb.Node[*pkgResolveScope] = (*constSpec)(nil)

func (spec *constSpec) processConstExpr(pscope procScope, name *ast.Ident, value ast.Expr) (*constExpr, bool) {
	val, ok := processExpr(pscope, value)
	return &constExpr{
		spec:  spec,
		name:  name,
		value: val,
	}, ok
}

func (cst *constExpr) buildDeclaration(ibld irBuilder) (*ir.ConstExpr, bool) {
	ext := &ir.ConstExpr{VName: cst.name}
	var ok bool
	ext.Decl, ok = irBuild[*ir.ConstSpec](ibld, cst.spec)
	if !ok {
		return ext, false
	}
	ext.Decl.Exprs = append(ext.Decl.Exprs, ext)
	return ext, true
}

func (cst *constExpr) buildExpression(ibld irBuilder, ext *ir.ConstExpr) bool {
	fScope, ok := ibld.Scope().newFileScope(cst.spec.bFile)
	if !ok {
		return false
	}
	ext.Val, ok = buildAExpr(fScope, cst.value)
	if !ok {
		return false
	}
	if ext.Decl.Type == nil {
		return true
	}
	targetType := ext.Decl.Type.Typ
	if ir.IsNumber(ext.Val.Type().Kind()) {
		ext.Val, ok = castNumber(fScope, ext.Val, targetType)
	}
	if !ok {
		return false
	}
	return assignableToAt(fScope, ext.Val.Source(), ext.Val.Type(), targetType)
}

func (cst *constExpr) pNode() processNode {
	return newProcessNode[iConstExpr](token.CONST, cst.name, cst)
}

func constDeclarator(ext *ir.ConstSpec) irb.Declarator {
	return func(decls *ir.Declarations) {
		decls.Consts = append(decls.Consts, ext)
	}
}

func (cst *constExpr) String() string {
	return fmt.Sprintf("const %s %s", cst.name.Name, cst.value.String())
}
