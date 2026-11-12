package hostio_test

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/gomlx/compute/dtypes/bfloat16"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gomlx/compute/gobackend"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/fmt/fmtarray"
	"github.com/gx-org/gx/src/gx/ast"
)

func mustHostArray[T gotype.Supported](t *testing.T, dtype ir.Type, flat []T, dims ...int) *hostio.HostArray {
	t.Helper()
	var typ ir.Type = dtype
	if len(dims) > 0 {
		axes := make([]any, len(dims))
		for i, d := range dims {
			axes[i] = d
		}
		typ = irhelper.ArrayType(dtype, axes...)
	}
	arr, err := hostio.NewHostArrayFromFlat(typ, flat, dims...)
	if err != nil {
		t.Fatalf("NewHostArrayFromFlat(%v, %v, %v) failed: %v", typ, flat, dims, err)
	}
	return arr
}

func testDTypeCase[T gotype.Supported](
	t *testing.T,
	dtype ir.Type,
	scalarVal T,
	vecVals []T,
	matVals []T,
	wantFloat *big.Float,
) {
	t.Helper()
	t.Run(dtype.Kind().String(), func(t *testing.T) {
		scalar := mustHostArray(t, dtype, []T{scalarVal})
		vec := mustHostArray(t, dtype, vecVals, len(vecVals))
		mat := mustHostArray(t, dtype, matVals, 2, 2)

		wantUntypedAtom := any(scalarVal)
		if v, ok := wantUntypedAtom.(ir.Int); ok {
			wantUntypedAtom = int64(v)
		}

		gotAtom, err := scalar.ToAtom()
		if err != nil {
			t.Fatalf("scalar.ToAtom() unexpected error: %v", err)
		}
		if gotAtom != wantUntypedAtom {
			t.Errorf("scalar.ToAtom() = %v (%T), want %v (%T)", gotAtom, gotAtom, wantUntypedAtom, wantUntypedAtom)
		}
		if _, err := vec.ToAtom(); err == nil || !strings.Contains(err.Error(), "not atomic") {
			t.Errorf("vec.ToAtom() error = %v, want 'not atomic' error", err)
		}

		gotTypedAtom, err := hostio.ToAtom[T](scalar)
		if err != nil {
			t.Fatalf("hostio.ToAtom[%T](scalar) unexpected error: %v", scalarVal, err)
		}
		if gotTypedAtom != scalarVal {
			t.Errorf("hostio.ToAtom[%T](scalar) = %v, want %v", scalarVal, gotTypedAtom, scalarVal)
		}
		if _, err := hostio.ToAtom[T](vec); err == nil || !strings.Contains(err.Error(), "is not an atom") {
			t.Errorf("hostio.ToAtom[%T](vec) error = %v, want 'is not an atom' error", scalarVal, err)
		}

		gotVecFlat, err := hostio.Flat[T](vec)
		if err != nil {
			t.Fatalf("hostio.Flat[%T](vec) unexpected error: %v", scalarVal, err)
		}
		if !slices.Equal(gotVecFlat, vecVals) {
			t.Errorf("hostio.Flat[%T](vec) = %v, want %v", scalarVal, gotVecFlat, vecVals)
		}

		gotFloat, err := scalar.ToFloatNumber()
		if wantFloat == nil {
			if err == nil || !strings.Contains(err.Error(), "cannot convert non-algebra type") {
				t.Errorf("scalar.ToFloatNumber() = (%v, %v), want 'cannot convert non-algebra type' error", gotFloat, err)
			}
		} else {
			if err != nil {
				t.Fatalf("scalar.ToFloatNumber() unexpected error: %v", err)
			}
			if gotFloat.Cmp(wantFloat) != 0 {
				t.Errorf("scalar.ToFloatNumber() = %v, want %v", gotFloat, wantFloat)
			}
		}
		if _, err := vec.ToFloatNumber(); err == nil {
			t.Errorf("vec.ToFloatNumber() succeeded, want error for non-scalar array")
		}

		wantScalarSource := fmtarray.SDataPrint([]T{scalarVal}, nil)
		if got := scalar.SourceString(nil); got != wantScalarSource {
			t.Errorf("scalar.SourceString(nil) = %q, want %q", got, wantScalarSource)
		}
		wantScalarDefine := fmt.Sprintf("%s(%s)", dtype.ReferString(nil), wantScalarSource)
		if got := scalar.DefineString(nil); got != wantScalarDefine {
			t.Errorf("scalar.DefineString(nil) = %q, want %q", got, wantScalarDefine)
		}

		wantVecSource := fmtarray.SDataPrint(vecVals, []int{len(vecVals)})
		if got := vec.SourceString(nil); got != wantVecSource {
			t.Errorf("vec.SourceString(nil) = %q, want %q", got, wantVecSource)
		}
		wantVecDefine := fmt.Sprintf("[%d]%s%s", len(vecVals), dtype.ReferString(nil), wantVecSource)
		if got := vec.DefineString(nil); got != wantVecDefine {
			t.Errorf("vec.DefineString(nil) = %q, want %q", got, wantVecDefine)
		}

		wantMatSource := fmtarray.SDataPrint(matVals, []int{2, 2})
		if got := mat.SourceString(nil); got != wantMatSource {
			t.Errorf("mat.SourceString(nil) = %q, want %q", got, wantMatSource)
		}
		wantMatDefine := fmt.Sprintf("[2][2]%s%s", dtype.ReferString(nil), wantMatSource)
		if got := mat.DefineString(nil); got != wantMatDefine {
			t.Errorf("mat.DefineString(nil) = %q, want %q", got, wantMatDefine)
		}

		gotScalarGoVal, err := scalar.GoValue()
		if err != nil {
			t.Fatalf("scalar.GoValue() unexpected error: %v", err)
		}
		if gotScalarGoVal != wantUntypedAtom {
			t.Errorf("scalar.GoValue() = %v (%T), want %v (%T)", gotScalarGoVal, gotScalarGoVal, wantUntypedAtom, wantUntypedAtom)
		}

		gotVecGoVal, err := vec.GoValue()
		if err != nil {
			t.Fatalf("vec.GoValue() unexpected error: %v", err)
		}
		wantVecSprint := fmtarray.Sprint(vecVals, []int{len(vecVals)})
		if got := fmt.Sprint(gotVecGoVal); got != wantVecSprint {
			t.Errorf("fmt.Sprint(vec.GoValue()) = %q, want %q", got, wantVecSprint)
		}

		gotMatGoVal, err := mat.GoValue()
		if err != nil {
			t.Fatalf("mat.GoValue() unexpected error: %v", err)
		}
		wantMatSprint := fmtarray.Sprint(matVals, []int{2, 2})
		if got := fmt.Sprint(gotMatGoVal); got != wantMatSprint {
			t.Errorf("fmt.Sprint(mat.GoValue()) = %q, want %q", got, wantMatSprint)
		}

		bck, err := gobackend.New("")
		if err != nil {
			t.Fatalf("gobackend.New failed: %v", err)
		}
		devScalar, err := scalar.ToDevice(bck, 0)
		if err != nil {
			t.Fatalf("scalar.ToDevice failed: %v", err)
		}
		defer devScalar.Finalize()
		gotDevScalarGoVal, err := devScalar.GoValue()
		if err != nil {
			t.Fatalf("devScalar.GoValue() unexpected error: %v", err)
		}
		if gotDevScalarGoVal != wantUntypedAtom {
			t.Errorf("devScalar.GoValue() = %v (%T), want %v (%T)", gotDevScalarGoVal, gotDevScalarGoVal, wantUntypedAtom, wantUntypedAtom)
		}
		roundTripScalar, err := devScalar.ToHostArray()
		if err != nil {
			t.Fatalf("devScalar.ToHostArray() unexpected error: %v", err)
		}
		gotRoundTripAtom, err := hostio.ToAtom[T](roundTripScalar)
		if err != nil {
			t.Fatalf("hostio.ToAtom[%T](roundTripScalar) unexpected error: %v", scalarVal, err)
		}
		if gotRoundTripAtom != scalarVal {
			t.Errorf("hostio.ToAtom[%T](roundTripScalar) = %v, want %v", scalarVal, gotRoundTripAtom, scalarVal)
		}

		devVec, err := vec.ToDevice(bck, 0)
		if err != nil {
			t.Fatalf("vec.ToDevice failed: %v", err)
		}
		defer devVec.Finalize()
		gotDevVecGoVal, err := devVec.GoValue()
		if err != nil {
			t.Fatalf("devVec.GoValue() unexpected error: %v", err)
		}
		if got := fmt.Sprint(gotDevVecGoVal); got != wantVecSprint {
			t.Errorf("fmt.Sprint(devVec.GoValue()) = %q, want %q", got, wantVecSprint)
		}
		roundTripVec, err := devVec.ToHostArray()
		if err != nil {
			t.Fatalf("devVec.ToHostArray() unexpected error: %v", err)
		}
		gotRoundTripVecFlat, err := hostio.Flat[T](roundTripVec)
		if err != nil {
			t.Fatalf("hostio.Flat[%T](roundTripVec) unexpected error: %v", scalarVal, err)
		}
		if !slices.Equal(gotRoundTripVecFlat, vecVals) {
			t.Errorf("hostio.Flat[%T](roundTripVec) = %v, want %v", scalarVal, gotRoundTripVecFlat, vecVals)
		}

		devMat, err := mat.ToDevice(bck, 0)
		if err != nil {
			t.Fatalf("mat.ToDevice failed: %v", err)
		}
		defer devMat.Finalize()
		roundTripMat, err := devMat.ToHostArray()
		if err != nil {
			t.Fatalf("devMat.ToHostArray() unexpected error: %v", err)
		}
		gotRoundTripMatFlat, err := hostio.Flat[T](roundTripMat)
		if err != nil {
			t.Fatalf("hostio.Flat[%T](roundTripMat) unexpected error: %v", scalarVal, err)
		}
		if !slices.Equal(gotRoundTripMatFlat, matVals) {
			t.Errorf("hostio.Flat[%T](roundTripMat) = %v, want %v", scalarVal, gotRoundTripMatFlat, matVals)
		}
	})
}

