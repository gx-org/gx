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

package generics_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir/generics/testgenerics"
)

func TestInfer(t *testing.T) {
	testbuild.Run(t,
		testgenerics.Infer{
			Src: "func F(int32) int32",
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"int32(2)"},
					Want: "func(int32) int32",
				},
				testgenerics.Call{
					// Infer only checks generic arguments.
					// No error will be detected and the function signature will not change.
					// The error will be reported later by the compiler
					// when all arguments will be checked against their matching parameters.
					Args: []string{"int64(2)"},
					Want: "func(int32) int32",
				},
			},
		},
		testgenerics.Infer{
			Src: `
type someInt interface{ int32 | int64 }

func F[T someInt](T) T
`,
			Calls: []testgenerics.Call{
				testgenerics.Call{
					Args: []string{"int32(2)"},
					Want: "func[](int32) int32",
				},
				testgenerics.Call{
					Args: []string{"int64(2)"},
					Want: "func[](int64) int64",
				},
			},
		},
	)
}
