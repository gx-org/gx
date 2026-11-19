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

type stubValue struct {
	typ ir.Type
	val int
}

func (v stubValue) Type() ir.Type                 { return v.typ }
func (v stubValue) SourceString(*ir.File) string  { return v.typ.ReferString(nil) }
func (v stubValue) ToHost() (hostio.Value, error) { return v, nil }
func (v stubValue) Finalize() error               { return nil }

type stubNamedValue struct {
	stubValue
	under hostio.Value
}

func (v stubNamedValue) Underlying() hostio.Value { return v.under }

type stubStructValue struct {
	stubValue
	fields map[string]ir.Element
}

func (v stubStructValue) FieldValue(name string) ir.Element { return v.fields[name] }

type stubTypeEval struct {
	file *ir.File
	env  map[string]ir.Element
}

func (e *stubTypeEval) File() *ir.File { return e.file }

func (e *stubTypeEval) EvalExpr(expr ir.Expr) (ir.Element, error) {
	switch x := expr.(type) {
	case *ir.Ident:
		if el, ok := e.env[x.Src.Name]; ok {
			return el, nil
		}
		return nil, fmt.Errorf("undefined identifier %s", x.Src.Name)
	case *ir.NumberCastExpr:
		return e.EvalExpr(x.X)
	case *ir.NumberInt:
		return stubValue{typ: ir.IntType(), val: int(x.Val.Int64())}, nil
	default:
		return nil, fmt.Errorf("unsupported expr %T", expr)
	}
}

func (e *stubTypeEval) Sub(*ir.File, map[string]ir.Element) (ir.Evaluator, error) {
	return e, nil
}

func (e *stubTypeEval) Compare(x, y ir.Element) (bool, error) {
	if ir.IsUnknown(x.Type(), y.Type()) {
		return true, nil
	}
	return x.(stubValue).val == y.(stubValue).val, nil
}

type recordingConverter struct {
	ev      *stubTypeEval
	order   []string
	convert func(*ir.Field, hostio.Value) ir.Element
}

func (c *recordingConverter) ElementFromReceiverValue(_ ir.Evaluator, field *ir.Field, val hostio.Value) (ir.Element, error) {
	c.order = append(c.order, "recv:"+field.Name.Name)
	if c.convert != nil {
		return c.convert(field, val), nil
	}
	return val, nil
}

func (c *recordingConverter) ElementFromArgValue(_ ir.Evaluator, field *ir.Field, _ int, val hostio.Value) (ir.Element, error) {
	c.order = append(c.order, "arg:"+field.Name.Name)
	var el ir.Element = val
	if c.convert != nil {
		el = c.convert(field, val)
	}
	c.ev.env[field.Name.Name] = el
	return el, nil
}

func newNamedStruct(file *ir.File, name string, genExprs ...ir.Expr) *ir.NamedType {
	nt := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.IdentAST(name)},
		File:       file,
		Underlying: ir.TypeExpr(nil, &ir.StructType{}),
	}
	for _, expr := range genExprs {
		nt.GenParams.Values = append(nt.GenParams.Values, ir.NewNonTypeGenericValue(nil, expr))
	}
	return nt
}

func newAnonymousStruct(name string, typ ir.Type) *ir.StructType {
	grp := &ir.FieldGroup{
		Src:  &ast.Field{},
		Type: ir.TypeExpr(typ.Refer(nil), typ),
	}
	st := irhelper.StructType(irhelper.Field(name, typ, grp))
	st.Fields.Src.List = []*ast.Field{grp.Src}
	return st
}

