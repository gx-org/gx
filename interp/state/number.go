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

package state

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/build/ir"
)

// Number is a GX number.
type Number struct {
	state  *State
	number ir.Expr
}

var _ Element = (*Number)(nil)

// Number returns a new float number state element.
func (s *State) Number(nb ir.Expr) *Number {
	return &Number{state: s, number: nb}
}

// State owning the element.
func (n *Number) State() *State {
	return n.state
}

// Flatten returns the number in a slice of elements.
func (n *Number) Flatten() ([]Element, error) {
	return []Element{n}, nil
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
