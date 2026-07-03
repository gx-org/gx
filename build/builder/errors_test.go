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

package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/stdlib"
)

func TestErrorBuiltin(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func someError() error
`,
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.ErrorType()),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `
type ArgError struct {
}

// gx:compeval
func (ArgError) Error() string {
	return "Hello"
}

// gx:compeval
func f() error {
	return ArgError{}
}`,
		},
	)
}

func TestCPErrors(t *testing.T) {
	testbuild.RunWith(t,
		[]importers.Importer{stdlib.Importer()},
		testbuild.Decl{
			Src: `
import "cperrors"

//gx:compeval
func argError() (int, error) {
	return 2, cperrors.Argf(1, "some error about argument 1")
}

func f(a, b [2]float32) [argError()]float32

func g() [2]float32 {
	a := [2]float32{1, 2}
	return f(a, a) // ERROR some error about argument 1
}

`,
		},
	)
}

func TestErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type S struct{}

func New() S {
	return &S{} // ERROR unary operator & not supported
}
`,
		},
		testbuild.Decl{
			Src: `
type S struct{}

func New() S {
	return *S{} // ERROR expression *<expr> not supported
}
`,
		},
		testbuild.Decl{
			Src: `
func New() float32 {
	1 := 2 // ERROR non-name on left side of :=
	return 1
}
`,
		},
		testbuild.Decl{
			Src: `
func New() float32 {
	1 = 2 // ERROR non-name on left side of =
	return 1
}
`,
		},
		testbuild.Decl{
			Src: `
type ArgError struct {
	Error error
}

// gx:compeval
func f() error {
	return ArgError{Error: nil} // ERROR cannot use ArgError as error value in return statement: error does not implement ArgError (missing method Error)
}`,
		},
		testbuild.Decl{
			Src: `
type ArgError struct {}

func (ArgError) Error() int32 {
	return 0
}

func f() error {
	return ArgError{} // ERROR cannot use ArgError as error value in return statement: error does not implement ArgError (wrong type for method Error)
}`,
		},
	)
}
