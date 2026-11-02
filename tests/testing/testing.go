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

// Package testing provides functions to run gx tests.
package testing

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
)

// NumberLines returns a string where lines are prefixed by their number.
func NumberLines(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return s
	}
	padding := int(math.Ceil(math.Log10(float64(len(lines)))))
	paddingS := fmt.Sprintf("%%0%dd ", padding)
	for i, line := range lines {
		lines[i] = fmt.Sprintf(paddingS, i+1) + line
	}
	return strings.Join(lines, "\n")
}

// FetchAtom fetches an atomic value from a device.
func FetchAtom[T gotype.Supported](t *testing.T, atom types.Atom[T]) T {
	if atom == nil {
		t.Fatalf("cannot fetch value from a nil atom")
	}
	value, err := atom.Fetch()
	if err != nil {
		t.Fatalf("cannot fetch value:\n%+v", err)
	}
	return value.Value()
}

// FetchArray fetches an array from a device.
func FetchArray[T gotype.Supported](t *testing.T, array types.Array[T]) []T {
	if array == nil {
		t.Fatalf("cannot fetch value from a nil array")
	}
	value, err := array.Fetch()
	if err != nil {
		t.Fatalf("cannot fetch value:\n%+v", err)
	}
	return value.CopyFlat()
}
