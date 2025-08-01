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

package interp

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

type (
	valuer interface {
		array(fitp *FileScope, expr *ir.ArrayLitExpr) (ir.Element, error)
	}

	valuerT[T dtype.GoDataType] struct {
		kind         ir.Kind
		toAtomValue  func(tp ir.Type, val T) (*values.HostArray, error)
		toArrayValue func(tp ir.Type, val []T, dims []int) (*values.HostArray, error)
	}
)

func goValueFromElement[T dtype.GoDataType](el evaluator.NumericalElement) (T, bool, error) {
	var t T
	canonicalElt, ok := el.(elements.ElementWithConstant)
	if !ok {
		return t, false, nil
	}
	cst := canonicalElt.NumericalConstant()
	if cst == nil {
		return t, false, nil
	}
	var err error
	t, err = values.ToAtom[T](cst)
	return t, err == nil, err
}

func goSliceFromArrayElement[T dtype.GoDataType](el evaluator.NumericalElement) ([]T, bool, error) {
	canonicalElt, ok := el.(elements.ElementWithConstant)
	if !ok {
		return nil, false, nil
	}
	array := canonicalElt.NumericalConstant().Handle().(*kernels.Buffer).KernelValue().(interface{ Flat() []T })
	return array.Flat(), true, nil
}

func goSliceFromElements[T dtype.GoDataType](els []evaluator.NumericalElement) ([]T, bool, error) {
	var vals []T
	for _, el := range els {
		var subVals []T
		var ok bool
		var err error
		if el.Type().Kind() == ir.ArrayKind {
			subVals, ok, err = goSliceFromArrayElement[T](el)
		} else {
			subVals = make([]T, 1)
			subVals[0], ok, err = goValueFromElement[T](el)
		}
		if err != nil {
			return nil, false, err
		}
		if !ok {
			return nil, false, nil
		}
		vals = append(vals, subVals...)
	}
	return vals, true, nil
}

func (v valuerT[T]) buildStaticArray(fitp *FileScope, lit *ir.ArrayLitExpr, axes, vals []evaluator.NumericalElement) (ir.Element, bool, error) {
	axesI64, ok, err := goSliceFromElements[int64](axes)
	if !ok || err != nil {
		return nil, false, err
	}
	valsT, ok, err := goSliceFromElements[T](vals)
	if !ok || err != nil {
		return nil, false, err
	}
	size := 1
	for _, axis := range axesI64 {
		size *= int(axis)
	}
	if len(vals) > 0 && len(valsT) != size {
		return nil, false, errors.Errorf("tensor has dimensions %v (size=%d) but has %d elements", axes, size, len(vals))
	}
	axesI := make([]int, len(axesI64))
	for i, ax := range axesI64 {
		axesI[i] = int(ax)
	}
	if len(valsT) == 0 {
		valsT = make([]T, size)
	}
	array, err := v.toArrayValue(lit.Type(), valsT, axesI)
	if err != nil {
		return nil, false, err
	}
	// All elements of the literal are scalars already known.
	node, err := fitp.Evaluator().ArrayOps().ElementFromArray(fitp, lit, array)
	if err != nil {
		return nil, false, err
	}
	return node, true, nil
}

func (v valuerT[T]) array(fitp *FileScope, lit *ir.ArrayLitExpr) (ir.Element, error) {
	axes, err := evalArrayAxes(fitp, lit, lit.Typ)
	if err != nil {
		return nil, err
	}
	irVals := lit.Values()
	elVals := make([]evaluator.NumericalElement, len(irVals))
	for i, expr := range irVals {
		elVals[i], err = evalNumExpr(fitp, expr)
		if err != nil {
			return nil, err
		}
	}
	staticArray, staticOk, err := v.buildStaticArray(fitp, lit, axes, elVals)
	if staticOk || err != nil {
		return staticArray, err
	}
	// Some values will be known at runtime. We create one node for each element
	// and concatenates everything into an array.
	array1d, err := fitp.Evaluator().ArrayOps().Concat(fitp, lit, elVals)
	if err != nil {
		return nil, err
	}
	if len(axes) == 1 {
		return array1d, nil
	}
	return array1d.Reshape(fitp, lit, axes)
}

