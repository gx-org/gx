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
	"go/ast"
	"go/token"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/state"
)

type (
	valuer interface {
		evaluator(errfs fmterr.FileSet) evaluator
		eval(Context, ir.Expr) (state.Element, error)
	}

	valuerT[T dtype.GoDataType] struct {
		kind         ir.Kind
		toAtomValue  func(tp ir.Type, val T) *values.HostArray
		toArrayValue func(tp ir.Type, val []T, dims []int) *values.HostArray
	}
)

type (
	evaluator interface {
		scalar(ctx Context, expr ir.StaticExpr) (state.Element, error)
		array(ctx Context, expr *ir.ArrayLitExpr) (state.Element, error)
	}

	evaluatorT[T dtype.GoDataType] struct {
		errfs  fmterr.FileSet
		valuer valuerT[T]
	}

	fetcher[T dtype.GoDataType] struct {
		kind ir.Kind
		ctx  Context
	}
)

func (v valuerT[T]) eval(ctx Context, expr ir.Expr) (el state.Element, err error) {
	val, unknowns, err := v.valueOf(ctx, expr)
	if err != nil {
		return nil, err
	}
	if len(unknowns) > 0 {
		names := make([]string, len(unknowns))
		return nil, errors.Errorf("%v are unknowns", names)
	}
	gxValue := v.toAtomValue(expr.Type(), val)
	return ctx.Evaluator().ElementFromValue(ctx, expr, gxValue)
}

func (v valuerT[T]) evaluator(errfs fmterr.FileSet) evaluator {
	return evaluatorT[T]{errfs: errfs, valuer: v}
}

func (v valuerT[T]) valueOf(ctx Context, expr ir.Expr) (T, []*ir.ValueRef, error) {
	return ir.Eval[T](fetcher[T]{kind: ir.Kind(v.kind.DType()), ctx: ctx}, expr)
}

func (f fetcher[T]) ToGoValue(val ir.RuntimeValue) (any, error) {
	hostVal, ok := val.(*values.HostArray)
	if !ok {
		return nil, errors.Errorf("cannot convert %T to a Go value", val)
	}
	return hostVal.ToAtom()
}

func (f fetcher[T]) Fetch(ident *ast.Ident) (atom ir.StaticValue, err error) {
	defer func() {
		if err != nil {
			err = f.ctx.FileSet().Errorf(ident, "cannot fetch %s: %v", ident.Name, err)
		}
	}()
	el, err := f.ctx.frame().find(ident)
	if err != nil {
		return
	}
	if number, ok := el.(*state.Number); ok {
		return number.Atomic(f)
	}
	val, err := state.ConstantScalarFromElement[T](el)
	if err != nil {
		return nil, err
	}
	return &ir.AtomicValueT[T]{
		Src: ident,
		Typ: ir.TypeFromKind(f.kind),
		Val: val,
	}, nil
}

func (f fetcher[T]) FileSet() *token.FileSet {
	return f.ctx.FileSet().FSet
}

func (e evaluatorT[T]) scalar(ctx Context, val ir.StaticExpr) (state.Element, error) {
	value, _, err := e.valuer.valueOf(ctx, val)
	if err != nil {
		return nil, err
	}
	atom := e.valuer.toAtomValue(val.Type(), value)
	return ctx.Evaluator().ElementFromValue(ctx, val, atom)
}

func toGoInt(val *values.HostArray) (int, error) {
	valT := val.Shape().DType
	switch valT {
	case dtype.Int32:
		return int(values.ToAtom[int32](val)), nil
	case dtype.Int64:
		return int(values.ToAtom[int64](val)), nil
	default:
		return -1, errors.Errorf("cannot cast type %s to int", valT.String())
	}
}

