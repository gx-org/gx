package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestCompEval(t *testing.T) {
	testAll(t,
		irDeclTest{
			src: `
//gx:compeval
func returnTwo() float32 {
	return 2
}
`,
			want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.CompEvalFuncType(
						irh.Fields(),
						irh.Fields(ir.Float32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.IntNumberAs(2, ir.Float32Type()),
							},
						},
					)},
			},
		},
	)
}

func TestBuiltin(t *testing.T) {
	testAll(t,
		irDeclTest{
			src: `func returnTwo() float32`,
			want: []ir.Node{
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Float32Type()),
					),
				},
			},
		},
	)
}

func TestBuiltinMethods(t *testing.T) {
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("A")},
		Underlying: ir.AtomTypeExpr(ir.Uint32Type()),
	}
	funF := &ir.FuncBuiltin{
		FType: irh.FuncType(
			nil,
			irh.Fields(typeA),
			irh.Fields(ir.Uint32Type()),
			irh.Fields(ir.Uint32Type()),
		),
	}
	typeA.Methods = []ir.PkgFunc{funF}
	testAll(t,
		irDeclTest{
			src: `
type A uint32
func (A) F(uint32) uint32
`,
			want: []ir.Node{
				typeA,
			},
		},
	)
}

func TestFuncDecl(t *testing.T) {
	aField := irh.Field("a", ir.Int32Type(), nil)
	returnTwoFunc := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "returnTwo"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields(),
			irh.Fields(ir.Float32Type()),
		),
		Body: irh.Block(
			&ir.ReturnStmt{
				Results: []ir.Expr{
					irh.IntNumberAs(2, ir.Float32Type()),
				},
			},
		)}
	testAll(t,
		irDeclTest{
			src: `
func returnTwo() float32 {
	return 2
}
`,
			want: []ir.Node{returnTwoFunc},
		},
		irDeclTest{
			src: `
func returnTwo() float32 {
	return 2
}

func call() float32 {
	return returnTwo()
}
`,
			want: []ir.Node{
				returnTwoFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(nil, nil, irh.Fields(), irh.Fields(ir.Float32Type())),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.CallExpr{
								Callee: &ir.FuncValExpr{
									X: irh.ValueRef(returnTwoFunc),
									F: returnTwoFunc,
									T: returnTwoFunc.FType,
								},
							}},
						},
					),
				},
			},
		},
		irDeclTest{
			src: `
func withArgs(a int32) int32 {
	return a
}
`,
			want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(aField),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.ValueRef(aField.Storage()),
							},
						},
					)},
			},
		},
		irDeclTest{
			src: `
func namedReturn() (a int32) {
	a = 2
	return
}
`,
			want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil, irh.Fields(),
						irh.Fields("a", ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.AssignExprStmt{List: []*ir.AssignExpr{
							{
								Storage: irh.Fields("a", ir.Int32Type()).Fields()[0].Storage(),
								X:       irh.IntNumberAs(2, ir.Int32Type()),
							},
						}},
						&ir.ReturnStmt{},
					)},
			},
		},
	)
	returnTupleFunc := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "returnTuple"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields(),
			irh.Fields(ir.Float32Type(), ir.Int32Type()),
		),
		Body: irh.Block(
			&ir.ReturnStmt{
				Results: []ir.Expr{
					irh.IntNumberAs(2, ir.Float32Type()),
					irh.IntNumberAs(3, ir.Int32Type()),
				},
			},
		)}
	testAll(t,
		irDeclTest{
			src: `
func returnTuple() (float32, int32) {
	return 2, 3
}

func call() (float32, int32) {
	return returnTuple()
}
`,
			want: []ir.Node{
				returnTupleFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Float32Type(), ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.CallExpr{
								Callee: &ir.FuncValExpr{
									X: irh.ValueRef(returnTupleFunc),
									F: returnTupleFunc,
									T: returnTupleFunc.FType,
								},
							}},
						},
					),
				},
			},
		},
	)
}

func TestCallWithLiterals(t *testing.T) {
	oneByOneLiteral := &ir.ArrayLitExpr{
		Typ: irh.ArrayType(ir.Float32Type(), 1, 1),
		Elts: []ir.AssignableExpr{
			&ir.ArrayLitExpr{
				Typ:  irh.ArrayType(ir.Float32Type(), 1),
				Elts: []ir.AssignableExpr{irh.IntNumberAs(2, ir.Float32Type())},
			},
		},
	}
	fDef := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "f"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields(oneByOneLiteral.Typ),
			irh.Fields(ir.Float32Type()),
		),
	}
	testAll(t,
		irDeclTest{
			src: `
func f([1][1]float32) float32

func call() float32 {
	return f([1][1]float32{{2}})
}
`,
			want: []ir.Node{
				fDef,
				&ir.FuncDecl{
					FType: irh.FuncType(nil, nil, irh.Fields(), irh.Fields(ir.Float32Type())),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.CallExpr{
								Callee: &ir.FuncValExpr{
									X: irh.ValueRef(fDef),
									F: fDef,
									T: fDef.FType,
								},
								Args: []ir.AssignableExpr{oneByOneLiteral},
							}},
						},
					),
				},
			},
		},
	)
}
