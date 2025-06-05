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

// Package parameterstest tests the Go bindings of the basic GX package.
package parameterstest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/tests/bindings/parameters/parameters_go_gx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var parametersGX *parameters_go_gx.Package

func TestAddInt(t *testing.T) {
	x := types.Int64(3)
	y := types.Int64(4)
	z, err := parametersGX.AddInt.Run(x, y)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, z)
	want := int64(7)
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect computation: got %T(%v) but want %T(%v)", got, got, want, want)
	}
}

func TestAddInts(t *testing.T) {
	x := types.ArrayInt64([]int64{1, 2, 3})
	y := types.ArrayInt64([]int64{-2, -4, -6})
	z, err := parametersGX.AddInts.Run(x, y)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchArray(t, z)
	want := []int64{-1, -2, -3}
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect computation: got %T(%v) but want %T(%v)", got, got, want, want)
	}
}

func TestAddFloat32(t *testing.T) {
	x := types.Float32(3)
	y := types.Float32(4)
	z, err := parametersGX.AddFloat32.Run(x, y)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, z)
	want := float32(7)
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect computation: got %T(%v) but want %T(%v)", got, got, want, want)
	}
}

func TestAddFloat32s(t *testing.T) {
	x := types.ArrayFloat32([]float32{1, 2, 3})
	y := types.ArrayFloat32([]float32{-2, -4, -6})
	z, err := parametersGX.AddFloat32s.Run(x, y)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchArray(t, z)
	want := []float32{-1, -2, -3}
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect computation: got %T(%v) but want %T(%v)", got, got, want, want)
	}
}

func TestLen(t *testing.T) {
	x := types.ArrayFloat32([]float32{1, 2, 3})
	length, err := parametersGX.Len.Run(x)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, length)
	want := int64(3)
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect computation: got %T(%v) but want %T(%v)", got, got, want, want)
	}
}

func TestIota(t *testing.T) {
	iota, err := parametersGX.Iota.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchArray(t, iota)
	want := []int64{0, 1, 2, 3, 4, 5}
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect computation: got %T(%v) but want %T(%v)", got, got, want, want)
	}
}

func TestAddToStruct(t *testing.T) {
	offset := types.Float32(10)
	st, err := parametersGX.NewStruct.Run(offset)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	for i := range int32(4) {
		ok := true
		if32 := float32(i)
		st, err = parametersGX.AddToStruct.Run(st)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		gotA := gxtesting.FetchArray(t, st.A)
		wantA := []float32{if32 + 1, offset.Value() + if32 + 1}
		if !cmp.Equal(gotA, wantA) {
			t.Errorf("incorrect A computation: got %T(%v) but want %T(%v)", gotA, gotA, wantA, wantA)
			ok = false
		}

		gotB := []float32{
			gxtesting.FetchAtom(t, st.B.At(0)),
			gxtesting.FetchAtom(t, st.B.At(1)),
			gxtesting.FetchAtom(t, st.B.At(2)),
		}
		wantB := []float32{11 + if32, 101 + if32, 1001 + if32}
		if !cmp.Equal(gotB, wantB) {
			t.Errorf("incorrect B computation: got %T(%v) but want %T(%v)", gotB, gotB, wantB, wantB)
			ok = false
		}

		gotC := []int32{
			gxtesting.FetchAtom(t, st.C.At(0).Val),
			gxtesting.FetchAtom(t, st.C.At(1).Val),
			gxtesting.FetchAtom(t, st.C.At(2).Val),
		}
		wantC := []int32{i + 1, i + 2, i + 3}
		if !cmp.Equal(gotC, wantC) {
			t.Errorf("incorrect C computation: got %T(%v) but want %T(%v)", gotC, gotC, wantC, wantC)
			ok = false
		}
		// TODO(b/395551138): disable the test on the SpecialValue field for now.
		if i < 5 {
			continue
		}
		gotSpecialValue := gxtesting.FetchAtom(t, st.SpecialValue)
		wantSpecialValue := wantA[1] - 1
		if !cmp.Equal(gotSpecialValue, wantSpecialValue) {
			t.Errorf("incorrect SpecialValue computation: got %T(%v) but want %T(%v)", gotSpecialValue, gotSpecialValue, wantSpecialValue, wantSpecialValue)
			ok = false
		}
		if !ok {
			break
		}
	}
}

