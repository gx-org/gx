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

package algexpr_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/algexpr/testalgexpr"
)

func TestMul(t *testing.T) {
	testbuild.Run(t,
		testalgexpr.Compare{
			X: "3*2",
			Y: "6",
		},
		testalgexpr.Compare{
			X: "1",
			Y: "1*1*1",
		},
		testalgexpr.Compare{
			X:        "3*2",
			Y:        "7",
			NotEqual: true,
		},
		testalgexpr.Compare{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
				"B": ir.IntType(),
				"C": ir.IntType(),
			},
			X: "A*B*C",
			Ys: []string{
				"(A*B)*C",
				"A*(B*C)",
				"(A*B*C)",
			},
		},
		testalgexpr.Compare{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
				"B": ir.IntType(),
				"C": ir.IntType(),
				"D": ir.IntType(),
			},
			X: "A*B*C*D",
			Ys: []string{
				"(A*B)*(C*D)",
				"A*(B*C)*D",
				"A*(B*C*D)",
				"(A*(B*C*D))",
			},
		},
		testalgexpr.Compare{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
			},
			X:  "A",
			Ys: []string{"A*1", "1*A"},
		},
		testalgexpr.Compare{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
			},
			X:  "6*A",
			Ys: []string{"3*A*2", "2*A*3"},
		},
	)
}
