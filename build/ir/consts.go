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

package ir

import (
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/internal/exprdeps"
)

var falseValue = &AtomicValueT[bool]{
	Val: false,
	Typ: BoolType(),
}

// False returns an atomic value equal to false.
func False() *AtomicValueT[bool] {
	return falseValue
}

var trueValue = &AtomicValueT[bool]{
	Val: true,
	Typ: BoolType(),
}

// True returns an atomic value equal to true.
func True() *AtomicValueT[bool] {
	return trueValue
}

// Deps returns the dependencies of a constant expression.
func (expr *ConstExpr) Deps() []*ast.Ident {
	if expr.Val == nil {
		return nil
	}
	return exprdeps.Idents(expr.Val.Expr())
}

func (decls *Declarations) findConstExpr(name string) *ConstExpr {
	for _, decl := range decls.Consts {
		for _, expr := range decl.Exprs {
			if expr.VName.Name == name {
				return expr
			}
		}
	}
	return nil
}

func (decls *Declarations) appendConstExprs(done map[string]bool, expr *ConstExpr) ([]*ConstExpr, error) {
	if done[expr.VName.Name] {
		return nil, nil
	}
	done[expr.VName.Name] = true
	deps := expr.Deps()
	var exprs []*ConstExpr
	for _, dep := range deps {
		exprDep := decls.findConstExpr(dep.Name)
		if exprDep == nil {
			return nil, errors.Errorf("cannot find expression definition for %s", dep.Name)
		}
		depExprs, err := decls.appendConstExprs(done, exprDep)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, depExprs...)
	}
	exprs = append(exprs, expr)
	return exprs, nil
}

// ConstExprs returns all constant expressions with respect to their dependency.
// That is expressions are ordered such that an expression is always after
// its dependencies. If a cycle exists, then the ordering between the expressions
// in that cycle is not defined.
func (decls *Declarations) ConstExprs() ([]*ConstExpr, error) {
	var exprs []*ConstExpr
	done := make(map[string]bool)
	for _, decl := range decls.Consts {
		for _, expr := range decl.Exprs {
			toAppend, err := decls.appendConstExprs(done, expr)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, toAppend...)
		}
	}
	return exprs, nil
}
