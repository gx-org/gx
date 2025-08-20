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

func TestLocalVar(t *testing.T) {
	aStorage1 := &ir.VarExpr{VName: irh.Ident("a")}
	aDeclSpec := &ir.VarSpec{
		TypeV: ir.Int32Type(),
		Exprs: []*ir.VarExpr{aStorage1},
	}
	aStorage1.Decl = aDeclSpec

	aStorage2 := &ir.VarExpr{VName: irh.Ident("a")}
	bStorage := &ir.VarExpr{VName: irh.Ident("b")}
	abDeclSpec := &ir.VarSpec{
		TypeV: ir.Int32Type(),
		Exprs: []*ir.VarExpr{aStorage2, bStorage},
	}
	aStorage2.Decl = abDeclSpec
	bStorage.Decl = abDeclSpec

	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func F() int32 {
    var a int32
    return a
}
`,
			Want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.DeclStmt{
							Decls: []*ir.VarSpec{aDeclSpec},
						},
						&ir.ReturnStmt{
							Results: []ir.Expr{irh.ValueRef(aStorage1)},
						},
					),
				},
			},
		},
		testbuild.Decl{
			Src: `
func F() (int32, int32) {
    var a, b int32
    return a, b
}
`,
			Want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type(), ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.DeclStmt{
							Decls: []*ir.VarSpec{abDeclSpec},
						},
						&ir.ReturnStmt{
							Results: []ir.Expr{irh.ValueRef(aStorage2), irh.ValueRef(bStorage)},
						},
					),
				},
			},
		},
		testbuild.Decl{
			Src: `
func F() int32 {
    var a, a int32
    return a
}
`,
			Err: "redeclared in this block",
		},
	)
}
