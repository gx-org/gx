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

// Package irhelper provides helper functions to build IR programmatically.
package irhelper

import (
	"fmt"
	"go/ast"
	"math/big"
	"strings"

	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

// Ident returns an identifier.
func Ident(n string) *ast.Ident {
	return &ast.Ident{Name: n}
}

// ValueRef returns a reference to a value.
func ValueRef(store ir.Storage) *ir.ValueRef {
	return &ir.ValueRef{
		Src:  store.NameDef(),
		Stor: store,
	}
}

// Block returns a block of statement.
func Block(stmts ...ir.Stmt) *ir.BlockStmt {
	return &ir.BlockStmt{List: stmts}
}

type testAnnotations struct {
	anns annotations.Annotations
}

func (ta *testAnnotations) ShortString() string {
	return "irhelper.annotations"
}
func (ta *testAnnotations) Annotations() *annotations.Annotations {
	return &ta.anns
}

// Annotations builds an annotation set.
func Annotations[T fmt.Stringer](key annotations.Key, val T) annotations.Annotations {
	var anns testAnnotations
	annotations.Set(&anns, key, val)
	return anns.anns
}

// LocalVar returns a local variable storage given a name and a type.
func LocalVar(name string, typ ir.Type) *ir.LocalVarStorage {
	return &ir.LocalVarStorage{
		Src: Ident(name),
		Typ: typ,
	}
}

func fieldGroup(fields []*ir.Field, typeExpr *ir.TypeValExpr) *ir.FieldGroup {
	group := &ir.FieldGroup{
		Fields: fields,
		Type:   typeExpr,
	}
	for _, field := range fields {
		field.Group = group
	}
	return group
}

func toTypeRef(typ ir.Type) *ir.TypeValExpr {
	switch typ.Kind() {
	case irkind.Slice:
		return &ir.TypeValExpr{X: typ, Typ: typ}
	default:
		return ir.AtomTypeExpr(typ)
	}
}

// Field returns a field group of one field given a name and type.
func Field(name string, typ ir.Type, grp *ir.FieldGroup) *ir.Field {
	if grp == nil {
		grp = &ir.FieldGroup{
			Type: toTypeRef(typ),
		}
	}
	field := &ir.Field{
		Name:  &ast.Ident{Name: name},
		Group: grp,
	}
	field.Group.Fields = append(field.Group.Fields, field)
	return field
}

// Fields returns a list of fields.
func Fields(vals ...any) *ir.FieldList {
	list := &ir.FieldList{Src: &ast.FieldList{}}
	var fields []*ir.Field
	for _, val := range vals {
		switch valT := val.(type) {
		case string:
			if valT == "" {
				continue
			}
			fields = append(fields, &ir.Field{
				Name: Ident(valT),
			})
		case ir.Type:
			group := fieldGroup(fields, ir.AtomTypeExpr(valT))
			list.List = append(list.List, group)
			fields = []*ir.Field{}
		case *ir.TypeValExpr:
			group := fieldGroup(fields, valT)
			list.List = append(list.List, group)
			fields = []*ir.Field{}
		case *ir.FieldGroup:
			list.List = append(list.List, valT)
		case *ir.Field:
			list.List = append(list.List, valT.Group)
		default:
			panic(fmt.Sprintf("%T not supported", valT))
		}
	}
	return list
}

// FieldLit builds a field literal from a field name and a value.
func FieldLit(fields *ir.FieldList, name string, val ir.AssignableExpr) *ir.FieldLit {
	field := fields.FindField(name)
	if field == nil {
		field = &ir.Field{Name: Ident(fmt.Sprintf("ERROR: FIELD %q NOT FOUND", name))}
	}
	return &ir.FieldLit{
		FieldStorage: field.Storage(),
		X:            val,
	}

}

// AxisLenName returns a new axis group given a name.
func AxisLenName(name string) *ir.AxisStmt {
	return &ir.AxisStmt{
		Src: &ast.Ident{Name: name},
		Typ: ir.IntLenType(),
	}
}

// AxisGroup returns a new axis group given a name.
func AxisGroup(name string) *ir.AxisStmt {
	return &ir.AxisStmt{
		Src: &ast.Ident{Name: name},
		Typ: ir.IntLenSliceType(),
	}
}

// Axis builds an array axis length.
func Axis(ax any) ir.AxisLengths {
	switch axisT := ax.(type) {
	case int:
		return &ir.AxisExpr{
			X: IntNumberAs(int64(axisT), ir.IntLenType()),
		}
	case ir.AxisLengths:
		return axisT
	case ir.AssignableExpr:
		return &ir.AxisExpr{Src: axisT.Node().(ast.Expr), X: axisT}
	case string:
		if strings.HasPrefix(axisT, ir.DefineAxisGroup) {
			name := strings.TrimPrefix(axisT, ir.DefineAxisGroup)
			return AxisGroup(name)
		}
		if strings.HasSuffix(axisT, ir.DefineAxisGroup) {
			name := strings.TrimSuffix(axisT, ir.DefineAxisGroup)
			return &ir.AxisExpr{
				X: &ir.ValueRef{
					Src:  &ast.Ident{Name: name},
					Stor: AxisGroup(name),
				},
			}
		}
		if strings.HasSuffix(axisT, ir.DefineAxisLength) {
			name := strings.TrimSuffix(axisT, ir.DefineAxisLength)
			return &ir.AxisExpr{
				X: &ir.ValueRef{
					Src:  &ast.Ident{Name: name},
					Stor: AxisLenName(name),
				},
			}
		}
		panic(fmt.Sprintf("axis string %q not supported: pass a storage", axisT))
	}
	panic(fmt.Sprintf("axis type %T not supported", ax))
}

// TypeRef returns a reference to a type that defines itself
// (e.g. []intlen which defines a slice of axis lengths
// as opposed to intlen which references the axis length type).
func TypeRef(typ ir.Type) *ir.TypeValExpr {
	return &ir.TypeValExpr{X: typ, Typ: typ}
}

// ArrayType returns a new array type given a list of expressions to specify axis lengths.
func ArrayType(dtype ir.Type, axes ...any) ir.ArrayType {
	if len(axes) == 1 {
		rnk, rnkOk := axes[0].(ir.ArrayRank)
		if rnkOk {
			return ir.NewArrayType(&ast.ArrayType{}, dtype, rnk)
		}
	}
	rank := &ir.Rank{}
	for _, ax := range axes {
		rank.Ax = append(rank.Ax, Axis(ax))
	}
	return ir.NewArrayType(&ast.ArrayType{}, dtype, rank)
}

// TypeExpr encapsulates a type in an expression.
func TypeExpr(typ ir.Type) *ir.TypeValExpr {
	return &ir.TypeValExpr{X: typ, Typ: typ}
}

// SliceType returns a new slice type.
func SliceType(dtype *ir.TypeValExpr, rank int) *ir.SliceType {
	return &ir.SliceType{
		BaseType: ir.BaseType[*ast.ArrayType]{Src: &ast.ArrayType{}},
		DType:    dtype,
		Rank:     rank,
	}
}

// InferArrayType returns a new array type given a list of expressions to specify axis lengths.
func InferArrayType(dtype ir.Type, axes ...any) ir.ArrayType {
	rank := &ir.Rank{}
	for _, ax := range axes {
		rank.Ax = append(rank.Ax, &ir.AxisInfer{
			X: Axis(ax),
		})
	}
	return ir.NewArrayType(&ast.ArrayType{}, dtype, &ir.RankInfer{Rnk: rank})
}

// FloatNumberAs returns a integer number casted to a target type.
func FloatNumberAs(val float64, typ ir.Type) ir.AssignableExpr {
	return &ir.NumberCastExpr{
		X:   &ir.NumberFloat{Val: big.NewFloat(val)},
		Typ: typ,
	}
}

// IntNumberAs returns a integer number casted to a target type.
func IntNumberAs(val int64, typ ir.Type) ir.AssignableExpr {
	return &ir.NumberCastExpr{
		X:   &ir.NumberInt{Val: big.NewInt(val)},
		Typ: typ,
	}
}

// VarSpec returns the specification for static variables.
func VarSpec(names ...string) *ir.VarSpec {
	spec := &ir.VarSpec{
		TypeV: ir.IntLenType(),
		Exprs: make([]*ir.VarExpr, len(names)),
	}
	for i, name := range names {
		spec.Exprs[i] = &ir.VarExpr{
			Decl:  spec,
			VName: Ident(name),
		}
	}
	return spec
}

// ConstSpec returns the specification given an optional type and constant expressions.
func ConstSpec(typ ir.Type, exprs ...*ir.ConstExpr) *ir.ConstSpec {
	var typeExpr *ir.TypeValExpr
	if typ != nil {
		typeExpr = &ir.TypeValExpr{X: typ, Typ: typ}
	}
	spec := &ir.ConstSpec{
		Type:  typeExpr,
		Exprs: exprs,
	}
	for _, expr := range exprs {
		expr.Decl = spec
	}
	return spec
}

// TypeSet builds a set of types.
func TypeSet(types ...ir.Type) *ir.TypeSet {
	return ir.NewTypeSet(&ast.InterfaceType{}, types)
}

// StructType builds a structure type.
func StructType(fields ...*ir.Field) *ir.StructType {
	done := make(map[*ir.FieldGroup]bool)
	list := &ir.FieldList{}
	for _, field := range fields {
		if done[field.Group] {
			continue
		}
		list.List = append(list.List, field.Group)
		done[field.Group] = true
	}
	return &ir.StructType{
		BaseType: ir.BaseType[*ast.StructType]{Src: &ast.StructType{}},
		Fields:   list,
	}
}

// FuncType builds a compeval function type .
func FuncType(types, recv, params, results *ir.FieldList) *ir.FuncType {
	if params == nil {
		params = &ir.FieldList{}
	}
	return &ir.FuncType{
		BaseType:   ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{}},
		TypeParams: types,
		Receiver:   recv,
		Params:     params,
		Results:    results,
	}
}

