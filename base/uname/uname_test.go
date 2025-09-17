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

package uname_test

import (
	"testing"

	"github.com/gx-org/gx/base/uname"
)

func TestName(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{
			name: "a",
			want: "a",
		},
		{
			name: "a",
			want: "a1",
		},
		{
			name: "a",
			want: "a2",
		},
		{
			name: "b",
			want: "b",
		},
		{
			name: "b",
			want: "b1",
		},
		{
			name: "b",
			want: "b2",
		},
		{
			name: "c",
			want: "c",
		},
	}
	unames := uname.New()
	for i, test := range tests {
		got := unames.Name(test.name)
		if got != test.want {
			t.Errorf("test %d: for name %s, got %s but want %s", i, test.name, got, test.want)
		}
	}
}
