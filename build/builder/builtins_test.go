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

func TestBuiltinFuncs(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f() [2][2]int32 {
	a := [2][2]int32{}
	a = set(a, [2]int32{1, 2}, 0)
	a = set(a, [2]int32{3, 4}, 1)
	return a
}
`,
		},
		testbuild.Decl{
			Src: `
func f() int64 {
	a := [2]int32{0, 1}
	return len(a)
}
`,
		},
		testbuild.Decl{
			Src: `
func f() float32 {
	a := [2]int32{0, 1}
	return float32(len(a))
}
`,
		},
		testbuild.Decl{
			Src: `
func log([2]float32) [2]float32

func cast(int64) float32

func f(probs [2]float32) float32 {
	logprobs := log(probs)
	return cast(len(logprobs))
}
`,
		},
	)
}

func TestBuiltinErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f() int64 {
	return len() // ERROR incorrect number of arguments for len() (expected 1, found 0)
}
`,
		},
		testbuild.Decl{
			Src: `
func f() float32 {
	a := [2]int32{0, 1}
	return float32(len(a, a)) // ERROR incorrect number of arguments for len(a, a) (expected 1, found 2)
}
`,
		},
		testbuild.Decl{
			Src: `
type A struct{}

func f() float32 {
	a := A{}
	return float32(len(a)) // ERROR invalid argument: a (A) for built-in len
}
`,
		},
	)
}
