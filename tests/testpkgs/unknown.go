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

package testpkgs

import (
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
)

// TestUnknown declares the macro package in a test.
var TestUnknown = testbuild.DeclarePackage{
	Src: `
package testunknown

type U any
`,
	Post: func(pkg *ir.Package) {
		tp := pkg.Decls.TypeByName("U")
		if tp == nil {
			panic("cannot find Obj type")
		}
		tp.Underlying = ir.TypeFromExpr(ir.NewIdent(ir.UnknownType()))
	},
}
