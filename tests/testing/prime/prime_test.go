// Copyright 2025 Google LLC
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

package prime_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/tests/testing/prime"
)

func TestPrime(t *testing.T) {
	tests := []struct {
		Min  int
		Want []int
	}{
		{
			Min:  0,
			Want: []int{2, 3, 5},
		},
		{
			Min:  5,
			Want: []int{5, 7, 11, 13},
		},
		{
			Min:  7,
			Want: []int{7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97},
		},
	}
	for i, test := range tests {
		gen := prime.New(test.Min)
		got := []int{}
		for range test.Want {
			got = append(got, gen.Next())
		}
		if !cmp.Equal(test.Want, got) {
			t.Errorf("test %d: incorrect prime numbers: got %v but want %v", i, got, test.Want)
		}
	}
}
