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

	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
)

// IdentAST returns an identifier.
func IdentAST(n string) *ast.Ident {
	return &ast.Ident{Name: n}
}

// Ident returns a reference to a value.
func Ident(store ir.Storage) *ir.Ident {
	return &ir.Ident{
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
		Src: IdentAST(name),
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

// Field returns a field group of one field given a name and type.
func Field(name string, typ ir.Type, grp *ir.FieldGroup) *ir.Field {
	if grp == nil {
		grp = &ir.FieldGroup{
			Type: ir.TypeExpr(nil, typ),
		}
	}
	field := &ir.Field{
		Name:  &ast.Ident{Name: name},
		Group: grp,
	}
	field.Group.Fields = append(field.Group.Fields, field)
	return field
}

func cloneField(grp *ir.FieldGroup, field *ir.Field) *ir.Field {
	ret := *field
	ret.Group = grp
	ret.Orig = field
	return &ret
}

func cloneGroup(grp *ir.FieldGroup) *ir.FieldGroup {
	ret := *grp
	ret.Fields = make([]*ir.Field, len(grp.Fields))
	for i, field := range grp.Fields {
		ret.Fields[i] = cloneField(&ret, field)
	}
	return &ret
}

// CloneFields clones a list of fields
func CloneFields(list *ir.FieldList) *ir.FieldList {
	ret := *list
	ret.List = make([]*ir.FieldGroup, len(list.List))
	for i, grp := range list.List {
		ret.List[i] = cloneGroup(grp)
	}
	return &ret
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
				Name: IdentAST(valT),
			})
		case ir.Type:
			group := fieldGroup(fields, ir.TypeExpr(nil, valT))
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
		case *ir.GenericNonTypeParam:
			list.List = append(list.List, valT.OrigField().Group)
		default:
			panic(fmt.Sprintf("%T not supported", valT))
		}
	}
	return list
}

// FieldLit builds a field literal from a field name and a value.
func FieldLit(fields *ir.FieldList, name string, val ir.Expr) *ir.FieldLit {
	field := fields.FindField(name)
	if field == nil {
		field = &ir.Field{Name: IdentAST(fmt.Sprintf("ERROR: FIELD %q NOT FOUND", name))}
	}
	return &ir.FieldLit{
		FieldStorage: field.Storage(),
		X:            val,
	}

}

// AxisLenName returns a new axis group given a name.
func AxisLenName(name string) *ir.GenericNonTypeParam {
	field := Field(name, ir.IntLenType(), nil)
	return ir.NewGenericNonTypeParam(field)
}

// AxisGroup returns a new axis group given a name.
func AxisGroup(name string) *ir.GenericNonTypeParam {
	field := Field(name, ir.IntLenSliceType(), nil)
	return ir.NewGenericNonTypeParam(field)
}

// UnpackAxes unpacks a storage.
func UnpackAxes(ax any) *ir.UnpackExpr {
	switch axT := ax.(type) {
	case ir.Storage:
		return &ir.UnpackExpr{
			X: &ir.Ident{
				Src:  axT.NameDef(),
				Stor: axT,
			},
			EltTyp: ir.IntLenType(),
		}
	case *ir.Field:
		return UnpackAxes(axT.Storage())
	}
	panic(fmt.Sprintf("cannot unpack %T for an axis", ax))
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
	case ir.Expr:
		return &ir.AxisExpr{X: axisT}
	case *ir.Field:
		var x ir.Expr = &ir.Ident{Src: axisT.Name, Stor: axisT.Storage()}
		return &ir.AxisExpr{X: x}
	case *ir.GenericNonTypeParam:
		return &ir.AxisExpr{
			X: &ir.Ident{Src: axisT.NameDef(), Stor: axisT},
		}
	}
	panic(fmt.Sprintf("axis type %T not supported", ax))
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

// SliceType returns a new slice type.
func SliceType(dtype *ir.TypeValExpr, rank int) *ir.SliceType {
	return &ir.SliceType{
		BaseType: ir.BaseType[ast.Expr]{Src: &ast.ArrayType{}},
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
func FloatNumberAs(val float64, typ ir.Type) ir.Expr {
	return &ir.NumberCastExpr{
		X:   &ir.NumberFloat{Val: big.NewFloat(val)},
		Typ: typ,
	}
}

// IntNumberAs returns a integer number casted to a target type.
func IntNumberAs(val int64, typ ir.Type) ir.Expr {
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
			VName: IdentAST(name),
		}
	}
	return spec
}

