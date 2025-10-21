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
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/tests/testmacros"
)

func TestMacro(t *testing.T) {
	testbuild.Run(t,
		testmacros.DeclarePackage,
		testbuild.Decl{
			Src: `
import "testmacros"

//gx:=testmacros.ID(f)
func synthetic()

func f() int32 {
	return 2
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
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.IntNumberAs(2, ir.Int32Type()),
							},
						},
					)},
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.IntNumberAs(2, ir.Int32Type()),
							},
						},
					)},
			},
		},
		testbuild.Decl{
			Src: `
import "testmacros"

//gx:=testmacros.ID(f)
func synthetic()

func f(x int32) int32 {
	return x
}
`,
		},
		testbuild.Decl{
			Src: `
import "testmacros"

func f(x int32) int32 {
	return x
}

func F() int32 {
	return testmacros.ID(f)(2)
}
`,
		},
	)
}

func TestMacroWithGenerics(t *testing.T) {
	testbuild.Run(t,
		testmacros.DeclarePackage,
		testbuild.Decl{
			Src: `
import "testmacros"

type floats interface {
	float32 | float64
}

func f[T floats]() T {
	return 2
}

func g[T floats]() T {
	return testmacros.ID(f)[T]()
}
`,
		},
		testbuild.Decl{
			Src: `
import "testmacros"

type floats interface {
	float32 | float64
}

func f[T floats](x T) T {
	return 2*x
}

func g[T floats](x T) T {
	return testmacros.ID(f)[T](x)
}
`,
		},
	)
}

func TestMacroOnMethod(t *testing.T) {
	typeS := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("S")},
		Underlying: irh.TypeExpr(irh.StructType()),
	}
	fType := irh.FuncType(
		nil,
		irh.Fields(typeS),
		irh.Fields(),
		irh.Fields(ir.Int32Type()),
	)
	body := irh.SingleReturn(irh.IntNumberAs(2, ir.Int32Type()))
	typeS.Methods = []ir.PkgFunc{
		&ir.FuncDecl{
			FType: fType,
			Body:  body,
		},
		&ir.FuncDecl{
			FType: fType,
			Body:  body,
		},
	}
	testbuild.Run(t,
		testmacros.DeclarePackage,
		testbuild.Decl{
			Src: `
import "testmacros"

type S struct{}

//gx:=testmacros.ID(S.f)
func (S) synthetic()

func (S) f() int32 {
	return 2
}
`,
			Want: []ir.Node{
				typeS,
			},
		},
		testbuild.Decl{
			Src: `
import "testmacros"

type S struct{}

//gx:=testmacros.ID(S.f)
func synthetic()

func (S) f() int32 {
	return 2
}
`,
			Err: "synthetic requires a test.S type receiver",
		},
		testbuild.Decl{
			Src: `
import "testmacros"

type S struct{}

//gx:=testmacros.ID(f)
func (S) synthetic()

func f() int32 {
	return 2
}
`,
			Err: "synthetic requires no receiver",
		},
		testbuild.Decl{
			Src: `
import "testmacros"

type S struct{}

type T struct{}

//gx:=testmacros.ID(S.f)
func (T) synthetic()

func (S) f() int32 {
	return 2
}
`,
			Err: "cannot assign S.synthetic to T.synthetic",
		},
	)
}

func TestMacroWithErrors(t *testing.T) {
	testbuild.Run(t,
		testmacros.DeclarePackage,

		testbuild.Decl{
			Src: `
import "testmacros"

type floats interface {
	float32 | float64
}

func f[T floats](x T) T {
	return 2*x
}

func g[T floats](x T) T {
	return testmacros.ID(f)[T]()
}
`,
			Err: "not enough arguments in call to func[](x test.floats) test.floats",
		},
	)
}