func TestHostArrayAllDTypes(t *testing.T) {
	testDTypeCase(t, ir.BoolType(), true, []bool{true, false, true}, []bool{true, false, false, true}, nil)
	bfScalar := bfloat16.FromFloat32(1.5)
	bfVec := []bfloat16.BFloat16{bfloat16.FromFloat32(1.0), bfloat16.FromFloat32(2.5), bfloat16.FromFloat32(-0.5)}
	bfMat := []bfloat16.BFloat16{bfloat16.FromFloat32(1.0), bfloat16.FromFloat32(2.0), bfloat16.FromFloat32(3.0), bfloat16.FromFloat32(4.0)}
	testDTypeCase(t, ir.Bfloat16Type(), bfScalar, bfVec, bfMat, big.NewFloat(float64(bfScalar.Float32())))
	testDTypeCase(t, ir.Float32Type(), float32(2.5), []float32{1.25, -2.5, 3.0}, []float32{1, 2, 3, 4}, big.NewFloat(2.5))
	testDTypeCase(t, ir.Float64Type(), float64(-3.75), []float64{1.5, 2.25, -4.0}, []float64{1, 2, 3, 4}, big.NewFloat(-3.75))
	testDTypeCase(t, ir.IntType(), ir.Int(-123), []ir.Int{-10, 0, 20}, []ir.Int{1, 2, 3, 4}, big.NewFloat(-123))
	testDTypeCase(t, ir.Int32Type(), int32(-42), []int32{-1, 0, 7}, []int32{1, 2, 3, 4}, big.NewFloat(-42))
	largeInt64 := int64((1 << 53) + 1)
	testDTypeCase(t, ir.Int64Type(), largeInt64, []int64{10, -20, 30}, []int64{1, 2, 3, 4}, new(big.Float).SetInt64(largeInt64))
	testDTypeCase(t, ir.Uint32Type(), uint32(77), []uint32{1, 2, 3}, []uint32{1, 2, 3, 4}, big.NewFloat(77))
	largeUint64 := uint64((1 << 53) + 1)
	testDTypeCase(t, ir.Uint64Type(), largeUint64, []uint64{100, 200, 300}, []uint64{1, 2, 3, 4}, new(big.Float).SetUint64(largeUint64))
}

