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
		Expr(Evaluator, ast.Expr) (Expr, CompEvalError, error)
	}

	// Evaluator evaluates IR expressions into canonical values.
	Evaluator interface {
		File() *File
		EvalExpr(Expr) (Element, error)
		ToCompEvalError(ast.Expr, Element) (CompEvalError, error)
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
)

// InvalidIdent is used as non-nil invalid expression.
var InvalidIdent = &Ident{
	Src:  &ast.Ident{Name: "<<<invalid>>>"},
	Stor: InvalidType(),
}

// CompEvalExpr evaluates an expression at compile time and returns
// the result of the evaluation as an IR expression.
func CompEvalExpr(ev Fetcher, src ast.Node, x Expr) (Expr, bool) {
	if x.Type().Kind() == irkind.MetaType {
		return x, true
	}
	el, err := ev.EvalExpr(x)
	if err != nil {
		return InvalidIdent, ev.Err().Append(err)
	}
	res, cpErr, err := ToExpr(ev, x.Expr(), el)
	if err != nil {
		return InvalidIdent, ev.Err().AppendInternalf(x.Node(), "cannot convert element %T to an IR expression: %v", el, err)
	}
	if cpErr != nil {
		return InvalidIdent, ev.Err().AppendAt(src, cpErr)
	}
	return res, true
}

// ToExpr converts an element from the interpreter to an IR expression.
func ToExpr(ev Evaluator, src ast.Expr, el Element) (Expr, CompEvalError, error) {
	toExpr, ok := el.(WithExpr)
	if !ok {
		return nil, nil, errors.Errorf("cannot convert %T to an IR expression", el)
	}
	return toExpr.Expr(ev, src)
}

// ExprString first converts an element to an IR expression, then converts
// that expression into a GX source string.
func ExprString(ev Evaluator, src ast.Expr, el Element) string {
	expr, cpErr, err := ToExpr(ev, src, el)
	if uniErr := UnifyErr(cpErr, err); uniErr != nil {
		return fmt.Sprintf("<%s>", err)
	}
	return expr.SourceString(ev.File())
}
