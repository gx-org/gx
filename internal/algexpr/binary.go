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

type binaryExpr struct {
	op   token.Token
	typ  ir.Type
	x, y cmp.Expr
}

// NewBinary returns a new binary algebra expressions.
func NewBinary(op token.Token, typ ir.Type, x, y cmp.Expr) (cmp.Expr, error) {
	bin := &binaryExpr{op: op, typ: typ, x: x, y: y}
	switch op {
	case token.ADD:
		return &add{binaryExpr: bin}, nil
	case token.MUL:
		return &mul{binaryExpr: bin}, nil
	case token.EQL:
		return &eql{binaryExpr: bin}, nil
	}
	return bin, nil
}

func (e *binaryExpr) Simplify(srcf ir.SourceFile) (cmp.Comparable, error) {
	x, err := e.x.Simplify(srcf)
	if err != nil {
		return nil, err
	}
	y, err := e.y.Simplify(srcf)
	if err != nil {
		return nil, err
	}
	return newOp(e.op, e.typ, x, y), nil
}

func (e *binaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", e.op, e.x.String(), e.y.String())
}
