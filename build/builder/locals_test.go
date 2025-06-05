package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestLocalAssignment(t *testing.T) {
	idFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "id"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields("x", ir.Float32Type()),
			irh.Fields(ir.Float32Type()),
		),
	}
	xField := irh.Field("x", ir.Float32Type(), nil)
	yField := irh.Field("y", ir.Float32Type(), nil)
	testAll(t,
		irDeclTest{
			src: `
func id(x float32) float32

func call(x float32, y float32) float32 {
	return id(x/y)
}
`,
			want: []ir.Node{
				idFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(xField, yField),
						irh.Fields(ir.Float32Type()),
					),
					Body: irh.SingleReturn(
						&ir.CallExpr{
							Callee: &ir.FuncValExpr{
								X: irh.ValueRef(idFunc),
								F: idFunc,
								T: idFunc.FType,
							},
							Args: []ir.AssignableExpr{&ir.BinaryExpr{
								X:   irh.ValueRef(xField.Storage()),
								Y:   irh.ValueRef(yField.Storage()),
								Typ: ir.Float32Type(),
							}},
						},
					),
				},
			},
		},
	)
}
