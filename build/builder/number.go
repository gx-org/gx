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

package builder

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/build/ir"
)

type (
	evalType interface {
		float64 | int64
	}

	evaluator interface {
		scoper() scoper
		parse(*ast.BasicLit) ir.Atomic
		eval(expr ir.Expr) (ir.Atomic, []*ir.ValueRef, error)
		cast(expr ir.Expr) ir.Atomic
		want() ir.Type
	}

	evaluatorT[E evalType, T dtype.AlgebraType] struct {
		scope scoper
		tp    ir.Type
	}
)

func newEvaluatorT[E evalType, T dtype.AlgebraType](scope scoper, want ir.Type) evaluator {
	return &evaluatorT[E, T]{
		scope: scope,
		tp:    want,
	}
}

func newEvaluator(scope scoper, node nodePos, want ir.Type) (evaluator, bool) {
	switch want.Kind() {
	case ir.AxisIndexKind:
		return newEvaluatorT[int64, ir.Int](scope, want), true
	case ir.AxisLengthKind:
		return newEvaluatorT[int64, ir.Int](scope, want), true
	case ir.Float32Kind:
		return newEvaluatorT[float64, float32](scope, want), true
	case ir.Float64Kind:
		return newEvaluatorT[float64, float64](scope, want), true
	case ir.Int32Kind:
		return newEvaluatorT[int64, int32](scope, want), true
	case ir.Int64Kind:
		return newEvaluatorT[int64, int64](scope, want), true
	case ir.Uint32Kind:
		return newEvaluatorT[int64, uint32](scope, want), true
	case ir.Uint64Kind:
		return newEvaluatorT[int64, uint64](scope, want), true
	default:
		scope.err().Appendf(node.source(), "number cannot be casted to %s", want)
		return nil, false
	}
}

func (ev *evaluatorT[E, T]) want() ir.Type {
	return ev.tp
}

func (ev *evaluatorT[E, T]) scoper() scoper {
	return ev.scope
}

func (ev *evaluatorT[E, T]) parse(src *ast.BasicLit) ir.Atomic {
	errF := ev.scoper().err()
	// Check that we are not specifying an integer with a float.
	want := ev.tp.Kind()
	if src.Kind == token.FLOAT && (want != ir.Float32Kind && want != ir.Float64Kind) {
		errF.Appendf(src, "cannot use %s (untyped %s constant) as %s value", src.Value, src.Kind, want)
		return nil
	}

	var val T
	switch src.Kind {
	case token.INT:
		valI, err := strconv.ParseInt(src.Value, 10, 0)
		if err != nil {
			errF.Appendf(src, "cannot parse %s: %v", src.Value, err)
			return nil
		}
		val = T(valI)
	case token.FLOAT:
		valF, err := strconv.ParseFloat(src.Value, 64)
		if err != nil {
			errF.Appendf(src, "cannot parse %s: %v", src.Value, err)
			return nil
		}
		val = T(valF)
	default:
		errF.Appendf(src, "cannot parse literal %s: token %q unknown", src.Value, src.Kind)
	}
	return &ir.AtomicValueT[T]{
		Src: src,
		Val: val,
		Typ: ev.tp,
	}
}

func (ev *evaluatorT[E, T]) eval(expr ir.Expr) (ir.Atomic, []*ir.ValueRef, error) {
	val, unknowns, err := ir.Eval[T](ev.scope.evalFetcher(), expr)
	if err != nil {
		return nil, nil, err
	}
	if len(unknowns) > 0 {
		return nil, unknowns, nil
	}
	return &ir.AtomicValueT[T]{Val: val, Src: expr.Expr(), Typ: ev.want()}, nil, nil
}

func (ev *evaluatorT[E, T]) cast(expr ir.Expr) ir.Atomic {
	return &ir.AtomicExprT[T]{X: expr}
}

func castExprTo(eval evaluator, expr exprNode) (exprScalar, []*ir.ValueRef, bool) {
	caster, ok := expr.(exprNumber)
	if !ok {
		eval.scoper().err().AppendInternalf(expr.source(), "type %T cannot be casted to exprNumber", expr)
		return nil, nil, false
	}
	return caster.castTo(eval)
}

func buildNumberNode(scope scoper, expr exprNode, want ir.Type) (exprScalar, typeNode, bool) {
	wantArray, wantArrayOk := want.(*ir.ArrayType)
	if wantArrayOk {
		want = wantArray.DataType()
	}
	eval, ok := newEvaluator(scope, expr, want)
	if !ok {
		return nil, invalid, false
	}
	evalNode, _, castOk := castExprTo(eval, expr)
	typeNodeOut, typeNodeOk := toTypeNode(scope, eval.want())
	return evalNode, typeNodeOut, castOk && typeNodeOk
}

func buildDefaultNumberNode(scope scoper, expr exprNode) (exprNode, typeNode, bool) {
	want := ir.ScalarTypeK(ir.Float64Kind)
	// TODO(degris): all numbers are converted to float64. Convert integers to int64 instead.
	return buildNumberNode(scope, expr, want)
}
