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

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/require"
)

type compevalFactory struct {
	srcs []TestFactory
}

// CompEval returns a test factory to run GX code given a backend.
func CompEval(srcs ...TestFactory) TestFactory {
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

func (ft compevalFuncTest) Run(b *Builder) (*ir.Package, error) {
	bld := builder.New(b.Importers()...)
	hostEval := compeval.NewHostEvaluator(bld, compeval.RunFunc)
	itp, err := interp.New(hostEval, hostEval, cpevelements.MixedRunner(), nil)
	if err != nil {
		return nil, err
	}
	outs, err := itp.EvalFunc(ft.fun, &elements.InputElements{})
	var gotRequire string
	if rErr := require.ToError(err); rErr != nil {
		gotRequire = rErr.Err().Error()
		err = nil
	}
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
			s += fmt.Sprintf(" STRING FROM ELEMENT ERROR: %v", err)
		}
		return s
	})
	if gotRequire != "" {
		if got != "" {
			got += "\n"
		}
		got = fmt.Sprintf("%sREQUIRE ERROR: %s", got, gotRequire)
	}
	return ft.pkg, CheckOutput(ft.fun, outs, got)
}
