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
	"fmt"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
)

type sliceExpr struct {
	typ ir.Type
	els []cmp.Expr
}

// NewSlice of algebraic expressions.
func NewSlice(typ ir.Type, els []cmp.Expr) cmp.Expr {
	return &sliceExpr{typ: typ, els: els}
}

func (x *sliceExpr) Simplify(srcf ir.SourceFile) (cmp.Comparable, error) {
	els := make([]cmp.Comparable, len(x.els))
	for i, el := range x.els {
		var err error
		els[i], err = el.Simplify(srcf)
		if err != nil {
			return nil, err
		}
	}
	return &sliceCmp{typ: x.typ, els: els}, nil
}

func (x *sliceExpr) String() string {
	return fmt.Sprintf("[%s]", toString(x.els, ", "))
}

type sliceCmp struct {
	typ ir.Type
	els []cmp.Comparable
}

func (c *sliceCmp) Equal(other cmp.Comparable) bool {
	otherT, ok := other.(*sliceCmp)
	if !ok {
		return false
	}
	if len(c.els) != len(otherT.els) {
		return false
	}
	for i, el := range c.els {
		if !el.Equal(otherT.els[i]) {
			return false
		}
	}
	return true
}

func (c *sliceCmp) BuildIR() ir.Expr {
	elts := make([]ir.Expr, len(c.els))
	for i, el := range c.els {
		elts[i] = el.BuildIR()
	}
	return &ir.SliceLitExpr{
		Typ:  c.typ,
		Elts: elts,
	}
}

func (c *sliceCmp) String() string {
	return fmt.Sprintf("[%s]", toString(c.els, ", "))
}
