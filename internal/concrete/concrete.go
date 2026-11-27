// Copyright 2026 Google LLC
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

// Package concrete computes concrete types from generic types.
package concrete

import (
	"fmt"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/internal/undef"
)

// TypeElement is an element encapsulating a type.
// For example, the value of a generic type parameter in a frame (elements.GenericType).
type TypeElement interface {
	ir.Element
	Value() ir.Type
}

func isConcreteIntLiteral(expr ir.Expr) bool {
	switch exprT := expr.(type) {
	case *ir.NumberInt:
		return true
	case *ir.NumberCastExpr:
		_, isInt := exprT.X.(*ir.NumberInt)
		return isInt && irkind.IsConcrete(exprT.Typ.Kind())
	}
	return false
}

func canConvertToExpr(element ir.Element) bool {
	bare := ir.BareValue(element)
	if _, ok := bare.(ir.WithExpr); !ok {
		return false
	}
	if container, ok := bare.(coreiface.WithElements); ok {
		for _, child := range container.Elements() {
			if !canConvertToExpr(child) {
				return false
			}
		}
	}
	return true
}

func concreteTypeParam(fr ir.Evaluator, tp *ir.GenericTypeParam) (ir.Type, error) {
	typeExpr := &ir.Ident{
		Src:  tp.OrigField().Name,
		Stor: tp,
	}
	el, err := fr.EvalExpr(typeExpr)
	if err != nil {
		if undef.Is(err) {
			return tp, nil
		}
		return nil, fmterr.Internalf("cannot cast to %s: %v", tp.ReferString(fr.File()), err)
	}
	typeEl, isType := ir.BareValue(el).(TypeElement)
	if !isType {
		return nil, fmterr.Internalf("element %T is not a type", el)
	}
	return typeEl.Value(), nil
}

func evalAxisExpr(fr ir.Evaluator, axis ir.AxisLengths, x ir.Expr) ([]ir.AxisLengths, error) {
	if isConcreteIntLiteral(x) {
		return []ir.AxisLengths{axis}, nil
	}
	el, err := fr.EvalExpr(x)
	if err != nil {
		if undef.Is(err) {
			return []ir.AxisLengths{axis}, nil
		}
		return []ir.AxisLengths{axis}, err
	}
	if !canConvertToExpr(el) {
		return []ir.AxisLengths{axis}, nil
	}
	xs, err := ir.ToExpr(fr, nil, el)
	var axlens []ir.AxisLengths
	for _, x := range xs {
		sliceLit, isSliceLit := x.(*ir.SliceLitExpr)
		if !isSliceLit {
			axlens = append(axlens, &ir.AxisExpr{X: x})
			continue
		}
		for _, elt := range sliceLit.Elts {
			axlens = append(axlens, &ir.AxisExpr{X: elt})
		}
	}
	return axlens, err
}

func concreteAxisLength(fr ir.Evaluator, axis ir.AxisLengths) ([]ir.AxisLengths, error) {
	switch axisT := axis.(type) {
	case *ir.AxisExpr:
		return evalAxisExpr(fr, axis, axisT.X)
	case *ir.AxisInfer:
		if axisT.X == nil {
			return []ir.AxisLengths{axisT}, nil
		}
		return concreteAxisLength(fr, axisT.X)
	default:
		return []ir.AxisLengths{axis}, fmterr.Internalf("cannot convert axis %s:%T to a concrete axis", axisT.SourceString(fr.File()), axisT)
	}
}

func concreteAxisLengths(fr ir.Evaluator, axes []ir.AxisLengths) ([]ir.AxisLengths, error) {
	var crs []ir.AxisLengths
	for _, ax := range axes {
		crAxis, err := concreteAxisLength(fr, ax)
		if err != nil {
			return axes, err
		}
		crs = append(crs, crAxis...)
	}
	return crs, nil
}

func concreteRank(fr ir.Evaluator, rnk ir.ArrayRank) (ir.ArrayRank, error) {
	switch rnkT := rnk.(type) {
	case *ir.Rank:
		axes, err := concreteAxisLengths(fr, rnkT.Ax)
		return &ir.Rank{Src: rnkT.Src, Ax: axes}, err
	case *ir.RankInfer:
		if rnkT.Rank() == nil {
			return rnkT, nil
		}
		return concreteRank(fr, rnkT.Rank())
	default:
		return rnk, fmterr.Internalf("cannot convert rank %s:%T to a concrete rank", rnk.SourceString(fr.File()), rnkT)
	}
}

func concreteTypeArg(fr ir.Evaluator, tp ir.Type) (ir.Type, error) {
	switch tpT := tp.(type) {
	case *ir.NamedType:
		return NamedType(fr, tpT)
	case *ir.GenericTypeParam:
		cType, err := concreteTypeParam(fr, tpT)
		if err != nil {
			return nil, err
		}
		if ntp, ok := cType.(*ir.NamedType); ok {
			return NamedType(fr, ntp)
		}
		return cType, nil
	default:
		return Concrete(fr, tp)
	}
}