func TestAtomConstructors(t *testing.T) {
	bVal, err := hostio.AtomBoolValue(ir.BoolType(), true)
	if err != nil {
		t.Fatalf("AtomBoolValue failed: %v", err)
	}
	if got, err := hostio.ToAtom[bool](bVal); err != nil || !got {
		t.Errorf("ToAtom[bool](bVal) = (%v, %v), want (true, nil)", got, err)
	}

	fVal, err := hostio.AtomFloatValue[float32](ir.Float32Type(), 3.5)
	if err != nil {
		t.Fatalf("AtomFloatValue failed: %v", err)
	}
	if got, err := hostio.ToAtom[float32](fVal); err != nil || got != 3.5 {
		t.Errorf("ToAtom[float32](fVal) = (%v, %v), want (3.5, nil)", got, err)
	}

	iVal, err := hostio.AtomIntegerValue[ir.Int](ir.IntType(), -99)
	if err != nil {
		t.Fatalf("AtomIntegerValue failed: %v", err)
	}
	if got, err := hostio.ToAtom[ir.Int](iVal); err != nil || got != -99 {
		t.Errorf("ToAtom[ir.Int](iVal) = (%v, %v), want (-99, nil)", got, err)
	}

	fromAtomVal, err := hostio.FromAtom(int(42), ir.IntType())
	if err != nil {
		t.Fatalf("FromAtom failed: %v", err)
	}
	if got, err := hostio.ToAtom[int](fromAtomVal); err != nil || got != 42 {
		t.Errorf("ToAtom[int](fromAtomVal) = (%v, %v), want (42, nil)", got, err)
	}
}

