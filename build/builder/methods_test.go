package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestMethods(t *testing.T) {
	fieldI := irh.Field("i", ir.Int32Type(), nil)
	fieldF := irh.Field("f", ir.Float32Type(), nil)
	structA := irh.StructType(fieldI, fieldF)
	typeA := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: &ast.Ident{Name: "A"}},
		File:       wantFile,
		Underlying: irh.TypeExpr(structA),
	}
	aRecv := irh.Fields("a", typeA)
	fI := &ir.FuncDecl{
		FType: irh.FuncType(nil,
			aRecv,
			irh.Fields(),
			irh.Fields(ir.Int32Type()),
		),
		Body: irh.Block(
			&ir.ReturnStmt{
				Results: []ir.Expr{&ir.SelectorExpr{
					X:    irh.ValueRef(aRecv.Fields()[0].Storage()),
					Stor: fieldI.Storage(),
				}},
			},
		)}
	typeA.Methods = []ir.PkgFunc{fI}
	testAll(t,
		irDeclTest{
			src: `
type A struct {
	i int32
	f float32
}

func (a A) fI() int32 {
	return a.i
}
`,
			want: []ir.Node{typeA},
		},
		irDeclTest{
			src: `
type A struct {
	i int32
	f float32
}

func (a A) fI() int32 {
	return a.i
}

func call() int32 {
	a := A{i:2,f:3}
	return a.fI()
}
`,
			want: []ir.Node{
				typeA,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.AssignExprStmt{List: []*ir.AssignExpr{{
							X: &ir.StructLitExpr{
								Elts: []*ir.FieldLit{
									irh.FieldLit(structA.Fields, "i", irh.IntNumberAs(2, ir.Int32Type())),
									irh.FieldLit(structA.Fields, "f", irh.IntNumberAs(3, ir.Float32Type())),
								},
								Typ: typeA,
							},
							Storage: &ir.LocalVarStorage{Src: irh.Ident("a"), Typ: typeA},
						}}},
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.CallExpr{
								Callee: &ir.FuncValExpr{
									X: &ir.SelectorExpr{
										X:    irh.ValueRef(typeA),
										Stor: fI,
									},
									F: fI,
									T: fI.FuncType(),
								},
							}},
						},
					)},
			},
		},
	)
}

func TestMethodOnNamedTypes(t *testing.T) {
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("A")},
		Underlying: ir.AtomTypeExpr(ir.Int32Type()),
	}
	aRecv := irh.Fields("a", typeA)
	val := &ir.FuncDecl{
		FType: irh.FuncType(nil,
			aRecv,
			irh.Fields(),
			irh.Fields(ir.Int32Type()),
		),
		Body: irh.Block(
			&ir.ReturnStmt{
				Results: []ir.Expr{&ir.CastExpr{
					Typ: ir.Int32Type(),
					X:   irh.ValueRef(aRecv.Fields()[0].Storage()),
				}},
			},
		)}
	typeA.Methods = []ir.PkgFunc{val}
	testAll(t,

		irDeclTest{
			src: `
type A int32

func (a A) val() int32 {
	return int32(a)
}

func call() int32 {
	a := A(2)
	return a.val()
}
`,
			want: []ir.Node{
				typeA,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.AssignExprStmt{List: []*ir.AssignExpr{{
							X: &ir.CastExpr{
								X:   irh.IntNumberAs(2, typeA),
								Typ: typeA,
							},
							Storage: &ir.LocalVarStorage{Src: irh.Ident("a"), Typ: typeA},
						}}},
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.CallExpr{
								Callee: &ir.FuncValExpr{
									X: &ir.SelectorExpr{
										X:    irh.ValueRef(typeA),
										Stor: val,
									},
									F: val,
									T: val.FuncType(),
								},
							}},
						},
					)},
			},
		},
	)
}
