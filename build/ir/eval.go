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

package ir

import (
	"go/ast"
	"go/token"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/build/fmterr"
)

// Fetcher fetches scalar value from identifiers in the code.
// A fetcher is required to sample tensor dimensions and,
// consequently, to compare one type to another.
type Fetcher interface {
	FileSet() *token.FileSet
	ToGoValue(RuntimeValue) (any, error)
	Fetch(ident *ast.Ident) (Atomic, error)
}

// Eval evaluates a scalar expression and returns the value or an error.
func Eval[T dtype.GoDataType](fetcher Fetcher, expr Expr) (val T, unknowns []*ValueRef, err error) {
	kind := KindGeneric[T]()
	var valI any
	switch kind {
	case BoolKind:
		valI, unknowns, err = evalBool(fetcher, expr)
	case Float32Kind:
		valI, unknowns, err = evalScalar[float32](fetcher, expr)
	case Float64Kind:
		valI, unknowns, err = evalScalar[float64](fetcher, expr)
	case Int32Kind:
		valI, unknowns, err = evalScalar[int32](fetcher, expr)
	case Int64Kind:
		valI, unknowns, err = evalScalar[int64](fetcher, expr)
	case Uint32Kind:
		valI, unknowns, err = evalScalar[uint32](fetcher, expr)
	case Uint64Kind:
		valI, unknowns, err = evalScalar[uint64](fetcher, expr)
	default:
		err = errors.Errorf("cannot evaluate %s: %s not supported", expr.String(), kind)
	}
	if err != nil {
		return
	}
	val = valI.(T)
	return
}

func evalBinaryExprBoolT[T dtype.GoDataType](fetcher Fetcher, expr *BinaryExpr) (val bool, unknowns []*ValueRef, err error) {
	valX, unknownsX, err := Eval[T](fetcher, expr.X)
	if err != nil {
		return
	}
	valY, unknownsY, err := Eval[T](fetcher, expr.Y)
	if err != nil {
		return
	}
	unknowns = append(unknownsX, unknownsY...)
	if len(unknowns) > 0 {
		return
	}
	switch expr.Src.Op {
	case token.NEQ:
		val = valX != valY
	case token.EQL:
		val = valX == valY
	default:
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "binary operator %s not supported", expr.Src.Op)
	}
	return
}

func evalBinaryExprBool(fetcher Fetcher, expr *BinaryExpr) (val bool, unknowns []*ValueRef, err error) {
	kind := expr.X.Type().Kind()
	switch kind {
	case BoolKind:
		return evalBinaryExprBoolT[bool](fetcher, expr)
	case Float32Kind:
		return evalBinaryExprBoolT[float32](fetcher, expr)
	case Float64Kind:
		return evalBinaryExprBoolT[float64](fetcher, expr)
	case Int32Kind:
		return evalBinaryExprBoolT[int32](fetcher, expr)
	case Int64Kind:
		return evalBinaryExprBoolT[int64](fetcher, expr)
	case Uint32Kind:
		return evalBinaryExprBoolT[uint32](fetcher, expr)
	case Uint64Kind:
		return evalBinaryExprBoolT[uint64](fetcher, expr)
	case AxisIndexKind:
		return evalBinaryExprBoolT[Int](fetcher, expr)
	case AxisLengthKind:
		return evalBinaryExprBoolT[Int](fetcher, expr)
	default:
		err = errors.Errorf("cannot evaluate binary expression for kind %s", kind.String())
		return
	}
}

func evalBool(fetcher Fetcher, expr Expr) (val any, unknowns []*ValueRef, err error) {
	switch exprT := expr.(type) {
	case *BinaryExpr:
		return evalBinaryExprBool(fetcher, exprT)
	default:
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot evaluate expression: %T not supported", exprT)
		return
	}
}

func evalScalar[T scalarEval](fetcher Fetcher, expr Expr) (val any, unknowns []*ValueRef, err error) {
	switch exprT := expr.(type) {
	case *Number:
		val, err = parseNumber[T](fetcher, exprT)
		return
	case *CastExpr:
		return evalScalar[T](fetcher, exprT.X)
	case *UnaryExpr:
		return evalUnaryExpr[T](fetcher, exprT)
	case *ParenExpr:
		return Eval[T](fetcher, exprT.X)
	case *BinaryExpr:
		return evalBinaryExpr[T](fetcher, exprT)
	case *AtomicValueT[T]:
		return exprT.Val, nil, nil
	case *AtomicExprT[T]:
		return Eval[T](fetcher, exprT.X)
	case *ValueRef:
		return evalValueRef[T](fetcher, exprT)
	case RuntimeValueExpr:
		val, err := fetcher.ToGoValue(exprT.Value())
		return val, nil, err
	default:
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot evaluate expression: %T not supported for %s eval", exprT, reflect.TypeFor[T]().String())
		return
	}
}