func TestToAtomAndFlatShapeAndDTypeChecks(t *testing.T) {
	vec1 := mustHostArray(t, ir.Int32Type(), []int32{42}, 1)
	if _, err := hostio.ToAtom[int32](vec1); err == nil || !strings.Contains(err.Error(), "is not an atom") {
		t.Errorf("hostio.ToAtom[int32]([1]int32) error = %v, want 'is not an atom' error", err)
	}
	f32Scalar := mustHostArray(t, ir.Float32Type(), []float32{1.5})
	if _, err := hostio.ToAtom[int32](f32Scalar); err == nil || !strings.Contains(err.Error(), "array data type") {
		t.Errorf("hostio.ToAtom[int32](float32 scalar) error = %v, want 'array data type' error", err)
	}
	if _, err := hostio.Flat[int32](f32Scalar); err == nil || !strings.Contains(err.Error(), "array data type") {
		t.Errorf("hostio.Flat[int32](float32 scalar) error = %v, want 'array data type' error", err)
	}
	boolVec4 := mustHostArray(t, ir.BoolType(), []bool{true, false, true, false}, 4)
	if _, err := hostio.ToAtom[int32](boolVec4); err == nil || !strings.Contains(err.Error(), "is not an atom") {
		t.Errorf("hostio.ToAtom[int32]([4]bool) error = %v, want 'is not an atom' error", err)
	}
}

func TestHostArrayNamedTypeDefineString(t *testing.T) {
	file := &ir.File{
		Package: &ir.Package{
			Name: &ast.Ident{Name: "pkg"},
		},
	}
	named := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.IdentAST("MyVec")},
		File:       file,
		Underlying: ir.TypeExpr(nil, irhelper.ArrayType(ir.Int32Type(), 2)),
	}
	arr, err := hostio.NewHostArrayFromFlat(named, []int32{10, 20}, 2)
	if err != nil {
		t.Fatalf("NewHostArrayFromFlat failed: %v", err)
	}
	if got, want := arr.DefineString(file), "MyVec{10, 20}"; got != want {
		t.Errorf("arr.DefineString(file) = %q, want %q", got, want)
	}
	goVal, err := arr.GoValue()
	if err != nil {
		t.Fatalf("arr.GoValue() failed: %v", err)
	}
	if got, want := fmt.Sprint(goVal), "[2]int32{10, 20}"; got != want {
		t.Errorf("fmt.Sprint(arr.GoValue()) = %q, want %q", got, want)
	}
}

func TestHostArrayInvalidSizeAndUnsupportedDType(t *testing.T) {
	if _, err := hostio.NewHostArrayFromFlat(irhelper.ArrayType(ir.Int32Type(), 3), []int32{1, 2}, 3); err == nil {
		t.Errorf("NewHostArrayFromFlat succeeded on mismatched slice length, want error")
	}
	if _, err := hostio.NewHostArrayFromData[int32](ir.Int32Type(), nil); err == nil {
		t.Errorf("NewHostArrayFromData succeeded on nil data, want error")
	}
	if _, err := hostio.NewHostArrayFromData(ir.Int32Type(), &hostio.HostData[int32]{
		Shape: shapes.Make(dtypes.Float32),
		Flat:  []int32{1},
	}); err == nil {
		t.Errorf("NewHostArrayFromData succeeded on mismatched shape DType, want error")
	}
	for _, unsupported := range []dtypes.DType{dtypes.Int8, dtypes.F8E4M3FN, dtypes.InvalidDType} {
		if _, err := hostio.NewHostArrayFromData(ir.Int32Type(), &hostio.HostData[int32]{
			Shape: shapes.Shape{DType: unsupported},
			Flat:  []int32{1},
		}); err == nil {
			t.Errorf("NewHostArrayFromData for %s succeeded, want error", unsupported)
		}
	}
}
