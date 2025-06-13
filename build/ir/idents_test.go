// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
