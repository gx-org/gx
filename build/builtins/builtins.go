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

// Package builtins provides helper functions for builtins functions.
package builtins

import (
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

var (
	// GenericArrayType returns a generic array type.
	GenericArrayType = &ir.ArrayType{RankF: &ir.GenericRank{}}

	// GenericSliceType returns a generic slice type.
	GenericSliceType = &ir.SliceType{}

	// PositionsType returns a type for a slice of indices.
	PositionsType = &ir.ArrayType{
		DType: ir.DefaultIntType,
		RankF: &ir.GenericRank{},
	}
)

// ToBinaryExpr returns a binary expression from two expressions.
func ToBinaryExpr(op token.Token, x, y ir.Expr) *ir.BinaryExpr {
	return &ir.BinaryExpr{
		Src: &ast.BinaryExpr{
			X:  x.Expr(),
			Y:  y.Expr(),
			Op: op,
		},
		X:   x,
		Y:   y,
		Typ: ir.AxisLengthType(),
	}
}

// Fields returns a list of fields to specify parameters or results.
func Fields(types ...ir.Type) *ir.FieldList {
	l := &ir.FieldList{
		List: make([]*ir.FieldGroup, len(types)),
	}
	for i, tp := range types {
		l.List[i] = &ir.FieldGroup{Type: tp}
	}
	return l
}

// InferFromNumericalType returns an argument type if that argument
// is numerical.
func InferFromNumericalType(fetcher ir.Fetcher, call *ir.CallExpr, argNum int, numberTarget ir.Type) (ir.Type, ir.Type, error) {
	arg := call.Args[argNum]
	argType := arg.Type()
	argKind := argType.Kind()
	if argKind == ir.NumberKind {
		target := numberTarget
		if target == nil {
			target = ir.DefaultFloatType
		}
		return target, target, nil
	}
	if ir.IsDataType(argKind) {
		return argType, argType, nil
	}
	arrayType, arrayOk := argType.(*ir.ArrayType)
	if !arrayOk {
		return nil, nil, fmterr.Errorf(fetcher.FileSet(), call.Args[argNum].Source(), "argument type %s not supported", arg.Type().String())
	}
	return arrayType, arrayType.DataType(), nil
}

func joinSignature[T any](sig []T, f func(T) string) string {
	w := strings.Builder{}
	w.WriteString("(")
	for i, item := range sig {
		if i > 0 {
			w.WriteString(", ")
		}
		w.WriteString(f(item))
	}
	w.WriteString(")")
	return w.String()
}
func fmtType(typ ir.Type) string {
	if typ == nil {
		return "?"
	}
	return typ.String()
}

func fmtExprType(e ir.Expr) string { return fmtType(e.Type()) }

// BuildFuncParams takes a function call and list of required argument types
// and returns a list of parameters for the function.
// Arguments with NumberKind are replaced by the target type.
// It returns an error if a call's arguments don't match the given signature.
func BuildFuncParams(fetcher ir.Fetcher, call *ir.CallExpr, name string, sig []ir.Type) ([]ir.Type, error) {
	if len(sig) != len(call.Args) {
		actual := joinSignature[ir.Expr](call.Args, fmtExprType)
		wanted := joinSignature[ir.Type](sig, fmtType)
		return nil, fmterr.Errorf(fetcher.FileSet(), call.Source(), "wrong number of arguments in call to %s: got %s but want %s", name, actual, wanted)
	}
	params := make([]ir.Type, len(sig))
	for i, want := range sig {
		got := call.Args[i].Type()
		if want == nil {
			params[i] = got
			continue
		}
		params[i] = want
		ok := false
		switch want.Kind() {
		case ir.TensorKind:
			ok = got.Kind() == ir.TensorKind
			params[i] = got
		case ir.SliceKind:
			ok = got.Kind() == ir.SliceKind
			params[i] = got
		default:
			assignable, err := got.AssignableTo(fetcher, want)
			if err != nil {
				return nil, err
			}
			ok = assignable
		}
		if !ok {
			actual := joinSignature[ir.Expr](call.Args, fmtExprType)
			wanted := joinSignature[ir.Type](sig, fmtType)
			return nil, fmterr.Errorf(fetcher.FileSet(), call.Source(), "signature mismatch in call to %s: got %s but want %s", name, actual, wanted)
		}
	}
	return params, nil
}

// NarrowType converts an abstract type into more concrete type, typically *google3/third_party/gxlang/gx/build/ir/ir.ArrayType.
func NarrowType[T ir.Type](fetcher ir.Fetcher, call *ir.CallExpr, arg ir.Type) (t T, err error) {
	var ok bool
	t, ok = arg.(T)
	if !ok {
		err = fmterr.Errorf(fetcher.FileSet(), call.Source(), "cannot convert %T to %s", arg, reflect.TypeFor[T]().String())
	}
	return
}

// NarrowTypes converts abstract types into more concrete types, typically *ir.ArrayType.
func NarrowTypes[T ir.Type](fetcher ir.Fetcher, call *ir.CallExpr, args []ir.Type) ([]T, error) {
	res := make([]T, len(args))
	for i, arg := range args {
		var err error
		res[i], err = NarrowType[T](fetcher, call, arg)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// UniqueAxesFromExpr returns the set of unique axis indices found in a slice literal.
func UniqueAxesFromExpr(fetcher ir.Fetcher, expr ir.Expr) (map[int]struct{}, error) {
	sliceExpr, ok := expr.(*ir.SliceExpr)
	if !ok {
		return nil, fmterr.Errorf(fetcher.FileSet(), expr.Source(), "expected axes slice literal, but got %s", expr.String())
	}

	axes := map[int]struct{}{}
	for _, val := range sliceExpr.Vals {
		axis, unknown, err := ir.Eval[ir.Int](fetcher, val)
		if err != nil {
			return nil, err
		}
		if len(unknown) > 0 {
			return nil, fmterr.Errorf(fetcher.FileSet(), expr.Source(), "expected axis literals, but got un-evalable expression %s", expr.String())
		}
		if _, exists := axes[int(axis)]; exists {
			return nil, fmterr.Errorf(fetcher.FileSet(), expr.Source(), "axis index %d specified more than once", axis)
		}
		axes[int(axis)] = struct{}{}
	}
	return axes, nil
}

// EvalShape computes a shape given a rank.
func EvalShape(fetcher ir.Fetcher, rank *ir.Rank) ([]int, bool, error) {
	if rank == nil || len(rank.Axes) == 0 {
		return nil, false, nil
	}
	shape := make([]int, len(rank.Axes))
	for i, axis := range rank.Axes {
		val, unknowns, err := ir.Eval[ir.Int](fetcher, axis.Expr())
		if err != nil {
			return nil, false, err
		}
		if len(unknowns) > 0 {
			return nil, false, nil
		}
		shape[i] = int(val)
	}
	return shape, true, nil
}

// RankFromExpr returns a rank if it can be computed from the expression.
func RankFromExpr(src ast.Expr, expr ir.Expr) ir.ArrayRank {
	sliceExpr, ok := expr.(*ir.SliceExpr)
	if !ok {
		return &ir.GenericRank{}
	}
	axes := make([]ir.AxisLength, len(sliceExpr.Vals))
	for i, val := range sliceExpr.Vals {
		axes[i] = &ir.AxisExpr{
			Src: src,
			X:   val,
		}
	}
	return &ir.Rank{Axes: axes}
}

// RankOf returns the list of axes of an array.
func RankOf(fetcher ir.Fetcher, src ir.SourceNode, a *ir.ArrayType) (*ir.Rank, error) {
	rank, ok := a.Rank().(ir.ResolvedRank)
	if !ok {
		return nil, fmterr.Errorf(fetcher.FileSet(), src.Source(), "array axes have not been resolved")
	}
	return rank.Resolved(), nil
}
