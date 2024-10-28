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

package fmtarray_test

import (
	"strings"
	"testing"

	"github.com/gx-org/gx/fmt/fmtarray"
)

func buildData(axes []int) []int32 {
	total := int32(1)
	for _, axisSize := range axes {
		total *= int32(axisSize)
	}
	data := make([]int32, total)
	for i := range total {
		data[i] = i
	}
	return data
}

func TestFmtArray1Axis(t *testing.T) {
	tests := []struct {
		data []int32
		axes []int
		want string
	}{
		{
			data: []int32{42},
			want: "int32(42)",
		},
		{
			data: []int32{1, 2, 3, 4, 5, 6},
			axes: []int{6},
			want: "[6]int32{1, 2, 3, 4, 5, 6}",
		},
		{
			axes: []int{2, 3},
			want: `
[2][3]int32{
	{0, 1, 2},
	{3, 4, 5},
}
`,
		},
		{
			axes: []int{2, 3, 4},
			want: `
[2][3][4]int32{
	{
		{0, 1, 2, 3},
		{4, 5, 6, 7},
		{8, 9, 10, 11},
	},
	{
		{12, 13, 14, 15},
		{16, 17, 18, 19},
		{20, 21, 22, 23},
	},
}
`,
		},
		{
			axes: []int{3, 2, 3, 2},
			want: `
[3][2][3][2]int32{
	{
		{
			{0, 1},
			{2, 3},
			{4, 5},
		},
		{
			{6, 7},
			{8, 9},
			{10, 11},
		},
	},
	{
		{
			{12, 13},
			{14, 15},
			{16, 17},
		},
		{
			{18, 19},
			{20, 21},
			{22, 23},
		},
	},
	{
		{
			{24, 25},
			{26, 27},
			{28, 29},
		},
		{
			{30, 31},
			{32, 33},
			{34, 35},
		},
	},
}
`,
		},
	}
	for i, test := range tests {
		if test.data == nil {
			test.data = buildData(test.axes)
		}
		test.want = strings.TrimSpace(test.want)
		got := fmtarray.Sprint[int32](test.data, test.axes)
		if got != test.want {
			t.Errorf("test %d: incorrect array formatting:\naxes: %v\ndata: %v\ngot:\n%s\nwant:\n%s\n", i, test.axes, test.data, got, test.want)
		}
	}
}
