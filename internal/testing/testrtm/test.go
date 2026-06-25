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

package testrtm

import (
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
)

type factory struct {
	rtm  *api.Runtime
	srcs []testbuild.TestFactory
}

// Factory returns a test factory to run GX code given a backend.
func Factory(rtm *api.Runtime, srcs ...testbuild.TestFactory) testbuild.TestFactory {
	return factory{
		rtm:  rtm,
		srcs: srcs,
	}
}

func (f factory) compile(bld *testbuild.Builder, srcTest testbuild.WithName) ([]testbuild.Test, error) {
	pkg, err := srcTest.Run(bld)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, nil
	}
	fns, err := FindTests(pkg)
	if err != nil {
		return nil, err
	}
	options, err := BuildCompileOptions(f.rtm, pkg)
	if err != nil {
		return nil, err
	}
	var tests []testbuild.Test
	for _, fn := range fns {
		tests = append(tests, funcTest{
			factory: &f,
			parent:  srcTest,
			pkg:     pkg,
			options: options,
			fun:     fn,
		})
	}
	return tests, nil
}

func (f factory) BuildTests(imps []importers.Importer) ([]testbuild.Test, error) {
	bld := testbuild.NewLocalBuilder(imps...)
	var tests []testbuild.Test
	for _, src := range f.srcs {
		srcTests, err := src.BuildTests(imps)
		if err != nil {
			return nil, err
		}
		for _, srcTest := range srcTests {
			srcTests, err = f.compile(bld, srcTest.(testbuild.WithName))
			if err != nil {
				return tests, err
			}
			tests = append(tests, srcTests...)
		}
	}
	return tests, nil
}

type funcTest struct {
	factory *factory
	parent  testbuild.WithName
	pkg     *ir.Package
	options []options.PackageOption
	fun     *ir.FuncDecl
}

var _ testbuild.WithName = funcTest{}

func (ft funcTest) Source() string {
	return ft.parent.Source()
}

func (ft funcTest) Name() string {
	return ft.parent.Name() + "/" + ft.fun.Name()
}

func (ft funcTest) Run(bld *testbuild.Builder) (*ir.Package, error) {
	tRunner, err := NewRunner(ft.factory.rtm, 0)
	if err != nil {
		return nil, err
	}
	if err := tRunner.Test(ft.fun, ft.options); err != nil {
		return nil, err
	}
	return ft.pkg, nil
}