func TestSliceSliceArg(t *testing.T) {
	t.SkipNow() // TODO: run a Go/XLA hybrid graph?

	wants := [][]float32{
		{1, 2},
		{3, 4},
		{5, 6},
	}
	values := make([]types.Array[float32], len(wants))
	for i, want := range wants {
		values[i] = types.ArrayFloat32(want)
	}
	arg, err := types.NewSlice[types.Array[float32]](nil, values)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range wants {
		hostI := types.Int32(int32(i))
		gotDevice, err := parametersGX.SliceSliceArg.Run(arg, hostI)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		got := gxtesting.FetchArray(t, gotDevice)
		if !cmp.Equal(got, want) {
			t.Errorf("incorrect computation: got %T(%v) but want %T(%v)", got, got, want, want)
		}
	}
}

func TestSliceArrayArg(t *testing.T) {
	t.SkipNow() // GX does not support slicing with an index given at runtime.
	wants := [][]float32{
		{1, 2},
		{3, 4},
		{5, 6},
	}
	flat := []float32{}
	for _, want := range wants {
		flat = append(flat, want...)
	}
	arg := types.ArrayFloat32(flat, 3, 2)
	for i, want := range wants {
		hostI := types.Int32(int32(i))
		gotDevice, err := parametersGX.SliceArrayArg.Run(arg, hostI)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		got := gxtesting.FetchArray(t, gotDevice)
		if !cmp.Equal(got, want) {
			t.Errorf("incorrect computation for index %d: got %T(%v) but want %T(%v)", i, got, got, want, want)
		}
	}
}

func TestSliceArrayArgConstIndex(t *testing.T) {
	wants := [][]float32{
		{1, 2},
		{3, 4},
		{5, 6},
	}
	flat := []float32{}
	for _, want := range wants {
		flat = append(flat, want...)
	}
	arg := types.ArrayFloat32(flat, 3, 2)
	gotDev0, gotDev1, gotDev2, err := parametersGX.SliceArrayArgConstIndex.Run(arg)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	for i, gotDev := range []types.Array[float32]{gotDev0, gotDev1, gotDev2} {
		want := wants[i]
		got := gxtesting.FetchArray(t, gotDev)
		if !cmp.Equal(got, want) {
			t.Errorf("incorrect computation for index %d: got %T(%v) but want %T(%v)", i, got, got, want, want)
		}
	}
}

func TestSetNotInSlice(t *testing.T) {
	// GX does not make the difference between dynamic assignment, that is
	// assignment that needs to be done at every function call
	// and static assignment, that is an assignment that only needs to be done
	// at compile/trace time.
	// TestSetNotInSlice does not need code on the accelerator to run but does need
	// a dynamic assignment.
	t.SkipNow()
	offset := types.Float32(10)
	aStruct, err := parametersGX.NewStruct.Run(offset)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	const want = 3
	notInSlice, err := parametersGX.NewNotInSlice.Run(types.Int32(want))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	aStruct, err = aStruct.SetNotInSlice().Run(notInSlice)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, aStruct.D.Val)
	if got != want {
		t.Errorf("incorrect value: got %d but want %d", got, want)
	}
}

func setupTest(dev *api.Device) error {
	gxPackage, err := parameters_go_gx.Load(dev.Runtime())
	if err != nil {
		return err
	}
	parametersGX = gxPackage.BuildFor(
		dev,
		parameters_go_gx.Size.Set(5),
	)
	return nil
}
