package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestNamedTypes(t *testing.T) {
	aField := irh.Field("a", ir.Int32Type(), nil)
	bField := irh.Field("b", ir.Float32Type(), nil)
	cField := irh.Field("c", ir.Float64Type(), nil)
	testAll(t,
		irDeclTest{
			src: `type A intlen`,
			want: []ir.Node{&ir.NamedType{
				Src:        &ast.TypeSpec{Name: irh.Ident("A")},
				Underlying: ir.AtomTypeExpr(ir.IntLenType()),
			}},
		},
		irDeclTest{
			src: `type B interface {
				float32 | float64
			}`,
			want: []ir.Node{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.Ident("B")},
				Underlying: irh.TypeExpr(irh.TypeSet(
					ir.Float32Type(),
					ir.Float64Type(),
				))}},
		},
		irDeclTest{
			src: `type A struct {
				a int32
				b float32
				c float64
			}`,
			want: []ir.Node{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.Ident("A")},
				Underlying: irh.TypeExpr(irh.StructType(
					aField, bField, cField,
				))}},
		},
		irDeclTest{
			src: `type A [2][3]float32`,
			want: []ir.Node{&ir.NamedType{
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
	testAll(t,
		irDeclTest{
			src: `
type A float32

func f() [2][3]A
`,
			want: []ir.Node{
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
