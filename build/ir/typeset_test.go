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
	"fmt"
	"testing"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
)

func TestTypeSetSlice(t *testing.T) {
	tests := []struct {
		typeSet *ir.TypeSet
		want    ir.Type
		ok      bool
	}{
		{
			typeSet: irhelper.TypeSet(
				irhelper.ArrayType(ir.Float32Type(), 2, 3, 4),
				irhelper.ArrayType(ir.Float32Type(), 5, 6, 7),
			),
			want: irhelper.TypeSet(
				irhelper.ArrayType(ir.Float32Type(), 3, 4),
				irhelper.ArrayType(ir.Float32Type(), 6, 7),
			),
			ok: true,
		},
		{
			typeSet: irhelper.TypeSet(ir.Int32Type()),
			want:    irhelper.TypeSet(),
			ok:      false,
		},
	}
	fetcher, err := newFetcherTesting()
	if err != nil {
		t.Fatal(err)
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("test%d", i), func(t *testing.T) {
			got, gotOk := test.typeSet.ElementType()
			if gotOk != test.ok {
				t.Errorf("unexpected ok value: got %v but want %v", gotOk, test.ok)
			}
			eq, err := test.want.Equal(fetcher, got)
			if err != nil {
				t.Error(err)
				return
			}
			if !eq {
				t.Errorf("unexpected type: got %s but want %s", got.String(), test.want.String())
			}
		})
	}
}
