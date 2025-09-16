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

func TestFunctionLiteral(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f() int32 {
	fn := func() int32 {
		return 10
	}
	return fn()
}
`,
		},
		testbuild.Decl{
			Src: `
func f(x float32) (float32, func() int32) {
	fn := func() int32 {
		return 10
	}
	return x, fn
}
`,
		},
		testbuild.Decl{
			Src: `
func f(x float32) (float32, func() int32) {
	fn := func() int32 {
		return 10
	}
	return x, fn
}

func g() (float32, int32) {
	x, fn := f(10.0)
	return x, fn()
}
`,
		},
		testbuild.Decl{
			Src: `
func f(a float32) func() float32 {
	return func() float32 {
		return a
	}
}

func g() float32 {
	fn := f(10)
	a := fn()
	return a
}
`,
		},
		testbuild.Decl{
			Src: `
func f(a float32) func() float32 {
	return func() float32 {
		return a
	}
}

func g() float32 {
	fn := f(10)
	b := fn()
	return b
}
`,
		},
		testbuild.Decl{
			Src: `
func f(x float32) float32 {
	return x
}

func F(x float32) func() float32 {
	return func() float32 {
		return f(x)
	}
}
			`,
		},
	)
}
