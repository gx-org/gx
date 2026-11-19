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
		testgrad.Reverse{
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
		testgrad.Reverse{
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
		testgrad.Reverse{
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
		testgrad.Reverse{
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
		testgrad.Reverse{
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
		testgrad.Reverse{
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
		testgrad.Reverse{
			Src: `
func F(x float32) float32 {
	return 2*x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := 2*x
	selfVJPFunc := func(res float32) float32 {
		bck0y := 2*res
		return bck0y
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
		testgrad.Reverse{
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
		testgrad.Reverse{
			Src: `
func F(x float32) float32 {
	return x*x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x*x
	selfVJPFunc := func(res float32) float32 {
		bck0y := x*res
		bck0x := res*x
		return bck0x+bck0y
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
		return bck0x
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		bck0y := x*res
		return bck0y
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.Reverse{
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
		bck1y := x*res
		bck1x := res*fwd0
		return bck1x+bck1y
	}
	return fwd1, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
			Src: `
func F(x float32) float32 {
	return 1/x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := 1/x
	selfVJPFunc := func(res float32) float32 {
		bck0y := -res/(x*x)
		return bck0y
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
			Src: `
func F(x float32) float32 {
	return 1.0/x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := 1.0/x
	selfVJPFunc := func(res float32) float32 {
		bck0y := -res/(x*x)
		return bck0y
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
		return bck0x
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		bck0y := -(x*res)/(y*y)
		return bck0y
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.Reverse{
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
		bck0y := x*bck1
		bck0x := bck1*x
		return bck0x+bck0y
	}
	return fwd1, selfVJPFunc
}`,
		},
		testgrad.Reverse{
			Src: `
func F(x float32) [2]float32 {
	return [2]float32{x*x, 2}
}
`,
			Want: `
func vjpF(x float32) ([2]float32, func(res [2]float32) float32) {
	fwd0 := x*x
	selfVJPFunc := func(res [2]float32) float32 {
		bck0y := x*res[0]
		bck0x := res[0]*x
		return bck0x+bck0y
	}
	return [2]float32{fwd0, 2}, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
		bck1y := 2*res[0][1]
		bck2y := 3*res[1][0]
		bck3y := 4*res[1][1]
		bck4y := 5*res[2][0]
		bck5y := 6*res[2][1]
		return ((res[0][0])+bck1y)+((bck2y+bck3y)+(bck4y+bck5y))
	}
	return [3][2]float32{[2]float32{fwd0, fwd1}, [2]float32{fwd2, fwd3}, [2]float32{fwd4, fwd5}}, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
			Src: `
func F(x float32) float64 {
	return float64(x)
}
`,
			Want: `
func vjpF(x float32) (float64, func(res float64) float32) {
	fwd0 := float64(x)
	selfVJPFunc := func(res float64) float32 {
		bck0 := float32(res)
		return bck0
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
			Src: `
func F(x, y float32) float64 {
	return float64(x) * float64(y)
}
`,
			Want: `
func vjpF(x, y float32) (float64, func(res float64) float32, func(res float64) float32) {
	fwd0 := float64(x)
	fwd1 := float64(y)
	fwd2 := fwd0*fwd1
	selfVJPFuncWRTx := func(res float64) float32 {
		bck2x := res*fwd1
		bck0 := float32(bck2x)
		return bck0
	}
	selfVJPFuncWRTy := func(res float64) float32 {
		bck2y := fwd0*res
		bck1 := float32(bck2y)
		return bck1
	}
	return fwd2, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.Reverse{
			Src: `
func F(x float32) float64 {
	return 0 * float64(x)
}
`,
			Want: `
func vjpF(x float32) (float64, func(res float64) float32) {
	fwd0 := float64(x)
	fwd1 := 0*fwd0
	selfVJPFunc := func(res float64) float32 {
		return 0
	}
	return fwd1, selfVJPFunc
}
`,
		},
	)
}

func TestVJPFunctions(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Reverse{
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
	fwd0, fVJP := grad.Reverse(f)(x)
	selfVJPFunc := func(res float32) float32 {
		bck0 := fVJP(res)
		return bck0
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
	fwd0, fVJP := grad.Reverse(f)(x)
	fwd1, fVJP1 := grad.Reverse(f)(x)
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
				"grad.Reverse(f)": `
func(x float32) (float32, func(res float32) float32) {
	selfVJPFunc := func(res float32) float32 {
		return res
	}
	return x, selfVJPFunc
}
`,
			},
		},
		testgrad.Reverse{
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
	fwd0, fVJP := grad.Reverse(f)(x)
	fwd1, fVJP1 := grad.Reverse(f)(x)
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
		testgrad.Reverse{
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
	fwd0, fVJP := grad.Reverse(f)(x)
	fwd1, gVJP := grad.Reverse(g)(x)
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
		testgrad.Reverse{
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
	fwd0, gVJP := grad.Reverse(g)(x)
	fwd1, fVJP := grad.Reverse(f)(fwd0)
	selfVJPFunc := func(res float32) float32 {
		bck1 := fVJP(res)
		bck0 := gVJP(bck1)
		return bck0
	}
	return fwd1, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
	fwd0, f0VJP := grad.Reverse(f0)(x)
	fwd1, f1VJP := grad.Reverse(f1)(fwd0)
	fwd2, f2VJP := grad.Reverse(f2)(fwd1)
	fwd3, f3VJP := grad.Reverse(f3)(fwd2)
	fwd4, f4VJP := grad.Reverse(f4)(fwd3)
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
		testgrad.Reverse{
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
		testgrad.Reverse{
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
		bck0y := 2*res
		return bck0y
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		bck1y := 3*res
		return bck1y
	}
	return fwd2, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.Reverse{
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
	fwd0, hVJPa, hVJPb := grad.Reverse(h)(x, x)
	selfVJPFunc := func(res float32) float32 {
		bck0a := hVJPa(res)
		bck0b := hVJPb(res)
		return bck0a+bck0b
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
	fwd0, fVJP := grad.Reverse(f)(x)
	fwd1, gVJP := grad.Reverse(g)(x)
	fwd2, hVJPa, hVJPb := grad.Reverse(h)(fwd0, fwd1)
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
		testgrad.Reverse{
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
	fwd0, fVJP := grad.Reverse(f)(x)
	fwd1, gVJP := grad.Reverse(g)(x)
	fwd2 := fwd0*fwd1
	selfVJPFunc := func(res float32) float32 {
		bck2y := fwd0*res
		bck1 := gVJP(bck2y)
		bck2x := res*fwd1
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
		testgrad.Reverse{
			Imports: []string{"math"},
			Src: `
func F(x float32) float32 {
	return math.Sin(x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, SinVJP := grad.Reverse(math.Sin)(x)
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
		testgrad.Reverse{
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
		testgrad.Reverse{
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
	fwd0, gVJP := grad.Reverse(g)(x)
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
		testgrad.Reverse{
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
		bck0y := 2*res
		bcka := bck0y
		return bcka
	}
	return a, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
		bck1y := 3*res
		bcka := bck1y
		return bcka
	}
	return a1, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
		bck2y := a1*res
		bck2x := res*x
		bck1y := 3*bck2x
		bcka := bck1y
		return bcka+bck2y
	}
	return fwd2, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
		bck0y := 2*res1
		bckb := bck0y
		bckc := res2
		return bcka+(bckb+bckc)
	}
	return a, b, c, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
		testgrad.Reverse{
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
		bck0y := x*res
		bcka := bck0y
		bck0x := res*a
		bcka1 := (bck0x+bcka)
		return bcka1
	}
	return a1, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
		testgrad.Reverse{
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
		testgrad.Reverse{
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
		testgrad.Reverse{
			Src: `
func F[A int](x [A]float32) [A]float32 {
	return x
}
`,
			Want: `
func vjpF[A int](x [A]float32) ([A]float32, func(res [A]float32) [A]float32) {
	selfVJPFunc := func(res [A]float32) [A]float32 {
		return res
	}
	return x, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
			Src: `
func F[S []int](x [S...]float32) [S...]float32 {
	return x
}
`,
			Want: `
func vjpF[S []int](x [S...]float32) ([S...]float32, func(res [S...]float32) [S...]float32) {
	selfVJPFunc := func(res [S...]float32) [S...]float32 {
		return res
	}
	return x, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
			Src: `
type floats interface {
	float32 | float64
}

func F[T floats, S []int](x [S...]T) [S...]T {
	return x
}
`,
			Want: `
func vjpF[T floats, S []int](x [S...]T) ([S...]T, func(res [S...]T) [S...]T) {
	selfVJPFunc := func(res [S...]T) [S...]T {
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
		testgrad.Reverse{
			Src: `
func F[S []int](fwd0 [S...]float32) [S...]float32 {
	return 2*fwd0
}
`,
			Want: `
func vjpF[S []int](fwd0 [S...]float32) ([S...]float32, func(res [S...]float32) [S...]float32) {
	fwd01 := 2*fwd0
	selfVJPFunc := func(res [S...]float32) [S...]float32 {
		bck0y := 2*res
		return bck0y
	}
	return fwd01, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
			Src: `
func F[fwd0 []int](x [fwd0...]float32) [fwd0...]float32 {
	return 2*x
}
`,
			Want: `
func vjpF[fwd0 []int](x [fwd0...]float32) ([fwd0...]float32, func(res [fwd0...]float32) [fwd0...]float32) {
	fwd0_1 := 2*x
	selfVJPFunc := func(res [fwd0...]float32) [fwd0...]float32 {
		bck0y := 2*res
		return bck0y
	}
	return fwd0_1, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
			Src: `
type floats interface {
	float32 | float64
}

func F[fwd0 floats, S []int](x [S...]fwd0) [S...]fwd0 {
	return 2*x
}
`,
			Want: `
func vjpF[fwd0 floats, S []int](x [S...]fwd0) ([S...]fwd0, func(res [S...]fwd0) [S...]fwd0) {
	fwd0_1 := 2*x
	selfVJPFunc := func(res [S...]fwd0) [S...]fwd0 {
		bck0y := 2*res
		return bck0y
	}
	return fwd0_1, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
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
		bck0y := x*res
		bck0x := res*x
		bckx := (bck0x+bck0y)
		bck0y1 := x*res
		bck0x1 := res*x
		bckx1 := (bck0x1+bck0y1)
		bckx2 := (bckx1+bckx)
		bckx2_1 := bckx2
		bck0y2 := x*res
		bck0x2 := res*x
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

func TestVJPDependsOn(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Reverse{
			Src: `
func f(x float32) float32 {
	return x
}

func F(x, y float32) float32 {
	return f(x)
}
`,
			Want: `
func vjpF(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0, fVJP := grad.Reverse(f)(x)
	selfVJPFuncWRTx := func(res float32) float32 {
		bck0 := fVJP(res)
		return bck0
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		return 0
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.Reverse{
			Src: `
func f(x float32) float32 {
	return x
}

func F(x, y float32) float32 {
	return f(-x)
}
`,
			Want: `
func vjpF(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0 := -x
	fwd1, fVJP := grad.Reverse(f)(fwd0)
	selfVJPFuncWRTx := func(res float32) float32 {
		bck1 := fVJP(res)
		bck0 := -bck1
		return bck0
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		return 0
	}
	return fwd1, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.Reverse{
			Src: `
func f(x float32) float32 {
	return x
}

func F(x, y float32) float32 {
	return f(x+y)
}
`,
			Want: `
func vjpF(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0 := x+y
	fwd1, fVJP := grad.Reverse(f)(fwd0)
	selfVJPFuncWRTx := func(res float32) float32 {
		bck1 := fVJP(res)
		return bck1
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		bck11 := fVJP(res)
		return bck11
	}
	return fwd1, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.Reverse{
			Src: `
func F(x, y float32) float64 {
	return float64(x)
}
`,
			Want: `
func vjpF(x, y float32) (float64, func(res float64) float32, func(res float64) float32) {
	fwd0 := float64(x)
	selfVJPFuncWRTx := func(res float64) float32 {
		bck0 := float32(res)
		return bck0
	}
	selfVJPFuncWRTy := func(res float64) float32 {
		return 0
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
	)
}

func TestVJPSelectors(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Reverse{
			Src: `
type S struct {
	a float32
}

func F(s S, x float32) float32 {
	return s.a * x
}
`,
			Want: `
func vjpF(s S, x float32) (float32, func(res float32) S, func(res float32) float32) {
	fwd0 := s.a*x
	selfVJPFuncWRTs := func(res float32) S {
		bck0x := res*x
		return S{
			a: bck0x,
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck0y := s.a*res
		return bck0y
	}
	return fwd0, selfVJPFuncWRTs, selfVJPFuncWRTx
}
`,
		},
		testgrad.Reverse{
			Src: `
type S struct {
	w [2]float32
	b float32
}

func F(s S) float32 {
	return s.b
}
`,
			Want: `
func vjpF(s S) (float32, func(res float32) S) {
	selfVJPFunc := func(res float32) S {
		return S{
			w: [2]float32(0),
			b: res,
		}
	}
	return s.b, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
			Src: `
type S struct {
	a float32
}

func f(s S, x float32) float32 {
	return s.a * x
}

func F(s S, x float32) float32 {
	return f(s, x)
}
`,
			Want: `
func vjpF(s S, x float32) (float32, func(res float32) S, func(res float32) float32) {
	fwd0, fVJPs, fVJPx := grad.Reverse(f)(s, x)
	selfVJPFuncWRTs := func(res float32) S {
		bck0s := fVJPs(res)
		return S{
			a: bck0s.a,
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck0x := fVJPx(res)
		return bck0x
	}
	return fwd0, selfVJPFuncWRTs, selfVJPFuncWRTx
}
`,
		},
		testgrad.Reverse{
			Src: `
type Layer struct {
	w float32
}

func f(l Layer, x float32) float32 {
	return l.w*x
}

type S struct {
	l1 Layer
	l2 Layer
}

func F(s S, x float32) float32 {
	return f(s.l1, x) + f(s.l2, x)
}
`,
			Want: `
func vjpF(s S, x float32) (float32, func(res float32) S, func(res float32) float32) {
	fwd0, fVJPl, fVJPx := grad.Reverse(f)(s.l1, x)
	fwd1, fVJPl1, fVJPx1 := grad.Reverse(f)(s.l2, x)
	fwd2 := fwd0+fwd1
	selfVJPFuncWRTs := func(res float32) S {
		bck0l := fVJPl(res)
		bck1l := fVJPl1(res)
		return S{
			l1: Layer{
				w: bck0l.w,
			},
			l2: Layer{
				w: bck1l.w,
			},
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck1x := fVJPx1(res)
		bck0x := fVJPx(res)
		return bck0x+bck1x
	}
	return fwd2, selfVJPFuncWRTs, selfVJPFuncWRTx
}
`,
		},
		testgrad.Reverse{
			Src: `
type Layer struct {
	w float32
}

func f(l Layer, x float32) float32 {
	return l.w*x
}

type S struct {
	l Layer
}

func F(s S, x float32) float32 {
	return f(s.l, x) + s.l.w*x
}
`,
			Want: `
func vjpF(s S, x float32) (float32, func(res float32) S, func(res float32) float32) {
	fwd0, fVJPl, fVJPx := grad.Reverse(f)(s.l, x)
	fwd1 := s.l.w*x
	fwd2 := fwd0+fwd1
	selfVJPFuncWRTs := func(res float32) S {
		bck0l := fVJPl(res)
		bck1x := res*x
		return S{
			l: Layer{
				w: bck0l.w+bck1x,
			},
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck1y := s.l.w*res
		bck0x := fVJPx(res)
		return bck0x+bck1y
	}
	return fwd2, selfVJPFuncWRTs, selfVJPFuncWRTx
}
`,
		},
		testgrad.Reverse{
			Src: `
type Layer struct {
	w float32
}

type S struct {
	l1 Layer
	l2 Layer
}

func F(s S, x float32) float32 {
	return s.l1.w * x + s.l2.w * x
}
`,
			Want: `
func vjpF(s S, x float32) (float32, func(res float32) S, func(res float32) float32) {
	fwd0 := s.l1.w*x
	fwd1 := s.l2.w*x
	fwd2 := fwd0+fwd1
	selfVJPFuncWRTs := func(res float32) S {
		bck0x := res*x
		bck1x := res*x
		return S{
			l1: Layer{
				w: bck0x,
			},
			l2: Layer{
				w: bck1x,
			},
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck1y := s.l2.w*res
		bck0y := s.l1.w*res
		return bck0y+bck1y
	}
	return fwd2, selfVJPFuncWRTs, selfVJPFuncWRTx
}
`,
		},
		testgrad.Reverse{
			Src: `
type Layer struct {
	w float32
}

type S struct {
	l1 Layer
	l2 Layer
}

func F(s S, x float32) float32 {
	w1 := s.l1.w
	w2 := s.l2.w
	return w1 * x + w2 * x
}
`,
			Want: `
func vjpF(s S, x float32) (float32, func(res float32) S, func(res float32) float32) {
	w1 := s.l1.w
	w2 := s.l2.w
	fwd0 := w1*x
	fwd1 := w2*x
	fwd2 := fwd0+fwd1
	selfVJPFuncWRTs := func(res float32) S {
		bck0x := res*x
		bckw1 := bck0x
		bck1x := res*x
		bckw2 := bck1x
		return S{
			l1: Layer{
				w: bckw1,
			},
			l2: Layer{
				w: bckw2,
			},
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck1y := w2*res
		bck0y := w1*res
		return bck0y+bck1y
	}
	return fwd2, selfVJPFuncWRTs, selfVJPFuncWRTx
}
`,
		},
	)
}

func TestVJPReceivers(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Reverse{
			GradOf: "S.F",
			Src: `
type S struct {
	a float32
}

func (s S) F(x float32) float32 {
	return s.a * x
}
`,
			Want: `
func (s S) vjpF(x float32) (float32, func(res float32) S, func(res float32) float32) {
	fwd0 := s.a*x
	selfVJPFuncWRTs := func(res float32) S {
		bck0x := res*x
		return S{
			a: bck0x,
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck0y := s.a*res
		return bck0y
	}
	return fwd0, selfVJPFuncWRTs, selfVJPFuncWRTx
}
`,
		},
		testgrad.Reverse{
			Src: `
type S struct {
	a float32
}

func (s S) f(x float32) float32 {
	return s.a * x
}

func F(a, x float32) float32 {
	return (S{a: 2 * a}.f)(x)
}
`,
			Want: `
func vjpF(a, x float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0 := 2*a
	fwd1, fVJPs, fVJPx := grad.Reverse(S{
		a: fwd0,
	}.f)(x)
	selfVJPFuncWRTa := func(res float32) float32 {
		bck1s := fVJPs(res)
		bck0y := 2*bck1s.a
		return bck0y
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck1x := fVJPx(res)
		return bck1x
	}
	return fwd1, selfVJPFuncWRTa, selfVJPFuncWRTx
}
`,
		},
	)
}

func TestVJPStructAccumulation(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Reverse{
			Src: `
type S struct {
	w float32
	b float32
}

func (s S) forward(x float32) float32 {
	return s.w*x + s.b
}

func F(s S, x, y float32) float32 {
	err := s.forward(x) - y
	return err * err
}
`,
			Want: `
func vjpF(s S, x, y float32) (float32, func(res float32) S, func(res float32) float32, func(res float32) float32) {
	fwd0, forwardVJPs, forwardVJPx := grad.Reverse(s.forward)(x)
	fwd1 := fwd0-y
	err := fwd1
	fwd2 := err*err
	selfVJPFuncWRTs := func(res float32) S {
		bck2y := err*res
		bck0s := forwardVJPs(bck2y)
		bckerr := bck0s
		bck2x := res*err
		bck0s1 := forwardVJPs(bck2x)
		bckerr1 := bck0s1
		return S{
			w: bckerr1.w+bckerr.w,
			b: bckerr1.b+bckerr.b,
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck2y1 := err*res
		bck0x := forwardVJPx(bck2y1)
		bckerr2 := bck0x
		bck2x1 := res*err
		bck0x1 := forwardVJPx(bck2x1)
		bckerr3 := bck0x1
		return bckerr3+bckerr2
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		bck2y2 := err*res
		bck1y := -bck2y2
		bckerr4 := bck1y
		bck2x2 := res*err
		bck1y1 := -bck2x2
		bckerr5 := bck1y1
		return bckerr5+bckerr4
	}
	return fwd2, selfVJPFuncWRTs, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.Reverse{
			Src: `
type S struct {
	w float32
	b float32
}

func (s S) forward(x float32) float32 {
	return s.w*x + s.b
}

func F(s S, x float32) float32 {
	err := s.forward(x)
	z := err * err
	return z
}
`,
			Want: `
func vjpF(s S, x float32) (float32, func(res float32) S, func(res float32) float32) {
	fwd0, forwardVJPs, forwardVJPx := grad.Reverse(s.forward)(x)
	err := fwd0
	fwd1 := err*err
	z := fwd1
	selfVJPFuncWRTs := func(res float32) S {
		bck1y := err*res
		bck0s := forwardVJPs(bck1y)
		bckerr := bck0s
		bck1x := res*err
		bck0s1 := forwardVJPs(bck1x)
		bckerr1 := bck0s1
		bckz := S{
			w: bckerr1.w+bckerr.w,
			b: bckerr1.b+bckerr.b,
		}
		return S{
			w: bckz.w,
			b: bckz.b,
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck1y1 := err*res
		bck0x := forwardVJPx(bck1y1)
		bckerr2 := bck0x
		bck1x1 := res*err
		bck0x1 := forwardVJPx(bck1x1)
		bckerr3 := bck0x1
		bckz1 := (bckerr3+bckerr2)
		return bckz1
	}
	return z, selfVJPFuncWRTs, selfVJPFuncWRTx
}
`,
		},
		testgrad.Reverse{
			Src: `
type S struct {
	w float32
	b float32
}

func h(a, b S) float32 {
	return a.w + b.b
}

func F(s S) float32 {
	return h(s, s)
}
`,
			Want: `
func vjpF(s S) (float32, func(res float32) S) {
	fwd0, hVJPa, hVJPb := grad.Reverse(h)(s, s)
	selfVJPFunc := func(res float32) S {
		bck0a := hVJPa(res)
		bck0b := hVJPb(res)
		return S{
			w: bck0a.w+bck0b.w,
			b: bck0a.b+bck0b.b,
		}
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.Reverse{
			Src: `
type S struct {
	a float32
}

type Pair struct {
	x float32
	y float32
}

func (s S) f(x float32) float32 {
	return s.a * x
}

func F(s S, x float32) ([2]float32, Pair) {
	return [2]float32{s.f(x), s.f(x)}, Pair{x: s.f(x), y: s.f(x)}
}
`,
			Want: `
func vjpF(s S, x float32) ([2]float32, Pair, func(res [2]float32, res1 Pair) S, func(res [2]float32, res1 Pair) float32) {
	fwd0, fVJPs, fVJPx := grad.Reverse(s.f)(x)
	fwd1, fVJPs1, fVJPx1 := grad.Reverse(s.f)(x)
	fwd2, fVJPs2, fVJPx2 := grad.Reverse(s.f)(x)
	fwd3, fVJPs3, fVJPx3 := grad.Reverse(s.f)(x)
	selfVJPFuncWRTs := func(res [2]float32, res1 Pair) S {
		bck0s := fVJPs(res[0])
		bck1s := fVJPs1(res[1])
		bck2s := fVJPs2(res1.x)
		bck3s := fVJPs3(res1.y)
		return S{
			a: bck0s.a+(bck1s.a+(bck2s.a+bck3s.a)),
		}
	}
	selfVJPFuncWRTx := func(res [2]float32, res1 Pair) float32 {
		bck0x := fVJPx(res[0])
		bck1x := fVJPx1(res[1])
		bck2x := fVJPx2(res1.x)
		bck3x := fVJPx3(res1.y)
		return (bck0x+bck1x)+(bck2x+bck3x)
	}
	return [2]float32{fwd0, fwd1}, Pair{
		x: fwd2,
		y: fwd3,
	}, selfVJPFuncWRTs, selfVJPFuncWRTx
}
`,
		},
		testgrad.Reverse{
			Src: `
type Inner struct {
	w float32
	b float32
}

type Outer struct {
	in Inner
	c  float32
}

func fOuter(o Outer, x float32) float32 {
	return o.in.w*x + o.in.b + o.c
}

func fInner(in Inner, x float32) float32 {
	return in.w*x + in.b
}

func F(o Outer, x float32) float32 {
	return fOuter(o, x) + fInner(o.in, x) + o.in.w*x
}
`,
			Want: `
func vjpF(o Outer, x float32) (float32, func(res float32) Outer, func(res float32) float32) {
	fwd0, fOuterVJPo, fOuterVJPx := grad.Reverse(fOuter)(o, x)
	fwd1, fInnerVJPin, fInnerVJPx := grad.Reverse(fInner)(o.in, x)
	fwd2 := fwd0+fwd1
	fwd3 := o.in.w*x
	fwd4 := fwd2+fwd3
	selfVJPFuncWRTo := func(res float32) Outer {
		bck0o := fOuterVJPo(res)
		bck1in := fInnerVJPin(res)
		bck3x := res*x
		return Outer{
			in: Inner{
				w: (bck0o.in.w+bck1in.w)+bck3x,
				b: bck0o.in.b+bck1in.b,
			},
			c: bck0o.c,
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck3y := o.in.w*res
		bck1x := fInnerVJPx(res)
		bck0x := fOuterVJPx(res)
		return (bck0x+bck1x)+bck3y
	}
	return fwd4, selfVJPFuncWRTo, selfVJPFuncWRTx
}
`,
		},
		testgrad.Reverse{
			Src: `
type Inner struct {
	w float32
	b float32
}

type Outer struct {
	in Inner
	c  float32
}

func fOuter(o Outer, x float32) float32 {
	return o.in.w*x + o.in.b + o.c
}

func fInner(in Inner, x float32) float32 {
	return in.w*x + in.b
}

func F(o Outer, x float32) float32 {
	err := fOuter(o, x) + fInner(o.in, x)
	z := err * err
	return z
}
`,
			Want: `
func vjpF(o Outer, x float32) (float32, func(res float32) Outer, func(res float32) float32) {
	fwd0, fOuterVJPo, fOuterVJPx := grad.Reverse(fOuter)(o, x)
	fwd1, fInnerVJPin, fInnerVJPx := grad.Reverse(fInner)(o.in, x)
	fwd2 := fwd0+fwd1
	err := fwd2
	fwd3 := err*err
	z := fwd3
	selfVJPFuncWRTo := func(res float32) Outer {
		bck3y := err*res
		bck0o := fOuterVJPo(bck3y)
		bckerr := bck0o
		bck3x := res*err
		bck0o1 := fOuterVJPo(bck3x)
		bckerr1 := bck0o1
		bckz := Outer{
			in: Inner{
				w: bckerr1.in.w+bckerr.in.w,
				b: bckerr1.in.b+bckerr.in.b,
			},
			c: bckerr1.c+bckerr.c,
		}
		bck3y1 := err*res
		bck1in := fInnerVJPin(bck3y1)
		bckerr2 := bck1in
		bck3x1 := res*err
		bck1in1 := fInnerVJPin(bck3x1)
		bckerr3 := bck1in1
		bckz1 := Inner{
			w: bckerr3.w+bckerr2.w,
			b: bckerr3.b+bckerr2.b,
		}
		return Outer{
			in: Inner{
				w: bckz.in.w+bckz1.w,
				b: bckz.in.b+bckz1.b,
			},
			c: bckz.c,
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck3y2 := err*res
		bck1x := fInnerVJPx(bck3y2)
		bck0x := fOuterVJPx(bck3y2)
		bckerr4 := (bck0x+bck1x)
		bck3x2 := res*err
		bck1x1 := fInnerVJPx(bck3x2)
		bck0x1 := fOuterVJPx(bck3x2)
		bckerr5 := (bck0x1+bck1x1)
		bckz2 := (bckerr5+bckerr4)
		return bckz2
	}
	return z, selfVJPFuncWRTo, selfVJPFuncWRTx
}
`,
		},
		testgrad.Reverse{
			GradOf: "S.F",
			Src: `
type S struct {
	w float32
	b float32
}

func (s S) g(x float32) float32 {
	return s.w*x + s.b
}

func (s S) F(x float32) float32 {
	err := s.g(x)
	return err * err
}
`,
			Want: `
func (s S) vjpF(x float32) (float32, func(res float32) S, func(res float32) float32) {
	fwd0, gVJPs, gVJPx := grad.Reverse(s.g)(x)
	err := fwd0
	fwd1 := err*err
	selfVJPFuncWRTs := func(res float32) S {
		bck1y := err*res
		bck0s := gVJPs(bck1y)
		bckerr := bck0s
		bck1x := res*err
		bck0s1 := gVJPs(bck1x)
		bckerr1 := bck0s1
		return S{
			w: bckerr1.w+bckerr.w,
			b: bckerr1.b+bckerr.b,
		}
	}
	selfVJPFuncWRTx := func(res float32) float32 {
		bck1y1 := err*res
		bck0x := gVJPx(bck1y1)
		bckerr2 := bck0x
		bck1x1 := res*err
		bck0x1 := gVJPx(bck1x1)
		bckerr3 := bck0x1
		return bckerr3+bckerr2
	}
	return fwd1, selfVJPFuncWRTs, selfVJPFuncWRTx
}
`,
		},
	)
}

func TestVJPErrors(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Reverse{
			Src: `
type S struct {}

func F(s S, x float32) float32 {
	return s * x // ERROR invalid operation: operator * not defined on type S
}
`,
		},
	)
}
