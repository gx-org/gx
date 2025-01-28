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

func processIdentTypeExpr(_ owner, ident *ast.Ident) (typeNode, bool) {
	scalarType := ir.AtomicFromString(ident.Name)
	if scalarType.Kind() != ir.InvalidKind {
		return &builtinType[ir.Type]{ext: scalarType}, true
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

func (n *identTypeExpr) irType() ir.Type {
	return n.ext.Typ
}

func (n *identTypeExpr) isGeneric() bool {
	return n.typ.isGeneric()
}

func (n *identTypeExpr) kind() ir.Kind {
	return n.irType().Kind()
}

func (n *identTypeExpr) String() string {
	if n.typ == nil {
		return "<unresolved type>"
	}
	return n.typ.String()
}

func (n *identTypeExpr) resolveConcreteType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	prev := scope.namespace().fetch(n.src.Name)
	if prev != nil {
		// Type is defined in the package.
		typ, ok := prev.typeF(scope)
		if deferred, ok := typ.(*deferredType); ok {
			// deferredType prevents caching of resolved concrete types.
			return deferred.resolveConcreteType(scope)
		}
		n.typ = typ
		n.ext.Typ = n.typ.irType()
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
	n.ext.Typ = ir.TypeFromKind(kind)
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
	n.ext.Typ = n.typ.irType()
	return &n.ext
}

func (n *valueRef) resolveType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	var ok bool
	n.typ, ok = fetchType(scope, scope.namespace(), n.ext.Src)
	return n.typ, ok
}

func (n *valueRef) String() string {
	return n.ext.Src.Name
}
