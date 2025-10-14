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

func TestRoot(t *testing.T) {
	unames := uname.New()
	for _, name := range []string{"b0", "c1", "c1_1"} {
		unames.Register(name)
	}
	a := unames.Root("a")
	b := unames.Root("b")
	c := unames.Root("c")
	tests := []struct {
		root *uname.Root
		want string
	}{
		{
			root: a,
			want: "a0",
		},
		{
			root: a,
			want: "a1",
		},
		{
			root: b,
			want: "b0_1",
		},
		{
			root: c,
			want: "c0",
		},
		{
			root: c,
			want: "c1_2",
		},
	}
	for i, test := range tests {
		got := test.root.Next().String()
		if got != test.want {
			t.Errorf("test %d: for root %s, got %s but want %s", i, test.root.Root(), got, test.want)
		}
	}
}

func TestNameFor(t *testing.T) {
	unames := uname.New()
	for _, name := range []string{"b1", "b2", "b2_1"} {
		unames.Register(name)
	}
	a := unames.Root("a")
	names := make([]uname.Name, 6)
	for i := range len(names) {
		names[i] = a.Next()
	}
	b := unames.Root("b")
	for i, want := range []string{"b0", "b1_1", "b2_2"} {
		got := names[i].NameFor(b).String()
		if got != want {
			t.Errorf("index %d: got %s but want %s", i, got, want)
		}
	}
}

func TestNameSub(t *testing.T) {
	unames := uname.New()
	for _, name := range []string{"a5X2", "a5X3", "a5X3_1"} {
		unames.Register(name)
	}
	a := unames.Root("a")
	var name uname.Name
	for range 6 {
		name = a.Next()
	}
	root := name.Sub("X")
	for i, want := range []string{"a5X0", "a5X1", "a5X2_1", "a5X3_2"} {
		got := root.Next().String()
		if got != want {
			t.Errorf("index %d: got %s but want %s", i, got, want)
		}
	}

}
