package values_test

import (
	"slices"
	"testing"

	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

func mustAtomInt(t *testing.T, v int32) hostio.Value {
	t.Helper()
	val, err := hostio.AtomIntegerValue[int32](ir.Int32Type(), v)
	if err != nil {
		t.Fatal(err)
	}
	return val
}

func mustSlice(t *testing.T, els ...hostio.Value) *values.Slice {
	t.Helper()
	val, err := values.NewSlice(ir.IntSliceType(), els)
	if err != nil {
		t.Fatal(err)
	}
	return val
}

func TestFlatten(t *testing.T) {
	v1, v2, v3 := mustAtomInt(t, 1), mustAtomInt(t, 2), mustAtomInt(t, 3)
	s1, err := values.NewString(ir.StringType(), "gx")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		in   []hostio.Value
		want []hostio.Value
	}{
		{name: "empty input", in: nil, want: nil},
		{name: "non-slice values", in: []hostio.Value{v1, v2, s1}, want: []hostio.Value{v1, v2, s1}},
		{name: "empty slices", in: []hostio.Value{mustSlice(t), mustSlice(t, mustSlice(t))}, want: nil},
		{name: "empty slice mixed with values", in: []hostio.Value{v1, mustSlice(t), v2}, want: []hostio.Value{v1, v2}},
		{name: "flat slice", in: []hostio.Value{mustSlice(t, v1, v2)}, want: []hostio.Value{v1, v2}},
		{name: "nested slices", in: []hostio.Value{v1, mustSlice(t, v2, mustSlice(t, v3, mustSlice(t)), s1)}, want: []hostio.Value{v1, v2, v3, s1}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := values.Flatten(test.in); !slices.Equal(got, test.want) {
				t.Errorf("values.Flatten(%v) = %v, want %v", test.in, got, test.want)
			}
		})
	}
}
