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
		bck0y := -res
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
		bck0y := -res
		return res+bck0y
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
		bck0x := res*x
		bck0y := 2*res
		return bck0y
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
		bck0 := -res
		return bck0
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
		bck0x := res*x
		bck0y := x*res
		return bck0x+bck0y
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
		bck0x := res*y
		bck0y := x*res
		return bck0x
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		bck0x1 := res*y
		bck0y1 := x*res
		return bck0y1
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
		bck1x := res*fwd0
		bck1y := x*res
		return bck1x+bck1y
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
		bck0x := res/x
		bck0y := -res/(x*x)
		return bck0y
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return 1.0/x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := 1.0/x
	selfVJPFunc := func(res float32) float32 {
		bck0x := res/x
		bck0y := -res/(x*x)
		return bck0y
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
		bck0x := res/y
		bck0y := -(x*res)/(y*y)
		return bck0x
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		bck0x1 := res/y
		bck0y1 := -(x*res)/(y*y)
		return bck0y1
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	return -(x*x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x*x
	fwd1 := -fwd0
	selfVJPFunc := func(res float32) float32 {
		bck1 := -res
		bck0x := bck1*x
		bck0y := x*bck1
		return bck0x+bck0y
	}
	return fwd1, selfVJPFunc
}`,
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
		bck1 := fVJP1(res)
		bck0 := fVJP(res)
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

func F(x float32) float32 {
	return f(x)-f(x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, fVJP := grad.VJP(f)(x)
	fwd1, fVJP1 := grad.VJP(f)(x)
	fwd2 := fwd0-fwd1
	selfVJPFunc := func(res float32) float32 {
		bck2y := -res
		bck1 := fVJP1(bck2y)
		bck0 := fVJP(res)
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
	return f(x)+g(x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, fVJP := grad.VJP(f)(x)
	fwd1, gVJP := grad.VJP(g)(x)
	fwd2 := fwd0+fwd1
	selfVJPFunc := func(res float32) float32 {
		bck1 := gVJP(res)
		bck0 := fVJP(res)
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
		bck1x := res*y
		bck1y := 3*res
		bck0x := res*x
		bck0y := 2*res
		return bck0y
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		bck1x1 := res*y
		bck1y1 := 3*res
		bck0x1 := res*x
		bck0y1 := 2*res
		return bck1y1
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
		bck2x := res*fwd1
		bck2y := fwd0*res
		bck1 := gVJP(bck2y)
		bck0 := fVJP(bck2x)
		return bck0+bck1
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

func TestVJPAssignments(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	a := 2*x
	return a
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := 2*x
	a := fwd0
	selfVJPFunc := func(res float32) float32 {
		bck0x := res*x
		bck0y := 2*res
		bcka := bck0y
		return bcka
	}
	return a, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	a := 2*x
	a = 3*x
	return a
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := 2*x
	a := fwd0
	fwd1 := 3*x
	a1 := fwd1
	selfVJPFunc := func(res float32) float32 {
		bck1x := res*x
		bck1y := 3*res
		bcka := bck1y
		return bcka
	}
	return a1, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	a := 2*x
	a = 3*x
	return a*x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := 2*x
	a := fwd0
	fwd1 := 3*x
	a1 := fwd1
	fwd2 := a1*x
	selfVJPFunc := func(res float32) float32 {
		bck2x := res*x
		bck2y := a1*res
		bck1x := bck2x*x
		bck1y := 3*bck2x
		bcka := bck1y
		return bcka+bck2y
	}
	return fwd2, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) (float32, float32, float32) {
	a, b, c := x, 2*x, 2+x
	return a, b, c
}
`,
			Want: `
func vjpF(x float32) (float32, float32, float32, func(res float32, res1 float32, res2 float32) float32) {
	fwd0 := 2*x
	fwd1 := 2+x
	a, b, c := x, fwd0, fwd1
	selfVJPFunc := func(res float32, res1 float32, res2 float32) float32 {
		bcka := res
		bck0x := res1*x
		bck0y := 2*res1
		bckb := bck0y
		bckc := res2
		return bcka+(bckb+bckc)
	}
	return a, b, c, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	a := x
	a = x+a
	return a
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	a := x
	fwd0 := x+a
	a1 := fwd0
	selfVJPFunc := func(res float32) float32 {
		bcka := res
		bcka1 := (res+bcka)
		return bcka1
	}
	return a1, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	a := x
	a = x*a
	return a
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	a := x
	fwd0 := x*a
	a1 := fwd0
	selfVJPFunc := func(res float32) float32 {
		bck0x := res*a
		bck0y := x*res
		bcka := bck0y
		bcka1 := (bck0x+bcka)
		return bcka1
	}
	return a1, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	a := x
	x = 2
	return a * x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	a := x
	x1 := float32(2)
	fwd0 := a*x1
	selfVJPFunc := func(res float32) float32 {
		bck0x := res*x1
		bck0y := a*res
		bcka := bck0x
		return bcka
	}
	return fwd0, selfVJPFunc
}
`,
		},
	)
}

func TestVJPArrays(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
func F(x [2]float32) [2]float32 {
	return x
}
`,
			Want: `
func vjpF(x [2]float32) ([2]float32, func(res [2]float32) [2]float32) {
	selfVJPFunc := func(res [2]float32) [2]float32 {
		return res
	}
	return x, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x [2][3]float32) [2][3]float32 {
	return x
}
`,
			Want: `
func vjpF(x [2][3]float32) ([2][3]float32, func(res [2][3]float32) [2][3]float32) {
	selfVJPFunc := func(res [2][3]float32) [2][3]float32 {
		return res
	}
	return x, selfVJPFunc
}
`,
		},
	)
}
