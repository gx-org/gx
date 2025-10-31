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

package exprdeps_test

import (
	"go/ast"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/internal/exprdeps"
)

func names(vals []*ast.Ident) []string {
	ss := make([]string, len(vals))
	for i, val := range vals {
		ss[i] = val.Name
	}
	return ss
}

func TestIdents(t *testing.T) {
	xVar := &ast.Ident{Name: "x"}
	yVar := &ast.Ident{Name: "y"}
	tests := []struct {
		expr ast.Expr
		want []string
	}{
		{
			expr: xVar,
			want: []string{"x"},
		},
		{
			expr: &ast.BinaryExpr{
				X: xVar,
				Y: yVar,
			},
			want: []string{"x", "y"},
		},
		{
			expr: &ast.BinaryExpr{
				X: xVar,
				Y: xVar,
			},
			want: []string{"x"},
		},
	}
	for i, test := range tests {
		refs := exprdeps.Idents(test.expr)
		got := names(refs)
		if !cmp.Equal(got, test.want) {
			t.Errorf("test %d: incorrect identifier list: got %v but want %v", i, got, test.want)
		}
	}
}
