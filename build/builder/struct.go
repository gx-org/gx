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
	"github.com/gx-org/gx/build/ir/irkind"
)

// structType represents a structure type.
type structType struct {
	src *ast.StructType

	nameToField map[string]*field
	fields      []*field
	fieldList   *fieldList
}

var _ typeExprNode = (*structType)(nil)

func processStructType(own typeProcScope, src *ast.StructType) (*structType, bool) {
	n := &structType{
		src:         src,
		nameToField: make(map[string]*field),
	}
	var ok bool
	n.fieldList, ok = processFieldList(own, src.Fields, func(block procScope, field *field) bool {
		return n.assign(block, field)
	})
	return n, ok
}

func (n *structType) assign(block procScope, fld *field) bool {
	name := fld.src.Name
	if prev, ok := n.nameToField[name]; ok {
		block.Err().Appendf(
			fld.src,
			"%s redeclared in this block\n\t%s: other declaration of %s",
			name,
			fmterr.PosString(block.Err().FSet().FSet, prev.src.Pos()),
			name,
		)
		return false
	}
	n.nameToField[name] = fld
	n.fields = append(n.fields, fld)
	return true
}

func (n *structType) source() ast.Node {
	return n.src
}

func (n *structType) buildTypeExpr(scope resolveScope) (*ir.TypeValExpr, bool) {
	ext := &ir.StructType{BaseType: ir.BaseType[*ast.StructType]{Src: n.src}}
	ephemeral, ok := newEphemeralResolveScope(scope, n.src)
	if !ok {
		return ir.TypeExpr(nil, ir.InvalidType()), false
	}
	stypeScope := newDefineScope(ephemeral, nil, nil)
	var fieldsOk bool
	ext.Fields, fieldsOk = n.fieldList.buildFieldList(stypeScope)
	return ir.TypeExpr(nil, ext), fieldsOk
}

func (n *structType) String() string {
	return "struct"
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
		src       *ast.CompositeLit
		nameToElt map[string]int
		typ       typeExprNode
		fields    []fieldExpr
	}
)

func processCompositeLitStruct(pscope procScope, src *ast.CompositeLit, typeExpr ast.Expr) (exprNode, bool) {
	n := &structLiteral{
		src:       src,
		fields:    make([]fieldExpr, len(src.Elts)),
		nameToElt: make(map[string]int),
	}
	typScope := defaultTypeProcScope(pscope)
	var typOk bool
	n.typ, typOk = processTypeExpr(typScope, typeExpr)
	eltsOk := true
	for i, elt := range src.Elts {
		switch eltT := elt.(type) {
		case *ast.KeyValueExpr:
			field, fieldOk := n.processKeyValueField(pscope, eltT)
			eltsOk = fieldOk && eltsOk
			if prevIndex, prevOk := n.nameToElt[field.ident.Name]; prevOk {
				eltsOk = false
				prev := n.fields[prevIndex]
				pscope.Err().Appendf(
					field.ident,
					"%s redefined in this literal\n\t%s: other declaration of %s",
					field.ident.Name,
					fmterr.PosString(pscope.Err().FSet().FSet, prev.ident.Pos()),
					field.ident.Name,
				)
				continue
			}
			n.nameToElt[field.ident.Name] = i
			n.fields[i] = field
		default:
			pscope.Err().Appendf(elt, "literal element type %T not supported", elt)
			eltsOk = false
		}
	}
	return n, typOk && eltsOk
}

func (n *structLiteral) processKeyValueField(pscope procScope, expr *ast.KeyValueExpr) (fieldExpr, bool) {
	field := fieldExpr{}
	var ok bool
	field.expr, ok = processExpr(pscope, expr.Value)
	key := expr.Key
	switch keyT := key.(type) {
	case *ast.Ident:
		field.ident = keyT
	default:
		pscope.Err().Appendf(key, "literal element key %T not supported", key)
		ok = false
	}
	return field, ok
}

func (n *structLiteral) source() ast.Node {
	return n.src
}

func (n *structLiteral) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	ext := &ir.StructLitExpr{Src: n.src, Typ: ir.InvalidType()}
	typeExpr, typOk := n.typ.buildTypeExpr(rscope)
	if !typOk {
		return ext, false
	}
	ext.Typ = typeExpr.Val()
	underlying := ir.Underlying(ext.Typ)
	structType, ok := underlying.(*ir.StructType)
	if !ok {
		return ext, rscope.Err().Appendf(n.source(), "%s (type %T) is not a structure type", ext.Typ.String(), underlying.Type().String())
	}
	ext.Elts = make([]*ir.FieldLit, structType.Fields.Len())
	fieldsOk := true
	for i, field := range structType.Fields.Fields() {
		name := field.Name.Name
		valueFieldIndex, valueOk := n.nameToElt[name]
		if !valueOk {
			fieldsOk = rscope.Err().Appendf(ext.Src, "field %s has not been assigned", name)
			continue
		}
		fieldLit := &ir.FieldLit{FieldStorage: field.Storage()}
		ext.Elts[i] = fieldLit
		fieldNameExpr := n.fields[valueFieldIndex]
		fieldLit.X, valueOk = buildAExpr(rscope, fieldNameExpr.expr)
		if !valueOk {
			fieldsOk = false
			continue
		}
		if irkind.IsNumber(fieldLit.X.Type().Kind()) {
			fieldLit.X, valueOk = castNumber(rscope, fieldLit.X, field.Type())
			if !valueOk {
				fieldsOk = false
				continue
			}
		}
		valueOk = assignableToAt(rscope, fieldNameExpr.ident, fieldLit.X.Type(), field.Type())
		fieldsOk = valueOk && fieldsOk
	}
	for _, fieldNameExpr := range n.fields {
		if field := structType.Fields.FindField(fieldNameExpr.ident.Name); field == nil {
			return ext, rscope.Err().Appendf(fieldNameExpr.ident, "type %s has no field or method %s", ext.Typ.String(), fieldNameExpr.ident.Name)
		}
	}
	return ext, fieldsOk

}

func (n *structLiteral) String() string {
	return n.typ.String()
}
