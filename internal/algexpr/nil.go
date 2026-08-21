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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
)

type nilExpr struct {
	x *ir.NilCastExpr
}

// NewNil returns a algebraic expression representing nil.
func NewNil(x *ir.NilCastExpr) cmp.Expr {
	return &nilExpr{x: x}
}

func (n *nilExpr) Simplify(srcf ir.SourceFile) (cmp.Comparable, error) {
	return n, nil
}

func (n *nilExpr) Equal(other cmp.Comparable) bool {
	_, isNil := other.(*nilExpr)
	return isNil
}

func (n *nilExpr) BuildIR() ir.Expr {
	return n.x
}

func (n *nilExpr) String() string {
	return "nil"
}
