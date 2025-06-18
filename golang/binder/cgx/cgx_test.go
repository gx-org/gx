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

package cgx_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/golang/binder/cgx"
)

type item struct {
	A int32
	B string
}

// checkHandleCount compares the current handle count to a reference.
// Cannot use the cgx/testing.CheckHandleCount because of dependency cycle.
func checkHandleCount(t *testing.T, startCount int) {
	endCount := cgx.HandleCount()
	if endCount != startCount {
		t.Errorf("handles are leaking: started with %d and ended with %d", startCount, endCount)
	}
}

func TestWrapIntrinsicPointer(t *testing.T) {
	defer checkHandleCount(t, cgx.HandleCount())
	var want float32 = 0.42
	w := cgx.Wrap[*float32](&want)
	v := cgx.Unwrap[*float32](w)
	if *v != want {
		t.Errorf("wrong value: got %v, want %v", *v, want)
	}
	cgx.Release(w)
}

func TestWrapStructPointer(t *testing.T) {
	defer checkHandleCount(t, cgx.HandleCount())
	type test struct {
		A int32
		B string
	}
	want := &test{A: 42, B: "more data"}
	w := cgx.Wrap[*test](want)
	v := cgx.Unwrap[*test](w)
	if !cmp.Equal(*v, *want) {
		t.Errorf("wrong value: got %v, want %v", *v, *want)
	}
	cgx.Release(w)
}

func TestWrapInterface(t *testing.T) {
	defer checkHandleCount(t, cgx.HandleCount())
	buffer := bytes.NewBufferString("the quick brown fox")
	want := buffer.Bytes()
	w := cgx.Wrap[io.Reader](buffer)
	v := cgx.Unwrap[io.Reader](w)
	if got, err := io.ReadAll(v); err != nil {
		t.Error(err)
	} else if !cmp.Equal(got, want) {
		t.Errorf("wrong value: got %s, want %s", got, want)
	}
	cgx.Release(w)
}

func TestWrapInterfacesSlice(t *testing.T) {
	defer checkHandleCount(t, cgx.HandleCount())
	readers := []io.Reader{
		bytes.NewBufferString("the"),
		bytes.NewBufferString("quick"),
		bytes.NewBufferString("brown"),
		bytes.NewBufferString("fox"),
	}
	w := cgx.WrapSlice[io.Reader](readers)
	gotText := []string{}
	for _, rw := range w {
		reader := cgx.Unwrap[io.Reader](rw)
		got, err := io.ReadAll(reader)
		if err != nil {
			t.Error(err)
			continue
		}
		gotText = append(gotText, string(got))
		cgx.Release(rw)
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
		_ = cgx.Wrap(value)
	}
}

func BenchmarkUnwrap(b *testing.B) {
	handle := cgx.Wrap(&fake{})
	b.ReportAllocs()
	for range b.N {
		_ = cgx.Unwrap[*fake](handle)
	}
}