func TestInputTypeError(t *testing.T) {
	file := &ir.File{Package: &ir.Package{Name: &ast.Ident{Name: "pkg"}}}
	sa := newNamedStruct(file, "SA")
	sb := newNamedStruct(file, "SB")
	anonymousStruct := newAnonymousStruct("v", ir.Float32Type())

	for _, tc := range []struct {
		name string
		arg  string
		want ir.Type
		got  ir.Element
		msg  string
	}{
		{"named_with_arg", "x", sa, stubValue{typ: sb}, "cannot use pkg.SB as pkg.SA for argument x"},
		{"named_unnamed", "", sa, stubValue{typ: sb}, "cannot use pkg.SB as pkg.SA"},
		{"unknown_struct", "b", sa, stubValue{typ: ir.UnknownType()}, "cannot use hostio_test.stubValue as pkg.SA for argument b"},
		{"unknown_anonymous_struct", "b", anonymousStruct, stubValue{typ: ir.UnknownType()}, "cannot use hostio_test.stubValue as " + anonymousStruct.ReferString(nil) + " for argument b"},
		{"nil_non_struct", "n", ir.Int32Type(), nil, "cannot use <nil> as int32 for argument n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := hostio.InputTypeError(tc.arg, tc.want, tc.got)
			if err == nil || err.Error() != tc.msg {
				t.Errorf("InputTypeError(%q, %s, %v) = %v, want %q", tc.arg, tc.want.ReferString(nil), tc.got, err, tc.msg)
			}
		})
	}
}

