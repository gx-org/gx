// Copyright 2026 Google LLC
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

// Package cmp provides helpers to compare one IR expression to another.
package cmp

import (
	"fmt"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
)

// Expr is an algebraic expression that can be simplified into a comparable expression.
type Expr interface {
	// Simplify the expression.
	Simplify(ir.SourceFile) (Comparable, error)
	String() string
}

// Comparable is an element that can be compared to another.
type Comparable interface {
	Equal(other Comparable) bool
	BuildIR() ir.Expr
	String() string
}

// Canonical is a canonical value with a IR representation.
type Canonical interface {
	ir.Element
	ir.WithExpr
	ir.StringShorter
	// AlgExpr converts the element into an algebra expression.
	AlgExpr(ir.Evaluator) (Expr, error)
}

// ToAlgExpr converts an element to an algebra expression.
func ToAlgExpr(eva ir.Evaluator, el ir.Element) (Expr, error) {
	elCan, err := cast.To[Canonical](el)
	if err != nil {
		return nil, err
	}
	return elCan.AlgExpr(eva)
}

// Equal compares if elements are equal.
func Equal(eva ir.Evaluator, x, y ir.Element) (bool, error) {
	xAlg, err := ToAlgExpr(eva, x)
	if err != nil {
		return false, err
	}
	yAlg, err := ToAlgExpr(eva, y)
	if err != nil {
		return false, err
	}
	return Compare(eva, xAlg, yAlg)
}

// CanonicalString returns the canonical string representation of an element.
// Only used for debugging.
func CanonicalString(eva ir.Evaluator, el ir.Element) string {
	elAlg, err := ToAlgExpr(eva, el)
	if err != nil {
		return fmt.Sprintf("unknown: %q", err.Error())
	}
	longS := elAlg.String()
	shortS, err := SimplifyString(eva, elAlg)
	if err != nil {
		return err.Error()
	}
	if longS == shortS {
		return longS
	}
	return fmt.Sprintf("{%s => %s}", longS, shortS)
}

// Compare x to y after simplifying the expressions.
func Compare(srcf ir.SourceFile, x, y Expr) (bool, error) {
	sX, err := x.Simplify(srcf)
	if err != nil {
		return false, err
	}
	sY, err := y.Simplify(srcf)
	if err != nil {
		return false, err
	}
	return sX.Equal(sY), nil
}

// SimplifyIR a algebra expression to an IR expression.
func SimplifyIR(srcf ir.SourceFile, x Expr) (ir.Expr, error) {
	s, err := x.Simplify(srcf)
	if err != nil {
		return nil, err
	}
	return s.BuildIR(), nil
}

// SimplifyString returns the string of the simplified algebraic expression.
func SimplifyString(srcf ir.SourceFile, x Expr) (string, error) {
	s, err := x.Simplify(srcf)
	if err != nil {
		return "", err
	}
	return s.BuildIR().SourceString(srcf.File()), nil
}
