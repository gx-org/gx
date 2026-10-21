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

// TestPkg1 declares a basic package used to test imports.
var TestPkg1 = testbuild.DeclarePackage{
	Src: `
package pkg1

// Int is a named int type.
type Int int32

// AB is a struct with an A and B.
type AB struct {
	A int32
	B float32
}

// Const42 always returns 42.
func Const42() int32 {
	return 42
}

// CallConst42 calls the Const42 functions.
func CallConst42() int32 {
	return Const42()
}
`,
}
