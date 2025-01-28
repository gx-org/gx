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

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// structType represents a structure type.
type structType struct {
	ext *ir.StructType

	nameToField map[string]*field
	fields      []*field
	fieldList   *fieldList
	typ         typeNode
}

var (
	_ selector         = (*structType)(nil)
	_ concreteTypeNode = (*structType)(nil)
)

func processStructType(own owner, expr *ast.StructType) (*structType, bool) {
	n := &structType{
		ext: &ir.StructType{
			Src: expr,
			Fields: &ir.FieldList{
				Src: expr.Fields,
			},
		},
		nameToField: make(map[string]*field),
	}
	var ok bool
	n.fieldList, ok = processFieldList(own, expr.Fields, func(block *scopeFile, field *field) bool {
		return n.assign(block, field)
	})
	return n, ok
}

func importStructType(scope scoper, ext *ir.StructType) (*structType, bool) {
	fieldList, ok := importFieldList(scope, ext.Fields)
	n := &structType{
		ext:         ext,
		nameToField: make(map[string]*field),
		fieldList:   fieldList,
	}
	for _, group := range n.fieldList.list {
		for _, field := range group.list {
			n.nameToField[field.ext.Name.Name] = field
			n.fields = append(n.fields, field)
		}
	}
	n.typ = n
	return n, ok
}

func (n *structType) assign(block *scopeFile, fld *field) bool {
	name := fld.ext.Name.Name
	if prev, ok := n.nameToField[name]; ok {
		block.err().Appendf(
			fld.ext.Name,
			"%s redeclared in this block\n\t%s: other declaration of %s",
			name,
			fmterr.PosString(block.err().FSet().FSet, prev.ext.Name.Pos()),
			name,
		)
		return false
	}
	n.nameToField[name] = fld
	n.fields = append(n.fields, fld)
	return true
}

func (n *structType) irType() ir.Type {
	n.ext.Fields = n.fieldList.irType()
	return n.ext
}

func (n *structType) convertibleTo(scope scoper, typ typeNode) (bool, error) {
	return n.irType().ConvertibleTo(scope.evalFetcher(), typ.irType())
}

func (n *structType) kind() ir.Kind {
	return n.irType().Kind()
}

func (n *structType) source() ast.Node {
	return n.ext.Source()
}

func (n *structType) isGeneric() bool {
	return false
}

func (n *structType) String() string {
	return n.ext.String()
}

func (n *structType) resolveConcreteType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	var ok bool
	n.fieldList, ok = n.fieldList.resolveType(scope)
	if !ok {
		n.typ, _ = invalidType()
		return n.typ, false
	}
	// Build GX struct type.
	n.ext.Fields = n.fieldList.irType()
	n.typ = n
	return n.typ, true
}

func (n *structType) selectField(ident *ast.Ident) *field {
	return n.nameToField[ident.Name]
}

func (n *structType) buildSelectNode(scope scoper, expr *ast.SelectorExpr) selectNode {
	field := n.selectField(expr.Sel)
	if field == nil {
		return nil
	}
	return buildFieldSelectorExpr(expr, n, field)
}

// ----------------------------------------------------------------------------
// Structure literal.

// structLiteral defines the value of a struct.
type (
	fieldExpr struct {
		ident *ast.Ident
		expr  exprNode
	}

	structLiteral struct {
		ext        ir.StructLitExpr
		nameToElt  map[string]int
		typeExpr   typeNode
		structType *structType
		fields     []fieldExpr
		typ        typeNode
	}
)

