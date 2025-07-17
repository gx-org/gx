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
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestVars(t *testing.T) {
	vrA := &ir.VarExpr{
		Decl:  &ir.VarSpec{TypeV: ir.IntLenType()},
		VName: irh.Ident("a"),
	}
	vrA.Decl.Exprs = append(vrA.Decl.Exprs, vrA)
	testbuild.Run(t,
		testbuild.Decl{
			Src: `var a intlen`,
			Want: []ir.Node{
				vrA.Decl,
			},
		},
		testbuild.Decl{
			Src: `
import "dtypes"

var a intlen

func f[T dtypes.Floats]([a]float32) [a]float32
`,
			Err: "package dtypes",
		},
	)
}
