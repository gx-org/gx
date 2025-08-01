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
	"slices"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
)

type (
	// Canonical is a canonical value with a IR representation.
	Canonical interface {
		Expr() (AssignableExpr, error)
	}

	// Importer imports packages given their path.
	Importer interface {
		// Import a package given its path.
		Import(pkgPath string) (*Package, error)
	}

	// Element is a value returned by the evaluator.
	Element interface {
		Type() Type
	}

	// Evaluator evaluates IR expressions into canonical values.
	Evaluator interface {
		File() *File
		EvalExpr(Expr) (Element, error)
	}

	// Fetcher represents a scope in the compiler.
	Fetcher interface {
		Evaluator
		fmterr.ErrAppender
		BuildExpr(ast.Expr) (Expr, bool)
		IsDefined(string) bool
	}
)

func appendIdent(done map[string]bool, ids []*ValueRef, id *ValueRef) []*ValueRef {
	if done[id.Src.Name] {
		return ids
	}
	done[id.Src.Name] = true
	return append(ids, id)
}

func idents(done map[string]bool, expr Expr) ([]*ValueRef, error) {
	if done == nil {
		done = make(map[string]bool)
	}
	switch exprT := expr.(type) {
	case *ValueRef:
		return appendIdent(done, nil, exprT), nil
	case *NumberInt:
		return nil, nil
	case *NumberFloat:
		return nil, nil
	case *NumberCastExpr:
		return idents(done, exprT.X)
	case *ParenExpr:
		return idents(done, exprT.X)
	case *UnaryExpr:
		return idents(done, exprT.X)
	case *BinaryExpr:
		xDeps, err := idents(nil, exprT.X)
		if err != nil {
			return nil, err
		}
		yDeps, err := idents(nil, exprT.Y)
		if err != nil {
			return nil, err
		}
		var all []*ValueRef
		for _, id := range slices.Concat(xDeps, yDeps) {
			all = appendIdent(done, all, id)
		}
		return all, nil
	case AtomicValue:
		return nil, nil
	default:
		return nil, errors.Errorf("cannot get constant expression dependencies: expression %T not supported", expr)
	}
}

// Idents returns a slice of all identifiers used in an expression.
func Idents(expr Expr) ([]*ValueRef, error) {
	return idents(nil, expr)
}