// ConstSpec returns the specification given an optional type and constant expressions.
func ConstSpec(typ ir.Type, exprs ...*ir.ConstExpr) *ir.ConstSpec {
	var typeExpr *ir.TypeValExpr
	if typ != nil {
		typeExpr = ir.TypeExpr(nil, typ)
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
func TypeSet(types ...ir.Type) *ir.Interface {
	return ir.NewInterface(&ast.InterfaceType{}, types, nil)
}

// StructType builds a structure type.
func StructType(fields ...*ir.Field) *ir.StructType {
	stype := &ir.StructType{
		BaseType: ir.BaseType[*ast.StructType]{Src: &ast.StructType{}},
		Fields:   &ir.FieldList{},
	}
	done := make(map[*ir.FieldGroup]bool)
	for i, field := range fields {
		if done[field.Group] {
			continue
		}
		stype.Fields.List = append(stype.Fields.List, field.Group)
		done[field.Group] = true
		field.Pos = i
	}
	return stype
}

// FTypeOption modifies function types.
type FTypeOption func(ftype *ir.FuncType) *ir.FuncType

// FuncType builds a compeval function type.
func FuncType(types, recv, params, results *ir.FieldList, opts ...FTypeOption) *ir.FuncType {
	if params == nil {
		params = &ir.FieldList{}
	}
	ftype := &ir.FuncType{
		BaseType:   ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{}},
		TypeParams: types,
		Receiver:   recv,
		Params:     params,
		Results:    results,
	}
	ftype.GenericValues = make([]ir.GenericValue, ftype.TypeParams.Len())
	for _, opt := range opts {
		ftype = opt(ftype)
	}
	return ftype
}

// FuncTypeFrom builds a function type from an existing type.
func FuncTypeFrom(orig *ir.FuncType, opts ...FTypeOption) *ir.FuncType {
	ret := *orig
	ftype := &ret
	ftype.GenericValues = make([]ir.GenericValue, ftype.TypeParams.Len())
	for _, opt := range opts {
		ftype = opt(ftype)
	}
	return ftype
}

func cloneFType(ftype *ir.FuncType) *ir.FuncType {
	res := *ftype
	return &res
}

type typeParamSetter struct {
	vals []any
}

func (ts *typeParamSetter) set(ftype *ir.FuncType) *ir.FuncType {
	fields := ftype.TypeParams.Fields()
	if len(ts.vals) != len(fields) {
		panic(fmt.Sprintf("expect %d values to set %d fields but got %d values", len(fields), len(fields), len(ts.vals)))
	}
	genVals := make([]ir.GenericValue, ftype.Origin().TypeParams.Len())
	for i, field := range fields {
		if ir.IsAxisSpecType(field.Type()) {
			genVals[i] = setNonTypeParam(field, ts.vals[i])
		} else {
			genVals[i] = ir.NewTypeGenericValue(
				ir.NewGenericTypeParam(field),
				ts.vals[i].(ir.Type),
			)
		}
	}
	res := cloneFType(ftype)
	res.TypeParams = nil
	res.GenericValues = genVals
	return res
}

func setAxisLengthSlice(genAxis *ir.GenericNonTypeParam, vals []int) ir.GenericValue {
	slice := &ir.SliceLitExpr{
		Typ:  ir.IntLenSliceType(),
		Elts: make([]ir.Expr, len(vals)),
	}
	for i, val := range vals {
		slice.Elts[i] = IntNumberAs(int64(val), ir.IntLenType())
	}
	return ir.NewAxisGenericValue(genAxis, slice)
}

func setAxisLength(genAxis *ir.GenericNonTypeParam, val int) ir.GenericValue {
	return ir.NewAxisGenericValue(genAxis, IntNumberAs(int64(val), ir.IntLenType()))
}

func setAxisLengthFromField(genAxis *ir.GenericNonTypeParam, val ir.Storage) ir.GenericValue {
	return ir.NewAxisGenericValue(genAxis, &ir.Ident{
		Src:  val.NameDef(),
		Stor: val,
	})
}

func setNonTypeParam(field *ir.Field, val any) ir.GenericValue {
	genAxis := ir.NewGenericNonTypeParam(field)
	switch valT := val.(type) {
	case int:
		return setAxisLength(genAxis, valT)
	case []int:
		return setAxisLengthSlice(genAxis, valT)
	case *ir.Field:
		return setAxisLengthFromField(genAxis, valT.Storage())
	case ir.Storage:
		return setAxisLengthFromField(genAxis, valT)
	case *ir.UnpackExpr:
		return ir.NewAxisGenericValue(genAxis, valT)
	default:
		panic(fmt.Sprintf("cannot build a function type option: type %T not supported", valT))
	}
}

// SetTypeParams returns an option to set type params in a function type.
func SetTypeParams(vals ...any) FTypeOption {
	return (&typeParamSetter{vals: vals}).set
}

// CompEvalFuncType builds a compeval function type.
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
	x := &ir.Ident{Src: id, Stor: fn}
	return ir.NewFuncValExpr(x, fn)
}

// FuncExpr builds a function expression from a function.
func FuncExpr(fn ir.Func) *ir.FuncValExpr {
	return ir.NewFuncValExpr(Ident(fn.(ir.Storage)), fn)
}
