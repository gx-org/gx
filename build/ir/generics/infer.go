// Copyright 2025 Google LLC
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

package generics

import (
	"go/ast"
	"reflect"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

type (
	unifier struct {
		fetcher ir.Fetcher
		defined []ir.GenericValue
	}

	argUnifier struct {
		*unifier
		isVarArgs bool
		param     *ir.Field
		arg       ir.Expr
	}
)

var _ = (ir.Unifier)(&argUnifier{})

func (uni *unifier) specialiseRemainingNumbers() bool {
	for i, v := range uni.defined {
		if v == nil {
			// The type has not been defined with inference.
			continue
		}
		knd := v.DefinedType().Kind()
		if !irkind.IsNumber(knd) {
			continue
		}
		genericIdentType, ok := v.Generic().(*ir.GenericTypeParam)
		if !ok {
			continue
		}
		tp := ir.DefaultNumberType(knd)
		uni.defined[i] = ir.NewTypeGenericValue(genericIdentType, tp)
	}
	return true
}

func (uni *argUnifier) Node() ast.Node {
	return uni.arg.Node()
}

func (uni *argUnifier) Err() *fmterr.Appender {
	return uni.fetcher.Err()
}

func (uni *argUnifier) specialiseNumber(genType *ir.GenericTypeParam, defined ir.GenericValue, typ ir.Type) ir.Type {
	definedType := defined.DefinedType()
	if !irkind.IsNumber(definedType.Kind()) {
		return definedType
	}
	if irkind.IsNumber(typ.Kind()) {
		return definedType
	}
	if !ir.CanBeNumber(typ) {
		return definedType
	}
	genericIdentType, ok := defined.Generic().(*ir.GenericTypeParam)
	if !ok {
		return definedType
	}
	uni.defined[genType.OrigField().Pos] = ir.NewTypeGenericValue(genericIdentType, typ)
	return typ
}

func invalidGenericType(genType *ir.GenericTypeParam) *ir.TypeGenericValue {
	return ir.NewTypeGenericValue(genType, ir.InvalidType())
}

func (uni *argUnifier) DefineType(genType *ir.GenericTypeParam, typ ir.Type) bool {
	field := genType.OrigField()
	pos := field.Pos
	ok, cpErr, err := ir.AssignableTo(uni.fetcher, typ, field.Type())
	if err != nil {
		uni.defined[pos] = invalidGenericType(genType)
		return uni.fetcher.Err().AppendAt(uni.arg.Node(), err)
	}
	if cpErr != nil {
		uni.defined[pos] = invalidGenericType(genType)
		return uni.fetcher.Err().AppendAt(uni.arg.Node(), cpErr)
	}
	if !ok {
		uni.defined[pos] = invalidGenericType(genType)
		from := uni.fetcher.File()
		field := genType.OrigField()
		return uni.fetcher.Err().Appendf(uni.arg.Node(), "%s does not satisfy %s for %s", typ.ReferString(from), field.Type().ReferString(from), field.Name.Name)
	}
	defined := uni.defined[pos]
	if defined == nil {
		uni.defined[pos] = ir.NewTypeGenericValue(genType, typ)
		return true
	}
	definedType := uni.specialiseNumber(genType, defined, typ)
	if !ir.IsValid(definedType) || !ir.IsValid(typ) {
		return false
	}
	eq, cpErr, err := typ.AssignableTo(uni.fetcher, definedType)
	if err != nil {
		return uni.fetcher.Err().AppendAt(uni.arg.Node(), err)
	}
	if cpErr != nil {
		return uni.fetcher.Err().AppendAt(uni.arg.Node(), cpErr)
	}
	if !eq {
		from := uni.fetcher.File()
		return uni.fetcher.Err().Appendf(uni.arg.Node(), "type %s does not match type %s for %s", typ.ReferString(from), definedType.ReferString(from), genType.OrigField().Name.Name)
	}
	return true
}

func (uni *argUnifier) defineVarArgsNonType(genAxis *ir.GenericNonTypeParam, expr ir.Expr) bool {
	nonTypeSlice, isSlice := ir.Underlying(genAxis.Type()).(*ir.SliceType)
	if !isSlice {
		from := uni.fetcher.File()
		elType := expr.Type().ReferString(from)
		return uni.fetcher.Err().Appendf(uni.arg.Node(), "cannot assign %s to %s (expect []%s)", elType, genAxis.Type().ReferString(from), elType)
	}
	sliceElement, ok := nonTypeSlice.ElementType()
	if !ok {
		return false
	}
	if !AssignTo(uni.fetcher, expr, genAxis, sliceElement) {
		return false
	}
	pos := genAxis.OrigField().Pos
	defined := uni.defined[pos]
	var sliceLit *ir.SliceLitExpr
	if defined == nil {
		sliceLit = &ir.SliceLitExpr{
			Src:  expr.Expr(),
			Typ:  genAxis.Type(),
			Elts: []ir.Expr{},
		}
		uni.defined[pos] = ir.NewAxisGenericValue(genAxis, sliceLit)
	} else {
		var sliceLitOk bool
		sliceLit, sliceLitOk = defined.Value().(*ir.SliceLitExpr)
		if !sliceLitOk {
			return uni.fetcher.Err().AppendInternalf(uni.arg.Node(), "cannot convert %T to %s", defined.Value(), reflect.TypeFor[*ir.SliceLitExpr]().String())
		}
	}
	sliceLit.Elts = append(sliceLit.Elts, expr)
	return true
}

func (uni *argUnifier) DefineNonType(genAxis *ir.GenericNonTypeParam, expr ir.Expr) bool {
	if uni.isVarArgs {
		return uni.defineVarArgsNonType(genAxis, expr)
	}
	if !AssignTo(uni.fetcher, expr, genAxis, genAxis.Type()) {
		return false
	}
	pos := genAxis.OrigField().Pos
	defined := uni.defined[pos]
	if defined == nil {
		uni.defined[pos] = ir.NewAxisGenericValue(genAxis, expr)
		return true
	}
	nonType, isNonType := defined.(*ir.NonTypeGenericValue)
	if !isNonType {
		from := uni.fetcher.File()
		return uni.fetcher.Err().AppendInternalf(uni.arg.Node(), "cannot compare value %s to type parameter %s (type: %s)", expr.SourceString(from), defined.Name(), defined.DefinedType().ReferString(from))
	}
	isEqual, cpErr, err := nonType.Equal(uni.fetcher, expr)
	if uniErr := ir.UnifyErr(cpErr, err); uniErr != nil {
		return uni.fetcher.Err().AppendAt(uni.arg.Node(), err)
	}
	if !isEqual {
		from := uni.fetcher.File()
		return uni.fetcher.Err().Appendf(uni.arg.Node(), "%s does not match inferred value %s for %s", expr.SourceString(from), nonType.Value().SourceString(from), nonType.Generic().NameDef().Name)
	}
	return true
}

func typeParametersMap(tparams *ir.FieldList) map[string]*ir.Field {
	fields := make(map[string]*ir.Field)
	for _, field := range tparams.Fields() {
		if !ir.ValidIdent(field.Name) {
			continue
		}
		fields[field.Name.Name] = field
	}
	return fields
}

// Infer the type parameters of a function given a list of argument expressions.
func Infer(fetcher ir.Fetcher, fExpr *ir.FuncValExpr, args []ir.Expr) (*ir.FuncValExpr, bool) {
	ftype := fExpr.FuncType()
	if ftype.TypeParams.Len() == 0 {
		// Nothing left to infer.
		return fExpr, true
	}
	uni := &unifier{
		fetcher: fetcher,
		defined: append([]ir.GenericValue{}, ftype.GenericValues...),
	}
	ok := true
	params := ftype.Params.Fields()
	for i := range max(len(args), len(params)) {
		param, isVarArgs := ftype.ArgIndexToParamField(i)
		argUni := &argUnifier{unifier: uni, param: param, arg: args[i], isVarArgs: isVarArgs}
		toUnify := param.Type()
		if isVarArgs {
			toUnify = ftype.VarArgs.Typ.DType.Val()
		}
		if argOk := toUnify.UnifyWith(argUni, argUni.arg.Type()); !argOk {
			ok = false
		}
	}
	if !ok {
		return fExpr, false
	}
	if !uni.specialiseRemainingNumbers() {
		return fExpr, false
	}
	spec := newSpecialiser(fetcher, fExpr.Node(), ftype, uni.defined)
	ftypeInfer := ftype.SpecialiseFType(spec, true)
	checkTypeParams(ftypeInfer, ftypeInfer.GenericValues)
	return fExpr.NewFType(ftypeInfer), ok
}
