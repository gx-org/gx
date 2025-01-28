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

func TestDefine(t *testing.T) {
	s := NewScope[any, int](nil, nil)
	s.Define("x", 1)
	s.Define("y", 2)

	if value, ok := s.Find("x"); value != 1 || !ok {
		t.Errorf("Find('y') = %v, %v, want 1, true", value, ok)
	}
	if value, ok := s.Find("y"); value != 2 || !ok {
		t.Errorf("Find('y') = %v, %v, want 2, true", value, ok)
	}
	if value, ok := s.Find("z"); value != 0 || ok {
		t.Errorf("Find('z') = %v, %v, want 0, false", value, ok)
	}
}

func TestAssign(t *testing.T) {
	s := NewScope[any, int](nil, nil)
	s.Define("z", -1)
	if err := s.Assign("z", 3); err != nil {
		t.Error(err)
	}

	if value, ok := s.Find("z"); value != 3 || !ok {
		t.Errorf("Find('z') = %v, %v, want 3, true", value, ok)
	}

	if err := s.Assign("q", 42); err == nil {
		t.Error("Assign() succeeded, expected failure")
	}
}

func TestDelete(t *testing.T) {
	s := NewScope[any, int](nil, nil)
	s.Define("x", 1)

	if err := s.Delete("x"); err != nil {
		t.Error(err)
	}
	if err := s.Delete("y"); err == nil {
		t.Error("Delete() succeeded, expected failure")
	}

	if value, ok := s.Find("x"); value != 0 || ok {
		t.Errorf("Find('x') = %v, %v, want 0, false", value, ok)
	}
}

func TestNestedScope(t *testing.T) {
	s1 := NewScope[any, int](nil, nil)
	s1.Define("x", 1)
	s1.Define("z", 20)

	s2 := s1.NewChild(nil)
	s2.Define("x", 10)
	s2.Define("y", 2)

	if value, ok := s1.Find("x"); value != 1 || !ok {
		t.Errorf("s1.Find('x') = %v, %v, want 1, true", value, ok)
	}
	if value, ok := s1.Find("y"); value != 0 || ok {
		t.Errorf("s1.Find('y') = %v, %v, want 0, false", value, ok)
	}
	if value, ok := s2.Find("x"); value != 10 || !ok {
		t.Errorf("s2.Find('x') = %v, %v, want 10, true", value, ok)
	}
	if value, ok := s2.Find("y"); value != 2 || !ok {
		t.Errorf("s2.Find('y') = %v, %v, want 2, true", value, ok)
	}

	if err := s2.Assign("z", 8); err != nil {
		t.Error(err)
	}
	if value, ok := s1.Find("z"); value != 8 || !ok {
		t.Errorf("s1.Find('z') = %v, %v, want 8, true", value, ok)
	}
	if value, ok := s2.Find("z"); value != 8 || !ok {
		t.Errorf("s2.Find('z') = %v, %v, want 8, true", value, ok)
	}
}

func TestImmutableScopeContext(t *testing.T) {
	s := NewImmutableScope(struct{ text string }{text: "foo"}, map[string]any{})
	s.Context().text = "bar"
	if s.Context().text != "foo" {
		t.Error("ImmutableScope.Context() was modified")
	}
}
