package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestStruct(t *testing.T) {
	aField := irh.Field("a", ir.Int32Type(), nil)
	bField := irh.Field("b", ir.Float32Type(), nil)
	structA := irh.StructType(aField, bField)
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("A")},
		Underlying: irh.TypeExpr(structA),
	}
	testAll(t,
		irDeclTest{
			src: `
type A struct {
	a int32
	b float32
}

func a() A {
	return A{
		a: 1,
		b: 2,
	}
}
`,
			want: []ir.Node{
				typeA,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(typeA),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.StructLitExpr{
								Elts: []*ir.FieldLit{
									irh.FieldLit(structA.Fields, "a", irh.IntNumberAs(1, ir.Int32Type())),
									irh.FieldLit(structA.Fields, "b", irh.IntNumberAs(2, ir.Float32Type())),
								},
								Typ: typeA,
							}},
						},
					)},
			},
		},
	)
}
