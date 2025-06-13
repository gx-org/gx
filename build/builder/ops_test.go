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

func TestBinaryOp(t *testing.T) {
	testbuild.Run(t,
		testbuild.Expr{
			Src: `[2]int64{3, 4} > 2`,
			Want: &ir.BinaryExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.Int64Type(), 2),
					Elts: []ir.AssignableExpr{
						irh.IntNumberAs(3, ir.Int64Type()),
						irh.IntNumberAs(4, ir.Int64Type()),
					},
				},
				Y:   irh.IntNumberAs(2, ir.Int64Type()),
				Typ: irh.ArrayType(ir.BoolType(), 2),
			},
			WantType: "[2]bool",
		},
	)
}
