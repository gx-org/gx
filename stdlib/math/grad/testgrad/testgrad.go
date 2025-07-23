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

// Package testgrad provides function to test autograd.
package testgrad

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
)

// Func tests the computation of the gradient of a function.
type Func struct {
	// Src declares the function (which must be named `F`) to compute the gradient of.
	Src string

	// Want stores the source code of the expected gradient of the function
	Want string

	// GradImportName is the name of the import of the grad package.
	// If empty, then the default import name is used.
	GradImportName string

	// Wants stores the source code of the expected synthetic auxiliary functions.
	Wants map[string]string

	// Err is the substring expected if the compiler returns an error.
	Err string
}

// Source code of the declarations.
func (tt Func) Source() string {
	return tt.Src
}

func (tt Func) checkError(err error) error {
	if tt.Err == "" {
		// No error expected
		return err
	}
	msg := err.Error()
	if strings.Contains(msg, tt.Err) {
		return nil
	}
	return errors.Errorf("got error:\n%s\nbut want an expected error which contains %q", err, tt.Err)
}

// Run builds the declarations as a package, then compare to an expected outcome.
func (tt Func) Run(b *testbuild.Builder) error {
	declImportName := ""
	callImportName := "grad"
	if tt.GradImportName != "" {
		declImportName = tt.GradImportName
		callImportName = tt.GradImportName
	}
	src := fmt.Sprintf(`
package test

import %s"grad"

//gx:=%s.Func(F, "x")
func gradF()

%s
`, declImportName+" ", callImportName, tt.Src)
	pkg, err := b.Build(src)
	if err != nil {
		return tt.checkError(err)
	}
	pkgIR := pkg.IR()
	if err := checkFunc(pkgIR, "gradF", tt.Want); err != nil {
		return err
	}
	for name, src := range tt.Wants {
		if err := checkFunc(pkgIR, name, src); err != nil {
			return err
		}
	}
	return nil
}

func listFunc(pkg *ir.Package) []string {
	fns := pkg.Decls.Funcs
	ss := make([]string, len(fns))
	for i, fn := range pkg.Decls.Funcs {
		ss[i] = fn.Name()
	}
	return ss
}

func checkFunc(pkg *ir.Package, name string, want string) error {
	gotF := pkg.FindFunc(name)
	if gotF == nil {
		return errors.Errorf("cannot find function %s. Available functions are %v", name, listFunc(pkg))
	}
	return testbuild.CompareString(gotF.String(), want)
}
