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

	"github.com/gx-org/gx/build/ir"
)

type selectorTypeExpr struct {
	src *ast.SelectorExpr

	x   exprNode
	typ *namedType
}

var (
	_ selector = (*selectorTypeExpr)(nil)
	_ typeNode = (*selectorTypeExpr)(nil)
)

func processTypeSelectorReference(owner owner, expr *ast.SelectorExpr) (typeNode, bool) {
	n := &selectorTypeExpr{
		src: expr,
	}
	var ok bool
	n.x, ok = processExpr(owner, expr.X)
	return n, ok
}

func (n *selectorTypeExpr) resolveConcreteType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	xType, ok := n.x.resolveType(scope)
	if !ok {
		return invalid, false
	}
	if xType.kind() != packageKind {
		scope.err().Appendf(n.source(), "%s.%s undefined", n.src.X, n.src.Sel.Name)
		return invalid, false
	}
	pckRef := xType.(*packageRef)
	typ, ok := fetchType(scope, pckRef.decl.pkg.base().ns, n.src.Sel)
	if !ok {
		return invalid, false
	}
	n.typ, ok = typ.(*namedType)
	if !ok {
		scope.err().Appendf(n.source(), "%s.%s undefined", n.src.X, n.src.Sel.Name)
	}
	return n.typ, ok
}

func (n *selectorTypeExpr) source() ast.Node {
	return n.src
}

func (n *selectorTypeExpr) kind() ir.Kind {
	return n.typ.kind()
}

func (n *selectorTypeExpr) convertibleTo(scope scoper, other typeNode) (bool, error) {
	return n.irType().ConvertibleTo(scope.evalFetcher(), other.irType())
}

func (n *selectorTypeExpr) irType() ir.Type {
	if n.typ == nil {
		return invalid.irType()
	}
	return n.typ.irType()
}

func (n *selectorTypeExpr) isGeneric() bool {
	return false
}

func (n *selectorTypeExpr) buildSelectNode(scope scoper, expr *ast.SelectorExpr) selectNode {
	return n.typ.buildSelectNode(scope, expr)
}

func (n *selectorTypeExpr) String() string {
	return n.src.Sel.Name
}

type (
	// selector node contains elements selectable by their names.
	// A selector can be a named type, a structure, or a package.
	// Elements that can be selected include:
	//  * methods for named types,
	//  * methods and fields for structures,
	//  * functions, constants, or variables for packages.
	// It returns the node being selected and its index.
	selector interface {
		buildSelectNode(scoper, *ast.SelectorExpr) selectNode
	}

	selectNode interface {
		resolveType(scope scoper) (typeNode, bool)
		buildExpr(x exprNode) ir.Expr
	}

	// selectedExpr references an attribute in a package or a structure.
	selectorExpr struct {
		ext ir.Expr
		src *ast.SelectorExpr
		sel selectNode
		x   exprNode
		typ typeNode
	}
)

var (
	_ exprNode   = (*selectorExpr)(nil)
	_ selectNode = (*fieldSelectorExpr)(nil)
	_ selectNode = (*methodSelectorExpr)(nil)
)

func processSelectorReference(owner owner, expr *ast.SelectorExpr) (*selectorExpr, bool) {
	n := &selectorExpr{
		src: expr,
	}
	var ok bool
	n.x, ok = processExpr(owner, expr.X)
	return n, ok
}

func (n *selectorExpr) source() ast.Node {
	return n.expr()
}

func (n *selectorExpr) expr() ast.Expr {
	return n.src
}

func (n *selectorExpr) String() string {
	return n.src.Sel.Name
}

func (n *selectorExpr) buildExpr() ir.Expr {
	if n.ext != nil || n.sel == nil {
		return n.ext
	}
	n.ext = n.sel.buildExpr(n.x)
	return n.ext
}

func (n *selectorExpr) selectNode(scope scoper, typ typeNode) selectNode {
	selector, ok := typ.(selector)
	if !ok {
		scope.err().Appendf(n.source(), "undefined: %s.%s", n.x.String(), n.src.Sel.Name)
		return nil
	}
	return selector.buildSelectNode(scope, n.src)
}

func (n *selectorExpr) resolveType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	exprType, ok := n.x.resolveType(scope)
	if !ok {
		n.typ, _ = invalidType()
		return n.typ, false
	}
	if n.sel = n.selectNode(scope, exprType); n.sel == nil {
		n.typ, _ = invalidType()
		return n.typ, false
	}
	n.typ, ok = n.sel.resolveType(scope)
	return n.typ, ok
}

// fieldSelectorExpr references a field on a structure.
type fieldSelectorExpr struct {
	ext        ir.FieldSelectorExpr
	structType *structType
	field      *field
}

func buildFieldSelectorExpr(expr *ast.SelectorExpr, st *structType, field *field) selectNode {
	return &fieldSelectorExpr{
		ext: ir.FieldSelectorExpr{
			Src:    expr,
			Struct: st.ext,
			Field:  field.ext,
		},
		structType: st,
		field:      field,
	}
}

func (n *fieldSelectorExpr) buildExpr(x exprNode) ir.Expr {
	n.ext.X = x.buildExpr()
	n.ext.Typ = n.field.group.typ.irType()
	return &n.ext
}

func (n *fieldSelectorExpr) resolveType(scope scoper) (typeNode, bool) {
	return typeNodeOk(n.field.group.typ)
}

// methodSelectorExpr references a method on a structure.
type methodSelectorExpr struct {
	ext       ir.MethodSelectorExpr
	namedType *namedType
}

func buildMethodSelectorExpr(expr *ast.SelectorExpr, st *namedType, fn ir.Func) selectNode {
	return &methodSelectorExpr{
		ext: ir.MethodSelectorExpr{
			Src:   expr,
			Named: &st.repr,
			Func:  fn,
		},
		namedType: st,
	}
}

func (n *methodSelectorExpr) buildExpr(x exprNode) ir.Expr {
	n.ext.X = x.buildExpr()
	n.ext.Typ = n.namedType.methods[n.ext.Func.Name()].irFunc().Type()
	return &n.ext
}

func (n *methodSelectorExpr) resolveType(scope scoper) (typeNode, bool) {
	return n.namedType.methods[n.ext.Func.Name()].resolveType(scope)
}
