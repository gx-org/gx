// Copyright 2026 Google LLC
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

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/src/gx/ast"
)

func newNamedType(name string, typ ir.Type) *ir.NamedType {
	return &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.IdentAST(name)},
		File:       testFile,
		Underlying: ir.TypeExpr(nil, typ),
	}
}

func TestNamedTypeStructZero(t *testing.T) {
	a := irhelper.Field("a", ir.Float32Type(), nil)
	stype := irhelper.StructType(a)
	for _, named := range []*ir.NamedType{
		newNamedType("S", stype),
		// Named type defined from another named type.
		newNamedType("T", newNamedType("S", stype)),
	} {
		zero := named.Zero()
		lit, ok := zero.(*ir.StructLitExpr)
		if !ok {
			t.Fatalf("got %T, but want %s.Zero() to return %T", zero, named.Name(), lit)
		}
		if lit.Type() != named {
			t.Errorf("got %s, but want %s.Zero().Type() to be %s", lit.Type().ReferString(nil), named.Name(), named.Name())
		}
		elt := lit.Elts[0]
		if elt.Field != a {
			t.Errorf("got field %s, but want %s.Zero() to have a field literal for field a", elt.Field.Name.Name, named.Name())
		}
		if elt.X != nil {
			t.Errorf("got value %s, but want %s.Zero() to have a field literal with no value", elt.X.SourceString(nil), named.Name())
		}
	}
}

func TestNamedTypeArrayZero(t *testing.T) {
	for _, testCase := range []struct {
		named *ir.NamedType
		want  string
	}{
		{newNamedType("A", ir.Float64Type()), "0"},
		{newNamedType("A", irhelper.ArrayType(ir.Int32Type(), 2, 3)), "ir_test.A([2][3]int32(0))"},
		// Named type defined from another named type.
		{newNamedType("B", newNamedType("A", ir.Float64Type())), "0"},
	} {
		zero := testCase.named.Zero()
		if zero == nil {
			t.Fatalf("got nil, but want %s.Zero() to return an expression", testCase.named.Name())
		}
		if zero.Type() != testCase.named {
			t.Errorf("got %s, but want %s.Zero().Type() to be %s", zero.Type().ReferString(nil), testCase.named.Name(), testCase.named.Name())
		}
		if got := zero.SourceString(nil); got != testCase.want {
			t.Errorf("got %s, but want %s.Zero() = %s", got, testCase.named.Name(), testCase.want)
		}
	}
}

func TestNamedTypeUnsupportedZero(t *testing.T) {
	named := newNamedType("Ints", irhelper.SliceType(ir.TypeExpr(nil, ir.IntType()), 1))
	if zero := named.Zero(); zero != nil {
		t.Errorf("got %s, but want %s.Zero() = nil", zero.SourceString(nil), named.Name())
	}
}
