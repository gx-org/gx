package builder_test

import (
	"go/ast"
	"math/big"
	"testing"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
)

func TestImportType(t *testing.T) {
	typInt := &ir.NamedType{
		File:       &ir.File{Package: &ir.Package{Name: irhelper.Ident("dtype")}},
		Src:        &ast.TypeSpec{Name: irhelper.Ident("Int")},
		Underlying: irhelper.TypeExpr(ir.Int32Type()),
	}
	testAll(t,
		declarePackage{src: `
package dtype

type Int int32
`},
		irDeclTest{
			src: `
import "dtype"

func bla() dtype.Int
`,
			want: []ir.Node{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(typInt),
					),
				},
			},
		},
	)
}

func TestImportConst(t *testing.T) {
	pkgImportDecl := &ir.ImportDecl{
		Src:  &ast.ImportSpec{Name: irhelper.Ident("pkg")},
		Path: "pkg",
	}
	constExpr := &ir.ConstExpr{
		Decl:  &ir.ConstDecl{},
		VName: irhelper.Ident("MyConst"),
		Val:   &ir.NumberInt{Val: big.NewInt(42)},
	}
	constExpr.Decl.Exprs = []*ir.ConstExpr{constExpr}
	testAll(t,
		declarePackage{src: `
package pkg

const MyConst = 42

func F(int32) int32
`},
		irDeclTest{
			src: `
import "pkg"

func returnMyConst() int32 {
	return pkg.MyConst
}
`,
			want: []ir.Node{
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(ir.Int32Type()),
					),
					Body: irhelper.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								&ir.NumberCastExpr{
									X: &ir.SelectorExpr{
										X:    irhelper.ValueRef(pkgImportDecl),
										Stor: constExpr,
									},
									Typ: ir.Int32Type(),
								},
							},
						},
					)},
			},
		},
		irDeclTest{
			src: `
import "pkg"

func f(a int32) int32 {
	return pkg.F(a)
}
`,
			want: []ir.Node{
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(ir.Int32Type()),
					),
					Body: irhelper.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								&ir.NumberCastExpr{
									X: &ir.SelectorExpr{
										X:    irhelper.ValueRef(pkgImportDecl),
										Stor: constExpr,
									},
									Typ: ir.Int32Type(),
								},
							},
						},
					)},
			},
		},
	)
}
