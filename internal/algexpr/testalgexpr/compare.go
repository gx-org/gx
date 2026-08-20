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
	"strings"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
)

// Compare two GX algebraic expressions to one another.
type Compare struct {
	// FileSource are all declarations to include in the package
	// where expressions are evaluated.
	PkgDecl string

	// Vars in which the expressions is being defined.
	Vars map[string]ir.Type

	// GX expressions to compare.
	X, Y string

	// Ys are additional expressions to compare to X.
	Ys []string

	// NotEqual is set to true if two expressions are expected to be different.
	NotEqual bool
}

func (tt Compare) compareTo(eva *testbuild.Evaluator, x ir.Element, srcY string) error {
	if srcY == "" {
		return nil
	}
	y, err := evalExpr(eva, srcY)
	if err != nil {
		return err
	}
	eq, err := cmp.Equal(eva, x, y)
	if err != nil {
		return err
	}
	if eq != !tt.NotEqual {
		return fmt.Errorf("{Source: %s Canonical: %s} == {Source: %s Canonical: %s} returned %v but want %v", tt.X, cmp.CanonicalString(eva, x), srcY, cmp.CanonicalString(eva, y), eq, !tt.NotEqual)
	}
	return nil
}

// Run the test.
func (tt Compare) Run(b *testbuild.Builder) (*ir.Package, error) {
	pkg, err := b.Build("", fmt.Sprintf("package test\n%s", tt.PkgDecl))
	if err != nil {
		return pkg.IR(), err
	}
	ev, err := b.EvaluatorFor(pkg, tt.Vars)
	if err != nil {
		return nil, err
	}
	x, err := evalExpr(ev, tt.X)
	if err != nil {
		return pkg.IR(), err
	}
	for _, srcY := range append([]string{tt.Y}, tt.Ys...) {
		if err := tt.compareTo(ev, x, srcY); err != nil {
			return pkg.IR(), err
		}
	}
	return pkg.IR(), nil
}

// Source code of the test.
func (tt Compare) Source() string {
	all := tt.Ys
	if tt.Y != "" {
		all = append([]string{tt.Y}, tt.Ys...)
	}
	yS := strings.Join(all, ", ")
	if len(all) > 1 {
		yS = fmt.Sprintf("[%s]", yS)
	}
	return fmt.Sprintf("%s == %s", tt.X, yS)
}
