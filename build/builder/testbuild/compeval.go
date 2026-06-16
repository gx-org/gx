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

package testbuild

import (
	"fmt"
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/internal/interp/coreops"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

// CompEval declares some GX code and runs the compeval main function in that code.
type CompEval struct {
	// Src is the GX source code.
	Src string
	// EvalCanonical converts the output to a canonical value and use the string representation of that value.
	EvalCanonical bool
	// Want is the set of nodes that is expected from the compiler to build.
	// If nil (or length 0), the output of the compiler is not checked.
	Wants []string
}

// Source code of the declarations.
func (tt CompEval) Source() string {
	return tt.Src
}

func stringFromCanonical(el ir.Element) (string, error) {
	coreEl, err := cast.To[coreops.Element](el)
	if err != nil {
		return "", err
	}
	boolVal, err := cast.To[canonical.WithBool](coreEl.CanonicalExpr())
	if err != nil {
		return "", err
	}
	bl, err := boolVal.Bool()
	if err != nil {
		return "", err
	}
	return fmt.Sprint(bl), nil
}

func (tt CompEval) stringFromElement(ev ir.Evaluator, el ir.Element) (string, error) {
	if tt.EvalCanonical {
		return stringFromCanonical(el)
	}
	expr, err := ir.ToSingleExpr(ev, &ast.Ident{}, el)
	if err != nil {
		return "", err
	}
	return expr.SourceString(nil), nil
}

// Run builds the declarations as a package, then compare to an expected outcome.
func (tt CompEval) Run(b *Builder) error {
	bld := builder.NewWithLoader(&b.imp)
	pkg, err := build(bld, "", fmt.Sprintf(`
package test

%s
`, tt.Src))
	if err != nil {
		return err
	}
	const funcName = "test"
	irPkg := pkg.IR()
	fn := irPkg.FindFunc(funcName)
	if fn == nil {
		return errors.Errorf("%s function not found", funcName)
	}
	if !fn.FuncType().CompEval {
		return errors.Errorf("%s is not a compeval function", funcName)
	}
	fnDecl, isFuncDecl := fn.(*ir.FuncDecl)
	if !isFuncDecl {
		return errors.Errorf("%s needs a body", funcName)
	}
	hostEval := compeval.NewHostEvaluator(bld, cpevelements.NewMixFunc)
	itp, err := interp.New(hostEval, hostEval, cpevelements.MixedRunner(), nil)
	if err != nil {
		return err
	}
	outs, err := itp.EvalFunc(fnDecl, &elements.InputElements{})
	if err != nil {
		return err
	}
	if len(outs) != len(tt.Wants) {
		return errors.Errorf("%s returned %d elements but want %d", funcName, len(outs), len(tt.Wants))
	}
	const fileName = "src0.gx"
	file := irPkg.File(fileName)
	if file == nil {
		return errors.Errorf("cannot find file %s in package", fileName)
	}
	fitp, err := itp.ForFile(file)
	if err != nil {
		return err
	}
	for i, out := range outs {
		got, err := tt.stringFromElement(fitp, out)
		if err != nil {
			return err
		}
		want := tt.Wants[i]
		if got != want {
			return errors.Errorf("got expression %d:\n%s\nbut want:\n%s", i, got, want)
		}
	}
	return nil
}
