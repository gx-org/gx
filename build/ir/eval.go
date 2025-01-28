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
	"unsafe"

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
	Fetch(ident *ast.Ident) (StaticValue, error)
}

// Eval evaluates a scalar expression and returns the value or an error.
func Eval[T dtype.GoDataType](fetcher Fetcher, expr Expr) (val T, unknowns []*ValueRef, err error) {
	kind := KindGeneric[T]()
	var valI any
	switch kind {
	case BoolKind:
		valI, unknowns, err = evalBool(fetcher, expr)
	case Float32Kind:
		valI, unknowns, err = evalAlgebra[float32](fetcher, expr)
	case Float64Kind:
		valI, unknowns, err = evalAlgebra[float64](fetcher, expr)
	case Int32Kind:
		valI, unknowns, err = evalInteger[int32](fetcher, expr)
	case Int64Kind:
		valI, unknowns, err = evalInteger[int64](fetcher, expr)
	case Uint32Kind:
		valI, unknowns, err = evalInteger[uint32](fetcher, expr)
	case Uint64Kind:
		valI, unknowns, err = evalInteger[uint64](fetcher, expr)
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
	case IntIdxKind:
		return evalBinaryExprBoolT[Int](fetcher, expr)
	case IntLenKind:
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

func evalInteger[T dtype.IntegerType](fetcher Fetcher, expr Expr) (val any, unknowns []*ValueRef, err error) {
	switch exprT := expr.(type) {
	case *BinaryExpr:
		return evalBinaryIntegerExpr[T](fetcher, exprT)
	case *NumberCastExpr:
		return evalInteger[T](fetcher, exprT.X)
	}
	return evalAlgebra[T](fetcher, expr)
}

func checkEllipsis(fetcher Fetcher, a *AxisEllipsis) error {
	if a.X != nil {
		return nil
	}
	return fmterr.Errorf(fetcher.FileSet(), a.Source(), "missing axis length")
}

func evalAlgebra[T dtype.AlgebraType](fetcher Fetcher, expr Expr) (val any, unknowns []*ValueRef, err error) {
	if expr == nil {
		err = fmterr.Internalf(fetcher.FileSet(), nil, "cannot evaluate a nil expression")
		return
	}
	switch exprT := expr.(type) {
	case *NumberFloat:
		val, err = parseNumberFloat[T](fetcher, exprT)
		return
	case *NumberInt:
		val, err = parseNumberInt[T](fetcher, exprT)
		return
	case *CastExpr:
		return evalAlgebra[T](fetcher, exprT.X)
	case *UnaryExpr:
		return evalUnaryExpr[T](fetcher, exprT)
	case *ParenExpr:
		return Eval[T](fetcher, exprT.X)
	case *BinaryExpr:
		return evalBinaryAlgebraExpr[T](fetcher, exprT)
	case *AtomicValueT[T]:
		return exprT.Val, nil, nil
	case *StaticAtom:
		return Eval[T](fetcher, exprT.X)
	case *ValueRef:
		return evalValueRef[T](fetcher, exprT)
	case *AxisExpr:
		return evalAlgebra[T](fetcher, exprT.X)
	case *AxisEllipsis:
		if err = checkEllipsis(fetcher, exprT); err != nil {
			return
		}
		return evalAlgebra[T](fetcher, exprT.X)
	case *NumberCastExpr:
		return evalAlgebra[T](fetcher, exprT.X)
	case RuntimeValueExpr:
		val, err := fetcher.ToGoValue(exprT.Value())
		return val, nil, err
	case *PackageConstSelectorExpr:
		return evalAlgebra[T](fetcher, exprT.X)
	// Numbers are evaluated using float64 and int64 at the moment:
	case *AtomicValueT[float64]:
		return T(exprT.Val), nil, nil
	case *AtomicValueT[int64]:
		return T(exprT.Val), nil, nil
	case *ConstExpr:
		return evalAlgebra[T](fetcher, exprT.Value)
	default:
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot evaluate expression: %T not supported for %s eval", exprT, reflect.TypeFor[T]().String())
		return
	}
}

func evalValueRef[T dtype.GoDataType](fetcher Fetcher, expr *ValueRef) (val T, unknowns []*ValueRef, err error) {
	value, err := fetcher.Fetch(expr.Src)
	if err != nil {
		err = fmterr.Position(fetcher.FileSet(), expr.Source(), err)
		return
	}
	if value == nil {
		// The value has not been defined but is not assigned
		// (e.g. package static variables)
		unknowns = []*ValueRef{expr}
		return
	}
	typ := value.Type()
	if !SupportOperators(typ.Kind()) {
		err = errors.Errorf("%s of type %s is not a scalar", expr.Src.Name, typ.String())
		return
	}
	exprValue, ok := value.(Expr)
	if !ok {
		err = errors.Errorf("cannot cast %T to an IR expression", value)
		return
	}
	return Eval[T](fetcher, exprValue)
}

func evalUnaryExpr[T dtype.AlgebraType](fetcher Fetcher, expr *UnaryExpr) (val T, unknowns []*ValueRef, err error) {
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

func evalBinaryAlgebraVals[T dtype.AlgebraType](op token.Token, x, y T) (val T, err error) {
	switch op {
	case token.ADD:
		val = x + y
	case token.SUB:
		val = x - y
	case token.MUL:
		val = x * y
	case token.QUO:
		val = x / y
	default:
		err = errors.Errorf("binary operator %s not supported", op)
	}
	return
}

func evalBinaryIntegerVals[T dtype.IntegerType](op token.Token, x, y T) (val T, err error) {
	switch op {
	case token.REM:
		val = x % y
		return
	}
	return evalBinaryAlgebraVals(op, x, y)
}

func evalBinaryAlgebraExpr[T dtype.AlgebraType](fetcher Fetcher, expr *BinaryExpr) (val T, unknowns []*ValueRef, err error) {
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
	val, err = evalBinaryAlgebraVals(expr.Src.Op, valX, valY)
	if err != nil {
		err = fmterr.Position(fetcher.FileSet(), expr.Src, err)
	}
	return
}

func evalBinaryIntegerExpr[T dtype.IntegerType](fetcher Fetcher, expr *BinaryExpr) (val T, unknowns []*ValueRef, err error) {
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
	val, err = evalBinaryIntegerVals(expr.Src.Op, valX, valY)
	if err != nil {
		err = fmterr.Position(fetcher.FileSet(), expr.Src, err)
	}
	return
}

func parseNumberFloat[T dtype.AlgebraType](fetcher Fetcher, expr *NumberFloat) (val T, err error) {
	want := KindGeneric[T]()
	src := expr.Src
	if src.Kind == token.FLOAT && (want != Float32Kind && want != Float64Kind) {
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot use %s (untyped %s constant) as %s value", src.Value, src.Kind, want)
		return
	}
	valF, parseErr := strconv.ParseFloat(src.Value, 64)
	if parseErr != nil {
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot parse %s: %v", src.Value, parseErr)
		return
	}
	val = T(valF)
	return
}

func parseNumberInt[T dtype.AlgebraType](fetcher Fetcher, expr *NumberInt) (val T, err error) {
	want := KindGeneric[T]()
	src := expr.Src
	if src.Kind == token.FLOAT && (want != Float32Kind && want != Float64Kind) {
		err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot use %s (untyped %s constant) as %s value", src.Value, src.Kind, want)
		return
	}
	bitSize := int(unsafe.Sizeof(val) * 8)
	if want == Int32Kind || want == Int64Kind {
		valI, parseErr := strconv.ParseInt(src.Value, 10, bitSize)
		if parseErr != nil {
			err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot parse %s: %v", src.Value, parseErr)
			return
		}
		val = T(valI)
	} else {
		valU, parseErr := strconv.ParseUint(src.Value, 10, bitSize)
		if parseErr != nil {
			err = fmterr.Errorf(fetcher.FileSet(), expr.Source(), "cannot parse unsigned %s: %v", src.Value, parseErr)
			return
		}
		val = T(valU)
	}
	return
}
