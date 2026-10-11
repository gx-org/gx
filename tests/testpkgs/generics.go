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

package testpkgs

import (
	"github.com/gx-org/gx/build/builder/testbuild"
)

// TestGenerics declares a package with generic functions in a test.
//
// The signatures of the functions reference a package imported by testgenerics
// to check that callers do not need to import that package to instantiate the functions.
// A non-generic function is also included because its signature is instantiated the same way.
var TestGenerics = testbuild.DeclarePackage{
	Src: `
package testgenerics

import "testcpundef"

func Identity[A int](x [testcpundef.Undef(A)]float32) [testcpundef.Undef(A)]float32 {
	return x
}

func Ones() [testcpundef.Undef(2)]float32 {
	// The length of the array is a literal because the backends cannot evaluate
	// testcpundef.Undef(2) at runtime when its argument is a constant.
	return [4]float32{1, 1, 1, 1}
}
`,
}
