package ir_test

import (
	"fmt"
	"go/ast"
	"math/big"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/build/ir"
)

func valRef(name string) *ir.ValueRef {
	return &ir.ValueRef{
		Src: &ast.Ident{Name: name},
	}
}

func cExpr(name string, expr ir.AssignableExpr) *ir.ConstExpr {
	return &ir.ConstExpr{
		VName: &ast.Ident{Name: name},
		Val:   expr,
	}
}

func newDecl(file *ir.File, exprs ...*ir.ConstExpr) *ir.ConstDecl {
	decl := &ir.ConstDecl{
		FFile: file,
		Exprs: exprs,
	}
	for _, expr := range exprs {
		expr.Decl = decl
	}
	return decl
}

func TestConstsDeps(t *testing.T) {
	pkg := &ir.Package{}
	file := &ir.File{Package: pkg}
	pkg.Decls = &ir.Declarations{
		Consts: []*ir.ConstDecl{
			newDecl(file,
				cExpr("a", valRef("b")),
				cExpr("b", valRef("c")),
				cExpr("c", valRef("d")),
				cExpr("d", &ir.NumberFloat{}),
			),
		},
	}
	var got []string
	exprs, err := pkg.Decls.ConstExprs()
	if err != nil {
		t.Fatal(err)
	}
	for _, expr := range exprs {
		got = append(got, expr.VName.Name)
	}
	want := []string{"d", "c", "b", "a"}
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect const definition ordered: got %v but want %v", got, want)
	}
}

func TestConstsOrdering(t *testing.T) {
	tests := []struct {
		expr *ir.ConstExpr
		want []string
	}{
		{
			expr: cExpr("valref", valRef("dep")),
			want: []string{"dep"},
		},
		{
			expr: cExpr("numFloat", &ir.NumberFloat{}),
		},
		{
			expr: cExpr("numInt", &ir.NumberInt{
				Val: big.NewInt(0),
			}),
		},
		{
			expr: cExpr("atomic", &ir.AtomicValueT[float64]{}),
		},
		{
			expr: cExpr("cast", &ir.NumberCastExpr{
				X: valRef("dep"),
			}),
			want: []string{"dep"},
		},
		{
			expr: cExpr("unary", &ir.UnaryExpr{
				X: valRef("dep"),
			}),
			want: []string{"dep"},
		},
		{
			expr: cExpr("binary", &ir.BinaryExpr{
				X: valRef("depX"),
				Y: valRef("depY"),
			}),
			want: []string{"depX", "depY"},
		},
		{
			expr: cExpr("parenthesis", &ir.ParenExpr{
				X: &ir.BinaryExpr{
					X: valRef("depX"),
					Y: valRef("depY"),
				},
			}),
			want: []string{"depX", "depY"},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("test%d", i), func(t *testing.T) {
			deps, err := test.expr.Deps()
			if err != nil {
				t.Errorf("test %d: cannot get the dependencies of expression %s:\n%+v", i, test.expr.String(), err)
				return
			}
			got := names(deps)
			if len(got) == 0 && len(test.want) == 0 {
				return
			}
			if !cmp.Equal(got, test.want) {
				t.Errorf("test %d: expression %s has incorrect dependencies: got %v but want %v", i, test.expr.String(), got, test.want)
			}
		})
	}
}
