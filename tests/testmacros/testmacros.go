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

// Package testmacros defines macros for testing.
package testmacros

import (
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
)

// DeclarePackage declares the macro package in a test.
var DeclarePackage = testbuild.DeclarePackage{
	Src: `
package testmacros

//gx:irmacro
func ID(any) any

//gx:irmacro
func UpdateReturn(any, string) any

`,
	Post: func(pkg *ir.Package) {
		id := pkg.FindFunc("ID").(*ir.Macro)
		id.BuildSynthetic = ir.MacroImpl(newID)
		ret := pkg.FindFunc("UpdateReturn").(*ir.Macro)
		ret.BuildSynthetic = ir.MacroImpl(newUpdateReturn)
	},
}
