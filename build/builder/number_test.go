package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestNumber(t *testing.T) {
	valField := irh.Field("val", ir.Float32Type(), nil)
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("A")},
		Underlying: irh.TypeExpr(irh.StructType(valField)),
	}
	typeA.Methods = []ir.PkgFunc{
		&ir.FuncDecl{
			FType: irh.FuncType(
				nil,
				irh.Fields("a", typeA),
				irh.Fields(),
				irh.Fields(ir.Float32Type()),
			),
			Body: irh.Block(
				&ir.AssignExprStmt{List: []*ir.AssignExpr{{
					X: irh.FloatNumberAs(0, ir.Float32Type()),
					Storage: &ir.StructFieldStorage{Sel: &ir.SelectorExpr{
						X:    irh.ValueRef(typeA),
						Stor: valField.Storage(),
					}},
				}}},
				&ir.ReturnStmt{Results: []ir.Expr{
					&ir.SelectorExpr{
						X:    irh.ValueRef(typeA),
						Stor: valField.Storage(),
					},
				}},
			),
		},
	}
	testAll(t,
		irDeclTest{
			src: `
func id(int64) int64

func f() int64 {
	a := 2
	return id(a)
}
`,
		},
		irDeclTest{
			src: `
type A struct {
	val float32
}

func (a A) f() float32 {
	a.val = 0.0	
	return a.val
}
`,
		},
	)
}
