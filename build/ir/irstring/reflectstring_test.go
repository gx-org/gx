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

package irstring_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/build/ir/irstring"
)

func TestReflectString(t *testing.T) {
	done := map[any]bool{}
	a := irhelper.Field("a", ir.Float32Type(), nil).Storage()
	b := irhelper.Field("b", ir.Float32Type(), a.Field.Group).Storage()
	tests := []struct {
		code ir.IR
		want string
	}{
		{
			code: &ir.FuncDecl{
				FType: &ir.FuncType{
					Params:  irhelper.Fields(a.Field),
					Results: irhelper.Fields(ir.Float32Type()),
				},
				Body: &ir.BlockStmt{List: []ir.Stmt{
					&ir.ReturnStmt{
						Results: []ir.Expr{
							&ir.BinaryExpr{
								X:   irhelper.Ident(a),
								Y:   irhelper.Ident(b),
								Typ: ir.Float32Type(),
							},
						},
					},
				},
				},
			},
			want: `
FuncDecl {
	FType: FuncType {
		Params: FieldList{
			a,b float32
		}
		Results: FieldList{
			float32
		}
	}
	Body: BlockStmt {
		List: [
			ReturnStmt {
				Results: [
					BinaryExpr {
						X: a->FieldStorage[float32]
						Y: b->FieldStorage[float32]
						Typ: float32
					}
				]
			}
		]
	}
}`,
		},
		{
			code: &ir.NamedType{
				Underlying: ir.TypeExpr(nil, irhelper.StructType(
					irhelper.Fields("a", "b", ir.Float32Type()).List[0].Fields...,
				)),
			},
			want: `
NamedType {
	Underlying: StructType {
		BaseType: BaseType[*go/ast.StructType] {
		}
		Fields: FieldList{
			a,b float32
		}
	}
}
`,
		},
		{
			code: irhelper.IntNumberAs(2, ir.IntLenType()),
			want: `
NumberCastExpr {
	X: NumberInt {
		Val: 2
	}
	Typ: intlen
}`,
		},
	}
	for i, test := range tests {
		got := irstring.ReflectString(done, test.code)
		want := strings.TrimSpace(test.want)
		if got != want {
			t.Errorf("test %d: got:\n%s\nwant:\n%s\ndiff:\n%s", i, got, want, cmp.Diff(got, want))
		}
	}
}
