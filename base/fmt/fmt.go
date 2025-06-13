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

// Package fmt provides utility methods for building string representations of GX objects.
package fmt

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"slices"
	"strings"
)

// Number adds a number prefix to all lines in a string.
func Number(x string) string {
	lines := slices.Collect(strings.Lines(x))
	numDigits := int(math.Log10(float64(len(lines)))) + 1
	fmtString := fmt.Sprintf("%%0%dd %%s", numDigits)
	var s strings.Builder
	for i, line := range lines {
		s.WriteString(fmt.Sprintf(fmtString, i+1, line))
	}
	return s.String()
}

// IndentSkip skips some lines and indent the rest with a tabulation.
func IndentSkip(skip int, x string) string {
	var y strings.Builder
	n := 0
	for line := range strings.Lines(x) {
		if n >= skip {
			y.WriteString("\t")
		}
		y.WriteString(line)
		n++
	}
	return y.String()
}

// Indent the given string by a tabulation.
func Indent(x string) string {
	return IndentSkip(0, x)
}

func sliceString(x any) string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("%T{\n", x))
	val := reflect.ValueOf(x)
	for i := range val.Len() {
		s.WriteString(Indent(fmt.Sprintf("%d: %s,\n", i, String(val.Index(i).Interface()))))
	}
	s.WriteString("}")
	return s.String()
}

// Func returns the name of a function.
func Func(f any) string {
	if f == nil {
		return "<nil>"
	}
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// String returns a human-friendly debugging string representation of a GX object.
func String(x any) string {
	if x == nil {
		return "nil"
	}
	if reflect.ValueOf(x).IsNil() {
		return fmt.Sprintf("%T(nil)", x)
	}
	if reflect.TypeOf(x).Kind() == reflect.Slice {
		return sliceString(x)
	}
	strng, ok := x.(fmt.Stringer)
	if ok {
		return strng.String()
	}
	return fmt.Sprintf("%T", x)
}
