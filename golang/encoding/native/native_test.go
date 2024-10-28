// Copyright 2024 Google LLC
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

package native_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/golang/encoding/native"
	"github.com/gx-org/gx/tests/bindings/encoding/encoding_go_gx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var encodingGX *encoding_go_gx.Compiler

func multiply[T dtype.AlgebraType](f int, x T) T {
	return x * T(f)
}

func scalarsData(factor int) map[string]any {
	return map[string]any{
		"Int":     multiply[int32](factor, 42),
		"Float32": multiply[float32](factor, 4.2),
		"f64":     multiply[float64](factor, 4.2),
	}
}

func goSlicesData(factor int) map[string]any {
	return map[string]any{
		"ArrayF32": []float32{
			multiply[float32](factor, 10),
			multiply[float32](factor, 20),
		},
	}
}

func gxArrayData(factor int) map[string]any {
	return map[string]any{
		"ArrayF32": array[float32]([]float32{
			multiply[float32](factor, 10),
			multiply[float32](factor, 20),
		}),
	}
}

func encodingData(factor int) map[string]any {
	return map[string]any{
		"Scalars":      scalarsData(factor),
		"DataAsSlices": goSlicesData(factor),
		"DataAsArrays": gxArrayData(factor),
	}
}

func checkScalars(t *testing.T, factor int, scalars *encoding_go_gx.Scalars) {
	intGot := gxtesting.FetchAtom[int32](t, scalars.Int)
	intWant := multiply[int32](factor, 42)
	if intGot != intWant {
		t.Errorf("got %d but want %d", intGot, intWant)
	}

	f32Got := gxtesting.FetchAtom[float32](t, scalars.Float32)
	f32Want := multiply[float32](factor, 4.2)
	if f32Got != f32Want {
		t.Errorf("got %f but want %f", f32Got, f32Want)
	}

	f64Got := gxtesting.FetchAtom[float64](t, scalars.Float64)
	f64Want := multiply[float64](factor, 4.2)
	if f64Got != f64Want {
		t.Errorf("got %f but want %f", f64Got, f64Want)
	}
}

func checkArray(t *testing.T, factor int, arr *encoding_go_gx.Arrays) {
	arrayGot := gxtesting.FetchArray(t, arr.ArrayF32)
	arrayWant := []float32{
		multiply[float32](factor, 10),
		multiply[float32](factor, 20),
	}
	if !cmp.Equal(arrayGot, arrayWant) {
		t.Errorf("got %v but want %v", arrayGot, arrayWant)
	}
	shapeWant := &shape.Shape{DType: dtype.Float32, AxisLengths: []int{2}}
	if !cmp.Equal(arr.ArrayF32.Shape(), shapeWant) {
		t.Errorf("got %v but want %v", arr.ArrayF32.Shape(), shapeWant)
	}
}

func checkEncoding(t *testing.T, factor int, enc *encoding_go_gx.Encoding) {
	checkScalars(t, factor, enc.Scalars)
	checkArray(t, factor, enc.DataAsSlices)
	checkArray(t, factor, enc.DataAsArrays)
}

type array[T dtype.AlgebraType] []T

var _ shape.ArrayI[ir.Int] = (array[ir.Int])(nil)

func (a array[T]) Flat() []T { return a }

func (a array[T]) Shape() []int { return []int{len(a)} }

func TestUnmarshalNativeEncoding(t *testing.T) {
	encodingStruct := encodingGX.Factory.NewEncoding()
	const factor = 1
	if err := native.Unmarshal(encodingStruct, encodingData(factor)); err != nil {
		t.Fatalf("%+v", fmterr.ToStackTraceError(err))
	}
	checkEncoding(t, factor, encodingStruct)
}

func TestUnmarshalNativeSliceEncoding(t *testing.T) {
	data := map[string]any{
		"Instance": encodingData(-1),
		"Slice": []any{
			encodingData(0),
			encodingData(1),
			encodingData(2),
		},
	}
	onDev := encodingGX.Factory.NewSlice()
	if err := native.Unmarshal(onDev, data); err != nil {
		t.Fatalf("%+v", fmterr.ToStackTraceError(err))
	}
	checkEncoding(t, -1, onDev.Instance)
	const wantLen = 3
	gotLen := 0
	if onDev.Slice != nil {
		gotLen = onDev.Slice.Size()
	}
	if gotLen != wantLen {
		t.Errorf("incorrect slice length: want %d got %d", wantLen, gotLen)
		return
	}
	checkEncoding(t, 0, onDev.Slice.At(0))
	checkEncoding(t, 1, onDev.Slice.At(1))
	checkEncoding(t, 2, onDev.Slice.At(2))
}

func TestRunUnmarshalledScalarsStruct(t *testing.T) {
	const factor = 1
	scalars := encodingGX.Factory.NewScalars()
	if err := native.Unmarshal(scalars, scalarsData(factor)); err != nil {
		t.Fatalf("%+v", err)
	}
	checkScalars(t, factor, scalars)
	gotTotalDevice, err := scalars.GetTotal().Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	gotTotal, err := gotTotalDevice.FetchValue()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	wantTotal := float32(50.4)
	if !cmp.Equal(gotTotal, wantTotal) {
		t.Errorf("got %v but want %v", gotTotal, wantTotal)
	}
}

func TestMain(m *testing.M) {
	bck := backend.New(builder.New([]builder.Importer{
		embedpkg.New(),
	}))
	gxPackage, err := encoding_go_gx.Load(bck)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", fmterr.ToStackTraceError(err))
		return
	}
	dev, err := bck.Platform().Device(0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", fmterr.ToStackTraceError(err))
		return
	}
	encodingGX = gxPackage.CompilerFor(dev)
	os.Exit(m.Run())
}
