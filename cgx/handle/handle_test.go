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

package handle_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/cgx/handle"
)

type item struct {
	A int32
	B string
}

// checkHandleCount compares the current handle count to a reference.
// Cannot use the cgx/testing.CheckHandleCount because of dependency cycle.
func checkHandleCount(t *testing.T, startCount int) {
	endCount := handle.Count()
	if endCount != startCount {
		t.Errorf("handles are leaking: started with %d and ended with %d", startCount, endCount)
	}
}

func TestWrapIntrinsicPointer(t *testing.T) {
	defer checkHandleCount(t, handle.Count())
	var want float32 = 0.42
	w := handle.Wrap[*float32](&want)
	v := handle.Unwrap[*float32](w)
	if *v != want {
		t.Errorf("wrong value: got %v, want %v", *v, want)
	}
	handle.Release(w)
}

func TestWrapStructPointer(t *testing.T) {
	defer checkHandleCount(t, handle.Count())
	type test struct {
		A int32
		B string
	}
	want := &test{A: 42, B: "more data"}
	w := handle.Wrap[*test](want)
	v := handle.Unwrap[*test](w)
	if !cmp.Equal(*v, *want) {
		t.Errorf("wrong value: got %v, want %v", *v, *want)
	}
	handle.Release(w)
}

func TestWrapInterface(t *testing.T) {
	defer checkHandleCount(t, handle.Count())
	buffer := bytes.NewBufferString("the quick brown fox")
	want := buffer.Bytes()
	w := handle.Wrap[io.Reader](buffer)
	v := handle.Unwrap[io.Reader](w)
	if got, err := io.ReadAll(v); err != nil {
		t.Error(err)
	} else if !cmp.Equal(got, want) {
		t.Errorf("wrong value: got %s, want %s", got, want)
	}
	handle.Release(w)
}

func TestWrapInterfacesSlice(t *testing.T) {
	defer checkHandleCount(t, handle.Count())
	readers := []io.Reader{
		bytes.NewBufferString("the"),
		bytes.NewBufferString("quick"),
		bytes.NewBufferString("brown"),
		bytes.NewBufferString("fox"),
	}
	w := handle.WrapSlice[io.Reader](readers)
	var gotText []string
	for _, rw := range w {
		reader := handle.Unwrap[io.Reader](rw)
		got, err := io.ReadAll(reader)
		if err != nil {
			t.Error(err)
			continue
		}
		gotText = append(gotText, string(got))
		handle.Release(rw)
	}

	const want = "the quick brown fox"
	if got := strings.Join(gotText, " "); !cmp.Equal(got, want) {
		t.Errorf("wrong value: got %s, want %s", got, want)
	}
}

type fake struct{}

func BenchmarkWrap(b *testing.B) {
	value := &fake{}
	b.ReportAllocs()
	for range b.N {
		_ = handle.Wrap(value)
	}
}

func BenchmarkUnwrap(b *testing.B) {
	h := handle.Wrap(&fake{})
	b.ReportAllocs()
	for range b.N {
		_ = handle.Unwrap[*fake](h)
	}
}
