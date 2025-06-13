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

package scope

import (
	"testing"
)

type tester interface {
	run(t *testing.T, s Scope[int], testID int)
}

type valueWant struct {
	key string
	val int
	ok  bool
}

func (want valueWant) run(t *testing.T, s Scope[int], testID int) {
	gotVal, gotOk := s.Find(want.key)
	if gotVal != want.val || gotOk != want.ok {
		t.Errorf("test %d: Find(%s) = %v,%v but want %v,%v", testID, want.key, gotVal, gotOk, want.val, want.ok)
	}
}

type assignWant struct {
	key       string
	canAssign bool
}

func (want assignWant) run(t *testing.T, s Scope[int], testID int) {
	got := s.CanAssign(want.key)
	if got != want.canAssign {
		t.Errorf("test %d: CanAssign(%s) = %v but want %v", testID, want.key, got, want.canAssign)
	}
}

func testAll(t *testing.T, s Scope[int], all ...tester) {
	t.Helper()
	for i, tester := range all {
		tester.run(t, s, i)
	}
}

func TestDefine(t *testing.T) {
	s := NewScope[int](nil)
	s.Define("x", 1)
	s.Define("y", 2)

	testAll(t, s,
		valueWant{key: "x", val: 1, ok: true},
		valueWant{key: "y", val: 2, ok: true},
		valueWant{key: "z"},
	)
}

func TestAssign(t *testing.T) {
	s := NewScope[int](nil)
	s.Define("z", -1)
	if err := s.Assign("z", 3); err != nil {
		t.Error(err)
	}

	testAll(t, s, valueWant{key: "z", val: 3, ok: true})

	if err := s.Assign("q", 42); err == nil {
		t.Error("Assign() succeeded, expected failure")
	}
}

func TestNestedScope(t *testing.T) {
	s1 := NewScope[int](nil)
	s1.Define("x", 1)
	s1.Define("z", 20)

	s2 := NewScope(s1)
	s2.Define("x", 10)
	s2.Define("y", 2)

	testAll(t, s1,
		valueWant{key: "x", val: 1, ok: true},
		valueWant{key: "z", val: 20, ok: true},
		valueWant{key: "y"},
	)

	testAll(t, s2,
		valueWant{key: "x", val: 10, ok: true},
		valueWant{key: "z", val: 20, ok: true},
		valueWant{key: "y", val: 2, ok: true},
	)

	if err := s2.Assign("z", 8); err != nil {
		t.Error(err)
	}
	testAll(t, s2, valueWant{key: "z", val: 8, ok: true})
	testAll(t, s1, valueWant{key: "z", val: 8, ok: true})

	if err := s2.Assign("x", 100); err != nil {
		t.Error(err)
	}
	testAll(t, s1, valueWant{key: "x", val: 1, ok: true})
	testAll(t, s2, valueWant{key: "x", val: 100, ok: true})
}

func strToInt(s string) (int, bool) {
	if len(s) != 1 {
		return 0, false
	}
	return int(s[0]), true
}

type scopeF func(string) (int, bool)

func (f scopeF) Find(key string) (int, bool) {
	return f(key)
}

func (f scopeF) CanAssign(key string) bool {
	return false
}

func TestRONestedScope(t *testing.T) {
	s1 := NewScope[int](nil)
	s1.Define("xx", 10)
	s1.Define("zz", 20)

	s2 := NewReadOnly(s1, scopeF(strToInt))
	s3 := NewScope[int](s2)
	s3.Define("a", -1)
	s3.Define("b", -2)
	s4 := NewScope[int](s3)
	s4.Define("c", -3)
	s4.Define("d", -4)
	testAll(t, s4,
		valueWant{key: "xx", val: 10, ok: true},
		valueWant{key: "yy"},
		valueWant{key: "zz", val: 20, ok: true},
		valueWant{key: "A", val: 65, ok: true},
		valueWant{key: "B", val: 66, ok: true},
		valueWant{key: "a", val: -1, ok: true},
		valueWant{key: "b", val: -2, ok: true},
		valueWant{key: "c", val: -3, ok: true},
		valueWant{key: "d", val: -4, ok: true},
		assignWant{key: "xx", canAssign: false},
		assignWant{key: "yy", canAssign: false},
		assignWant{key: "zz", canAssign: false},
		assignWant{key: "A", canAssign: false},
		assignWant{key: "B", canAssign: false},
		assignWant{key: "a", canAssign: true},
		assignWant{key: "b", canAssign: true},
		assignWant{key: "c", canAssign: true},
		assignWant{key: "d", canAssign: true},
	)
}

func TestReadOnly(t *testing.T) {
	s1 := NewScope[int](nil)
	s1.Define("xx", 10)
	s1.Define("zz", 20)
	s2 := NewScope[int](s1.ReadOnly())
	s2.Define("a", -1)
	s2.Define("b", -2)
	testAll(t, s2,
		assignWant{key: "xx", canAssign: false},
		assignWant{key: "zz", canAssign: false},
		assignWant{key: "a", canAssign: true},
		assignWant{key: "b", canAssign: true},
	)
}
