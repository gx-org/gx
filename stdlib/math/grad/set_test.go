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
		testgrad.Func{
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
		testgrad.Func{
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
		return bck1*x+x*bck1
	}
	return fwd1, selfVJPFunc
}
`,
		},
		testgrad.Func{
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
		return bck1*x+x*bck1
	}
	return fwd1, selfVJPFunc
}
`,
		},
		testgrad.Func{
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
		return bck1*x+x*bck1
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
		testgrad.Func{
			Src: `
func gradOfGx(x, y float32) float32 {
	return x
}

func gradOfGy(x, y float32) float32 {
	return y
}

//gx:@grad.SetFor(gradOfGx, "x")
//gx:@grad.SetFor(gradOfGy, "y")
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
		testgrad.Func{
			WRT: "y",
			Src: `
func gradOfGx(x, y float32) float32 {
	return x
}

func gradOfGy(x, y float32) float32 {
	return y
}

//gx:@grad.SetFor(gradOfGx, "x")
//gx:@grad.SetFor(gradOfGy, "y")
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
		testgrad.Func{
			Src: `
func g(float32) float32

//gx:@grad.SetFor(g, "x")
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

func TestSetErrors(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Func{
			Src: `
//gx:@grad.Set(missingGrad)
func F(x float32) float32 {
	return 42
}`,
			Err: "undefined: missingGrad",
		},
		testgrad.Func{
			Src: `
var v int32
//gx:@grad.Set(v)
func F(x float32) float32 {
	return 42
}`,
			Err: "int32 is not a function",
		},
		testgrad.Func{
			Src: `
func g(x float64) float32

//gx:@grad.Set(g)
func F(x, y float32) float32
`,
			Err: "cannot set gradient of F: requires a single argument",
		},
		testgrad.Func{
			Src: `
func gradOfF(x, y float32) float32

//gx:@grad.SetFor(gradOfF, "z")
func F(x, y float32) float32 {
	return x	
}
`,
			Err: "function F has no parameter z",
		},
		testgrad.Func{
			Src: `
func gradOfF(x, y float32) float32

//gx:@grad.SetFor(gradOfF, "x")
//gx:@grad.SetFor(gradOfF, "x")
func F(x, y float32) float32 {
	return x	
}
`,
			Err: "gradient for parameter x has already been set",
		},
		testgrad.Func{
			Src: `
func g(float32) float32

//gx:@grad.Set(g)
//gx:@grad.SetFor(g, "x")
func F(x float32) float32 {
	return x
}`,
			Err: "gradient for parameter x has already been set",
		},
	)
}