func newValuer(fitp *FileScope, expr ir.Expr, kind ir.Kind) (v valuer, err error) {
	switch kind {
	case ir.IntIdxKind:
		v = valuerT[ir.Int]{kind: kind, toAtomValue: values.AtomIntegerValue[ir.Int], toArrayValue: values.ArrayIntegerValue[ir.Int]}
	case ir.IntLenKind:
		v = valuerT[ir.Int]{kind: kind, toAtomValue: values.AtomIntegerValue[ir.Int], toArrayValue: values.ArrayIntegerValue[ir.Int]}
	case ir.BoolKind:
		v = valuerT[bool]{kind: kind, toAtomValue: values.AtomBoolValue, toArrayValue: values.ArrayBoolValue}
	case ir.Bfloat16Kind:
		v = valuerT[dtype.Bfloat16T]{kind: kind, toAtomValue: values.AtomBfloat16Value, toArrayValue: values.ArrayBfloat16Value}
	case ir.Float32Kind:
		v = valuerT[float32]{kind: kind, toAtomValue: values.AtomFloatValue[float32], toArrayValue: values.ArrayFloatValue[float32]}
	case ir.Float64Kind:
		v = valuerT[float64]{kind: kind, toAtomValue: values.AtomFloatValue[float64], toArrayValue: values.ArrayFloatValue[float64]}
	case ir.Int32Kind:
		v = valuerT[int32]{kind: kind, toAtomValue: values.AtomIntegerValue[int32], toArrayValue: values.ArrayIntegerValue[int32]}
	case ir.Int64Kind:
		v = valuerT[int64]{kind: kind, toAtomValue: values.AtomIntegerValue[int64], toArrayValue: values.ArrayIntegerValue[int64]}
	case ir.Uint32Kind:
		v = valuerT[uint32]{kind: kind, toAtomValue: values.AtomIntegerValue[uint32], toArrayValue: values.ArrayIntegerValue[uint32]}
	case ir.Uint64Kind:
		v = valuerT[uint64]{kind: kind, toAtomValue: values.AtomIntegerValue[uint64], toArrayValue: values.ArrayIntegerValue[uint64]}
	default:
		err = fmterr.Errorf(fitp.File().FileSet(), expr.Source(), "%s cannot be converted to backend numerical: not supported", kind)
	}
	return
}

func evalArrayLiteral(fitp *FileScope, expr *ir.ArrayLitExpr) (ir.Element, error) {
	_, dtype := ir.Shape(expr.Type())
	valuer, err := newValuer(fitp, expr, dtype.Kind())
	if err != nil {
		return nil, err
	}
	return valuer.array(fitp, expr)
}

func toAtomElementInt[T dtype.IntegerType](fitp *FileScope, src elements.ExprAt, val T) (evaluator.NumericalElement, error) {
	hostVal, err := values.AtomIntegerValue(src.Node().Type(), val)
	if err != nil {
		return nil, err
	}
	return fitp.Evaluator().ElementFromAtom(fitp, src.Node(), hostVal)
}

func toAtomElementFloat[T dtype.Float](fitp *FileScope, src elements.ExprAt, val T) (evaluator.NumericalElement, error) {
	hostVal, err := values.AtomFloatValue(src.Node().Type(), val)
	if err != nil {
		return nil, err
	}
	return fitp.Evaluator().ElementFromAtom(fitp, src.Node(), hostVal)
}

func toAtomElementBool(fitp *FileScope, src elements.ExprAt, val bool) (evaluator.NumericalElement, error) {
	hostVal, err := values.AtomBoolValue(src.Node().Type(), val)
	if err != nil {
		return nil, err
	}
	return fitp.Evaluator().ElementFromAtom(fitp, src.Node(), hostVal)
}

func evalAtomicValue(fitp *FileScope, expr ir.AtomicValue) (evaluator.NumericalElement, error) {
	kind := expr.Type().Kind()
	exprAt := elements.NewExprAt(fitp.File(), expr)
	switch kind {
	case ir.IntIdxKind:
		return toAtomElementInt(fitp, exprAt, expr.(*ir.AtomicValueT[ir.Int]).Val)
	case ir.IntLenKind:
		return toAtomElementInt(fitp, exprAt, expr.(*ir.AtomicValueT[ir.Int]).Val)
	case ir.BoolKind:
		return toAtomElementBool(fitp, exprAt, expr.(*ir.AtomicValueT[bool]).Val)
	case ir.Float32Kind:
		return toAtomElementFloat(fitp, exprAt, expr.(*ir.AtomicValueT[float32]).Val)
	case ir.Float64Kind:
		return toAtomElementFloat(fitp, exprAt, expr.(*ir.AtomicValueT[float64]).Val)
	case ir.Int32Kind:
		return toAtomElementInt(fitp, exprAt, expr.(*ir.AtomicValueT[int32]).Val)
	case ir.Int64Kind:
		return toAtomElementInt(fitp, exprAt, expr.(*ir.AtomicValueT[int64]).Val)
	case ir.Uint32Kind:
		return toAtomElementInt(fitp, exprAt, expr.(*ir.AtomicValueT[uint32]).Val)
	case ir.Uint64Kind:
		return toAtomElementInt(fitp, exprAt, expr.(*ir.AtomicValueT[uint64]).Val)
	default:
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Source(), "%s cannot be converted to backend numerical: not supported", kind)
	}
}
