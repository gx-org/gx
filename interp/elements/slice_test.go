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
	if got, want := slice.DefineString(nil), "[]int{1, {2}}"; got != want {
		t.Errorf("slice.DefineString(nil) = %q, want %q", got, want)
	}
	host, err := slice.ToHost(kernels.DefaultAllocator())
	if err != nil {
		t.Fatalf("slice.ToHost() returned an error: %v", err)
	}
	if got, want := ir.DefineString(nil, host), slice.DefineString(nil); got != want {
		t.Errorf("slice.ToHost().DefineString(nil) = %q, want %q", got, want)
	}
	if err := slice.Finalize(); err != nil {
		t.Errorf("slice.Finalize() returned an error: %v", err)
	}
}

func TestSliceString(t *testing.T) {
	v1, v2, v3 := mustAtomInt(t, 1), mustAtomInt(t, 2), mustAtomInt(t, 3)
	v4, v5, v6 := mustAtomInt(t, 4), mustAtomInt(t, 5), mustAtomInt(t, 6)

	slice1DType := &ir.SliceType{DType: ir.TypeExpr(nil, ir.Int32Type()), Rank: 1}
	slice2DType := &ir.SliceType{DType: ir.TypeExpr(nil, ir.Int32Type()), Rank: 2}

	s1D, err := elements.NewSlice(slice1DType, []ir.Element{v1, v2, v3})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := s1D.String(), "[]int32{1, 2, 3}"; got != want {
		t.Errorf("s1D.String() = %q, want %q", got, want)
	}

	sEmpty, err := elements.NewSlice(slice1DType, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sEmpty.String(), "[]int32{}"; got != want {
		t.Errorf("sEmpty.String() = %q, want %q", got, want)
	}

	sub1, err := elements.NewSlice(slice1DType, []ir.Element{v1, v2, v3})
	if err != nil {
		t.Fatal(err)
	}
	sub2, err := elements.NewSlice(slice1DType, []ir.Element{v4})
	if err != nil {
		t.Fatal(err)
	}
	sub3, err := elements.NewSlice(slice1DType, []ir.Element{v5, v6})
	if err != nil {
		t.Fatal(err)
	}
	s2D, err := elements.NewSlice(slice2DType, []ir.Element{sub1, sub2, sub3})
	if err != nil {
		t.Fatal(err)
	}
	want2D := "[][]int32{\n\t{1, 2, 3},\n\t{4},\n\t{5, 6},\n}"
	if got := s2D.String(); got != want2D {
		t.Errorf("s2D.String() =\n%s\nwant:\n%s", got, want2D)
	}

	str1, err := values.NewString(ir.StringType(), "Hello")
	if err != nil {
		t.Fatal(err)
	}
	str2, err := values.NewString(ir.StringType(), "world")
	if err != nil {
		t.Fatal(err)
	}
	sliceStrType := &ir.SliceType{DType: ir.TypeExpr(nil, ir.StringType()), Rank: 1}
	sStr, err := elements.NewSlice(sliceStrType, []ir.Element{str1, str2})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sStr.String(), `[]string{"Hello", "world"}`; got != want {
		t.Errorf("sStr.String() = %q, want %q", got, want)
	}
}
