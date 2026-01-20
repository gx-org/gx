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

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/math/grad"
	"github.com/gx-org/gx/stdlib/math/grad/testgrad"

	_ "embed"
)

//go:embed grad.gx
var gradSrc []byte

var (
	declareGradPackage = testbuild.DeclarePackage{
		Path: "math",
		Src:  string(gradSrc),
		Post: func(pkg *ir.Package) {
			irVJP := pkg.FindFunc("Reverse").(*ir.Macro)
			irVJP.BuildSynthetic = ir.MacroImpl(grad.Reverse)
			irFunc := pkg.FindFunc("Func").(*ir.Macro)
			irFunc.BuildSynthetic = ir.MacroImpl(grad.FuncGrad)
			irSet := pkg.FindFunc("Set").(*ir.Annotator)
			irSet.Annotate = ir.AnnotatorImpl(grad.SetGrad)
			irSetFor := pkg.FindFunc("SetFor").(*ir.Annotator)
			irSetFor.Annotate = ir.AnnotatorImpl(grad.SetGradFor)
		},
	}

	declareMathPackage = testbuild.DeclarePackage{
		Src: `
package math

import "math/grad"

func Cos([___M]float32) [M___]float32

//gx:@grad.Set(Cos)
func Sin([___M]float32) [M___]float32
`,
	}
)

func TestGrad(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Func{
			Src: `
func F(x float32) float32 {
	return 2.0
}
`,
			Want: `
func gradF(x float32) (float32, float32) {
	y, xVJP := grad.Reverse(F)(x)
	res := float32(1)
	return y, xVJP(res)
}
`,
		},
		testgrad.Func{
			Src: `
func F(x float32) [2]float32 {
	return [2]float32{x, 2*x}
}
`,
			Want: `
func gradF(x float32) ([2]float32, float32) {
	y, xVJP := grad.Reverse(F)(x)
	res := [2]float32(1)
	return y, xVJP(res)
}
`,
		},
	)
}
