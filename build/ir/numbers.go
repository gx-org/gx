// Copyright 2026 Google LLC
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
	"reflect"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir/irkind"
)

// FileWithError is an interface able to return a file context and an error appender.
type FileWithError interface {
	File() *File
	fmterr.ErrAppender
}

// CastNumber builds a number cast expression.
func CastNumber(fetcher FileWithError, expr Expr, target Type) (*NumberCastExpr, bool) {
	cast := &NumberCastExpr{
		X:   expr,
		Typ: target,
	}
	if cast.Typ.Kind() == irkind.Unknown {
		// No specification on what we want.
		// For example:
		//   a := 5.2
		// Then we cast the number, 5.2 in this example, to a default type.
		cast.Typ = DefaultNumberType(expr.Type().Kind())
	}
	if cast.Typ.Kind() == irkind.Array {
		// The required type is an array. For example:
		//   a := 5 * [2]float32{1, 2}
		// We cast the number, 5 in this example, to the data type of the array.
		underlying := Underlying(cast.Typ)
		arrayType, ok := underlying.(ArrayType)
		if !ok {
			return cast, fetcher.Err().AppendInternalf(expr.Node(), "type %T has %s but does not implement %s", underlying, irkind.Array.String(), reflect.TypeFor[ArrayType]().Name())
		}
		cast.Typ = arrayType.DataType()
	}
	if !CanBeNumber(cast.Typ) {
		from := fetcher.File()
		// Return an error for code like:
		// func f() string { return 3 }
		return cast, fetcher.Err().Appendf(expr.Node(), "cannot use a number as %v", cast.Typ.ReferString(from))
	}
	if IsFloat(expr.Type()) && IsInteger(cast.Typ) {
		from := fetcher.File()
		return cast, fetcher.Err().Appendf(expr.Node(), "cannot use %s (untyped FLOAT constant) as %s value", expr.SourceString(from), cast.Typ.ReferString(from))
	}
	return cast, true
}
