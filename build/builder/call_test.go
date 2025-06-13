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

func TestCallOperators(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func cast[S interface{float64}](S) uint64

func f() uint64 {
	v := cast[float64](1.0)
	return v
}
`,
		},
	)
}

func TestCallSliceArgs(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
var A, B intlen

func f[T int32]([A][B][1]float32, []intidx) T

func fail(x [A][B][1]float32) float32 {
	return float32(f[int32](x, []intidx{0}))
}
`,
		},
	)
}
