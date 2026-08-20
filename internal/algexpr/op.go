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
	"go/ast"
	"go/token"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
)

type opCmp struct {
	op  token.Token
	typ ir.Type
	xs  []cmp.Comparable
}

func newOp(op token.Token, typ ir.Type, xs ...cmp.Comparable) *opCmp {
	return &opCmp{op: op, typ: typ, xs: xs}
}

func (c *opCmp) Equal(other cmp.Comparable) bool {
	otherT, ok := other.(*opCmp)
	if !ok {
		return false
	}
	if c.op != otherT.op {
		return false
	}
	if len(c.xs) != len(otherT.xs) {
		return false
	}
	for i, x := range c.xs {
		if !x.Equal(otherT.xs[i]) {
			return false
		}
	}
	return true
}

func (c *opCmp) String() string {
	return fmt.Sprintf("(%s %s)", c.op, toString(c.xs, " "))
}

func (c *opCmp) toOneLessIR() ir.Expr {
	return (&opCmp{
		op:  c.op,
		xs:  c.xs[1:],
		typ: c.typ,
	}).BuildIR()
}

func (c *opCmp) toUnaryIR() ir.Expr {
	switch c.op {
	case token.SUB:
		return &ir.UnaryExpr{
			Src: &ast.UnaryExpr{Op: c.op},
			X:   c.xs[0].BuildIR(),
		}
	case token.NEQ:
		return &ir.UnaryExpr{
			Src: &ast.UnaryExpr{Op: c.op},
			X:   c.xs[0].BuildIR(),
		}
	}
	return c.xs[0].BuildIR()
}

func (c *opCmp) BuildIR() ir.Expr {
	if len(c.xs) == 1 {
		return c.toUnaryIR()
	}
	return &ir.BinaryExpr{
		Src: &ast.BinaryExpr{Op: c.op},
		X:   c.xs[0].BuildIR(),
		Y:   c.toOneLessIR(),
		Typ: c.typ,
	}
}

func unpackOp(op token.Token, cs ...cmp.Comparable) []cmp.Comparable {
	var r []cmp.Comparable
	for _, c := range cs {
		opCmp, isOp := c.(*opCmp)
		if !isOp || opCmp.op != op {
			r = append(r, c)
			continue
		}
		r = append(r, unpackOp(op, opCmp.xs...)...)
	}
	return r
}
