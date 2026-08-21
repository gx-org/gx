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

// Package algexpr compares algebraic expression to one another.
package algexpr

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/interp/engine"
)

func toString[T fmt.Stringer](xs []T, sep string) string {
	all := make([]string, len(xs))
	for i, x := range xs {
		all[i] = x.String()
	}
	return strings.Join(all, sep)
}

func filterOut(cs []cmp.Comparable, out cmp.Comparable) []cmp.Comparable {
	var keep []cmp.Comparable
	for _, c := range cs {
		if c.Equal(out) {
			continue
		}
		keep = append(keep, c)
	}
	return keep
}

type opF func(typ ir.Type, x, y *big.Float) *big.Float

func evalValues(op opF, typ ir.Type, cs []cmp.Comparable) []cmp.Comparable {
	var r []cmp.Comparable
	var last *big.Float
	for _, c := range cs {
		val, isVal := c.(engine.ScalarConstant)
		if !isVal {
			r = append(r, c)
			continue
		}
		if last == nil {
			last = val.Number().Float()
			continue
		}
		last = op(typ, last, val.Number().Float())
	}
	if last != nil {
		r = append(r, numbers.NewConstantFromFloat(typ, last))
	}
	return r
}
