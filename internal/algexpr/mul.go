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

package algexpr

import (
	"math/big"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/numbers"
)

type mul struct {
	*binaryExpr
}

func mulFloat(typ ir.Type, x, y *big.Float) *big.Float {
	return (&big.Float{}).Mul(x, y)
}

func (m *mul) Simplify(srcf ir.SourceFile) (cmp.Comparable, error) {
	x, err := m.x.Simplify(srcf)
	if err != nil {
		return nil, err
	}
	y, err := m.y.Simplify(srcf)
	if err != nil {
		return nil, err
	}
	xs := unpackOp(m.op, x, y)
	xs = evalValues(mulFloat, m.typ, xs)
	one := numbers.One(m.typ)
	xs = filterOut(xs, one)
	switch len(xs) {
	case 0:
		return one, nil
	case 1:
		return xs[0], nil
	default:
		return &opCmp{op: m.op, xs: xs}, nil
	}
}
