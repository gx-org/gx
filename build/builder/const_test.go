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
	"go/ast"
	"math/big"
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestConst(t *testing.T) {
	cstA := &ir.ConstExpr{
		Decl:  &ir.ConstSpec{},
		VName: irh.Ident("cstA"),
		Val:   &ir.NumberInt{Val: big.NewInt(5)},
	}
	cstA.Decl.Exprs = append(cstA.Decl.Exprs, cstA)
	testbuild.Run(t,
		testbuild.Decl{
			Src: `const cstA = 5`,
			Want: []ir.IR{
				cstA.Decl,
			},
		},
		testbuild.Decl{
			Src: `
const cstA = 5
type Array [cstA]float32
`,
			Want: []ir.IR{
				&ir.NamedType{
					Src: &ast.TypeSpec{Name: irh.Ident("Array")},
					Underlying: ir.TypeExpr(nil, irh.ArrayType(
						ir.Float32Type(),
						&ir.NumberCastExpr{
							X:   irh.ValueRef(cstA),
							Typ: ir.IntLenType(),
						},
					)),
				},
				irh.ConstSpec(nil, cstA),
			},
		},
		testbuild.Decl{
			Src: `
const cstB = cstA
const cstA = 5
`,
			Want: []ir.IR{
				irh.ConstSpec(nil,
					&ir.ConstExpr{
						VName: irh.Ident("cstB"),
						Val:   irh.ValueRef(cstA),
					},
				),
				irh.ConstSpec(nil, cstA),
			},
		},
		testbuild.Decl{
			Src: `
const cstA = 2 * (3 + 4)
`,
			Want: []ir.IR{
				irh.ConstSpec(nil,
					&ir.ConstExpr{
						VName: irh.Ident("cstA"),
						Val: &ir.BinaryExpr{
							X: &ir.NumberInt{Val: big.NewInt(2)},
							Y: &ir.ParenExpr{
								X: &ir.BinaryExpr{
									X:   &ir.NumberInt{Val: big.NewInt(3)},
									Y:   &ir.NumberInt{Val: big.NewInt(4)},
									Typ: ir.NumberIntType(),
								},
							},
							Typ: ir.NumberIntType(),
						},
					},
				),
			},
		},
		testbuild.Decl{
			Src: `
const (
	cstA = 5.2
	cstB = 2 / cstA
)
`,
		},
		testbuild.Decl{
			Src: `
const (
	cstA = 0.6
	cstB = 1 / cstA
	cstC = 2.3
	cstD = 1 / cstC
)
`,
		},
	)
}

func TestConstWithType(t *testing.T) {
	cstIntA := &ir.ConstExpr{
		Decl:  &ir.ConstSpec{Type: ir.TypeExpr(nil, ir.Int32Type())},
		VName: irh.Ident("cstIntA"),
		Val: &ir.NumberCastExpr{
			X:   &ir.NumberInt{Val: big.NewInt(5)},
			Typ: ir.Int32Type(),
		},
	}
	cstIntA.Decl.Exprs = append(cstIntA.Decl.Exprs, cstIntA)
	testbuild.Run(t,
		testbuild.Decl{
			Src: `const cstIntA int32 = 5`,
			Want: []ir.IR{
				cstIntA.Decl,
			},
		},
	)
}
