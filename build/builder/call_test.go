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

func TestCallNoReturn(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func g(a int32) {
	a = a + 2
}

func f() int32 {
	a := int32(0)
	g(a)
	return a
}
`,
		},
	)
}

func TestCallMultipleReturns(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func g() (uint64, uint32)

func f() (uint64, uint32) {
	a, b := g()
	return a, b
}
`,
		},
		testbuild.Decl{
			Src: `
func g() (uint64, uint32)

func f() (uint64, uint32) {
	a, _ := g()
	return a, uint32(a)
}
`,
		},
		testbuild.Decl{
			Src: `
func g() (uint64, uint32)

func f() (a uint64, b uint32) {
	a, b = g()
	return
}
`,
		},
		testbuild.Decl{
			Src: `
func g() (int32, [2]float32)

func f() (a, b [2]float32) {
	_, a = g()
	_, b = g()
	return
}`,
		},
	)
}

func TestCallOperators(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func cast[S interface{float64}](S) uint64

func f() uint64 {
	v := cast[float64](1.0)
	return v
}
`,
		},
	)
}

func TestCallSliceArgs(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
var A, B intlen

func f[T int32]([A][B][1]float32, []intidx) T

func fail(x [A][B][1]float32) float32 {
	return float32(f[int32](x, []intidx{0}))
}
`,
		},
	)
}

func TestCallKeyword(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f() string {
	return "undefined"
}

func TestF() string {
	trace(f)
	return f()
}
`,
		},
	)
}

func TestCallSameName(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type S struct {
	f float32
}

func inc(a float32) float32 {
	return a + 1
}

func F(a S) float32 {
	return inc(a.f)
}
`,
		},
	)
}

func TestCallErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func g() (float32, float32)

func f() float32 {
	return g()()
}
`,
			Err: "multiple value g() in single-value context",
		},
		testbuild.Decl{
			Src: `
func f() float32 {
	c, d = somefunc()
	return c
}
`,
			Err: "undefined: somefunc",
		},
		testbuild.Decl{
			Src: `
func g() (int32, float32, int64)

func f() (int32, float32) {
	a, b := g()
	return a, b
}
`,
			Err: "assignment mismatch: 2 variable(s) but g() returns 3 values",
		},
		testbuild.Decl{
			Src: `
func g(int32, float64) int64

func f() int64 {
	return g(1)
}
`,
			Err: "not enough arguments in call to g (expected 2, found 1)",
		},
		testbuild.Decl{
			Src: `
func g(int32, float64) int64

func f() int64 {
	return g(1, 2, 3)
}
`,
			Err: "too many arguments in call to g (expected 2, found 3)",
		},
	)
}
