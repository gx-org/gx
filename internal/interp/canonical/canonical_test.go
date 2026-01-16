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

package canonical_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"math/big"
	"testing"

	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/numbers"
	"github.com/gx-org/gx/tests/testing/prime"
)

func newFloat(f float64) *numbers.Float {
	bf := big.NewFloat(f)
	return numbers.NewFloat(elements.NewExprAt(nil, &ir.NumberFloat{
		Src: &ast.BasicLit{Value: bf.String()},
	}), bf)
}

func newInt64(i int64) canonical.Canonical {
	val, err := values.AtomIntegerValue(ir.Int64Type(), i)
	if err != nil {
		panic(err)
	}
	can, err := cpevelements.NewAtom(elements.NewExprAt(nil, &ir.NumberCastExpr{
		X: &ir.NumberInt{
			Src: &ast.BasicLit{Value: fmt.Sprint(i)},
		},
		Typ: ir.Int64Type(),
	}), val)
	if err != nil {
		panic(err)
	}
	return can
}

type cVar struct {
	name string
}

func newVar(name string) *cVar {
	return &cVar{name: name}
}

func (cv *cVar) Compare(x canonical.Comparable) (bool, error) {
	xT, ok := x.(*cVar)
	if !ok {
		return false, nil
	}
	return cv.name == xT.name, nil
}

func (cv *cVar) Simplify() canonical.Simplifier {
	return cv
}

func (cv *cVar) ShortString() string {
	return cv.name
}

func (cv *cVar) String() string {
	return cv.name
}

func TestCanonicalEval(t *testing.T) {
	tests := []struct {
		Expr       canonical.Simplifier
		Want       float64
		Str        string
		Simplified string
	}{
		{
			Expr: newFloat(42),
			Want: 42,
			Str:  "42",
		},
		{
			Expr: canonical.FromBinary(token.ADD, newFloat(5), newFloat(2)),
			Want: 7,
			Str:  "(+ 2 5)",
		},
		{
			Expr: canonical.FromBinary(token.MUL, newFloat(5), newFloat(2)),
			Want: 10,
			Str:  "(* 2 5)",
		},
		{
			Expr: canonical.FromBinary(token.SUB, newFloat(5), newFloat(2)),
			Want: 3,
			Str:  "(+ (- 2) 5)",
		},
		{
			Expr: canonical.FromBinary(token.QUO, newFloat(10), newFloat(2)),
			Want: 5,
			Str:  "(* (/ 2) 10)",
		},
		{
			Expr: canonical.NewExpr(token.ADD, newFloat(5), newFloat(2)),
			Want: 7,
			Str:  "(+ 2 5)",
		},
		{
			Expr: canonical.NewExpr(token.MUL, newFloat(5), newFloat(2)),
			Want: 10,
			Str:  "(* 2 5)",
		},
		{
			Expr: canonical.NewExpr(token.SUB, newFloat(5), newFloat(2)),
			Want: -7,
			Str:  "(- 2 5)",
		},
		{
			Expr: canonical.NewExpr(token.QUO, newFloat(10), newFloat(2)),
			Want: 0.1 * 0.5,
			Str:  "(/ 10 2)",
		},
		{
			Expr: canonical.NewExpr(token.MUL,
				newFloat(5),
				canonical.NewExpr(token.MUL, newFloat(4), newFloat(3)),
			),
			Want:       60,
			Str:        "(* (* 3 4) 5)",
			Simplified: "(* 3 4 5)",
		},
		{
			Expr: canonical.NewExpr(token.MUL,
				newFloat(5),
				canonical.NewExpr(token.MUL,
					canonical.NewExpr(token.MUL,
						newFloat(10),
						newFloat(4),
					),
					newFloat(3)),
			),
			Want:       600,
			Str:        "(* (* (* 10 4) 3) 5)",
			Simplified: "(* 10 3 4 5)",
		},
		{
			Expr: canonical.NewExpr(token.MUL,
				newFloat(1),
				newFloat(4),
				newFloat(5),
				newFloat(6),
			),
			Want:       120,
			Str:        "(* 1 4 5 6)",
			Simplified: "(* 4 5 6)",
		},
		{
			Expr: canonical.NewExpr(token.MUL,
				newInt64(1),
				newFloat(4),
				newFloat(5),
				newFloat(6),
			),
			Want:       120,
			Str:        "(* 4 5 6 int64(1))",
			Simplified: "(* 4 5 6)",
		},
		{
			Expr: canonical.NewExpr(token.ADD,
				newVar("x"),
				newFloat(1),
			),
			Str:        "(+ 1 x)",
			Simplified: "(+ 1 x)",
		},
		{
			Expr: canonical.NewExpr(token.SUB,
				newVar("x"),
				newFloat(1),
			),
			Str:        "(- 1 x)",
			Simplified: "(- 1 x)",
		},
		{
			Expr: canonical.NewExpr(token.MUL,
				newInt64(1),
				newFloat(4),
				newVar("x"),
			),
			Str:        "(* 4 int64(1) x)",
			Simplified: "(* 4 x)",
		},
		{
			Expr: canonical.NewExpr(token.QUO,
				canonical.NewExpr(token.MUL,
					newInt64(1),
					newFloat(4),
					newVar("x"),
				)),
			Str:        "(/ (* 4 int64(1) x))",
			Simplified: "(/ (* 4 x))",
		},
	}
	for i, test := range tests {
		reprGot := test.Expr.String()
		if reprGot != test.Str {
			t.Errorf("test %d: incorrect expression representation: got %s but want %s", i, reprGot, test.Str)
		}
		want := big.NewFloat(test.Want)
		valueGot := test.Expr.(canonical.Evaluable).Float()
		if valueGot == nil && test.Want != 0 {
			t.Errorf("test %d: incorrect expression value %s: got nil but want %v", i, test.Expr.String(), want)
		}
		if valueGot != nil && valueGot.Cmp(want) != 0 {
			t.Errorf("test %d: incorrect expression value %s: got %v but want %v", i, test.Expr.String(), valueGot, want)
		}
		if test.Simplified == "" {
			continue
		}
		simplified := test.Expr.Simplify()
		simplifiedGot := simplified.(canonical.Evaluable).Float()
		if simplifiedGot != nil && simplifiedGot.Cmp(want) != 0 {
			t.Errorf("test %d: incorrect simplified expression value: got %v but want %v", i, simplifiedGot, want)
		}
		simplifiedReprGot := simplified.String()
		if simplifiedReprGot != test.Simplified {
			t.Errorf("test %d: incorrect simplified expression representation: got %s but want %s", i, simplifiedReprGot, test.Simplified)
		}
	}
}

