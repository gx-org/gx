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
)

func TestStructTypeZero(t *testing.T) {
	a := irhelper.Field("a", ir.Float32Type(), nil)
	blank := irhelper.Field("_", ir.IntType(), nil)
	b := irhelper.Field("b", irhelper.ArrayType(ir.Int32Type(), 2, 3), nil)
	stype := irhelper.StructType(a, blank, b)
	zero := stype.Zero()
	lit, ok := zero.(*ir.StructLitExpr)
	if !ok {
		t.Fatalf("Expected Zero() to return a %T; got %T", lit, zero)
	}
	if lit.Type() != stype {
		t.Errorf("Expected Zero().Type() to be the structure type; got %s", lit.Type().ReferString(nil))
	}
	// Blank fields are skipped and no value is assigned to the other fields.
	wantFields := []*ir.Field{a, b}
	if len(lit.Elts) != len(wantFields) {
		t.Fatalf("Expected %d field literals; got %d", len(wantFields), len(lit.Elts))
	}
	for i, elt := range lit.Elts {
		if elt.Field != wantFields[i] {
			t.Errorf("Expected field literal %d to be field %s; got %s", i, wantFields[i].Name.Name, elt.Field.Name.Name)
		}
		if elt.X != nil {
			t.Errorf("Expected field literal %d to have no value; got %s", i, elt.X.SourceString(nil))
		}
	}
}
