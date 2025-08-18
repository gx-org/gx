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

var declareGradPackage = testbuild.DeclarePackage{
	Path: "math",
	Src:  string(gradSrc),
	Post: func(pkg *ir.Package) {
		irFunc := pkg.FindFunc("Func").(*ir.Macro)
		irFunc.BuildSynthetic = cpevelements.MacroImpl(grad.FuncGrad)
		irSet := pkg.FindFunc("Set").(*ir.Macro)
		irSet.BuildSynthetic = cpevelements.MacroImpl(grad.SetGrad)
	},
}

func TestGradExpressions(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Func{
			Src: `
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
			Src: `
func F(x float32) [2]float32 {
	return [...]float32{x, 2.0}
}
`,
			Want: `
func gradF(x float32) [2]float32 {
	return [2]float32{1, 0}
}
`,
		},
		testgrad.Func{
			Src: `
func F(x [2]float32) [2]float32 {
	return 2+x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return [2]float32(1)
}
`,
		},
		testgrad.Func{
			Src: `
func F(x [2]float32) [2]float32 {
	return x+2
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return [2]float32(1)
}
`,
		},
		testgrad.Func{
			Src: `
func F(x [2]float32) [2]float32 {
	return 2-x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return [2]float32(-1)
}
`,
		},
		testgrad.Func{
			Src: `
func F(x [2]float32) [2]float32 {
	return x-2
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return [2]float32(1)
}
`,
		},
		testgrad.Func{
			Src: `
func F(x float32) float32 {
	return 2*x
}
`,
			Want: `
func gradF(x float32) float32 {
	return 2
}
`,
		},
		testgrad.Func{
			Src: `
func F(x [2]float32) [2]float32 {
	return 2*x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return [2]float32(2)
}
`,
		},
		testgrad.Func{
			Src: `
func F(x [2]float32) [2]float32 {
	return x*x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return x+x
}
`,
		},
		testgrad.Func{
			Src: `
func F(x [2]float32) [2]float32 {
	return x*x*x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return (x+x)*x+x*x
}
`,
		},
		testgrad.Func{
			Src: `
func F(x [2]float32) [2]float32 {
	return 1/x
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return -1/(x*x)
}
`,
		},
		testgrad.Func{
			Src: `
func F(x [2]float32) [2]float32 {
	return 1/(x*x)
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return -1*(x+x)/((x*x)*(x*x))
}
`,
		},
		testgrad.Func{
			Src: `
func F(x [2]float32) [2]float32 {
	return (2*x)/((x+1)*(x+1))
}
`,
			Want: `
func gradF(x [2]float32) [2]float32 {
	return ((2)*((x+1)*(x+1))-(2*x)*((x+1)+(x+1)))/(((x+1)*(x+1))*((x+1)*(x+1)))
}
`,
		},
	)
}

func TestGradFunc(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Func{
			Src: `
func g() float32 {
	return 2
}

func F(x float32) float32 {
	return g()
}
`,
			Want: `
func gradF(x float32) float32 {
	return 0
}
`,
		},
		testgrad.Func{
			Src: `
func g(y float32) float32 {
	return y	
}

func F(x float32) float32 {
	return g(x)
}
`,
			Want: `
func gradF(x float32) float32 {
	return __grad_Func_g_y(x)
}
`,
			Wants: map[string]string{
				"__grad_Func_g_y": `
func __grad_Func_g_y(y float32) float32 {
	return 1
}
`,
			},
		},
		testgrad.Func{
			Src: `
func g(x float32) float32 {
	return x	
}

func F(x float32) float32 {
	return g(x*x)
}
`,
			Want: `
func gradF(x float32) float32 {
	return __grad_Func_g_x(x*x)*(x+x)
}
`,
			Wants: map[string]string{
				"__grad_Func_g_x": `
func __grad_Func_g_x(x float32) float32 {
	return 1
}
`,
			},
		},
		testgrad.Func{
			GradImportName: "other",
			Src: `
func g(x float32) float32 {
	return x	
}

func F(x float32) float32 {
	return g(x*x)
}
`,
			Want: `
func gradF(x float32) float32 {
	return __other_Func_g_x(x*x)*(x+x)
}
`,
			Wants: map[string]string{
				"__other_Func_g_x": `
func __other_Func_g_x(x float32) float32 {
	return 1
}
`,
			},
		},
		testgrad.Func{
			GradOf: []string{"F", "G"},
			Src: `
func h(x float32) float32 {
	return x	
}

func F(x float32) float32 {
	return h(x*x)
}

func G(x float32) float32 {
	return h(x*x)
}
`,
			Wants: map[string]string{
				"gradF": `
func gradF(x float32) float32 {
	return __grad_Func_h_x(x*x)*(x+x)
}
`,
				"gradG": `
func gradG(x float32) float32 {
	return __grad_Func_h_x(x*x)*(x+x)
}
`,
				"__grad_Func_h_x": `
func __grad_Func_h_x(x float32) float32 {
	return 1
}
`,
			},
		},
	)
}

func TestGradStatements(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Func{
			Src: `
func F(x float32) float32 {
	a := x*x
	return a
}
`,
			Want: `
func gradF(x float32) float32 {
	__grad_a := x+x
	a := x*x
	return __grad_a
}
`,
		},
		testgrad.Func{
			Src: `
func F(x float32) float32 {
	a := 2*x
	b := 3*x
	return a*b
}
`,
			Want: `
func gradF(x float32) float32 {
	__grad_a := float32(2)
	a := 2*x
	__grad_b := float32(3)
	b := 3*x
	return __grad_a*b+a*__grad_b
}
`,
		},
		testgrad.Func{
			Src: `
func F(x float32) float32 {
	a, b := 2*x, 3*x
	return a*b
}
`,
			Want: `
func gradF(x float32) float32 {
	__grad_a, __grad_b := float32(2), float32(3)
	a, b := 2*x, 3*x
	return __grad_a*b+a*__grad_b
}
`,
		},
		testgrad.Func{
			Src: `
func F(x float32) float32 {
	x = 2
	return x
}
`,
			Want: `
func gradF(x float32) float32 {
	__grad_x := float32(0)
	x = 2
	return __grad_x
}
`,
		},
		testgrad.Func{
			Src: `
func F(x float32) (float32, float32) {
	x = x*x
	x1 := x
	x = x+x
	x2 := x
	return x1, x2
}
`,
			Want: `
func gradF(x float32) (float32, float32) {
	__grad_x := x+x
	x = x*x
	__grad_x1 := __grad_x
	x1 := x
	__grad_x = __grad_x+__grad_x
	x = x+x
	__grad_x2 := __grad_x
	x2 := x
	return __grad_x1, __grad_x2
}
`,
		},
	)
}

func TestParameters(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Func{
			Src: `
func F(x, y float32) float32 {
	return x*y
}
`,
			Want: `
func gradF(x, y float32) float32 {
	return y
}
`,
		},
	)
}

func TestGradErrors(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Func{
			Src: `
func F(y float32) float32 {
	return y
}
`,
			Err: "no parameter named x in F",
		},
		testgrad.Func{
			Src: `
func F(x float32) float32
`,
			Err: "cannot compute the gradient of function",
		},
	)
}
