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
	"math"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/tests/bindings/rand/rand_go_gx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var (
	randGX       *rand_go_gx.Package
	randHandleGX *rand_go_gx.PackageHandle
)

func TestSample(t *testing.T) {
	tests := []struct {
		desc string
		seed int64
		want []float32
	}{
		{
			desc: "first call to seed value",
			seed: 0,
			want: []float32{0.25327194, 0.30324578, 0.5043973},
		},
		{
			desc: "second call with the same seed value: should return the same values",
			seed: 0,
			want: []float32{0.25327194, 0.30324578, 0.5043973},
		},
		{
			desc: "call with a different seed: values should be different",
			seed: 1,
			want: []float32{0.7208755, 0.3596096, 0.6699227},
		},
		{
			desc: "call with the same last seed: values should be identical",
			seed: 1,
			want: []float32{0.7208755, 0.3596096, 0.6699227},
		},
		{
			desc: "back to the first seed",
			seed: 0,
			want: []float32{0.25327194, 0.30324578, 0.5043973},
		},
	}
	for _, test := range tests {
		vals, err := randGX.Sample(types.Int64(test.seed))
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
	values, err := randGX.SampleBool(types.Int64(3141))
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

func kolmogorovSmirnov(values []float64) float64 {
	n := len(values)
	slices.Sort(values)

	// Find the maximum absolute difference between the empirical and theoretical CDFs.
	var result float64
	for i := 0; i < n; i++ {
		empiricalCDF := float64(i+1) / float64(n)
		theoreticalCDF := float64(values[i]-values[0]) / (values[n-1] - values[0])
		diff := math.Abs(empiricalCDF - theoreticalCDF)
		if diff > result {
			result = diff
		}
	}
	return result
}

func TestSampleUniformFloat64(t *testing.T) {
	values, err := randGX.SampleUniformFloat64(types.Int64(0))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchArray(t, values)

	// Set the critical value for the Kolmogorov-Smirnov test using a significance level of 0.05.
	//   https://people.cs.pitt.edu/~lipschultz/cs1538/prob-table_KS.pdf
	criticalValue := 1.35810 / math.Sqrt(float64(len(got)))
	if ksStatistic := kolmogorovSmirnov(got); ksStatistic > criticalValue {
		t.Errorf("Kolmogorov-Smirnoff statistic: %f > critical value %f", ksStatistic, criticalValue)
	}
}

func TestDeviceValueAsSeed(t *testing.T) {
	seed := types.Int64(0)
	seedDevice, err := seed.SendTo(randHandleGX.Device())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := randGX.New(seedDevice); err != nil {
		t.Error(err)
	}
}

func setupTest(dev *api.Device) error {
	var err error
	randHandleGX, err = rand_go_gx.BuildHandleFor(dev)
	randGX = randHandleGX.Factory.Package
	return err
}
