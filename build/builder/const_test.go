package builder_test

import (
	"go/ast"
	"math/big"
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestConst(t *testing.T) {
	cstA := &ir.ConstExpr{
		Decl:  &ir.ConstDecl{},
		VName: irh.Ident("cstA"),
		Val:   &ir.NumberInt{Val: big.NewInt(5)},
	}
	cstIntA := &ir.ConstExpr{
		Decl:  &ir.ConstDecl{},
		VName: irh.Ident("cstA"),
		Val:   irh.IntNumberAs(5, ir.Int32Type()),
	}
	cstA.Decl.Exprs = append(cstA.Decl.Exprs, cstA)
	testAll(t,
		irDeclTest{
			src: `const cstA = 5`,
			want: []ir.Node{
				cstA.Decl,
			},
		},
		irDeclTest{
			src: `const cstIntA int32 = 5`,
			want: []ir.Node{
				cstIntA.Decl,
			},
		},
		irDeclTest{
			src: `
const cstA = 5
type Array [cstA]float32
`,
			want: []ir.Node{
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
		irDeclTest{
			src: `
const cstB = cstA
const cstA = 5
`,
			want: []ir.Node{
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
