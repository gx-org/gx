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
	"testing"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
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

func stringFromElement(ev ir.Evaluator, el ir.Element) (string, error) {
	algX, err := cmp.ToAlgExpr(ev, el)
	if err != nil {
		return "", err
	}
	simpX, err := cmp.SimplifyIR(ev, algX)
	if err != nil {
		return "", err
	}
	return ir.SourceString(ev.File(), simpX), nil
}

// Run builds the declarations as a package, then compare to an expected outcome.
func (tt CompEval) Run(b *Builder) (*ir.Package, error) {
	bld := builder.New(b.Importers()...)
	pkg, err := build(bld, "", fmt.Sprintf(`
package test

%s
`, tt.Src))
	if err != nil {
		return nil, err
	}
	const funcName = "test"
	irPkg := pkg.IR()
	fn := irPkg.FindFunc(funcName)
	if fn == nil {
		return nil, errors.Errorf("%s function not found", funcName)
	}
	if !fn.FuncType().CompEval {
		return nil, errors.Errorf("%s is not a compeval function", funcName)
	}
	fnDecl, isFuncDecl := fn.(*ir.FuncDecl)
	if !isFuncDecl {
		return nil, errors.Errorf("%s needs a body", funcName)
	}
	hostEval := compeval.NewHostEvaluator(bld, compeval.RunFunc)
	itp, err := interp.New(hostEval, hostEval, cpevelements.MixedRunner(), nil)
	if err != nil {
		return nil, err
	}
	outs, err := itp.EvalFunc(fnDecl, &elements.InputElements{})
	if err != nil {
		return nil, err
	}
	if len(outs) != len(tt.Wants) {
		return nil, errors.Errorf("%s returned %d elements but want %d", funcName, len(outs), len(tt.Wants))
	}
	const fileName = "src0.gx"
	file := irPkg.File(fileName)
	if file == nil {
		return nil, errors.Errorf("cannot find file %s in package", fileName)
	}
	fitp, err := itp.ForFile(file)
	if err != nil {
		return nil, err
	}
	for i, out := range outs {
		got, err := stringFromElement(fitp, out)
		if err != nil {
			return nil, err
		}
		want := tt.Wants[i]
		if got != want {
			return nil, errors.Errorf("got expression %d:\n%s\nbut want:\n%s", i, got, want)
		}
	}
	return irPkg, nil
}

type compevalFactory struct {
	srcs []TestFactory
}

// CompEvalFactory returns a test factory to run GX code given a backend.
func CompEvalFactory(srcs ...TestFactory) TestFactory {
	return compevalFactory{
		srcs: srcs,
	}
}

func (f compevalFactory) compile(bld *Builder, srcTest WithName) ([]Test, error) {
	pkg, err := srcTest.Run(bld)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, nil
	}
	fns := FindTests(pkg, true)
	var tests []Test
	for _, fn := range fns {
		tests = append(tests, compevalFuncTest{
			factory: &f,
			parent:  srcTest,
			pkg:     pkg,
			fun:     fn,
		})
	}
	return tests, nil
}

func (f compevalFactory) BuildTests(t *testing.T, imps []importers.Importer) ([]Test, error) {
	bld := NewLocalBuilder(imps...)
	var tests []Test
	for _, src := range f.srcs {
		srcTests, err := src.BuildTests(t, imps)
		if err != nil {
			return nil, err
		}
		for _, srcTest := range srcTests {
			testWithName := srcTest.(WithName)
			t.Run(testWithName.Name(), func(t *testing.T) {
				srcTests, err = f.compile(bld, testWithName)
				if err != nil {
					t.Error(err)
					return
				}
				tests = append(tests, srcTests...)
			})
		}
	}
	return tests, nil
}

type compevalFuncTest struct {
	factory *compevalFactory
	parent  WithName
	pkg     *ir.Package
	fun     *ir.FuncDecl
}

var _ WithName = compevalFuncTest{}

func (ft compevalFuncTest) Source() string {
	return ft.parent.Source()
}

func (ft compevalFuncTest) Name() string {
	return ft.parent.Name() + "/" + ft.fun.Name()
}

func (ft compevalFuncTest) Run(b *Builder) (*ir.Package, error) {
	bld := builder.New(b.Importers()...)
	hostEval := compeval.NewHostEvaluator(bld, compeval.RunFunc)
	itp, err := interp.New(hostEval, hostEval, cpevelements.MixedRunner(), nil)
	if err != nil {
		return nil, err
	}
	outs, err := itp.EvalFunc(ft.fun, &elements.InputElements{})
	if err != nil {
		return nil, err
	}
	fitp, err := itp.ForFile(ft.fun.File())
	if err != nil {
		return nil, err
	}
	got := BuildGot(outs, func(el ir.Element) string {
		s, err := stringFromElement(fitp, el)
		if err != nil {
			s += fmt.Sprintf(" EVAL ERROR: %v", err)
		}
		return s
	})
	return ft.pkg, CheckOutput(ft.fun, outs, got)
}
