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
	"github.com/gx-org/gx/stdlib/math/grad/testgrad"
)

func TestVJPExpressions(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return 2.0
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	selfVJPFunc := func(res float32) float32 {
		return 0
	}
	return 2.0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	selfVJPFunc := func(res float32) float32 {
		return res
	}
	return x, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return x+2
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x+2
	selfVJPFunc := func(res float32) float32 {
		return res
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return x+x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x+x
	selfVJPFunc := func(res float32) float32 {
		return res+res
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return x-2
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x-2
	selfVJPFunc := func(res float32) float32 {
		return res
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return x-x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x-x
	selfVJPFunc := func(res float32) float32 {
		return res-res
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return 2*x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := 2*x
	selfVJPFunc := func(res float32) float32 {
		return 2*res
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return -2
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	selfVJPFunc := func(res float32) float32 {
		return 0
	}
	return -2, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return -x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := -x
	selfVJPFunc := func(res float32) float32 {
		return -res
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return x*x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x*x
	selfVJPFunc := func(res float32) float32 {
		return res*x+x*res
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x, y float32) float32 {
	return x*y
}
`,
			Want: `
func vjpF(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0 := x*y
	selfVJPFuncWRTx := func(res float32) float32 {
		return res*y
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		return x*res
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return x*(2+x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := 2+x
	fwd1 := x*fwd0
	selfVJPFunc := func(res float32) float32 {
		return res*fwd0+x*(res)
	}
	return fwd1, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return 1/x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := 1/x
	selfVJPFunc := func(res float32) float32 {
		return -(1*res)/(x*x)
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x, y float32) float32 {
	return x/y
}
`,
			Want: `
func vjpF(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0 := x/y
	selfVJPFuncWRTx := func(res float32) float32 {
		return (res*y)/(y*y)
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		return -(x*res)/(y*y)
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
	)
}

func TestVJPFunctions(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
func f(x float32) float32 {
	return x
}

func F(x float32) float32 {
	return f(x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, fVJP := grad.VJP(f)(x)
	selfVJPFunc := func(res float32) float32 {
		bck0 := fVJP(res)
		return bck0
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func f(x float32) float32 {
	return x
}

func F(x float32) float32 {
	return f(x)+f(x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, fVJP := grad.VJP(f)(x)
	fwd1, fVJP1 := grad.VJP(f)(x)
	fwd2 := fwd0+fwd1
	selfVJPFunc := func(res float32) float32 {
		bck0 := fVJP(res)
		bck1 := fVJP1(res)
		return bck0+bck1
	}
	return fwd2, selfVJPFunc
}
`,
			WantExprs: map[string]string{
				"grad.VJP(f)": `
func(x float32) (float32, func(res float32) float32) {
	selfVJPFunc := func(res float32) float32 {
		return res
	}
	return x, selfVJPFunc
}
`,
			},
		},
		testgrad.VJP{
			Src: `
func f(x float32) float32 {
	return x
}

func g(x float32) float32 {
	return x
}

func F(x float32) float32 {
	return f(x)+g(x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, fVJP := grad.VJP(f)(x)
	fwd1, gVJP := grad.VJP(g)(x)
	fwd2 := fwd0+fwd1
	selfVJPFunc := func(res float32) float32 {
		bck0 := fVJP(res)
		bck1 := gVJP(res)
		return bck0+bck1
	}
	return fwd2, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func f(x float32) float32 {
	return x
}

func g(x float32) float32 {
	return x
}

func F(x float32) float32 {
	return f(g(x))
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, gVJP := grad.VJP(g)(x)
	fwd1, fVJP := grad.VJP(f)(fwd0)
	selfVJPFunc := func(res float32) float32 {
		bck1 := fVJP(res)
		bck0 := gVJP(bck1)
		return bck0
	}
	return fwd1, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func f0(x float32) float32 {
	return x
}

func f1(x float32) float32 {
	return x
}

func f2(x float32) float32 {
	return x
}

func f3(x float32) float32 {
	return x
}

func f4(x float32) float32 {
	return x
}

func F(x float32) float32 {
	return f4(f3(f2(f1(f0(x)))))
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, f0VJP := grad.VJP(f0)(x)
	fwd1, f1VJP := grad.VJP(f1)(fwd0)
	fwd2, f2VJP := grad.VJP(f2)(fwd1)
	fwd3, f3VJP := grad.VJP(f3)(fwd2)
	fwd4, f4VJP := grad.VJP(f4)(fwd3)
	selfVJPFunc := func(res float32) float32 {
		bck4 := f4VJP(res)
		bck3 := f3VJP(bck4)
		bck2 := f2VJP(bck3)
		bck1 := f1VJP(bck2)
		bck0 := f0VJP(bck1)
		return bck0
	}
	return fwd4, selfVJPFunc
}
`,
		},
	)
}

func TestVJPFunctionsFanInFanOut(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
func F(x, y float32) float32 {
	return x+y
}
`,
			Want: `
func vjpF(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0 := x+y
	selfVJPFuncWRTx := func(res float32) float32 {
		return res
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		return res
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x, y float32) float32 {
	return 2*x+3*y
}
`,
			Want: `
func vjpF(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0 := 2*x
	fwd1 := 3*y
	fwd2 := fwd0+fwd1
	selfVJPFuncWRTx := func(res float32) float32 {
		return 2*res
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		return 3*res
	}
	return fwd2, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.VJP{
			Src: `
func h(a, b float32) float32 {
	return a+b
}

func F(x float32) float32 {
	return h(x, x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, hVJPa, hVJPb := grad.VJP(h)(x, x)
	selfVJPFunc := func(res float32) float32 {
		bck0a := hVJPa(res)
		bck0b := hVJPb(res)
		return bck0a+bck0b
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func f(x float32) float32 {
	return x
}

func g(x float32) float32 {
	return x
}

func h(a, b float32) float32 {
	return a+b
}

func F(x float32) float32 {
	return h(f(x), g(x))
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, fVJP := grad.VJP(f)(x)
	fwd1, gVJP := grad.VJP(g)(x)
	fwd2, hVJPa, hVJPb := grad.VJP(h)(fwd0, fwd1)
	selfVJPFunc := func(res float32) float32 {
		bck2a := hVJPa(res)
		bck2b := hVJPb(res)
		bck1 := gVJP(bck2b)
		bck0 := fVJP(bck2a)
		return bck0+bck1
	}
	return fwd2, selfVJPFunc
}
`,
		},
	)
}

func TestVJPFunctionsInExpressions(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
func f(x float32) float32 {
	return x
}

func g(x float32) float32 {
	return x
}

func F(x float32) float32 {
	return f(x)*g(x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, fVJP := grad.VJP(f)(x)
	fwd1, gVJP := grad.VJP(g)(x)
	fwd2 := fwd0*fwd1
	selfVJPFunc := func(res float32) float32 {
		bck0 := fVJP(res)
		bck1 := gVJP(res)
		return bck0*fwd1+fwd0*bck1
	}
	return fwd2, selfVJPFunc
}
`,
		},
	)
}

func TestVJPPackageBuiltins(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		declareMathPackage,
		testgrad.VJP{
			Imports: []string{"math"},
			Src: `
func F(x float32) float32 {
	return math.Sin(x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, SinVJP := grad.VJP(math.Sin)(x)
	selfVJPFunc := func(res float32) float32 {
		bck0 := SinVJP(res)
		return bck0
	}
	return fwd0, selfVJPFunc
}
`,
		},
	)
}

func TestVJPGenerics(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		declareMathPackage,
		testgrad.VJP{
			Imports: []string{"math"},
			Src: `
type floats interface {
	float32 | float64
}

func F[T floats](x T) T {
	return x
}
`,
			Want: `
func vjpF[T floats](x T) (T, func(res T) T) {
	selfVJPFunc := func(res T) T {
		return res
	}
	return x, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Imports: []string{"math"},
			Src: `
type floats interface {
	float32 | float64
}

func g[T floats](x T) T {
	return x
}

func F[T floats](x T) T {
	return g(x)
}
`,
			Want: `
func vjpF[T floats](x T) (T, func(res T) T) {
	fwd0, gVJP := grad.VJP(g)[T](x)
	selfVJPFunc := func(res T) T {
		bck0 := gVJP(res)
		return bck0
	}
	return fwd0, selfVJPFunc
}
`,
		},
	)
}
