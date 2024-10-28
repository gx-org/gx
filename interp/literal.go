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
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/interp/state"
)

type (
	valuer interface {
		evaluator(errfs fmterr.FileSet) evaluator
		eval(*context, ir.Expr) (state.Element, error)
		toLiteral(ctx *context, typ ir.Type, arr kernels.Array) (state.Element, error)
	}

	valuerT[T dtype.GoDataType] struct {
		kind ir.Kind
	}
)

type (
	evaluator interface {
		scalar(ctx *context, expr ir.Atomic) (state.Element, error)
		array(ctx *context, expr ir.ArrayLitExpr) (state.Element, error)
	}

	evaluatorT[T dtype.GoDataType] struct {
		errfs  fmterr.FileSet
		valuer valuerT[T]
	}

	fetcher[T dtype.GoDataType] struct {
		kind ir.Kind
		ctx  *context
	}
)

func (v valuerT[T]) eval(ctx *context, expr ir.Expr) (el state.Element, err error) {
	val, unknowns, err := v.valueOf(ctx, expr)
	if err != nil {
		return nil, err
	}
	if len(unknowns) > 0 {
		names := make([]string, len(unknowns))
		return nil, errors.Errorf("%v are unknowns", names)
	}
	return state.NewAtomicLiteral(ctx.State(), expr.Type(), val)
}

func (v valuerT[T]) evaluator(errfs fmterr.FileSet) evaluator {
	return evaluatorT[T]{errfs: errfs, valuer: v}
}

func (v valuerT[T]) valueOf(ctx *context, expr ir.Expr) (T, []*ir.ValueRef, error) {
	return ir.Eval[T](fetcher[T]{kind: ir.Kind(v.kind.DType()), ctx: ctx}, expr)
}

func (v valuerT[T]) toLiteral(ctx *context, typ ir.Type, arr kernels.Array) (state.Element, error) {
	data := arr.(*kernels.ArrayT[T]).Flat()
	if len(data) != 1 {
		return nil, errors.Errorf("not supported for array literals")
	}
	return state.NewAtomicLiteral(ctx.State(), typ, data[0])
}

func (v valuerT[T]) toArrayFromScalar(ctx *context, typ ir.Type, val T, dims []int) (state.Element, error) {
	size := shape.Shape{DType: v.kind.DType(), AxisLengths: dims}.Size()
	vals := make([]T, size)
	for i := range vals {
		vals[i] = val
	}
	return state.NewArrayLiteral(ctx.State(), typ, vals, dims)
}

func (f fetcher[T]) ToGoValue(val ir.RuntimeValue) (any, error) {
	hostVal, ok := val.(*values.HostArray)
	if !ok {
		return nil, errors.Errorf("cannot convert %T to a Go value", val)
	}
	return hostVal.ToAtom()
}

func (f fetcher[T]) Fetch(ident *ast.Ident) (atom ir.Atomic, err error) {
	defer func() {
		if err != nil {
			err = f.ctx.FileSet().Errorf(ident, "cannot fetch %s: %v", ident.Name, err)
		}
	}()
	el, err := f.ctx.find(ident)
	if err != nil {
		return
	}
	val, err := state.ConstantScalarFromElement[T](el)
	if err != nil {
		return nil, err
	}
	return &ir.AtomicValueT[T]{
		Src: ident,
		Typ: ir.ScalarTypeK(f.kind),
		Val: val,
	}, nil
}

func (f fetcher[T]) FileSet() *token.FileSet {
	return f.ctx.FileSet().FSet
}