// NamedType resolves generic parameter values on a named type given the context.
func NamedType(fr ir.Evaluator, ntp *ir.NamedType) (*ir.NamedType, error) {
	if len(ntp.GenParams.Values) == 0 {
		return ntp, nil
	}
	values := make([]ir.GenericValue, len(ntp.GenParams.Values))
	for i, gv := range ntp.GenParams.Values {
		if gv == nil {
			continue
		}
		switch gvT := gv.(type) {
		case *ir.TypeGenericValue:
			cType, err := concreteTypeArg(fr, gvT.DefinedType())
			if err != nil {
				return nil, err
			}
			values[i] = ir.NewTypeExprGenericValue(gvT.Generic().(*ir.GenericTypeParam), ir.TypeExpr(gvT.Value().Expr(), cType))
		case *ir.NonTypeGenericValue:
			val := gvT.Value()
			el, err := fr.EvalExpr(val)
			if err == nil {
				var expr ir.Expr
				if expr, err = ir.ToSingleExpr(fr, val.Expr(), el); err == nil {
					values[i] = ir.NewNonTypeGenericValue(gvT.Generic().(*ir.GenericNonTypeParam), expr)
					continue
				}
			}
			if !undef.Is(err) {
				return nil, err
			}
			values[i] = gv
		default:
			return nil, fmterr.Internalf("cannot compute concrete generic value of %s:%T: not supported", gv.SourceString(fr.File()), gv)
		}
	}
	cNamed := *ntp
	cNamed.GenParams.Values = values
	return &cNamed, nil
}

// concretized maps named types to their concrete types.
// It is used to terminate the computation on recursive types.
type concretized map[*ir.NamedType]*ir.NamedType

func concreteNamedType(fr ir.Evaluator, ntp *ir.NamedType, done concretized) (ir.Type, error) {
	if _, isStruct := ir.Underlying(ntp).(*ir.StructType); isStruct && len(ntp.GenParams.Values) == 0 {
		// A structure type with no generic parameters is already concrete.
		return ntp, nil
	}
	if cNamed, ok := done[ntp]; ok {
		// Recursive type: return the concrete type being computed.
		return cNamed, nil
	}
	cNamed, err := NamedType(fr, ntp)
	if err != nil {
		return nil, err
	}
	res := *cNamed
	done[ntp] = &res
	under, err := concreteType(fr, ntp.Underlying.Val(), done)
	if err != nil {
		return nil, err
	}
	res.Underlying = ir.TypeExpr(ntp.Underlying.Expr(), under)
	return &res, nil
}

// concreteStructType returns a copy of a structure type where the type of each
// field is concrete.
func concreteStructType(fr ir.Evaluator, st *ir.StructType, done concretized) (*ir.StructType, error) {
	fields := &ir.FieldList{Src: st.Fields.Src}
	for _, group := range st.Fields.List {
		groupType, err := concreteType(fr, group.Type.Val(), done)
		if err != nil {
			return nil, err
		}
		cGroup := &ir.FieldGroup{
			Src:  group.Src,
			Type: ir.TypeExpr(group.Type.Expr(), groupType),
			Anns: group.Anns,
		}
		for _, field := range group.Fields {
			cField := *field
			cField.Orig = field.Origin()
			cField.Group = cGroup
			cGroup.Fields = append(cGroup.Fields, &cField)
		}
		fields.List = append(fields.List, cGroup)
	}
	cStruct := *st
	cStruct.Fields = fields
	return &cStruct, nil
}

// concreteSliceType returns a copy of a slice type where the type of the
// elements is concrete.
func concreteSliceType(fr ir.Evaluator, st *ir.SliceType, done concretized) (*ir.SliceType, error) {
	dtype, err := concreteType(fr, st.DType.Val(), done)
	if err != nil {
		return nil, err
	}
	cSlice := *st
	cSlice.DType = ir.TypeExpr(st.DType.Expr(), dtype)
	return &cSlice, nil
}

func concrete(fr ir.Evaluator, tp ir.Type, done concretized) (ir.Type, error) {
	if typeParam, isParam := tp.(*ir.GenericTypeParam); isParam {
		return concreteTypeParam(fr, typeParam)
	}
	if irkind.IsConcrete(tp.Kind()) || tp.Kind() == irkind.String || ir.IsUnknown(tp) {
		return tp, nil
	}
	switch tpT := tp.(type) {
	case *ir.NamedType:
		return concreteNamedType(fr, tpT, done)
	case *ir.StructType:
		return concreteStructType(fr, tpT, done)
	case *ir.SliceType:
		return concreteSliceType(fr, tpT, done)
	default:
		return tp, fmterr.Internalf("cannot compute the concrete atomic type of %s:%T: not supported", tp.ReferString(fr.File()), tpT)
	}
}

func concreteArrayType(fr ir.Evaluator, tp ir.ArrayType) (ir.ArrayType, error) {
	dtype, err := Concrete(fr, tp.DataType())
	if err != nil {
		return tp, err
	}
	rank, err := concreteRank(fr, tp.Rank())
	if err != nil {
		return tp, err
	}
	return ir.NewArrayType(tp.ArrayType(), dtype, rank), nil
}

// Concrete computes the concrete type.
func Concrete(fr ir.Evaluator, tp ir.Type) (ir.Type, error) {
	return concreteType(fr, tp, concretized{})
}

func concreteType(fr ir.Evaluator, tp ir.Type, done concretized) (_ ir.Type, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot resolve type %s: %w", tp.ReferString(fr.File()), err)
		}
	}()
	arrayType, isArray := tp.(ir.ArrayType)
	if !isArray {
		return concrete(fr, tp, done)
	}
	if arrayType.Rank().IsAtomic() {
		// Avoid a infinite loop on the data type of an array.
		dtype, err := concrete(fr, arrayType.DataType(), done)
		if err != nil {
			return tp, err
		}
		if _, isParam := dtype.(*ir.GenericTypeParam); isParam {
			return ir.NewArrayType(arrayType.ArrayType(), dtype, arrayType.Rank()), nil
		}
		return dtype, nil
	}
	return concreteArrayType(fr, arrayType)
}
