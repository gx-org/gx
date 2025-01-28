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

// Package randtest tests calling the rand package from GX standard library.
package randtest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/tests/bindings/rand/rand_go_gx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var randGX *rand_go_gx.Package

func TestSample(t *testing.T) {
	tests := []struct {
		desc string
		seed int64
		want []float32
	}{
		{
			desc: "first call to seed value",
			seed: 0,
			want: []float32{0.30324587, 0.8322963, 0.032319535},
		},
		{
			desc: "second call with the same seed value: should return the same values",
			seed: 0,
			want: []float32{0.30324587, 0.8322963, 0.032319535},
		},
		{
			desc: "call with a different seed: values should be different",
			seed: 1,
			want: []float32{0.35960966, 0.5141481, 0.86482173},
		},
		{
			desc: "call with the same last seed: values should be identical",
			seed: 1,
			want: []float32{0.35960966, 0.5141481, 0.86482173},
		},
		{
			desc: "back to the first seed",
			seed: 0,
			want: []float32{0.30324587, 0.8322963, 0.032319535},
		},
	}
	for _, test := range tests {
		vals, err := randGX.Sample.Run(types.Int64(test.seed))
		if err != nil {
			t.Fatalf("%+v", err)
		}
		got := gxtesting.FetchArray(t, vals)
		if !cmp.Equal(test.want, got) {
			t.Errorf("test %s:\nincorrect random values: got %v but want %v", test.desc, got, test.want)
		}
	}
}

func TestSampleBool(t *testing.T) {
	values, err := randGX.SampleBool.Run(types.Int64(3141))
	if err != nil {
		t.Fatalf("%+v", err)
	}

	ts, fs := 0, 0
	got := gxtesting.FetchArray(t, values)
	for _, value := range got {
		if value {
			ts++
		} else {
			fs++
		}
	}
	if ts < 45 || fs < 45 {
		t.Errorf("Expected roughly 50/50 distribution of true/false, got %d/%d", ts, fs)
	}
}

func TestDeviceValueAsSeed(t *testing.T) {
	seed := types.Int64(0)
	seedDevice, err := seed.SendTo(randGX.Device)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := randGX.New.Run(seedDevice); err != nil {
		t.Error(err)
	}
}

func setupTest(dev *api.Device) error {
	gxPackage, err := rand_go_gx.Load(dev.Runtime())
	if err != nil {
		return err
	}
	randGX = gxPackage.BuildFor(dev)
	return nil
}
