package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestAssign(t *testing.T) {
	aStorage := &ir.AssignExpr{
		Storage: irh.LocalVar("a", ir.Float32Type()),
		X: &ir.CastExpr{
			X:   irh.IntNumberAs(2, ir.Float32Type()),
			Typ: ir.Float32Type(),
		},
	}
	bStorage := &ir.AssignExpr{
		Storage: irh.LocalVar("b", ir.Float32Type()),
		X: &ir.CastExpr{
			X:   irh.IntNumberAs(3, ir.Float32Type()),
			Typ: ir.Float32Type(),
		},
	}
	cStorage := irh.LocalVar("c", ir.Float32Type())
	dStorage := irh.LocalVar("d", ir.Float32Type())
	assign := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: irh.Ident("assign")},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields(),
			irh.Fields(ir.Float32Type(), ir.Float32Type()),
		),
		Body: irh.Block(
			&ir.AssignExprStmt{List: []*ir.AssignExpr{
				aStorage, bStorage,
			}},
			&ir.ReturnStmt{
				Results: []ir.Expr{
					irh.ValueRef(aStorage),
					irh.ValueRef(bStorage),
				},
			},
		)}
	callToAssign := &ir.CallExpr{
		Callee: &ir.FuncValExpr{
			X: irh.ValueRef(assign),
			F: assign,
			T: assign.FType,
		},
	}
	cAssignment := &ir.AssignCallResult{
		Storage:     cStorage,
		Call:        callToAssign,
		ResultIndex: 0,
	}
	dAssignment := &ir.AssignCallResult{
		Storage:     dStorage,
		Call:        callToAssign,
		ResultIndex: 1,
	}
	testAll(t,
		irDeclTest{
			src: `
func assign() (float32, float32) {
	a, b := float32(2), float32(3)
	return a, b
}
`,
			want: []ir.Node{assign},
		},
		irDeclTest{
			src: `
func assign() (float32, float32) {
	a, b := float32(2), float32(3)
	return a, b
}

func callAssign() (float32, float32) {
	c, d := assign()
	return c, d
}
`,
			want: []ir.Node{
				assign,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Float32Type(), ir.Float32Type()),
					),
					Body: irh.Block(
						&ir.AssignCallStmt{
							List: []*ir.AssignCallResult{
								cAssignment,
								dAssignment,
							},
							Call: callToAssign,
						},
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.ValueRef(cAssignment),
								irh.ValueRef(dAssignment),
							},
						},
					)},
			},
		},
		irDeclTest{
			src: `
func id(int64) int64

func f() int64 {
	a, b := 2, 3
	c := a+b
	return id(c)
}
`,
		},
		irDeclTest{
			src: `
func f() uint32 {
	a, b := 2, 3
	c := uint32(a+b)
	return c
}
`,
		},
		irDeclTest{
			src: `
func g(uint32) uint32
func f() uint32 {
	a := uint32(2)
	a = a + 2
	return g(a)
}
`,
		},
		irDeclTest{
			src: `
func f() int64 {
	true := 3
	return true
}
`,
		},
	)
}