func (e evaluatorT[T]) scalar(ctx *context, val ir.Atomic) (state.Element, error) {
	value, _, err := e.valuer.valueOf(ctx, val)
	if err != nil {
		return nil, err
	}
	return state.NewAtomicLiteral(ctx.State(), val.Type(), value)
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

func (e evaluatorT[T]) evalIntExpr(ctx *context, expr ir.Expr) (int, []*ir.ValueRef, error) {
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

func (e evaluatorT[T]) evalDim(ctx *context, dim ir.AxisLength) (int, []*ir.ValueRef, error) {
	switch dimT := dim.(type) {
	case *ir.AxisExpr:
		return e.evalIntExpr(ctx, dimT.X)
	case *ir.AxisEllipsis:
		return e.evalIntExpr(ctx, dimT.X.X)
	default:
		return 0, nil, ctx.FileSet().Errorf(dim.Source(), "dimension of type %T not supported", dimT)

	}
}

func (e evaluatorT[T]) evalArrayElementsDims(ctx *context, scalars []ir.Atomic) ([]T, error) {
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

func (e evaluatorT[T]) evalArrayDims(ctx *context, typ *ir.ArrayType) (int, []int, error) {
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

func elementsToStaticScalars(exprs []ir.Expr) []ir.Atomic {
	scalars := make([]ir.Atomic, len(exprs))
	for i, expr := range exprs {
		// TODO(degris): some expression are not ir.Scalar but can still be evaluated
		// as scalars at JIT time (e.g. static variables).
		// At the moment, we fall back to concatenation which is suboptimal.
		scalar, ok := expr.(ir.Atomic)
		if !ok {
			return nil
		}
		scalars[i] = scalar
	}
	return scalars
}

func (e evaluatorT[T]) array(ctx *context, val ir.ArrayLitExpr) (state.Element, error) {
	valT, ok := val.(*ir.ArrayLitExprT[T])
	if !ok {
		return nil, errors.Errorf("value of type %T not supported", val)
	}
	size, dims, err := e.evalArrayDims(ctx, valT.Typ)
	if err != nil {
		return nil, err
	}
	if len(valT.Vals) > 0 && len(valT.Vals) != size {
		return nil, errors.Errorf("tensor has dimensions %v (size=%d) but has %d elements", dims, size, len(valT.Vals))
	}
	if scalars := elementsToStaticScalars(valT.Vals); scalars != nil {
		// All elements of the literal are scalars already known.
		// We can pack them into an array, build a local tensor
		// and send everything at once to the device.
		vals, err := e.evalArrayElementsDims(ctx, scalars)
		if err != nil {
			return nil, err
		}
		var literal state.Element
		if len(valT.Vals) == 0 {
			var zero T
			literal, err = e.valuer.toArrayFromScalar(ctx, val.Type(), zero, dims)
		} else {
			literal, err = state.NewArrayLiteral(ctx.State(), val.Type(), vals, dims)
		}
		return literal, err
	}
	// Some values will be known at runtime. We create one node for each element
	// and concatenates everything into an array.
	nodes := make([]graph.Node, len(valT.Vals))
	dtype := dtype.Generic[T]()
	for i, expr := range valT.Vals {
		array, _, err := ctx.evalNumericalExpr(expr)
		if err != nil {
			return nil, err
		}
		// Reshape scalars to 1-element array to work with Concat.
		flat, err := ctx.state.BackendGraph().Core().NewReshape(array, []int{1})
		if err != nil {
			return nil, err
		}
		nodes[i] = flat
	}
	array1d, err := ctx.state.BackendGraph().Core().NewConcat(0, nodes)
	if err != nil {
		return nil, err
	}
	targetShape := &shape.Shape{
		DType:       dtype,
		AxisLengths: dims,
	}
	if len(dims) == 1 {
		return ctx.state.ElementFromNode(ctx.exprAt(val), array1d, targetShape)
	}
	reshapeNode, err := ctx.state.BackendGraph().Core().NewReshape(array1d, dims)
	if err != nil {
		return nil, err
	}
	return ctx.state.ElementFromNode(ctx.exprAt(val), reshapeNode, targetShape)
}

func newValuer(ctx *context, expr ir.Expr, kind ir.Kind) (v valuer, err error) {
	switch kind {
	case ir.BoolKind:
		v = valuerT[bool]{kind: kind}
	case ir.Float32Kind:
		v = valuerT[float32]{kind: kind}
	case ir.Float64Kind:
		v = valuerT[float64]{kind: kind}
	case ir.Int32Kind:
		v = valuerT[int32]{kind: kind}
	case ir.Int64Kind:
		v = valuerT[int64]{kind: kind}
	case ir.Uint32Kind:
		v = valuerT[uint32]{kind: kind}
	case ir.Uint64Kind:
		v = valuerT[uint64]{kind: kind}
	case ir.AxisIndexKind:
		v = valuerT[ir.Int]{kind: kind}
	case ir.AxisLengthKind:
		v = valuerT[ir.Int]{kind: kind}
	default:
		err = ctx.FileSet().Errorf(expr.Source(), "%s cannot be converted to backend numerical: not supported", kind)
	}
	return
}

func toLocalTensor(
	ctx *context,
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

func evalScalarLiteral(ctx *context, expr ir.Atomic) (state.Element, error) {
	return toLocalTensor(
		ctx, expr, expr.Type().Kind(),
		func(eval evaluator) (state.Element, error) {
			return eval.scalar(ctx, expr)
		})
}

func evalArrayLiteral(ctx *context, expr ir.ArrayLitExpr) (state.Element, error) {
	_, dtype := ir.Shape(expr.Type())
	return toLocalTensor(
		ctx, expr, dtype.Kind(),
		func(eval evaluator) (state.Element, error) {
			return eval.array(ctx, expr)
		})
}

func evalValue(ctx *context, val ir.Expr) (state.Element, error) {
	switch valT := val.(type) {
	case ir.Atomic:
		return toLocalTensor(
			ctx, valT, valT.Type().Kind(),
			func(eval evaluator) (state.Element, error) {
				return eval.scalar(ctx, valT)
			})

	case ir.ArrayLitExpr:
		_, dtype := ir.Shape(val.Type())
		return toLocalTensor(
			ctx, valT, dtype.Kind(),
			func(eval evaluator) (state.Element, error) {
				return eval.array(ctx, valT)
			})
	}
	return nil, ctx.FileSet().Pos(val.Source()).Errorf("cannot evaluate %T into a XLA node: type not supported", val.Type())
}
