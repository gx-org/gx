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

package ir

import (
	"fmt"
	"strings"

	"github.com/gx-org/gx/internal/base/cast"
)

// StringSourcer returns the GX source code of the implementation.
type StringSourcer interface {
	SourceString(from *File) string
}

// SourceString returns the source string of an element.
func SourceString(from *File, el Element) string {
	src, err := cast.To[StringSourcer](el)
	if err != nil {
		return err.Error()
	}
	return src.SourceString(from)
}

// StringDefiner returns the GX source code to define the implementation.
type StringDefiner interface {
	DefineString(from *File) string
}

// StringReferer returns the GX source to refer to the implementation.
type StringReferer interface {
	ReferString(from *File) string
}

func sourceStringLiteral(from *File, elts []Expr) string {
	ss := make([]string, len(elts))
	for i, elt := range elts {
		if elt == nil {
			ss[i] = "ERROR:NIL"
			continue
		}
		ss[i] = elt.SourceString(from)
	}
	return fmt.Sprintf("{%s}", strings.Join(ss, ", "))
}

type stringer func() string

func (f stringer) String() string {
	return f()
}

// StringerWithFrom returns an instance of the fmt.Stringer interface using one of the String interface above.
func StringerWithFrom(from *File, a any) fmt.Stringer {
	if a == nil {
		return nil
	}
	switch aT := a.(type) {
	case StringSourcer:
		return stringer(func() string {
			return aT.SourceString(from)
		})
	case StringReferer:
		return stringer(func() string {
			return aT.ReferString(from)
		})
	case StringDefiner:
		return stringer(func() string {
			return aT.DefineString(from)
		})
	default:
		return stringer(func() string {
			return fmt.Sprintf("%T", aT)
		})
	}
}

// Stringer returns an instance of the fmt.Stringer interface using one of the String interface above.
// Should be used only for testing.
func Stringer(a any) fmt.Stringer {
	return StringerWithFrom(nil, a)
}

// String returns the default string representation of a node.
// Should be used only for testing.
func String(a any) string {
	return Stringer(a).String()
}
