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

package elements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// Number is a GX number.
type Number struct {
	number ir.Expr
}

var _ Element = (*Number)(nil)

// NewNumber returns a new float number state element.
func NewNumber(nb ir.Expr) *Number {
	return &Number{number: nb}
}

// Flatten returns the number in a slice of elements.
func (n *Number) Flatten() ([]Element, error) {
	return []Element{n}, nil
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (n *Number) Unflatten(handles *Unflattener) (values.Value, error) {
	return nil, fmterr.Internal(errors.Errorf("%T does not support converting device handles into GX values", n), "")
}

// Atomic returns the number as an ir.Atomic expression.
func (n *Number) Atomic(fetcher ir.Fetcher) (ir.StaticValue, error) {
	kind := n.number.Type().Kind()
	switch kind {
	case ir.NumberFloatKind:
		return evalNumber[float64](n, fetcher)
	case ir.NumberIntKind:
		return evalNumber[int64](n, fetcher)
	default:
		return nil, errors.Errorf("number kind %s not supported", kind)
	}
}

func evalNumber[T dtype.GoDataType](n *Number, fetcher ir.Fetcher) (ir.StaticValue, error) {
	val, _, err := ir.Eval[T](fetcher, n.number)
	if err != nil {
		return nil, err
	}
	return &ir.AtomicValueT[T]{
		Src: n.number.Expr(),
		Typ: ir.TypeFromKind(n.number.Type().Kind()),
		Val: val,
	}, nil
}
