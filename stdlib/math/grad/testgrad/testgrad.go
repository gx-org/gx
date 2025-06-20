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

	"github.com/gx-org/gx/build/builder/testbuild"
)

// Func tests the computation of the gradient of a function.
type Func struct {
	// GradOf declares the function (which must be named `F`) to compute the gradient of.
	GradOf string

	// Want stores the source code of the expected gradient of the function
	Want string

	// Err is the substring expected if the compiler returns an error.
	Err string
}

// Source code of the declarations.
func (tt Func) Source() string {
	return tt.GradOf
}

// Run builds the declarations as a package, then compare to an expected outcome.
func (tt Func) Run(b *testbuild.Builder) error {
	pkg, err := b.Build(fmt.Sprintf(`
package test

import "grad"

//gx:=grad.Func(F, "x")
func gradF()

%s
`, tt.GradOf))
	if err != nil {
		return err
	}
	gotF := pkg.IR().FindFunc("gradF")
	if err := testbuild.CompareString(gotF.String(), tt.Want); err != nil {
		return err
	}
	return nil
}
