package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestNamedTypes(t *testing.T) {
	aField := irh.Field("a", ir.Int32Type(), nil)
	bField := irh.Field("b", ir.Float32Type(), nil)
	cField := irh.Field("c", ir.Float64Type(), nil)
	testbuild.Run(t,
		testbuild.DeclTest{
			Src: `type A intlen`,
			Want: []ir.Node{&ir.NamedType{
				Src:        &ast.TypeSpec{Name: irh.Ident("A")},
				Underlying: ir.AtomTypeExpr(ir.IntLenType()),
			}},
		},
		testbuild.DeclTest{
			Src: `type B interface {
				float32 | float64
			}`,
			Want: []ir.Node{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.Ident("B")},
				Underlying: irh.TypeExpr(irh.TypeSet(
					ir.Float32Type(),
					ir.Float64Type(),
				))}},
		},
		testbuild.DeclTest{
			Src: `type A struct {
				a int32
				b float32
				c float64
			}`,
			Want: []ir.Node{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.Ident("A")},
				Underlying: irh.TypeExpr(irh.StructType(
					aField, bField, cField,
				))}},
		},
		testbuild.DeclTest{
			Src: `type A [2][3]float32`,
			Want: []ir.Node{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.Ident("A")},
				Underlying: irh.TypeExpr(irh.ArrayType(
					ir.Float32Type(),
					irh.IntNumberAs(2, ir.IntLenType()),
					irh.IntNumberAs(3, ir.IntLenType()),
				)),
			}},
		},
	)
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("A")},
		Underlying: ir.AtomTypeExpr(ir.Float32Type()),
	}
	testbuild.Run(t,
		testbuild.DeclTest{
			Src: `
type A float32

func f() [2][3]A
`,
			Want: []ir.Node{
				typeA,
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(typeA,
							irh.IntNumberAs(2, ir.IntLenType()),
							irh.IntNumberAs(3, ir.IntLenType()),
						))),
				},
			},
		},
	)
}
