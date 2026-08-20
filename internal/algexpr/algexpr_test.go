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

func TestCompare(t *testing.T) {
	testbuild.Run(t,
		testalgexpr.Compare{
			X:        "3",
			Y:        "2",
			NotEqual: true,
		},
		testalgexpr.Compare{
			X: "2",
			Y: "2",
		},
		testalgexpr.Compare{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
			},
			X: "A",
			Y: "A",
		},
		testalgexpr.Compare{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
				"B": ir.IntType(),
			},
			X:        "A",
			Y:        "B",
			NotEqual: true,
		},
	)
}

func TestSimplifyConst(t *testing.T) {
	testbuild.Run(t,
		testalgexpr.Compare{
			PkgDecl: `
const A = 2
`,
			X:  "A",
			Ys: []string{"A", "2"},
		},
		testalgexpr.Compare{
			PkgDecl: `
const A = 2
const B = 3
`,
			X:  "A*B",
			Ys: []string{"A*B", "6"},
		},
	)
}

func TestToExpr(t *testing.T) {
	testbuild.Run(t,
		testalgexpr.Simplify{
			X:    "2+3",
			Want: "5",
		},
		testalgexpr.Simplify{
			X:    "2==2",
			Want: "true",
		},
		testalgexpr.Simplify{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
			},
			X:    "A==2",
			Want: "false",
		},
		testalgexpr.Simplify{
			X:    "2>1",
			Want: "true",
		},
		testalgexpr.Simplify{
			X:    "2>2",
			Want: "false",
		},
		testalgexpr.Simplify{
			X:    "2>3",
			Want: "false",
		},
		testalgexpr.Simplify{
			X:    "2<1",
			Want: "false",
		},
		testalgexpr.Simplify{
			X:    "2<2",
			Want: "false",
		},
		testalgexpr.Simplify{
			X:    "2<3",
			Want: "true",
		},
		testalgexpr.Simplify{
			X:    "2>=1",
			Want: "true",
		},
		testalgexpr.Simplify{
			X:    "2>=2",
			Want: "true",
		},
		testalgexpr.Simplify{
			X:    "2>=3",
			Want: "false",
		},
		testalgexpr.Simplify{
			X:    "2<=1",
			Want: "false",
		},
		testalgexpr.Simplify{
			X:    "2<=2",
			Want: "true",
		},
		testalgexpr.Simplify{
			X:    "2<=3",
			Want: "true",
		},
		testalgexpr.Simplify{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
			},
			X:    "A>2",
			Want: "A>2",
		},
		testalgexpr.Simplify{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
			},
			X:    "A<2",
			Want: "A<2",
		},
		testalgexpr.Simplify{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
			},
			X:    "A>=2",
			Want: "A>=2",
		},
		testalgexpr.Simplify{
			Vars: map[string]ir.Type{
				"A": ir.IntType(),
			},
			X:    "A<=2",
			Want: "A<=2",
		},
	)
}
