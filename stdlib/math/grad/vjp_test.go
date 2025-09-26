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
	fwd0 := float32(2.0)
	selfVJPFunc := func(res float32) float32 {
		return 0
	}
	return fwd0, selfVJPFunc
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
	fwd0, fVJP := grad.VJP(f, "x")(x)
	selfVJPFunc := func(res float32) float32 {
		bck0 := res*fVJP(x)
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

func g(x float32) float32 {
	return x
}

func F(x float32) float32 {
	return f(g(x))
}
`,
			Want: `
func vjpF(x float32) (float32, func(res float32) float32) {
	fwd0, gVJP := grad.VJP(g, "x")(x)
	fwd1, fVJP := grad.VJP(f, "x")(fwd0)
	selfVJPFunc := func(res float32) float32 {
		bck0 := res*fVJP(fwd0)
		bck1 := bck0*gVJP(x)
		return bck1
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
	fwd0, f0VJP := grad.VJP(f0, "x")(x)
	fwd1, f1VJP := grad.VJP(f1, "x")(fwd0)
	fwd2, f2VJP := grad.VJP(f2, "x")(fwd1)
	fwd3, f3VJP := grad.VJP(f3, "x")(fwd2)
	fwd4, f4VJP := grad.VJP(f4, "x")(fwd3)
	selfVJPFunc := func(res float32) float32 {
		bck0 := res*f4VJP(fwd3)
		bck1 := bck0*f3VJP(fwd2)
		bck2 := bck1*f2VJP(fwd1)
		bck3 := bck2*f1VJP(fwd0)
		bck4 := bck3*f0VJP(x)
		return bck4
	}
	return fwd4, selfVJPFunc
}
`,
		},
	)
}
