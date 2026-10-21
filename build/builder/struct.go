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
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/src/gx/ast"
)

// structType represents a structure type.
type structType struct {
	src *ast.StructType

	nameToField map[string]*field
	fieldList   *fieldList
}

var _ typeExprNode = (*structType)(nil)

func processStructType(own typeProcScope, src *ast.StructType, fieldNS *fieldNamespace) (*structType, bool) {
	if fieldNS == nil {
		return nil, own.Err().Appendf(src, "structure declaration not supported in this context")
	}
	n := &structType{
		src:         src,
		nameToField: fieldNS.names,
	}
	var ok bool
	n.fieldList, ok = processFieldList(own, src.Fields, fieldNS.assignField)
	return n, ok
}

func (n *structType) assign(block procScope, fld *field) bool {
	name := fld.src.Name
	if prev, ok := n.nameToField[name]; ok {
		block.Err().Appendf(
			fld.src,
			"%s redeclared in this block\n\t%s: other declaration of %s",
			name,
			fmterr.At(block.Err().FSet().FSet, prev.src),
			name,
		)
		return false
	}
	n.nameToField[name] = fld
	return true
}

func (n *structType) source() ast.Node {
	return n.src
}

func (n *structType) buildTypeExpr(scope resolveScope) (*ir.TypeValExpr, bool) {
	ext := &ir.StructType{BaseType: ir.BaseType[*ast.StructType]{Src: n.src}}
	ephemeral, ephemeralOk := newEphemeralResolveScope(scope, n.src)
	stypeScope := newDefineScope(ephemeral, nil)
	var fieldsOk bool
	ext.Fields, fieldsOk = n.fieldList.buildFieldList(stypeScope)
	return ir.TypeExpr(nil, ext), fieldsOk && ephemeralOk
}

func (n *structType) String() string {
	return "struct"
}

// ----------------------------------------------------------------------------
// Structure literal.

type structResolveScope struct {
	resolveScope
	sType *ir.StructType

	bodyCE *compileEvaluator
}

func compEvalForStructType(rscope compEvalScope, src ast.Node, gtype *ir.StructType) (*compileEvaluator, bool) {
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return nil, false
	}
	if gtype.Generics() == nil {
		return compEval, true
	}
	// Define all type parameters as generic.
	sigMap, ok := evalGenericValues(compEval, gtype)
	if !ok {
		return nil, false
	}
	return compEval.sub(nil, sigMap)
}

func newStructScope(rscope resolveScope, sType *ir.StructType) (*structResolveScope, bool) {
	s := &structResolveScope{
		resolveScope: rscope,
		sType:        sType,
	}
	if sType == nil {
		return s, true
	}
	var ok bool
	s.bodyCE, ok = compEvalForStructType(rscope, sType.BaseType.Node(), sType)
	return s, ok
}

func (s *structResolveScope) nspace() *scope.RWScope[ir.Element] {
	return s.bodyCE.fitp.Context().Scope()
}

func (s *structResolveScope) compEval() (*compileEvaluator, bool) {
	return s.bodyCE, true
}

func (s *structResolveScope) evalAxisExpr(x ir.Expr) (ir.Expr, bool) {
	return x, true
}

func (s *structResolveScope) update(stor ir.Storage, el ir.Element) bool {
	return true
}

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
	fieldNS := &fieldNamespace{names: make(map[string]*field)}
	typScope := defaultTypeProcScope(pscope)
	var typOk bool
	n.typ, typOk = processTypeExpr(typScope, typeExpr, fieldNS)
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
					fmterr.At(pscope.Err().FSet().FSet, prev.ident).String(),
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

func (n *structLiteral) buildFieldValue(defScope *defineLocalScope, field *ir.Field) (ir.Expr, bool) {
	name := field.Name.Name
	valueFieldIndex, hasValue := n.nameToElt[name]
	if !hasValue {
		return nil, true
	}
	fieldNameExpr := n.fields[valueFieldIndex]
	x, valueOk := buildExpr(defScope, fieldNameExpr.expr)
	if valueOk {
		valueOk = checkSingleValue(defScope, fieldNameExpr.expr.source(), x)
	}
	if !valueOk {
		return x, false
	}
	x, valueOk = castNilAndNumber(defScope, x, field.Type())
	if !valueOk {
		return x, false
	}
	valueOk = assignableToAt(defScope, fieldNameExpr.ident, x.Type(), field.Type())
	return x, valueOk
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
		return ext, rscope.Err().Appendf(n.source(), "%s (type %T) is not a structure type", ext.Typ.ReferString(rscope.fileScope().irFile()), underlying.Type().ReferString(rscope.fileScope().irFile()))
	}
	stScope, ok := newStructScope(rscope, structType)
	if !ok {
		return ext, false
	}
	defScope := newDefineScope(stScope, nil)
	ext.Elts = make([]*ir.FieldLit, structType.Fields.Len())
	fieldsOk := true
	for i, field := range structType.Fields.Fields() {
		if !ir.ValidIdent(field.Name) {
			continue
		}
		fieldLit := &ir.FieldLit{FieldStorage: field.Storage()}
		var valueOk bool
		fieldLit.X, valueOk = n.buildFieldValue(defScope, field)
		ext.Elts[i] = fieldLit
		fieldsOk = fieldsOk && valueOk
	}
	for _, fieldNameExpr := range n.fields {
		if field := structType.Fields.FindField(fieldNameExpr.ident.Name); field == nil {
			from := rscope.fileScope().irFile()
			return ext, stScope.Err().Appendf(fieldNameExpr.ident, "type %s has no field or method %s", ext.Typ.ReferString(from), fieldNameExpr.ident.Name)
		}
	}
	return ext, fieldsOk

}

func (n *structLiteral) String() string {
	return n.typ.String()
}
