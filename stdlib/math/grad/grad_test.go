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

package grad_test

import (
	"testing"

	_ "embed"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/stdlib/math/grad"
	"github.com/gx-org/gx/stdlib/math/grad/testgrad"
)

//go:embed grad.gx
var gradSrc []byte

func TestGradFunc(t *testing.T) {
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: string(gradSrc),
			Post: func(pkg *ir.Package) {
				id := pkg.FindFunc("Func").(*ir.Macro)
				id.BuildSynthetic = cpevelements.MacroImpl(grad.FuncGrad)
			},
		},
		testgrad.Func{
			GradOf: `
func F(x float32) float32 {
	return 2.0
}
`,
			Want: `
func gradF(x float32) float32 {
	return 0
}
`,
		},
		testgrad.Func{
			GradOf: `
func F(x float32) [2]float32 {
	return [...]float32{x, 2.0}
}
`,
			Want: `
func gradF(x float32) [2]float32 {
	return [2]float32{(float32)(1), 0}
}
`,
		},
		testgrad.Func{
			GradOf: `
func F(x [2]float32) [2]float32 {
	return 2+x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return 0+([2]float32)(1)
}
`,
		},
		testgrad.Func{
			GradOf: `
func F(x [2]float32) [2]float32 {
	return 2-x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return 0-([2]float32)(1)
}
`,
		},
		testgrad.Func{
			GradOf: `
func F(x [2]float32) [2]float32 {
	return 2*x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return 0*x+2*([2]float32)(1)
}
`,
		},
		testgrad.Func{
			GradOf: `
func F(x [2]float32) [2]float32 {
	return x*x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return ([2]float32)(1)*x+x*([2]float32)(1)
}
`,
		},
		testgrad.Func{
			GradOf: `
func F(x [2]float32) [2]float32 {
	return x*x*x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return (([2]float32)(1)*x+x*([2]float32)(1))*x+(x*x)*([2]float32)(1)
}
`,
		},
	)
}
