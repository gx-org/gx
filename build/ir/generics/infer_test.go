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

package generics_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir/generics/testgenerics"
)

func TestInfer(t *testing.T) {
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: `
package dtype

type (
	Floats interface {
		bfloat16 | float32 | float64
	}

	Ints interface {
		int32 | int64 | uint32 | uint64
	}

	Nums interface {
		Floats | Ints
	}
)
`,
		},
		testgenerics.Infer{
			Src: "func F(int32) int32",
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"int32(2)"},
					Want: "func(int32) int32",
				},
				testgenerics.Call{
					// Infer only checks generic arguments.
					// No error will be detected and the function signature will not change.
					// The error will be reported later by the compiler
					// when all arguments will be checked against their matching parameters.
					Args: []string{"int64(2)"},
					Want: "func(int32) int32",
				},
			},
		},
		testgenerics.Infer{
			Src: `
import "dtype"
func F[T dtype.Ints](T) T
`,
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"true"},
					Err:  "bool does not satisfy dtype.Ints",
				},
				testgenerics.Call{
					Args: []string{"int32(2)"},
					Want: "func[int32](int32) int32",
				},
				testgenerics.Call{
					Args: []string{"int64(2)"},
					Want: "func[int64](int64) int64",
				},
			},
		},
		testgenerics.Infer{
			Src: `
import "dtype"
func F[T dtype.Ints](x T, y T) T
`,
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"int32(2)", "int64(4)"},
					Err:  "type int64 does not match type int32",
				},
				testgenerics.Call{
					Args: []string{"int32(2)", "int32(4)"},
					Want: "func[int32](x int32, y int32) int32",
				},
			},
		},
		testgenerics.Infer{
			Src: `
import "dtype"
type S[T dtype.Ints] struct {
	a T
}
func F[T dtype.Ints](s S[T]) T
`,
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"S[int32]{a: 2}"},
					Want: "func[int32](s test.S[int32]) int32",
				},
				testgenerics.Call{
					Args: []string{"S[int64]{a: 2}"},
					Want: "func[int64](s test.S[int64]) int64",
				},
			},
		},
		testgenerics.Infer{
			Src: `
type T[S []int] struct {
	a [S...]float32
}
func F[S []int](t T[S]) [S...]float32
`,
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"T[[]int{2, 3}]{}"},
					Want: "func[[]int{2, 3}](t test.T[[]int{2, 3}]) [2][3]float32",
				},
			},
		},
		testgenerics.Infer{
			Src: `
import "dtype"
const N4 = 4
type S[T dtype.Floats, N int] struct {
	a [N]T
}
func F[T dtype.Floats, N int](s S[T, N]) [N]T
`,
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"S[float32, N4]{}"},
					Want: "func[float32][4](s test.S[float32, 4]) [4]float32",
				},
			},
		},
		testgenerics.Infer{
			Src: `
import "dtype"
type U[S []int, T dtype.Floats] struct {
	x [S...]T
}
type V[S []int, T dtype.Floats] struct {
	u U[S, T]
}
func F[S []int, T dtype.Floats](v V[S, T]) [S...]T
`,
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"V[[]int{2, 3}, float32]{}"},
					Want: "func[[]int{2, 3}][float32](v test.V[[]int{2, 3}, float32]) [2][3]float32",
				},
			},
		},
		testgenerics.Infer{
			Src: `
type T[S []int] struct {
	a [S...]float32
}
func F[S []int](t T[S], x [S...]float32) [S...]float32
`,
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"T[[]int{2, 3}]{}", "[2][3]float32{}"},
					Want: "func[[]int{2, 3}](t test.T[[]int{2, 3}], x [2][3]float32) [2][3]float32",
				},
				testgenerics.Call{
					Args: []string{"T[[]int{2, 3}]{}", "[4][5]float32{}"},
					Err:  "[]int{4, 5} does not match inferred value []int{2, 3} for S",
				},
			},
		},
		testgenerics.Infer{
			Src: `
import "dtype"
type S[T dtype.Ints] struct {
	a T
}
func F[T dtype.Ints](s1 S[T], s2 S[T]) T
`,
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"S[int32]{a: 1}", "S[int64]{a: 2}"},
					Err:  "type int64 does not match type int32 for T",
				},
			},
		},
		testgenerics.Infer{
			Src: `
import "dtype"
type S[T dtype.Ints] struct {
	a T
	b int32
}
type OneField struct {
	a int32
}
type DiffFirst struct {
	x int32
	b int32
}
type DiffSecond struct {
	a int32
	c int32
}
func F[T dtype.Ints](n OneField, s S[T], x T) T
`,
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"OneField{a: 1}", "int32(2)", "int64(4)"},
					Want: "func[int64](n test.OneField, s test.S[int64], x int64) int64",
				},
				testgenerics.Call{
					Args: []string{"OneField{a: 1}", "OneField{a: 1}", "int64(4)"},
					Want: "func[int64](n test.OneField, s test.S[int64], x int64) int64",
				},
				testgenerics.Call{
					Args: []string{"OneField{a: 1}", "DiffFirst{x: 1, b: 2}", "int64(4)"},
					Want: "func[int64](n test.OneField, s test.S[int64], x int64) int64",
				},
				testgenerics.Call{
					Args: []string{"OneField{a: 1}", "DiffSecond{a: 1, c: 2}", "int64(4)"},
					Want: "func[int64](n test.OneField, s test.S[int64], x int64) int64",
				},
			},
		},
	)
}