func TestCheckType(t *testing.T) {
	file := &ir.File{Package: &ir.Package{Name: &ast.Ident{Name: "pkg"}}}
	sa := newNamedStruct(file, "SA")
	sb := newNamedStruct(file, "SB")
	nField := irhelper.Field("N", ir.IntType(), nil)
	boxN := newNamedStruct(file, "Box", &ir.Ident{Src: irhelper.IdentAST("N"), Stor: nField.Storage()})
	box2 := newNamedStruct(file, "Box", irhelper.IntNumberAs(2, ir.IntType()))
	box3 := newNamedStruct(file, "Box", irhelper.IntNumberAs(3, ir.IntType()))
	boxSlice := func(a, b int64) *ir.NamedType {
		return newNamedStruct(file, "Box", &ir.SliceLitExpr{
			Typ:  ir.IntSliceType(),
			Elts: []ir.Expr{irhelper.IntNumberAs(a, ir.IntType()), irhelper.IntNumberAs(b, ir.IntType())},
		})
	}
	tParam := ir.NewGenericTypeParam(irhelper.Field("T", ir.AnyType(), nil))
	tagF32 := newNamedStruct(file, "Tag")
	tagF32.GenParams.Values = []ir.GenericValue{ir.NewTypeGenericValue(tParam, ir.Float32Type())}
	tagI32 := newNamedStruct(file, "Tag")
	tagI32.GenParams.Values = []ir.GenericValue{ir.NewTypeGenericValue(tParam, ir.Int32Type())}
	anonymousStruct := newAnonymousStruct("v", ir.Float32Type())
	holder := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.IdentAST("Holder")},
		File:       file,
		Underlying: ir.TypeExpr(nil, newAnonymousStruct("B", sb)),
	}
	pyObj := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.IdentAST("Obj")},
		File:       file,
		Underlying: ir.TypeExpr(nil, ir.UnknownType()),
	}

	for _, tc := range []struct {
		name    string
		field   *ir.Field
		got     hostio.Value
		wantErr string
	}{
		{"matching_named_struct", irhelper.Field("b", boxN, nil), stubValue{typ: box2}, ""},
		{"matching_concrete_gen_param", irhelper.Field("b", box2, nil), stubValue{typ: box2}, ""},
		{"mismatched_concrete_gen_param", irhelper.Field("b", box2, nil), stubValue{typ: box3}, "cannot use pkg.Box[3] as pkg.Box[2] for argument b"},
		{"mismatched_slice_gen_param", irhelper.Field("b", boxSlice(2, 3), nil), stubValue{typ: boxSlice(2, 4)}, "cannot use pkg.Box"},
		{"mismatched_concrete_type_gen_param", irhelper.Field("t", tagF32, nil), stubValue{typ: tagI32}, "cannot use pkg.Tag[int32] as pkg.Tag[float32] for argument t"},
		{"matching_anonymous_struct", irhelper.Field("s", anonymousStruct, nil), stubValue{typ: anonymousStruct}, ""},
		{"anonymous_struct_any_field", irhelper.Field("s", newAnonymousStruct("x", ir.AnyType()), nil), stubValue{typ: newAnonymousStruct("x", ir.Float32Type())}, ""},
		{"runtime_struct_field_valid", irhelper.Field("h", holder, nil), stubNamedValue{stubValue: stubValue{typ: holder}, under: stubStructValue{fields: map[string]ir.Element{"B": stubValue{typ: sb}}}}, ""},
		{"runtime_struct_field_mismatch", irhelper.Field("h", holder, nil), stubNamedValue{stubValue: stubValue{typ: holder}, under: stubStructValue{fields: map[string]ir.Element{"B": stubValue{typ: box2}}}}, "cannot use pkg.Box[2] as pkg.SB for argument h.B"},
		{"runtime_struct_field_missing", irhelper.Field("h", holder, nil), stubNamedValue{stubValue: stubValue{typ: holder}, under: stubStructValue{}}, "cannot use <nil> as pkg.SB for argument h.B"},
		{"unknown_for_pyobj", irhelper.Field("x", pyObj, nil), stubValue{typ: ir.UnknownType()}, ""},
		{"unknown_for_array", irhelper.Field("x", irhelper.ArrayType(ir.Float32Type(), 2), nil), stubValue{typ: ir.UnknownType()}, ""},
		{"wrong_named_struct", irhelper.Field("b", boxN, nil), stubValue{typ: sb}, "cannot use pkg.SB as pkg.Box[N] for argument b"},
		{"unknown_for_named_struct", irhelper.Field("b", boxN, nil), stubValue{typ: ir.UnknownType()}, "cannot use hostio_test.stubValue as pkg.Box[N] for argument b"},
		{"array_for_named_struct", irhelper.Field("b", sa, nil), stubValue{typ: irhelper.ArrayType(ir.Int32Type(), 2)}, "cannot use [2]int32 as pkg.SA for argument b"},
		{"unknown_for_anonymous_struct", irhelper.Field("s", anonymousStruct, nil), stubValue{typ: ir.UnknownType()}, "cannot use hostio_test.stubValue as struct"},
		{"named_for_anonymous_struct", irhelper.Field("s", anonymousStruct, nil), stubValue{typ: sa}, "cannot use pkg.SA as struct"},
		{"anonymous_struct_wrong_field_name", irhelper.Field("s", anonymousStruct, nil), stubValue{typ: newAnonymousStruct("w", ir.Float32Type())}, "cannot use struct {\n\tw float32\n} as struct"},
		{"anonymous_struct_wrong_field_type", irhelper.Field("s", anonymousStruct, nil), stubValue{typ: newAnonymousStruct("v", ir.Int32Type())}, "cannot use struct {\n\tv int32\n} as struct"},
		{"anonymous_struct_wrong_field_count", irhelper.Field("s", anonymousStruct, nil), stubValue{typ: irhelper.StructType()}, "cannot use struct {\n} as struct"},
		{"anonymous_struct_nested_mismatch", irhelper.Field("s", newAnonymousStruct("inner", sa), nil), stubValue{typ: newAnonymousStruct("inner", sb)}, "cannot use struct"},
		{"nil_value", irhelper.Field("b", sa, nil), nil, "cannot use <nil> as pkg.SA for argument b"},
		{"nil_unnamed_receiver", &ir.Field{Group: &ir.FieldGroup{Type: ir.TypeExpr(nil, sa)}}, nil, "cannot use <nil> as pkg.SA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := hostio.CheckType(tc.field, tc.got)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("CheckType() unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("CheckType() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestFuncInputsToElements(t *testing.T) {
	file := &ir.File{Package: &ir.Package{Name: &ast.Ident{Name: "pkg"}}}
	sa := newNamedStruct(file, "SA")
	sb := newNamedStruct(file, "SB")
	nField := irhelper.Field("N", ir.IntType(), nil)
	nIdent := &ir.Ident{Src: irhelper.IdentAST("N"), Stor: nField.Storage()}
	boxN := newNamedStruct(file, "Box", nIdent)
	box2 := newNamedStruct(file, "Box", irhelper.IntNumberAs(2, ir.IntType()))
	box3 := newNamedStruct(file, "Box", irhelper.IntNumberAs(3, ir.IntType()))
	arrN := irhelper.ArrayType(ir.Float32Type(), &ir.AxisExpr{X: nIdent})
	arr2 := irhelper.ArrayType(ir.Float32Type(), 2)
	arr3 := irhelper.ArrayType(ir.Float32Type(), 3)

	// func (b Box[N]) Method[N int](x [N]float32, s SA)
	method := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: irhelper.IdentAST("Method")},
		FType: &ir.FuncType{
			GenParams: ir.GenericParams{
				Fields: irhelper.Fields(nField),
			},
			Receiver: irhelper.Fields(irhelper.Field("b", boxN, nil)),
			Params: irhelper.Fields(
				irhelper.Field("x", arrN, nil),
				irhelper.Field("s", sa, nil),
			),
		},
	}
	// func Plain(s SA)
	plain := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: irhelper.IdentAST("Plain")},
		FType: &ir.FuncType{
			Params: irhelper.Fields(irhelper.Field("s", sa, nil)),
		},
	}
	anonymousStruct := newAnonymousStruct("v", ir.Float32Type())
	// func PlainAnonymousStruct(s struct{ v float32 })
	plainAnonymousStruct := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: irhelper.IdentAST("PlainAnonymousStruct")},
		FType: &ir.FuncType{
			Params: irhelper.Fields(irhelper.Field("s", anonymousStruct, nil)),
		},
	}
	// func GenericSliceFunc[S []int]()
	genericSliceFunc := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: irhelper.IdentAST("GenericSliceFunc")},
		FType: &ir.FuncType{
			GenParams: ir.GenericParams{
				Fields: irhelper.Fields(irhelper.Field("S", ir.IntSliceType(), nil)),
			},
		},
	}

	t.Run("generic_before_receiver_and_params", func(t *testing.T) {
		ev := &stubTypeEval{file: file, env: map[string]ir.Element{}}
		conv := &recordingConverter{ev: ev}
		in, err := hostio.FuncInputsToElements(conv, ev, method, stubValue{typ: box2}, []hostio.Value{
			stubValue{typ: ir.IntType(), val: 2},
			stubValue{typ: arr2},
			stubValue{typ: sa},
		})
		if err != nil {
			t.Fatalf("FuncInputsToElements() unexpected error: %v", err)
		}
		if in.Receiver == nil || len(in.Args) != 3 {
			t.Fatalf("FuncInputsToElements() = %+v, want non-nil receiver and 3 args", in)
		}
		wantOrder := []string{"arg:N", "recv:b", "arg:x", "arg:s"}
		if !slices.Equal(conv.order, wantOrder) {
			t.Errorf("conversion order = %v, want %v", conv.order, wantOrder)
		}
	})

	t.Run("unknown_allowed_for_array", func(t *testing.T) {
		ev := &stubTypeEval{file: file, env: map[string]ir.Element{}}
		conv := &recordingConverter{ev: ev}
		_, err := hostio.FuncInputsToElements(conv, ev, method, stubValue{typ: box2}, []hostio.Value{
			stubValue{typ: ir.IntType(), val: 2},
			stubValue{typ: ir.UnknownType()},
			stubValue{typ: sa},
		})
		if err != nil {
			t.Fatalf("FuncInputsToElements() unexpected error for unknown array arg: %v", err)
		}
	})

	t.Run("unknown_allowed_for_slice_generic_param", func(t *testing.T) {
		ev := &stubTypeEval{file: file, env: map[string]ir.Element{}}
		conv := &recordingConverter{ev: ev}
		if _, err := hostio.FuncInputsToElements(conv, ev, genericSliceFunc, nil, []hostio.Value{
			stubValue{typ: ir.UnknownType()},
		}); err != nil {
			t.Fatalf("FuncInputsToElements() unexpected error for unknown slice generic arg: %v", err)
		}
	})

	t.Run("anonymous_struct_arg_valid", func(t *testing.T) {
		ev := &stubTypeEval{file: file, env: map[string]ir.Element{}}
		conv := &recordingConverter{ev: ev}
		if _, err := hostio.FuncInputsToElements(conv, ev, plainAnonymousStruct, nil, []hostio.Value{
			stubValue{typ: anonymousStruct},
		}); err != nil {
			t.Fatalf("FuncInputsToElements() unexpected error for anonymous struct arg: %v", err)
		}
	})

	t.Run("validates_converted_element", func(t *testing.T) {
		ev := &stubTypeEval{file: file, env: map[string]ir.Element{}}
		conv := &recordingConverter{
			ev: ev,
			convert: func(field *ir.Field, val hostio.Value) ir.Element {
				if field.Type() == ir.IntType() && val.Type() == ir.Int64Type() {
					return stubValue{typ: ir.IntType(), val: val.(stubValue).val}
				}
				if field.Name.Name == "x" {
					return stubValue{typ: arr2}
				}
				return val
			},
		}
		_, err := hostio.FuncInputsToElements(conv, ev, method, stubValue{typ: box2}, []hostio.Value{
			stubValue{typ: ir.Int64Type(), val: 2},
			stubValue{typ: arr3},
			stubValue{typ: sa},
		})
		if err != nil {
			t.Fatalf("FuncInputsToElements() unexpected error after converter adaptation: %v", err)
		}
	})

	for _, tc := range []struct {
		name    string
		fn      ir.Func
		recv    hostio.Value
		args    []hostio.Value
		wantErr string
	}{
		{
			name:    "missing_receiver",
			fn:      method,
			args:    []hostio.Value{stubValue{typ: ir.IntType(), val: 2}, stubValue{typ: arr2}, stubValue{typ: sa}},
			wantErr: "nil value has been passed as a receiver",
		},
		{
			name:    "unexpected_receiver",
			fn:      plain,
			recv:    stubValue{typ: sa},
			args:    []hostio.Value{stubValue{typ: sa}},
			wantErr: "has no receiver",
		},
		{
			name:    "missing_param",
			fn:      plain,
			args:    nil,
			wantErr: "missing parameter(s): s",
		},
		{
			name:    "too_many_args",
			fn:      plain,
			args:    []hostio.Value{stubValue{typ: sa}, stubValue{typ: sa}},
			wantErr: "too many arguments",
		},
		{
			name:    "wrong_receiver_generic_dim",
			fn:      method,
			recv:    stubValue{typ: box3},
			args:    []hostio.Value{stubValue{typ: ir.IntType(), val: 2}, stubValue{typ: arr2}, stubValue{typ: sa}},
			wantErr: "cannot use pkg.Box[3] as pkg.Box[N] for argument b",
		},
		{
			name:    "wrong_array_generic_dim",
			fn:      method,
			recv:    stubValue{typ: box2},
			args:    []hostio.Value{stubValue{typ: ir.IntType(), val: 2}, stubValue{typ: arr3}, stubValue{typ: sa}},
			wantErr: "cannot use [3]float32 as [N]float32 for argument x",
		},
		{
			name:    "wrong_struct_arg",
			fn:      plain,
			args:    []hostio.Value{stubValue{typ: sb}},
			wantErr: "cannot use pkg.SB as pkg.SA for argument s",
		},
		{
			name:    "unknown_for_struct_arg",
			fn:      plain,
			args:    []hostio.Value{stubValue{typ: ir.UnknownType()}},
			wantErr: "cannot use hostio_test.stubValue as pkg.SA for argument s",
		},
		{
			name:    "unknown_for_anonymous_struct_arg",
			fn:      plainAnonymousStruct,
			args:    []hostio.Value{stubValue{typ: ir.UnknownType()}},
			wantErr: "cannot use hostio_test.stubValue as struct",
		},
		{
			name:    "wrong_generic_arg",
			fn:      method,
			recv:    stubValue{typ: box2},
			args:    []hostio.Value{stubValue{typ: ir.Float32Type()}, stubValue{typ: arr2}, stubValue{typ: sa}},
			wantErr: "cannot use float32 as int for argument N",
		},
		{
			name:    "unknown_for_struct_receiver",
			fn:      method,
			recv:    stubValue{typ: ir.UnknownType()},
			args:    []hostio.Value{stubValue{typ: ir.IntType(), val: 2}, stubValue{typ: arr2}, stubValue{typ: sa}},
			wantErr: "cannot use hostio_test.stubValue as pkg.Box[N] for argument b",
		},
		{
			name:    "nil_generic_arg",
			fn:      method,
			recv:    stubValue{typ: box2},
			args:    []hostio.Value{nil, stubValue{typ: arr2}, stubValue{typ: sa}},
			wantErr: "cannot use <nil> as int for argument N",
		},
		{
			name:    "nil_regular_arg",
			fn:      plain,
			args:    []hostio.Value{nil},
			wantErr: "cannot use <nil> as pkg.SA for argument s",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ev := &stubTypeEval{file: file, env: map[string]ir.Element{}}
			conv := &recordingConverter{ev: ev}
			_, err := hostio.FuncInputsToElements(conv, ev, tc.fn, tc.recv, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("FuncInputsToElements() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}
