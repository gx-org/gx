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

package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
)

func TestErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type S struct{}

func New() S {
	return &S{}
}
`,
			Err: "unary operator & not supported",
		},
		testbuild.Decl{
			Src: `
type S struct{}

func New() S {
	return *S{}
}
`,
			Err: "expression *<expr> not supported",
		},
		testbuild.Decl{
			Src: `
func New() float32 {
	1 := 2
	return 1
}
`,
			Err: "non-name on left side of :=",
		},
		testbuild.Decl{
			Src: `
func New() float32 {
	1 = 2
	return 1
}
`,
			Err: "non-name on left side of =",
		},
	)
}
