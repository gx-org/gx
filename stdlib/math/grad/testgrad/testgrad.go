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

	// GradOf lists all the functions for which we compute the gradient of.
	// If GradOf is empty, we compute the gradient of F.
	GradOf []string

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

func (tt Func) buildSourceCode() string {
	declImportName := ""
	callImportName := "grad"
	if tt.GradImportName != "" {
		declImportName = tt.GradImportName
		callImportName = tt.GradImportName
	}
	grads := &strings.Builder{}
	for _, fnName := range tt.GradOf {
		grads.WriteString(fmt.Sprintf(
			`
//gx:=%s.Func(%s, "x")
func grad%s()
`,
			callImportName, fnName, fnName))
	}
	return fmt.Sprintf(`
package test

import %s"grad"

%s

%s
`, declImportName+" ", grads.String(), tt.Src)
}

// Run builds the declarations as a package, then compare to an expected outcome.
func (tt Func) Run(b *testbuild.Builder) error {
	if len(tt.GradOf) == 0 {
		tt.GradOf = []string{"F"}
	}
	// Build the package.
	src := tt.buildSourceCode()
	pkg, err := b.Build(src)
	if err != nil {
		return tt.checkError(err)
	}
	pkgIR := pkg.IR()
	// Check the gradient of the default function F.
	// checkFunc returns a nil error if tt.Want is empty.
	if err := checkFunc(pkgIR, "gradF", tt.Want); err != nil {
		return err
	}
	// Check other functions we expect.
	for name, src := range tt.Wants {
		if err := checkFunc(pkgIR, name, src); err != nil {
			return err
		}
	}
	// Check that functions have been built multiple times.
	funcs := make(map[string]bool)
	for _, fn := range pkg.IR().Decls.Funcs {
		_, found := funcs[fn.Name()]
		if found {
			return errors.Errorf("function %s has been built more than one time", fn.Name())
		}
		funcs[fn.Name()] = true
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
	if want == "" {
		return nil
	}
	gotF := pkg.FindFunc(name)
	if gotF == nil {
		return errors.Errorf("cannot find function %s. Available functions are %v", name, listFunc(pkg))
	}
	return testbuild.CompareString(gotF.String(), want)
}
