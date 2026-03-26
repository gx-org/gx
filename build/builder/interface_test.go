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
)

func TestInterface(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type I interface {
	A() string
}

type S struct {}

func (S) A() string {
	return "Hi!"
}

func f() I {
	return S{}
}
`,
		},
	)
}

func TestInterfaceError(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type I interface {
	A() string
}

type S struct {}

func (S) B() string {
	return "Hi!"
}

func f() I {
	return S{} // ERROR cannot use S as I value in return statement: I does not implement S (missing method A)
}
`,
		},
		testbuild.Decl{
			Src: `
type I interface {
	A() string
}

type S struct {}

func (S) A() int32 {
	return 0
}

func f() I {
	return S{} // ERROR cannot use S as I value in return statement: I does not implement S (wrong type for method A)
}
`,
		},
	)
}
