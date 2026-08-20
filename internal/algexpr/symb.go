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

// Symbol is a symbol in a canonical algebra expression.
type Symbol interface {
	// Same returns a function able to compare a symbolic element to another.
	Same(other ir.Element) bool
	// IR returns the expression as an IR expression.
	IR() ir.Expr
}

// Same compares two elements and returns true if they are the same.
type Same func(other ir.Element) bool

type symb struct {
	el   ir.Element
	same Same
	expr ir.Expr
}

// NewSymbFrom returns a new symbol from a comparison function.
func NewSymbFrom(el ir.Element, same Same, x ir.Expr) cmp.Expr {
	return NewSymb(el, &symb{
		el:   el,
		same: same,
		expr: x,
	})
}

// Same returns a function able to compare a symbolic element to another.
func (s *symb) Same(other ir.Element) bool {
	return s.same(other)
}

// IR returns the expression as an IR expression.
func (s *symb) IR() ir.Expr {
	return s.expr
}

type symbExpr struct {
	el   ir.Element
	symb Symbol
}

// NewSymb returns an algebraic expression of a symbolic value.
func NewSymb(el ir.Element, symb Symbol) cmp.Expr {
	return &symbExpr{el: el, symb: symb}
}

func (x *symbExpr) Simplify(ir.SourceFile) (cmp.Comparable, error) {
	return x, nil
}

func (x *symbExpr) Equal(other cmp.Comparable) bool {
	otherT, ok := other.(*symbExpr)
	if !ok {
		return false
	}
	return x.symb.Same(otherT.el)
}

func (x *symbExpr) BuildIR() ir.Expr {
	return x.symb.IR()
}

func (x *symbExpr) String() string {
	return x.symb.IR().SourceString(nil)
}
