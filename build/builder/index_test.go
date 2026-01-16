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

func TestIndex(t *testing.T) {
	testbuild.Run(t,
		testbuild.Expr{
			Src: `[2]float32{3, 4}[1]`,
			Want: &ir.IndexExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.Float32Type(), 2),
					Elts: []ir.Expr{
						irh.IntNumberAs(3, ir.Float32Type()),
						irh.IntNumberAs(4, ir.Float32Type()),
					},
				},
				Index: irh.IntNumberAs(1, ir.Int64Type()),
				Typ:   ir.Float32Type(),
			},
			WantType: "float32",
		},
		testbuild.Expr{
			Src: `[2]float32{3, 4}[float32(1)]`,
			Err: "index float32(1) (of type float32) must be integer",
		},
	)
}
