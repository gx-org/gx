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
		testgrad.VJP{
			Src: `
func F(x float32) [2]float32 {
	return [2]float32{x*x, 2}
}
`,
			Want: `
func vjpF(x float32) ([2]float32, func(res [2]float32) float32) {
	fwd0 := x*x
	selfVJPFunc := func(res [2]float32) float32 {
		bck0x := res[0]*x
		bck0y := x*res[0]
		return bck0x+bck0y
	}
	return [2]float32{fwd0, 2}, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) [3][2]float32 {
	return [3][2]float32{
		{1*x,2*x},
		{3*x,4*x},
		{5*x,6*x},
	}
}
`,
			Want: `
func vjpF(x float32) ([3][2]float32, func(res [3][2]float32) float32) {
	fwd0 := 1*x
	fwd1 := 2*x
	fwd2 := 3*x
	fwd3 := 4*x
	fwd4 := 5*x
	fwd5 := 6*x
	selfVJPFunc := func(res [3][2]float32) float32 {
		bck0x := res[0][0]*x
		bck1x := res[0][1]*x
		bck1y := 2*res[0][1]
		bck2x := res[1][0]*x
		bck2y := 3*res[1][0]
		bck3x := res[1][1]*x
		bck3y := 4*res[1][1]
		bck4x := res[2][0]*x
		bck4y := 5*res[2][0]
		bck5x := res[2][1]*x
		bck5y := 6*res[2][1]
		return ((res[0][0])+bck1y)+((bck2y+bck3y)+(bck4y+bck5y))
	}
	return [3][2]float32{[2]float32{fwd0, fwd1}, [2]float32{fwd2, fwd3}, [2]float32{fwd4, fwd5}}, selfVJPFunc
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
		testgrad.VJP{
			Src: `
func F(x [_A]float32) [A]float32 {
	return x
}
`,
			Want: `
func vjpF(x [_A]float32) ([A]float32, func(res [A]float32) [A]float32) {
	selfVJPFunc := func(res [A]float32) [A]float32 {
		return res
	}
	return x, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x [___S]float32) [S___]float32 {
	return x
}
`,
			Want: `
func vjpF(x [___S]float32) ([S___]float32, func(res [S___]float32) [S___]float32) {
	selfVJPFunc := func(res [S___]float32) [S___]float32 {
		return res
	}
	return x, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
type floats interface {
	float32 | float64
}

func F[T floats](x [___S]T) [S___]T {
	return x
}
`,
			Want: `
func vjpF[T floats](x [___S]T) ([S___]T, func(res [S___]T) [S___]T) {
	selfVJPFunc := func(res [S___]T) [S___]T {
		return res
	}
	return x, selfVJPFunc
}
`,
		},
	)
}

func TestVJPDuplicatedNames(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
func F(fwd0 [___S]float32) [S___]float32 {
	return 2*fwd0
}
`,
			Want: `
func vjpF(fwd0 [___S]float32) ([S___]float32, func(res [S___]float32) [S___]float32) {
	fwd01 := 2*fwd0
	selfVJPFunc := func(res [S___]float32) [S___]float32 {
		bck0x := res*fwd0
		bck0y := 2*res
		return bck0y
	}
	return fwd01, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x [___fwd0]float32) [fwd0___]float32 {
	return 2*x
}
`,
			Want: `
func vjpF(x [___fwd0]float32) ([fwd0___]float32, func(res [fwd0___]float32) [fwd0___]float32) {
	fwd0_1 := 2*x
	selfVJPFunc := func(res [fwd0___]float32) [fwd0___]float32 {
		bck0x := res*x
		bck0y := 2*res
		return bck0y
	}
	return fwd0_1, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
type floats interface {
	float32 | float64
}

func F[fwd0 floats](x [___S]fwd0) [S___]fwd0 {
	return 2*x
}
`,
			Want: `
func vjpF[fwd0 floats](x [___S]fwd0) ([S___]fwd0, func(res [S___]fwd0) [S___]fwd0) {
	fwd0_1 := 2*x
	selfVJPFunc := func(res [S___]fwd0) [S___]fwd0 {
		bck0x := res*x
		bck0y := 2*res
		return bck0y
	}
	return fwd0_1, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func F(x float32) float32 {
	x = x*x
	x1 := x
	x = x+x
	x2 := x
	return x1+x2
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x*x
	x1 := fwd0
	x1_1 := x1
	fwd1 := x1+x1
	x2 := fwd1
	x2_1 := x2
	fwd2 := x1_1+x2_1
	selfVJPFunc := func(res float32) float32 {
		bck0x := res*x
		bck0y := x*res
		bckx := (bck0x+bck0y)
		bck0x1 := res*x
		bck0y1 := x*res
		bckx1 := (bck0x1+bck0y1)
		bckx2 := (bckx1+bckx)
		bckx2_1 := bckx2
		bck0x2 := res*x
		bck0y2 := x*res
		bckx3 := (bck0x2+bck0y2)
		bckx1_1 := bckx3
		return bckx1_1+bckx2_1
	}
	return fwd2, selfVJPFunc
}
`,
		},
	)
}
