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