func (e evaluatorT[T]) evalIntExpr(ctx Context, expr ir.Expr) (int, []*ir.ValueRef, error) {
	valuer, err := newValuer(ctx, expr, expr.Type().Kind())
	if err != nil {
		return -1, nil, err
	}
	element, err := valuer.eval(ctx, expr)
	if err != nil {
		return -1, nil, err
	}
	elWithConstant, ok := element.(state.ElementWithConstant)
	if !ok {
		return -1, nil, errors.Errorf("cannot cast %T to %s", element, reflect.TypeFor[state.ElementWithConstant]())
	}
	elConstant := elWithConstant.NumericalConstant()
	valInt, err := toGoInt(elConstant)
	return valInt, nil, err
}

func (e evaluatorT[T]) evalDim(ctx Context, dim ir.AxisLength) (int, []*ir.ValueRef, error) {
	switch dimT := dim.(type) {
	case *ir.AxisExpr:
		return e.evalIntExpr(ctx, dimT.X)
	case *ir.AxisEllipsis:
		return e.evalIntExpr(ctx, dimT.X)
	default:
		return 0, nil, ctx.FileSet().Errorf(dim.Source(), "dimension of type %T not supported", dimT)

	}
}

func (e evaluatorT[T]) evalArrayElementsDims(ctx Context, scalars []ir.StaticExpr) ([]T, error) {
	vals := make([]T, len(scalars))
	for i, val := range scalars {
		var err error
		vals[i], _, err = e.valuer.valueOf(ctx, val)
		if err != nil {
			return nil, err
		}
	}
	return vals, nil
}

func (e evaluatorT[T]) evalArrayDims(ctx Context, typ ir.ArrayType) (int, []int, error) {
	rank, err := rankOf(ctx, typ, typ)
	if err != nil {
		return -1, nil, err
	}
	dims := make([]int, len(rank.Axes))
	size := 1
	for i, dim := range rank.Axes {
		dimInt64, _, err := e.evalDim(ctx, dim)
		if err != nil {
			return -1, nil, err
		}
		dims[i] = int(dimInt64)
		size *= dims[i]
	}
	return size, dims, nil
}

func elementsToStaticScalars(exprs []ir.Expr) []ir.StaticExpr {
	scalars := make([]ir.StaticExpr, len(exprs))
	for i, expr := range exprs {
		// TODO(degris): some expression are not ir.Scalar but can still be evaluated
		// as scalars at JIT time (e.g. static variables).
		// At the moment, we fall back to concatenation which is suboptimal.
		scalar, ok := expr.(ir.StaticExpr)
		if !ok {
			return nil
		}
		scalars[i] = scalar
	}
	return scalars
}

func (e evaluatorT[T]) array(ctx Context, val *ir.ArrayLitExpr) (state.Element, error) {
	size, dims, err := e.evalArrayDims(ctx, val.Typ)
	if err != nil {
		return nil, err
	}
	if len(val.Vals) > 0 && len(val.Vals) != size {
		return nil, errors.Errorf("tensor has dimensions %v (size=%d) but has %d elements", dims, size, len(val.Vals))
	}
	if scalars := elementsToStaticScalars(val.Vals); scalars != nil {
		// All elements of the literal are scalars already known.
		// We can pack them into an array, build a local tensor
		// and send everything at once to the device.
		vals, err := e.evalArrayElementsDims(ctx, scalars)
		if err != nil {
			return nil, err
		}
		if len(vals) == 0 {
			vals = make([]T, shape.Size(dims))
		}
		arrayValue := e.valuer.toArrayValue(val.Type(), vals, dims)
		return ctx.Evaluator().ElementFromValue(ctx, val, arrayValue)
	}
	// Some values will be known at runtime. We create one node for each element
	// and concatenates everything into an array.
	els := make([]state.Element, len(val.Vals))
	for i, expr := range val.Vals {
		els[i], err = evalExpr(ctx, expr)
		if err != nil {
			return nil, err
		}
	}
	array1d, err := ctx.Evaluator().Concat(ctx, val, els)
	if err != nil {
		return nil, err
	}
	if len(dims) == 1 {
		return array1d, nil
	}
	return ctx.Evaluator().Reshape(ctx, val, array1d, dims)
}

