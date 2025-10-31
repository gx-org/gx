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
	"go/ast"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/build/ir"
)

func valRef(name string) ast.Expr {
	return &ast.Ident{Name: name}
}

func cExpr(name string, expr ast.Expr) *ir.ConstExpr {
	return &ir.ConstExpr{
		VName: &ast.Ident{Name: name},
		Val: &ir.ParenExpr{
			Src: &ast.ParenExpr{X: expr},
		},
	}
}

func newDecl(file *ir.File, exprs ...*ir.ConstExpr) *ir.ConstSpec {
	decl := &ir.ConstSpec{
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
		Consts: []*ir.ConstSpec{
			newDecl(file,
				cExpr("a", valRef("b")),
				cExpr("b", valRef("c")),
				cExpr("c", valRef("d")),
				cExpr("d", &ast.BasicLit{}),
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