// AxisLengths defines axis lengths for a function type.
func AxisLengths(ftype *ir.FuncType, axes ...string) *ir.FuncType {
	ftype.AxisLengths = make([]ir.AxisValue, len(axes))
	for i, axis := range axes {
		typ := ir.IntLenType()
		name := axis
		if strings.HasPrefix(axis, ir.DefineAxisGroup) {
			typ = ir.IntLenSliceType()
			name = strings.TrimPrefix(name, ir.DefineAxisGroup)
		}
		if strings.HasPrefix(axis, ir.DefineAxisLength) {
			name = strings.TrimPrefix(name, ir.DefineAxisLength)
		}
		ftype.AxisLengths[i] = ir.AxisValue{
			Axis: &ir.AxisStmt{
				Src: Ident(name),
				Typ: typ,
			},
		}
	}
	return ftype
}

// CompEvalFuncType builds a compeval function type .
func CompEvalFuncType(params, results *ir.FieldList) *ir.FuncType {
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{}},
		Params:   params,
		Results:  results,
		CompEval: true,
	}
}

// SingleReturn returns a block statement with a single return statement in it.
func SingleReturn(exprs ...ir.Expr) *ir.BlockStmt {
	return &ir.BlockStmt{
		List: []ir.Stmt{&ir.ReturnStmt{
			Results: exprs,
		}},
	}
}

// FuncDeclCallee returns a reference to call a function given its name.
func FuncDeclCallee(name string, ftype *ir.FuncType) *ir.FuncValExpr {
	id := &ast.Ident{Name: name}
	fn := &ir.FuncDecl{
		Src:   &ast.FuncDecl{Name: id},
		FType: ftype,
	}
	x := &ir.ValueRef{Src: id, Stor: fn}
	return ir.NewFuncValExpr(x, fn)
}

// FuncExpr builds a function expression from a function.
func FuncExpr(fn ir.Func) *ir.FuncValExpr {
	return ir.NewFuncValExpr(ValueRef(fn.(ir.Storage)), fn)
}
