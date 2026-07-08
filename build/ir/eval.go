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
	"fmt"
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir/irkind"
)

type (
	// Canonical is a canonical value with a IR representation.
	Canonical interface {
		Element
		WithExpr
	}

	// Importer imports packages given their path.
	Importer interface {
		// Import a package given its path.
		Import(pkgPath string) (*Package, error)
	}

	// FuncElement converts an element into an IR expressions.
	FuncElement interface {
		Func() Func
	}

	// WithExpr converts an element into an IR expressions.
	WithExpr interface {
		Expr(Evaluator, ast.Expr) ([]Expr, error)
	}

	// Evaluator evaluates IR expressions into canonical values.
	Evaluator interface {
		File() *File
		EvalExpr(Expr) (Element, error)
		Sub(*File, map[string]Element) (Evaluator, error)
	}

	// TypeCmp is the interface used to compare type to one another.
	TypeCmp interface {
		Evaluator
	}

	// CompEvalError is an error generated from evaluating GX code
	// and returned to the user as a compiler error.
	CompEvalError error

	// Fetcher represents a scope in the compiler.
	Fetcher interface {
		Evaluator
		fmterr.ErrAppender
	}

	// ErrSource appends errors at a code source location.
	ErrSource interface {
		fmterr.ErrAppender
		Source() ast.Expr
	}
)

// InvalidIdent is used as non-nil invalid expression.
var InvalidIdent = &Ident{
	Src:  &ast.Ident{Name: "<<<invalid>>>"},
	Stor: InvalidType(),
}

// CompEvalExpr evaluates an expression at compile time and returns
// the result of the evaluation as an IR expression.
func CompEvalExpr(ev Fetcher, src ast.Expr, x Expr) (_ []Expr, ok bool) {
	if x.Type().Kind() == irkind.MetaType {
		return []Expr{x}, true
	}
	el, err := ev.EvalExpr(x)
	if err != nil {
		return []Expr{InvalidIdent}, ev.Err().AppendAt(src, err)
	}
	res, err := ToExpr(ev, x.Expr(), el)
	if err != nil {
		return nil, ev.Err().AppendAt(x.Node(), err)
	}
	return res, true
}

// CompEvalExprSingle evaluates an expression at compile time and returns
// the result of the evaluation as a single IR expression.
func CompEvalExprSingle(ev Fetcher, src ast.Expr, x Expr) (Expr, bool) {
	exprs, ok := CompEvalExpr(ev, src, x)
	if !ok {
		return nil, ok
	}
	if len(exprs) != 1 {
		return InvalidIdent, ev.Err().AppendInternalf(src, "compeval of %s error: got %d expression(s) but want 1", x.SourceString(ev.File()), len(exprs))
	}
	return exprs[0], true

}

// ToExpr converts an element from the interpreter to an IR expression.
func ToExpr(ev Evaluator, src ast.Expr, el Element) ([]Expr, error) {
	toExpr, ok := el.(WithExpr)
	if !ok {
		return nil, errors.Errorf("cannot convert %T to an IR expression", el)
	}
	return toExpr.Expr(ev, src)
}

// ToSingleExpr converts an element to a single expression.
func ToSingleExpr(ev Evaluator, src ast.Expr, el Element) (Expr, error) {
	exprs, err := ToExpr(ev, src, el)
	if err != nil {
		return nil, err
	}
	if len(exprs) != 1 {
		return nil, fmterr.Internalf(ev.File().FileSet(), src, "got %d expression(s) but want 1", len(exprs))
	}
	return exprs[0], err
}

// ExprString first converts an element to an IR expression, then converts
// that expression into a GX source string.
func ExprString(ev Evaluator, src ast.Expr, el Element) string {
	expr, err := ToSingleExpr(ev, src, el)
	if err != nil {
		return fmt.Sprintf("<%s>", err)
	}
	return expr.SourceString(ev.File())
}