func processCompositeLitStruct(owner owner, lit *ast.CompositeLit, typeExpr ast.Expr) (exprNode, bool) {
	typeRef, ok := processTypeExpr(owner, typeExpr)
	n := &structLiteral{
		ext: ir.StructLitExpr{
			Src: lit,
		},
		typeExpr:  typeRef,
		fields:    make([]fieldExpr, len(lit.Elts)),
		nameToElt: make(map[string]int),
	}
	for i, elt := range n.ext.Src.Elts {
		switch eltT := elt.(type) {
		case *ast.KeyValueExpr:
			field, fieldOk := n.processKeyValueField(owner, eltT)
			ok = fieldOk && ok
			if prevIndex, prevOk := n.nameToElt[field.ident.Name]; prevOk {
				ok = false
				prev := n.fields[prevIndex]
				owner.err().Appendf(
					field.ident,
					"%s redefined in this literal\n\t%s: other declaration of %s",
					field.ident.Name,
					fmterr.PosString(owner.err().FSet().FSet, prev.ident.Pos()),
					field.ident.Name,
				)
				continue
			}
			n.nameToElt[field.ident.Name] = i
			n.fields[i] = field
		default:
			owner.err().Appendf(elt, "literal element type %T not supported", elt)
			ok = false
		}
	}
	return n, ok
}

func (n *structLiteral) processKeyValueField(owner owner, expr *ast.KeyValueExpr) (fieldExpr, bool) {
	field := fieldExpr{}
	var ok bool
	field.expr, ok = processExpr(owner, expr.Value)
	key := expr.Key
	switch keyT := key.(type) {
	case *ast.Ident:
		field.ident = keyT
	default:
		owner.err().Appendf(key, "literal element key %T not supported", key)
		ok = false
	}
	return field, ok
}

func (n *structLiteral) source() ast.Node {
	return n.expr()
}

func (n *structLiteral) expr() ast.Expr {
	return n.ext.Expr()
}

func (n *structLiteral) String() string {
	return n.typ.String()
}

func (n *structLiteral) buildExpr() ir.Expr {
	n.ext.Elts = make([]ir.FieldLit, len(n.fields))
	for i, field := range n.fields {
		n.ext.Elts[i] = ir.FieldLit{
			X: field.expr.buildExpr(),
		}
		structField := n.structType.selectField(field.ident)
		if structField == nil {
			continue
		}
		n.ext.Elts[i].Field = structField.ext
	}
	n.ext.Typ = n.typ.irType()
	return &n.ext
}

func (n *structLiteral) resolveFieldTypes(scope scoper) bool {
	ok := true
	for _, field := range n.structType.fields {
		var wantType typeNode
		wantType, ok = resolveType(scope, field.group, field.group.typ)
		if !ok {
			ok = false
			continue
		}
		name := field.ext.Name.Name
		valueFieldIndex, valueOk := n.nameToElt[name]
		if !valueOk {
			scope.err().Appendf(n.ext.Src, "field %s has not been assigned", name)
			ok = false
			continue
		}
		valueField := n.fields[valueFieldIndex]
		valueType, valueOk := valueField.expr.resolveType(scope)
		if !valueOk {
			ok = false
			continue
		}
		if ir.IsNumber(valueType.kind()) {
			var valueExpr exprNode
			valueExpr, valueType, valueOk = castNumber(scope, valueField.expr, wantType.irType())
			if !valueOk {
				ok = false
				continue
			}
			valueField = fieldExpr{ident: valueField.ident, expr: valueExpr}
			n.fields[valueFieldIndex] = valueField
		}
		posI := valueField.expr
		_, okI, err := assignableTo(scope, posI, valueType, wantType)
		if err != nil {
			scope.err().Append(err)
			ok = false
			continue
		}
		if !okI {
			scope.err().Appendf(valueField.expr.source(), "cannot use type %s as %s value", valueType, wantType)
			ok = false
			continue
		}
	}
	for _, valueField := range n.fields {
		if field := n.structType.selectField(valueField.ident); field == nil {
			scope.err().Appendf(valueField.ident, "type %s has no field or method %s", n.structType.String(), valueField.ident.Name)
			return false
		}
	}
	return ok
}

func (n *structLiteral) resolveType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	var ok bool
	if n.typ, ok = resolveType(scope, n, n.typeExpr); !ok {
		return n.typ, false
	}
	underlying := findUnderlying(n.typ)
	if n.structType, ok = underlying.(*structType); !ok {
		scope.err().Appendf(n.source(), "%s (type %T) is not a structure type", n.typ.String(), n.typ.irType())
		n.typ, _ = invalidType()
		return n.typ, false
	}
	return n.typ, n.resolveFieldTypes(scope)
}
