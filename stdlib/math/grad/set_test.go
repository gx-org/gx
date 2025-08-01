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
func gradOfG(x float32) float32 {
	return x
}

//gx@=grad.Set(gradOfG)
func g(x float32) float32 {
	return x	
}

func F(x float32) float32 {
	return g(x*x)
}
`,
			Want: `
func gradF(x float32) float32 {
	return gradOfG(x*x)*(x+x)
}
`,
		},
		testgrad.Func{
			Src: `
func gradOfG(x float32) float32

//gx@=grad.Set(gradOfG)
func g(x float32) float32

func F(x float32) float32 {
	return g(x*x)
}
`,
			Want: `
func gradF(x float32) float32 {
	return gradOfG(x*x)*(x+x)
}
`,
		},
	)
}
