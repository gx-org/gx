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

// Package dtypestest tests the Go bindings for all dtypes.
package dtypestest

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/tests/bindings/dtypes/dtypes_go_gx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var dtypes *dtypes_go_gx.Package

func TestBool(t *testing.T) {
	atom, err := dtypes.Bool(types.Bool(true))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	const want = false
	if got := gxtesting.FetchAtom(t, atom); got != want {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestFloat32(t *testing.T) {
	atom, err := dtypes.Float32(types.Float32(math.MaxFloat32))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	const want = -math.MaxFloat32
	if got := gxtesting.FetchAtom(t, atom); got != want {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestFloat64(t *testing.T) {
	atom, err := dtypes.Float64(types.Float64(math.MaxFloat64))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	const want = -math.MaxFloat64
	if got := gxtesting.FetchAtom(t, atom); got != want {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestInt32(t *testing.T) {
	atom, err := dtypes.Int32(types.Int32(math.MaxInt32))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	const want = -math.MaxInt32
	if got := gxtesting.FetchAtom(t, atom); got != want {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestInt64(t *testing.T) {
	atom, err := dtypes.Int64(types.Int64(math.MaxInt64))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	const want = -math.MaxInt64
	if got := gxtesting.FetchAtom(t, atom); got != want {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestUint32(t *testing.T) {
	atom, err := dtypes.Uint32(types.Uint32(math.MaxUint32))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	var want uint32 = math.MaxUint32
	want = -want
	if got := gxtesting.FetchAtom(t, atom); got != want {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestUint64(t *testing.T) {
	atom, err := dtypes.Uint64(types.Uint64(math.MaxUint64))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	var want uint64 = math.MaxUint64
	want = -want
	if got := gxtesting.FetchAtom(t, atom); got != want {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestBoolArray(t *testing.T) {
	arg := []bool{true, false, true, false, true, false}
	array, err := dtypes.ArrayBool(types.ArrayBool(arg, 2, 3))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	want := []bool{false, true, false, true, false, true}
	if got := gxtesting.FetchArray(t, array); !cmp.Equal(got, want) {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestFloat32Array(t *testing.T) {
	arg := types.ArrayFloat32([]float32{
		math.MaxFloat32, -math.MaxFloat32,
		math.SmallestNonzeroFloat32, -math.SmallestNonzeroFloat32,
	}, 2, 2)
	array, err := dtypes.ArrayFloat32(arg)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	want := []float32{
		-math.MaxFloat32, math.MaxFloat32,
		-math.SmallestNonzeroFloat32, math.SmallestNonzeroFloat32,
	}
	if got := gxtesting.FetchArray(t, array); !cmp.Equal(got, want) {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestFloat64Array(t *testing.T) {
	arg := types.ArrayFloat64([]float64{
		math.MaxFloat64, -math.MaxFloat64,
		math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64,
	}, 2, 2)
	array, err := dtypes.ArrayFloat64(arg)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	want := []float64{
		-math.MaxFloat64, math.MaxFloat64,
		-math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64,
	}
	if got := gxtesting.FetchArray(t, array); !cmp.Equal(got, want) {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestInt32Array(t *testing.T) {
	arg := types.ArrayInt32([]int32{math.MaxInt32, math.MinInt32 + 1}, 2)
	array, err := dtypes.ArrayInt32(arg)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	want := []int32{math.MinInt32 + 1, math.MaxInt32}
	if got := gxtesting.FetchArray(t, array); !cmp.Equal(got, want) {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestInt64Array(t *testing.T) {
	arg := types.ArrayInt64([]int64{math.MaxInt64, math.MinInt64 + 1}, 2)
	array, err := dtypes.ArrayInt64(arg)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	want := []int64{math.MinInt64 + 1, math.MaxInt64}
	if got := gxtesting.FetchArray(t, array); !cmp.Equal(got, want) {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestUint32Array(t *testing.T) {
	arg := types.ArrayUint32([]uint32{math.MaxUint32, 1}, 2)
	array, err := dtypes.ArrayUint32(arg)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	want := []uint32{1, math.MaxUint32}
	if got := gxtesting.FetchArray(t, array); !cmp.Equal(got, want) {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestUint64Array(t *testing.T) {
	arg := types.ArrayUint64([]uint64{math.MaxUint64, 1}, 2)
	array, err := dtypes.ArrayUint64(arg)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	want := []uint64{1, math.MaxUint64}
	if got := gxtesting.FetchArray(t, array); !cmp.Equal(got, want) {
		t.Errorf("got %v but want %v", got, want)
	}
}

func setupTest(dev *api.Device) error {
	var err error
	dtypes, err = dtypes_go_gx.BuildFor(dev)
	return err
}