var xVar = newVar("x")

type exprGenerator struct {
	num   int
	prime *prime.Prime
}

func (eg *exprGenerator) nextCanonical() canonical.Simplifier {
	eg.num++
	// Inject named variables once in a while.
	// Not too often so that most expressions can be evaluated and simplified.
	if eg.num%71 == 0 {
		return xVar
	}
	return newFloat(float64(eg.prime.Next()))
}

// buildAllExprs builds all possible combination of binary operations as deep as indicated by level.
func (eg *exprGenerator) buildAllExprs(level int) []canonical.Simplifier {
	if level == 0 {
		return []canonical.Simplifier{eg.nextCanonical()}
	}
	var exprs []canonical.Simplifier
	for _, op := range []token.Token{token.ADD, token.SUB, token.MUL, token.QUO} {
		for _, xi := range eg.buildAllExprs(level - 1) {
			for _, yi := range eg.buildAllExprs(level - 1) {
				exprs = append(exprs, canonical.FromBinary(op, yi, xi))
			}
		}
	}
	return exprs
}

var (
	relativePrecision = big.NewFloat(1e-11)
	zero              = big.NewFloat(0)
)

func TestCanonicalSimplify(t *testing.T) {
	exprs := (&exprGenerator{prime: prime.New(11)}).buildAllExprs(3)
	numValChecks := 0
	numNoValues := 0
	for i, expr := range exprs {
		wantVal := expr.(canonical.Evaluable).Float()
		wantRepr := expr.String()
		simplified := expr.Simplify()
		simplifiedVal := simplified.(canonical.Evaluable).Float()
		wantValAfter := expr.(canonical.Evaluable).Float()
		if wantValAfter == nil {
			numNoValues++
			continue
		}
		numValChecks++
		if wantVal.Cmp(wantValAfter) != 0 {
			t.Errorf("test %d: expression %s with value %s has been modified to %s and %s", i, wantRepr, wantVal.String(), expr.String(), wantValAfter.String())
			continue
		}
		diff := new(big.Float).Abs(new(big.Float).Sub(wantVal, simplifiedVal))
		if diff.Cmp(zero) == 0 {
			continue
		}
		relativeDiff := new(big.Float).Quo(diff, wantVal)
		if relativeDiff.Cmp(relativePrecision) > 0 {
			relPercent := new(big.Float).Mul(relativeDiff, big.NewFloat(100))
			t.Errorf("test %d: simplified expression %s value is not equal to the original expression %s value: got %s but want %s (relative difference=%s%%)", i, expr.String(), simplified.String(), simplifiedVal.String(), wantVal.String(), relPercent.String())
		}
	}
	// Check that the evaluation of expressions with named variables have been generated sometimes, but not too often.
	wantNumNoValues := 1651
	if numNoValues != wantNumNoValues {
		t.Errorf("the number of expressions with variables in the test changed to %d from %d", numNoValues, wantNumNoValues)
	}
	wantNumValChecks := 14733
	if numValChecks != wantNumValChecks {
		t.Errorf("the number of expressions that can be evaluated in the test changed to %d from %d", numValChecks, wantNumValChecks)
	}
}
