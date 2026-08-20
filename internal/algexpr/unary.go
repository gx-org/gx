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
	"go/token"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
)

type unaryExpr struct {
	op token.Token
	x  cmp.Expr
}

// NewUnary returns a new unary algebra expressions.
func NewUnary(op token.Token, x cmp.Expr) (cmp.Expr, error) {
	return &unaryExpr{op: op, x: x}, nil
}

func (e *unaryExpr) Simplify(srcf ir.SourceFile) (cmp.Comparable, error) {
	x, err := e.x.Simplify(srcf)
	if err != nil {
		return nil, err
	}
	return &opCmp{
		op: e.op,
		xs: []cmp.Comparable{x},
	}, nil
}

func (e *unaryExpr) String() string {
	return fmt.Sprintf("(%s %s)", e.op, e.x.String())
}
