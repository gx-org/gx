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
	"fmt"
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
	v, err := handle.Unwrap[*float32](w)
	if err != nil {
		t.Fatal(err)
	}
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
	v, err := handle.Unwrap[*test](w)
	if err != nil {
		t.Fatal(err)
	}
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
	v, err := handle.Unwrap[io.Reader](w)
	if err != nil {
		t.Fatal(err)
	}
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
		reader, err := handle.Unwrap[io.Reader](rw)
		if err != nil {
			t.Error(err)
			continue
		}
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

func TestTryUnwrap(t *testing.T) {
	defer checkHandleCount(t, handle.Count())
	want := &item{A: 42, B: "more data"}
	h := handle.Wrap[*item](want)
	defer handle.Release(h)
	if got, ok := handle.TryUnwrap[*item](h); !ok || got != want {
		t.Errorf("TryUnwrap[*item](%v) = %v, %t, want %v, true", h, got, ok, want)
	}
	if got, ok := handle.TryUnwrap[*float32](h); ok {
		t.Errorf("TryUnwrap[*float32](%v) = %v, true, want false for a handle to a *item", h, got)
	}

	released := handle.Wrap[*item](&item{})
	handle.Release(released)
	for _, tc := range []struct {
		desc string
		h    handle.Handle
	}{
		{desc: "zero", h: 0},
		{desc: "released", h: released},
		{desc: "unknown", h: ^handle.Handle(0)},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			if got, ok := handle.TryUnwrap[*item](tc.h); ok {
				t.Errorf("TryUnwrap[*item](%v) = %v, true, want false", tc.h, got)
			}
		})
	}
}

func catchPanic(t *testing.T, fn func()) (msg string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic, got none")
		}
		msg = fmt.Sprint(r)
	}()
	fn()
	return ""
}

func TestUnwrap(t *testing.T) {
	defer checkHandleCount(t, handle.Count())
	want := &item{A: 42, B: "more data"}
	h := handle.Wrap[*item](want)
	defer handle.Release(h)

	if got, err := handle.Unwrap[*item](h); err != nil || got != want {
		t.Errorf("Unwrap[*item](%v) = %v, %v, want %v, nil", h, got, err, want)
	}

	released := handle.Wrap[*item](&item{})
	handle.Release(released)
	const unknown = ^handle.Handle(0)

	for _, tc := range []struct {
		desc        string
		unwrap      func() error
		wantSubstrs []string
	}{
		{
			desc: "zero",
			unwrap: func() error {
				_, err := handle.Unwrap[*item](0)
				return err
			},
			wantSubstrs: []string{"invalid handle 0 (zero handle), want a handle to a *handle_test.item"},
		},
		{
			desc: "released",
			unwrap: func() error {
				_, err := handle.Unwrap[*item](released)
				return err
			},
			wantSubstrs: []string{fmt.Sprintf("invalid handle %v (released or never created), want a handle to a *handle_test.item", released)},
		},
		{
			desc: "unknown",
			unwrap: func() error {
				_, err := handle.Unwrap[*item](unknown)
				return err
			},
			wantSubstrs: []string{fmt.Sprintf("invalid handle %v (released or never created), want a handle to a *handle_test.item", unknown)},
		},
		{
			desc: "wrong_type",
			unwrap: func() error {
				_, err := handle.Unwrap[*float32](h)
				return err
			},
			wantSubstrs: []string{
				fmt.Sprintf("invalid handle %v:", h),
				"cannot convert *handle_test.item to *float32",
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.unwrap()
			if err == nil {
				t.Fatal("Unwrap succeeded, want an error")
			}
			got := err.Error()
			for _, want := range tc.wantSubstrs {
				if !strings.Contains(got, want) {
					t.Errorf("Unwrap error = %q, want substring %q", got, want)
				}
			}
		})
	}
}

func TestReleasePanics(t *testing.T) {
	defer checkHandleCount(t, handle.Count())
	handle.Release(0)

	released := handle.Wrap[*item](&item{})
	handle.Release(released)
	const unknown = ^handle.Handle(0)

	for _, tc := range []struct {
		desc string
		h    handle.Handle
	}{
		{desc: "released", h: released},
		{desc: "unknown", h: unknown},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got := catchPanic(t, func() { handle.Release(tc.h) })
			want := fmt.Sprintf("cgx: deleting unknown handle %v (already released or never created)", tc.h)
			if got != want {
				t.Errorf("Release(%v) panic = %q, want %q", tc.h, got, want)
			}
		})
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
		_, _ = handle.Unwrap[*fake](h)
	}
}
