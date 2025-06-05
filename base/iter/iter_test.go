// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package iter_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/base/iter"
)

func TestAll(t *testing.T) {
	var got []string
	for el := range iter.All(
		[]string{"a", "b", "c"},
		[]string{"d", "e", "f"},
	) {
		got = append(got, el)
	}
	want := []string{"a", "b", "c", "d", "e", "f"}
	if !cmp.Equal(got, want) {
		t.Errorf("got %v but want %v", got, want)
	}
}

func isEven(n int) bool {
	return n%2 == 0
}

func TestFilter(t *testing.T) {
	var got []int
	for el := range iter.Filter(isEven,
		[]int{0, 1, 2},
		[]int{3, 4, 5},
	) {
		got = append(got, el)
	}
	want := []int{0, 2, 4}
	if !cmp.Equal(got, want) {
		t.Errorf("got %v but want %v", got, want)
	}
}
