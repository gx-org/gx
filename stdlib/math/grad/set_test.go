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

func TestSet(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
func g(float32) float32

//gx:@grad.Set(g)
func F(x float32) float32 {
	return x
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := F(x)
	selfVJPFunc := func(res float32) float32 {
		return res*g(x)
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func gradOfG(x float32) float32 {
	return x
}

//gx:@grad.Set(gradOfG)
func g(x float32) float32 {
	return x	
}

func F(x float32) float32 {
	return g(x*x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x*x
	fwd1, gVJP := grad.VJP(g)(fwd0)
	selfVJPFunc := func(res float32) float32 {
		bck1 := gVJP(res)
		bck0x := bck1*x
		bck0y := x*bck1
		return bck0x+bck0y
	}
	return fwd1, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
func gradOfG(x float32) float32

//gx:@grad.Set(gradOfG)
func g(x float32) float32

func F(x float32) float32 {
	return g(x*x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x*x
	fwd1, gVJP := grad.VJP(g)(fwd0)
	selfVJPFunc := func(res float32) float32 {
		bck1 := gVJP(res)
		bck0x := bck1*x
		bck0y := x*bck1
		return bck0x+bck0y
	}
	return fwd1, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
//gx:@grad.Set(g)
func g(x float32) float32

func F(x float32) float32 {
	return g(x*x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := x*x
	fwd1, gVJP := grad.VJP(g)(fwd0)
	selfVJPFunc := func(res float32) float32 {
		bck1 := gVJP(res)
		bck0x := bck1*x
		bck0y := x*bck1
		return bck0x+bck0y
	}
	return fwd1, selfVJPFunc
}
`,
		},
	)
}

func TestSetFor(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
func gradOfGx(x, y float32) float32 {
	return x
}

func gradOfGy(x, y float32) float32 {
	return y
}

//gx:@grad.SetFor("x", gradOfGx)
//gx:@grad.SetFor("y", gradOfGy)
func F(x, y float32) float32 {
	return x	
}
`,
			Want: `
func vjpF(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0 := F(x, y)
	selfVJPFuncWRTx := func(res float32) float32 {
		return res*gradOfGx(x, y)
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		return res*gradOfGy(x, y)
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.VJP{
			Src: `
func gradOfGx(x, y float32) float32 {
	return x
}

func gradOfGy(x, y float32) float32 {
	return y
}

//gx:@grad.SetFor("x", gradOfGx)
//gx:@grad.SetFor("y", gradOfGy)
func F(x, y float32) float32 {
	return x	
}
`,
			Want: `
func vjpF(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0 := F(x, y)
	selfVJPFuncWRTx := func(res float32) float32 {
		return res*gradOfGx(x, y)
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		return res*gradOfGy(x, y)
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}
`,
		},
		testgrad.VJP{
			Src: `
func g(float32) float32

//gx:@grad.SetFor("x", g)
func F(x float32) float32 {
	return x
}`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0 := F(x)
	selfVJPFunc := func(res float32) float32 {
		return res*g(x)
	}
	return fwd0, selfVJPFunc
}
`,
		},
	)
}

func TestSetGeneric(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
type floats interface {
	float32 | float64
}

func g[T floats](T) T 

//gx:@grad.Set(g)
func F[T floats](x T) T {
	return x
}
`,
			Want: `
func vjpF[T floats](x T) (T, func(res T) T) {
	fwd0 := F[T](x)
	selfVJPFunc := func(res T) T {
		return res*g[T](x)
	}
	return fwd0, selfVJPFunc
}
`,
		},
		testgrad.VJP{
			Src: `
type floats interface {
	float32 | float64
}

func backward[T floats](x T) T 

//gx:@grad.Set(backward)
func forward[T floats](x T) T 

func F(x float32) float32 {
	return forward[float32](x)
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, forwardVJP := grad.VJP(forward)[float32](x)
	selfVJPFunc := func(res float32) float32 {
		bck0 := forwardVJP(res)
		return bck0
	}
	return fwd0, selfVJPFunc
}
`,
			WantExprs: map[string]string{
				"grad.VJP(forward)": `
func[T floats](x T) (T, func(res T) T) {
	fwd0 := forward[T](x)
	selfVJPFunc := func(res T) T {
		return res*backward[T](x)
	}
	return fwd0, selfVJPFunc
}
`,
			},
		},
		testgrad.VJP{
			Src: `
type floats interface {
	float32 | float64
}

func g[T floats](x [___M]T) [M___]T

//gx:@grad.Set(g)
func F[T floats](x [___M]T) [M___]T
`,
			Want: `
func vjpF[T floats](x [___M]T) ([M___]T, func(res [M___]T) [M___]T) {
	fwd0 := F[T](x)
	selfVJPFunc := func(res [M___]T) [M___]T {
		return res*g[T](x)
	}
	return fwd0, selfVJPFunc
}
`,
		},
	)
}

func TestSetErrors(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.VJP{
			Src: `
//gx:@grad.Set(missingGrad)
func F(x float32) float32 {
	return 42
}`,
			Err: "undefined: missingGrad",
		},
		testgrad.VJP{
			Src: `
var v int32
//gx:@grad.Set(v)
func F(x float32) float32 {
	return 42
}`,
			Err: "int32 is not a function",
		},
		testgrad.VJP{
			Src: `
func g(x float64) float32

//gx:@grad.Set(g)
func F(x, y float32) float32
`,
			Err: "use test.SetFor to set the gradient of a function with more than one parameter",
		},
		testgrad.VJP{
			Src: `
func gradOfF(x, y float32) float32

//gx:@grad.SetFor("z", gradOfF)
func F(x, y float32) float32 {
	return x
}
`,
			Err: "function F has no parameter z",
		},
		testgrad.VJP{
			Src: `
func gradOfF(x, y float32) float32

//gx:@grad.SetFor("x", gradOfF)
//gx:@grad.SetFor("x", gradOfF)
func F(x, y float32) float32 {
	return x
}
`,
			Err: "gradient for parameter x has already been set",
		},
		testgrad.VJP{
			Src: `
func g(float32) float32

//gx:@grad.Set(g)
//gx:@grad.SetFor("x", g)
func F(x float32) float32 {
	return x
}`,
			Err: "gradient for parameter x has already been set",
		},
	)
}
