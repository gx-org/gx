package elements_test

import (
	"slices"
	"testing"

	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/interp/elements"
)

func mustAtomInt(t *testing.T, v int32) hostio.Value {
	t.Helper()
	val, err := hostio.AtomIntegerValue[int32](ir.Int32Type(), v)
	if err != nil {
		t.Fatal(err)
	}
	return val
}

func mustSlice(t *testing.T, els ...hostio.Value) *elements.Slice {
	t.Helper()
	val, err := elements.NewSlice(ir.IntSliceType(), hostio.ToElements(els))
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
			got, err := elements.Flatten(test.in)
			if err != nil {
				t.Fatalf("elements.Flatten(%v) returned an error: %v", test.in, err)
			}
			if !slices.Equal(got, test.want) {
				t.Errorf("elements.Flatten(%v) = %v, want %v", test.in, got, test.want)
			}
		})
	}
}

func TestFlattenNotAValue(t *testing.T) {
	notAValue, err := elements.NewString("gx", ir.StringType())
	if err != nil {
		t.Fatal(err)
	}
	slice, err := elements.NewSlice(ir.IntSliceType(), []ir.Element{notAValue})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := elements.Flatten([]hostio.Value{slice}); err == nil {
		t.Errorf("elements.Flatten(%v) = %v, want an error", slice, got)
	}
}

func TestSliceValue(t *testing.T) {
	v1, v2 := mustAtomInt(t, 1), mustAtomInt(t, 2)
	slice := mustSlice(t, v1, mustSlice(t, v2))
	if got, want := slice.SourceString(nil), "[int32(1),[int32(2)]]"; got != want {
		t.Errorf("slice.SourceString(nil) = %q, want %q", got, want)
	}
	host, err := slice.ToHost(kernels.DefaultAllocator())
	if err != nil {
		t.Fatalf("slice.ToHost() returned an error: %v", err)
	}
	if got, want := host.SourceString(nil), slice.SourceString(nil); got != want {
		t.Errorf("slice.ToHost().SourceString(nil) = %q, want %q", got, want)
	}
	if err := slice.Finalize(); err != nil {
		t.Errorf("slice.Finalize() returned an error: %v", err)
	}
}
