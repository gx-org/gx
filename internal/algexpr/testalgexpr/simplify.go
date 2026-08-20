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

package testalgexpr

import (
	"fmt"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
)

// Simplify two GX algebraic expressions to one another.
type Simplify struct {
	// FileSource are all declarations to include in the package
	// where expressions are evaluated.
	PkgDecl string

	// Vars in which the expressions is being defined.
	Vars map[string]ir.Type

	// GX expressions to simplify.
	X string

	// Want the same expression but simplified.
	Want string
}

// Run the test.
func (tt Simplify) Run(b *testbuild.Builder) (*ir.Package, error) {
	pkg, err := b.Build("", fmt.Sprintf("package test\n%s", tt.PkgDecl))
	if err != nil {
		return pkg.IR(), err
	}
	ev, err := b.EvaluatorFor(pkg, tt.Vars)
	if err != nil {
		return nil, err
	}
	el, err := evalExpr(ev, tt.X)
	if err != nil {
		return pkg.IR(), err
	}
	algX, err := cmp.ToAlgExpr(ev, el)
	if err != nil {
		return pkg.IR(), err
	}
	simpX, err := cmp.SimplifyIR(ev, algX)
	if err != nil {
		return pkg.IR(), err
	}
	got := ir.SourceString(ev.File(), simpX)
	if got != tt.Want {
		return pkg.IR(), fmt.Errorf("simplification of {Source: %s Canonical: %s} returned expression %s but want %s", tt.X, cmp.CanonicalString(ev, el), got, tt.Want)
	}
	return pkg.IR(), nil
}

// Source code of the test.
func (tt Simplify) Source() string {
	return tt.X
}