func evalValueRef[T dtype.GoDataType](fetcher Fetcher, expr *ValueRef) (val T, unknowns []*ValueRef, err error) {
	valueExpr, err := fetcher.Fetch(expr.Src)
	if err != nil {
		err = fmterr.Position(fetcher.FileSet(), expr.Source(), err)
		return
	}
	if valueExpr == nil {
		// The value has not been defined but is not assigned
		// (e.g. package static variables)
		unknowns = []*ValueRef{expr}
		return
	}
	return Eval[T](fetcher, valueExpr)
}

func evalUnaryExpr[T scalarEval](fetcher Fetcher, expr *UnaryExpr) (val T, unknowns []*ValueRef, err error) {
	kind := KindGeneric[T]()
	switch kind {
	case Float32Kind:
		var valT float32
		valT, unknowns, err = evalUnarySupportedExpr[float32](fetcher, expr)
		val = T(valT)
		return
	case Float64Kind:
		var valT float64
		valT, unknowns, err = evalUnarySupportedExpr[float64](fetcher, expr)
		val = T(valT)
		return
	case Int32Kind:
		var valT int32
		valT, unknowns, err = evalUnarySupportedExpr[int32](fetcher, expr)
		val = T(valT)
		return
	case Int64Kind:
		var valT int64
		valT, unknowns, err = evalUnarySupportedExpr[int64](fetcher, expr)
		val = T(valT)
		return
	default:
		err = errors.Errorf("cannot use %s as %s value (overflows)", expr.String(), kind)
		return
	}
}

type scalarEval interface {
	dtype.Float | dtype.Signed | dtype.Unsigned
}

type unarySupported interface {
	dtype.Signed | dtype.Float
}

func evalUnarySupportedExpr[T unarySupported](fetcher Fetcher, expr *UnaryExpr) (val T, unknowns []*ValueRef, err error) {
	val, unknowns, err = Eval[T](fetcher, expr.X)
	if err != nil || len(unknowns) > 0 {
		return
	}
	switch expr.Src.Op {
	case token.SUB:
		val *= -1
	default:
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "unary operator %v not supported", expr.Src.Op)
	}
	return
}

func evalBinaryExpr[T scalarEval](fetcher Fetcher, expr *BinaryExpr) (val T, unknowns []*ValueRef, err error) {
	valX, unknownsX, err := Eval[T](fetcher, expr.X)
	if err != nil {
		return
	}
	valY, unknownsY, err := Eval[T](fetcher, expr.Y)
	if err != nil {
		return
	}
	unknowns = append(unknownsX, unknownsY...)
	if len(unknowns) > 0 {
		return
	}
	switch expr.Src.Op {
	case token.ADD:
		val = valX + valY
	case token.SUB:
		val = valX - valY
	case token.MUL:
		val = valX * valY
	case token.QUO:
		val = valX / valY
	default:
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "binary operator %s not supported", expr.Src.Op)
	}
	return
}

func parseNumber[T scalarEval](fetcher Fetcher, expr *Number) (val T, err error) {
	want := KindGeneric[T]()
	src := expr.Src
	if src.Kind == token.FLOAT && (want != Float32Kind && want != Float64Kind) {
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot use %s (untyped %s constant) as %s value", src.Value, src.Kind, want)
		return
	}
	switch src.Kind {
	case token.INT:
		valI, parseErr := strconv.ParseInt(src.Value, 10, 0)
		if parseErr != nil {
			err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot parse %s: %v", src.Value, parseErr)
			return
		}
		val = T(valI)
	case token.FLOAT:
		valF, parseErr := strconv.ParseFloat(src.Value, 64)
		if parseErr != nil {
			err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot parse %s: %v", src.Value, parseErr)
			return
		}
		val = T(valF)
	default:
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot parse literal %s: token %q unknown", src.Value, src.Kind)
	}
	return
}
