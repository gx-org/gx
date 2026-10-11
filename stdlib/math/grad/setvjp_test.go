// Copyright 2026 Google LLC
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

func TestSetVJP(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Reverse{
			Src: `
func someFunc(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	fwd0 := F(x, y)
	selfVJPFuncWRTx := func(res float32) float32 {
		wrtX := res
		return wrtX
	}
	selfVJPFuncWRTy := func(res float32) float32 {
		wrtY := res
		return wrtY
	}
	return fwd0, selfVJPFuncWRTx, selfVJPFuncWRTy
}

//gx:@grad.SetVJP(someFunc)
func F(x, y float32) float32 {
	return x+y
}
`,
			Want: `
func vjpF(x, y float32) (float32, func(res float32) float32, func(res float32) float32) {
	return someFunc(x, y)
}
`,
		},
		testgrad.Reverse{
			Src: `
func someFunc[Axis []int, Shape []int](x float32) (float32, func(res float32) float32) {
	fwd0 := F[Axis][Shape](x)
	vjp := func(res float32) float32 {
		return x
	}
	return fwd0, vjp
}

//gx:@grad.SetVJP(someFunc)
func F[Axis []int, Shape []int](x float32) float32 {
	return x
}
`,
			Want: `
func vjpF[Axis []int, Shape []int](x float32) (float32, func(res float32) float32) {
	return someFunc[Axis][Shape](x)
}
`,
		},
	)
}
