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

package testgrad

import (
	"fmt"
	"go/ast"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder/testbuild"
)

// VJP tests the computation of the Vector-Jacobian product of a function.
type VJP struct {
	// Src declares the function (which must be named `F`) to compute the VJP of.
	Src string

	// GradOf lists all the functions for which we compute the VJP of.
	// If GradOf is empty, we compute the VJP of F.
	GradOf []string

	// Want stores the source code of the expected VJP of the function
	Want string

	// GradImportName is the name of the import of the grad package.
	// If empty, then the default import name is used.
	GradImportName string

	// WantExprs compares expanded expressions to required GX source code.
	WantExprs map[string]string

	// List of imports to include in the source.
	Imports []string

	// Err is the substring expected if the compiler returns an error.
	Err string
}

// Source code of the declarations.
func (tt VJP) Source() string {
	return tt.Src
}

func (tt VJP) buildSourceCode() string {
	declImportName := ""
	callImportName := "grad"
	if tt.GradImportName != "" {
		declImportName = tt.GradImportName
		callImportName = tt.GradImportName
	}
	grads := &strings.Builder{}
	for _, fnName := range tt.GradOf {
		fmt.Fprintf(grads,
			`
//gx:=%s.VJP(%s)
func vjp%s()
`,
			callImportName, fnName, fnName)
	}
	var imports string
	for _, imp := range tt.Imports {
		imports += fmt.Sprintf("import \"%s\"\n", imp)
	}
	return fmt.Sprintf(`
package test

%s
import %s"math/grad"

%s

%s
`, imports, declImportName+" ", grads.String(), tt.Src)
}

var gradImport = &ast.ImportSpec{
	Path: &ast.BasicLit{Value: strconv.Quote("math/grad")},
}

// Run builds the declarations as a package, then compare to an expected outcome.
func (tt VJP) Run(b *testbuild.Builder) error {
	if len(tt.GradOf) == 0 {
		tt.GradOf = []string{"F"}
	}
	// Build the package.
	src := tt.buildSourceCode()
	pkg, err := b.Build("", src)
	if err != nil {
		return testbuild.CheckError(tt.Err, err)
	}
	pkgIR := pkg.IR()
	// Check the VJP of the default function F.
	// checkVJP returns a nil error if tt.Want is empty.
	if err := checkFunc(pkgIR, "vjpF", tt.Want); err != nil {
		return err
	}
	// Check other functions we expect.
	for expr, want := range tt.WantExprs {
		if err := testbuild.CheckExpandedExpr(pkg, expr, want, gradImport); err != nil {
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