func newValuer(ctx Context, expr ir.Expr, kind ir.Kind) (v valuer, err error) {
	switch kind {
	case ir.IntIdxKind:
		v = valuerT[ir.Int]{kind: kind, toAtomValue: values.AtomIntegerValue[ir.Int], toArrayValue: values.ArrayIntegerValue[ir.Int]}
	case ir.IntLenKind:
		v = valuerT[ir.Int]{kind: kind, toAtomValue: values.AtomIntegerValue[ir.Int], toArrayValue: values.ArrayIntegerValue[ir.Int]}
	case ir.BoolKind:
		v = valuerT[bool]{kind: kind, toAtomValue: values.AtomBoolValue, toArrayValue: values.ArrayBoolValue}
	case ir.Float32Kind:
		v = valuerT[float32]{kind: kind, toAtomValue: values.AtomFloatValue[float32], toArrayValue: values.ArrayFloatValue[float32]}
	case ir.Float64Kind:
		v = valuerT[float64]{kind: kind, toAtomValue: values.AtomFloatValue[float64], toArrayValue: values.ArrayFloatValue[float64]}
	case ir.Int32Kind:
		v = valuerT[int32]{kind: kind, toAtomValue: values.AtomIntegerValue[int32], toArrayValue: values.ArrayIntegerValue[int32]}
	case ir.Int64Kind:
		v = valuerT[int64]{kind: kind, toAtomValue: values.AtomIntegerValue[int64], toArrayValue: values.ArrayIntegerValue[int64]}
	case ir.StringKind:
		v = &stringValuer{}
	case ir.Uint32Kind:
		v = valuerT[uint32]{kind: kind, toAtomValue: values.AtomIntegerValue[uint32], toArrayValue: values.ArrayIntegerValue[uint32]}
	case ir.Uint64Kind:
		v = valuerT[uint64]{kind: kind, toAtomValue: values.AtomIntegerValue[uint64], toArrayValue: values.ArrayIntegerValue[uint64]}
	default:
		err = ctx.FileSet().Errorf(expr.Source(), "%s cannot be converted to backend numerical: not supported", kind)
	}
	return
}

func toLocalTensor(
	ctx Context,
	expr ir.Expr,
	kind ir.Kind,
	toNode func(evaluator) (state.Element, error),
) (state.Element, error) {
	toValue, err := newValuer(ctx, expr, kind)
	if err != nil {
		return nil, err
	}
	return toNode(toValue.evaluator(ctx.FileSet()))
}

func evalScalarLiteral(ctx Context, expr ir.StaticExpr) (state.Element, error) {
	return toLocalTensor(
		ctx, expr, expr.Type().Kind(),
		func(eval evaluator) (state.Element, error) {
			return eval.scalar(ctx, expr)
		})
}

func evalArrayLiteral(ctx Context, expr *ir.ArrayLitExpr) (state.Element, error) {
	_, dtype := ir.Shape(expr.Type())
	return toLocalTensor(
		ctx, expr, dtype.Kind(),
		func(eval evaluator) (state.Element, error) {
			return eval.array(ctx, expr)
		})
}

func evalValue(ctx Context, val ir.Expr) (state.Element, error) {
	switch valT := val.(type) {
	case ir.StaticExpr:
		return toLocalTensor(
			ctx, valT, valT.Type().Kind(),
			func(eval evaluator) (state.Element, error) {
				return eval.scalar(ctx, valT)
			})

	case *ir.ArrayLitExpr:
		_, dtype := ir.Shape(val.Type())
		return toLocalTensor(
			ctx, valT, dtype.Kind(),
			func(eval evaluator) (state.Element, error) {
				return eval.array(ctx, valT)
			})
	}
	return nil, ctx.FileSet().Pos(val.Source()).Errorf("cannot evaluate %T into a XLA node: type not supported", val.Type())
}
