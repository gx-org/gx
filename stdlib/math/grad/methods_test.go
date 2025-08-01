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

func TestGradMethods(t *testing.T) {
	testbuild.Run(t,
		declareGradPackage,
		testgrad.Method{
			Src: `
type S struct {}

func (S) F(x float32) float32 {
	return 2.0
}
`,
			Want: `
func (S) gradF(x float32) float32 {
	return 0
}
`,
		},
		testgrad.Method{
			Src: `
type S struct {
	y float32
}

func (s S) F(x float32) float32 {
	return x*s.y
}
`,
			Want: `
func (s S) gradF(x float32) float32 {
	return s.y
}
`,
		},
	)
}
