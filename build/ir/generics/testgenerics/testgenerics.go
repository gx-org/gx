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

// Package testgenerics provides tests for GX generic algorithms, for example
// for type inference and function specialisation.
package testgenerics

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir/generics"
	"github.com/gx-org/gx/build/ir"
)

type (
	// Call specifies how to call the generic function.
	Call struct {
		// Args are the argument expressions to pass to the function call.
		Args []string

		// Signature of the desired function after inference.
		Want string

		// Err is the substring expected if the compiler returns an error.
		Err string
	}

	// Infer tests inference for a function call.
	Infer struct {
		// Src of the function to test.
		Src string

		// Calls to test.
		Calls []Call
	}
)

// Source code of the declarations.
func (tt Infer) Source() string {
	return fmt.Sprintf(`
package test

%s
`, tt.Src)
}

func (call Call) run(pkg *builder.IncrementalPackage, fnValRef *ir.FuncValExpr) error {
	args := make([]ir.AssignableExpr, len(call.Args))
	for i, arg := range call.Args {
		argI, err := pkg.BuildExpr(arg)
		if err != nil {
			return errors.Errorf("cannot build argument %d defined as %q: %v", i, arg, err)
		}
		args[i] = argI.(ir.AssignableExpr)
	}
	fetcher, err := pkg.Fetcher()
	if err != nil {
		return err
	}
	fnGot, _ := generics.Infer(fetcher, fnValRef, args)
	if err := testbuild.CheckError(call.Err, fetcher.Err().Errors().ToError()); err != nil {
		return err
	}
	if call.Want == "" {
		return nil
	}
	got := fnGot.FuncType().String()
	if got != call.Want {
		return errors.Errorf("incorrect signature: got %s but want %s", got, call.Want)
	}
	return nil
}

// Run builds the declarations as a package, then compare to an expected outcome.
func (tt Infer) Run(b *testbuild.Builder) error {
	// Build the package.
	bld := builder.New(b.Loader())
	pkg := bld.NewIncrementalPackage("test")
	src := tt.Source()
	if err := pkg.Build(src); err != nil {
		return errors.Errorf("unexpected error in function signature: %v\nPackage source code:\n%s", err, src)
	}
	// Find the declared function.
	pkgIR := pkg.IR()
	funcs := pkgIR.Decls.Funcs
	if len(funcs) != 1 {
		return errors.Errorf("got %d function declarations but want only 1", len(funcs))
	}
	fn := funcs[0]
	fnValRef := ir.NewFuncValExpr(&ir.ValueRef{Stor: fn}, fn)
	// Evaluates all calls.
	errs := &fmterr.Errors{}
	for i, call := range tt.Calls {
		err := call.run(pkg, fnValRef)
		if err != nil {
			errs.Append(fmt.Errorf("call %d error: %v", i, err))
		}
	}
	return errs.ToError()
}
