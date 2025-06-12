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
		Decl:  &ir.ConstDecl{},
		VName: irh.Ident("cstA"),
		Val:   &ir.NumberInt{Val: big.NewInt(5)},
	}
	cstA.Decl.Exprs = append(cstA.Decl.Exprs, cstA)
	testbuild.Run(t,
		testbuild.DeclTest{
			Src: `const cstA = 5`,
			Want: []ir.Node{
				cstA.Decl,
			},
		},
		testbuild.DeclTest{
			Src: `
const cstA = 5
type Array [cstA]float32
`,
			Want: []ir.Node{
				&ir.NamedType{
					Src: &ast.TypeSpec{Name: irh.Ident("Array")},
					Underlying: irh.TypeExpr(irh.ArrayType(
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
		testbuild.DeclTest{
			Src: `
const cstB = cstA
const cstA = 5
`,
			Want: []ir.Node{
				irh.ConstSpec(nil,
					&ir.ConstExpr{
						VName: irh.Ident("cstB"),
						Val:   irh.ValueRef(cstA),
					},
				),
				irh.ConstSpec(nil, cstA),
			},
		},
	)
}

func TestConstWithType(t *testing.T) {
	cstIntA := &ir.ConstExpr{
		Decl:  &ir.ConstDecl{Type: irh.TypeRef(ir.Int32Type())},
		VName: irh.Ident("cstIntA"),
		Val:   &ir.NumberInt{Val: big.NewInt(5)},
	}
	cstIntA.Decl.Exprs = append(cstIntA.Decl.Exprs, cstIntA)
	testbuild.Run(t,
		testbuild.DeclTest{
			Src: `const cstIntA int32 = 5`,
			Want: []ir.Node{
				cstIntA.Decl,
			},
		},
	)
}
