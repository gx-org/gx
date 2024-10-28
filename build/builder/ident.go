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

// identTypeExpr is a reference to a type by an identifier.
type identTypeExpr struct {
	ext ir.TypeExpr
	src *ast.Ident
	typ typeNode
}

var (
	_ concreteTypeNode = (*identTypeExpr)(nil)
	_ selector         = (*identTypeExpr)(nil)
)

func processIdentTypeExpr(owner owner, ident *ast.Ident) (typeNode, bool) {
	scalarType := ir.ScalarTypeS(ident.Name)
	if scalarType.Kind() != ir.InvalidKind {
		return &builtinType[*ir.AtomicType]{ext: scalarType}, true
	}
	return &identTypeExpr{
		ext: ir.TypeExpr{Src: ident},
		src: ident,
	}, true
}

// pos returns the position of the identifier in the code.
func (n *identTypeExpr) source() ast.Node {
	return n.ext.Source()
}

func (n *identTypeExpr) buildType() ir.Type {
	return n.ext.Typ
}

func (n *identTypeExpr) isGeneric() bool {
	return n.typ.isGeneric()
}

func (n *identTypeExpr) kind() ir.Kind {
	return n.buildType().Kind()
}

func (n *identTypeExpr) String() string {
	return n.typ.String()
}

func (n *identTypeExpr) resolveConcreteType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	prev := scope.namespace().fetchIdentNode(n.src.Name)
	if prev != nil {
		// Type is defined in the package.
		var ok bool
		_, n.typ, ok = prev.typeF(scope)
		n.ext.Typ = n.typ.buildType()
		return n.typ, ok
	}
	name := n.src.Name
	kind := ir.KindFromString(name)
	if kind == ir.InvalidKind {
		// The name cannot be match to a builtin type.
		scope.err().Appendf(n.source(), "undeclared type identifier: %s", name)
		n.typ, n.ext.Typ = invalidType()
		return n.typ, false
	}
	n.ext.Typ = ir.ScalarTypeK(kind)
	var ok bool
	n.typ, ok = toTypeNode(scope, n.ext.Typ)
	if n.ext.Typ.Kind() == ir.InvalidKind {
		// Kind is not a scalar.
		scope.err().Appendf(n.source(), "%s is not a scalar kind", name)
		return n.typ, false
	}
	return n.typ, ok
}

func (n *identTypeExpr) buildSelectNode(scope scoper, sel *ast.SelectorExpr) selectNode {
	selector, ok := n.typ.(selector)
	if !ok {
		return nil
	}
	return selector.buildSelectNode(scope, sel)
}

// valueRef is a reference to a value by an identifier.
type valueRef struct {
	ext ir.ValueRef
	typ typeNode
}

func processIdentExpr(ident *ast.Ident) *valueRef {
	return &valueRef{
		ext: ir.ValueRef{
			Src: ident,
		},
	}
}

func (n *valueRef) ident() *ast.Ident {
	return n.ext.Src
}

func (n *valueRef) source() ast.Node {
	return n.expr()
}

func (n *valueRef) expr() ast.Expr {
	return n.ext.Expr()
}

func (n *valueRef) buildExpr() ir.Expr {
	if n.ext.Typ != nil {
		return &n.ext
	}
	n.ext.Typ = n.typ.buildType()
	return &n.ext
}

func (n *valueRef) resolveType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	var ok bool
	_, n.typ, ok = scope.namespace().fetch(scope, n.ext.Src)
	return n.typ, ok
}

func (n *valueRef) castTo(eval evaluator) (exprScalar, []*ir.ValueRef, bool) {
	expr, typ, ok := eval.scoper().namespace().fetch(eval.scoper(), n.ext.Src)
	if !ok {
		eval.scoper().err().Appendf(n.source(), "undefined: %s", n.ext.Src.Name)
		return nil, nil, false
	}
	if expr == nil || expr == n {
		eval.scoper().err().Appendf(n.source(), "identifier %s (type %s) cannot be casted to %s", n.ext.Src.Name, typ.String(), eval.want().String())
	}
	return castExprTo(eval, expr)
}

func (n *valueRef) String() string {
	return n.ext.Src.Name
}
