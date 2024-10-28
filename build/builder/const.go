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

	"github.com/gx-org/gx/build/ir"
)

type constDecl struct {
	ext ir.ConstDecl

	resolvedType typeNode
	declaredType typeNode

	exprs []*constExpr
}

func importConstDecl(scope *scopeFile, irDecl *ir.ConstDecl) bool {
	decl := &constDecl{
		ext:   *irDecl,
		exprs: make([]*constExpr, len(irDecl.Exprs)),
	}
	ok := true
	if irDecl.Type != nil {
		decl.resolvedType, ok = toTypeNode(scope, irDecl.Type)
	}
	for i, irExpr := range irDecl.Exprs {
		var exprOk bool
		decl.exprs[i], exprOk = importConstExpr(scope, decl, irExpr)
		ok = ok && exprOk
	}
	return ok
}

func processConstDecl(scope *scopeFile, decl *ast.GenDecl) bool {
	ok := true
	for _, spec := range decl.Specs {
		ok = processConst(scope, spec.(*ast.ValueSpec)) && ok
	}
	return ok
}

func processConst(scope *scopeFile, spec *ast.ValueSpec) bool {
	decl := &constDecl{
		ext: ir.ConstDecl{
			Src: spec,
		},
	}
	declaredTypeOk := true
	if spec.Type != nil {
		decl.declaredType, declaredTypeOk = processTypeExpr(scope, spec.Type)
	}
	valuesOk := true
	decl.exprs = make([]*constExpr, len(spec.Names))
	for i, name := range spec.Names {
		value, valueOk := processExpr(scope, spec.Values[i])
		valuesOk = valueOk && valuesOk
		decl.exprs[i] = &constExpr{
			ext: &ir.ConstExpr{
				Decl:  &decl.ext,
				VName: name,
			},
			decl:  decl,
			value: value,
		}
	}

	declareOk := true
	for _, expr := range decl.exprs {
		ident := expr.ext.VName
		resolver := func(scope scoper) (exprNode, typeNode, bool) {
			typ, ok := expr.resolveType(scope)
			return expr, typ, ok
		}
		if prev := scope.file().pkg.ns.assignTypeF(ident.Name, ident, resolver); prev != nil {
			appendRedeclaredError(scope.err(), ident, prev)
			declareOk = false
		}
	}
	scope.file().consts = append(scope.file().consts, decl)

	return declaredTypeOk && valuesOk && declareOk
}

func (decl *constDecl) source() ast.Node {
	return decl.ext.Src
}

func (decl *constDecl) resolveType(scope scoper) (typeNode, bool) {
	if decl.resolvedType != nil {
		return typeNodeOk(decl.resolvedType)
	}
	ok := true
	if decl.declaredType != nil {
		decl.resolvedType, ok = resolveType(scope, decl, decl.declaredType)
	}
	if !ok {
		return typeNodeOk(decl.resolvedType)
	}
	for _, expr := range decl.exprs {
		var exprOk bool
		expr.valueType, exprOk = expr.value.resolveType(scope)
		if !exprOk {
			ok = false
			continue
		}
		assignType, assignOk := assignableToAt(scope, expr, expr.valueType, decl.resolvedType)
		if !assignOk {
			ok = false
			continue
		}
		decl.resolvedType = assignType
	}
	if !ok {
		decl.resolvedType = invalid
	}
	return typeNodeOk(decl.resolvedType)
}

func (decl *constDecl) buildStmt() *ir.ConstDecl {
	decl.ext.Type = decl.resolvedType.buildType()
	decl.ext.Exprs = make([]*ir.ConstExpr, len(decl.exprs))
	for i, expr := range decl.exprs {
		decl.ext.Exprs[i] = expr.buildConstExpr()
	}
	return &decl.ext
}

func (decl *constDecl) wantType() ir.Type {
	if decl.declaredType == nil {
		return nil
	}
	if decl.declaredType.kind() == ir.InvalidKind {
		return nil
	}
	return decl.declaredType.buildType()
}

type constExpr struct {
	ext  *ir.ConstExpr
	decl *constDecl

	value     exprNode
	valueType typeNode
}

var _ exprNode = (*constExpr)(nil)

func importConstExpr(scope *scopeFile, decl *constDecl, cstExpr *ir.ConstExpr) (*constExpr, bool) {
	expr := &constExpr{
		decl:      decl,
		ext:       cstExpr,
		value:     &irExprNode{x: cstExpr.Value},
		valueType: decl.resolvedType,
	}
	prev := scope.pkg().ns.assign(cstExpr.VName, expr, decl.resolvedType)
	if prev != nil {
		return expr, scope.err().Appendf(cstExpr.Source(), "%s has already been registered", cstExpr.VName.Name)
	}
	scope.file().consts = append(scope.file().consts, decl)
	return expr, true
}

func (cst *constExpr) resolveType(scope scoper) (typeNode, bool) {
	declType, ok := cst.decl.resolveType(scope)
	if !ok {
		return invalid, false
	}
	if cst.valueType.kind() == ir.NumberKind && declType.kind() != ir.NumberKind {
		cst.value, cst.valueType, ok = buildNumberNode(scope, cst.value, declType.buildType())
	}
	return declType, ok
}

func (cst *constExpr) expr() ast.Expr {
	return cst.value.expr()
}

func (cst *constExpr) source() ast.Node {
	return cst.ext.VName
}

func (cst *constExpr) castTo(eval evaluator) (exprScalar, []*ir.ValueRef, bool) {
	return cst.value.(exprNumber).castTo(eval)
}

func (cst *constExpr) buildConstExpr() *ir.ConstExpr {
	cst.ext.Value = cst.value.buildExpr()
	return cst.ext
}

func (cst *constExpr) buildExpr() ir.Expr {
	return cst.buildConstExpr().Value
}

func (cst *constExpr) String() string {
	return fmt.Sprintf("const %s", cst.ext.String())
}
