package ir_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
)

func TestIdents(t *testing.T) {
	xVar := irhelper.LocalVar("x", ir.Int32Type())
	yVar := irhelper.LocalVar("y", ir.Int32Type())
	tests := []struct {
		expr ir.Expr
		want []string
	}{
		{
			expr: irhelper.ValueRef(xVar),
			want: []string{"x"},
		},
		{
			expr: &ir.BinaryExpr{
				X: irhelper.ValueRef(xVar),
				Y: irhelper.ValueRef(yVar),
			},
			want: []string{"x", "y"},
		},
		{
			expr: &ir.BinaryExpr{
				X: irhelper.ValueRef(xVar),
				Y: irhelper.ValueRef(xVar),
			},
			want: []string{"x"},
		},
	}
	for i, test := range tests {
		refs, err := ir.Idents(test.expr)
		if err != nil {
			t.Error(err)
			continue
		}
		got := names(refs)
		if !cmp.Equal(got, test.want) {
			t.Errorf("test %d: incorrect identifier list: got %v but want %v", i, got, test.want)
		}
	}
}
