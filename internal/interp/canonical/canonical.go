// Copyright 2025 Google LLC
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

// Package canonical implements canonical expressions that can be compared to one another.
package canonical

import (
	"fmt"
	"go/token"
	"math/big"
	"sort"
	"strings"
)

type (

	// Comparable defines an interface for comparing an expression against another.
	Comparable interface {
		Compare(x Comparable) bool
	}

	// Canonical is a canonical expression.
	Canonical interface {
		Comparable
		fmt.Stringer
	}

	// Evaluable is a canonical that can be evaluated to a float.
	Evaluable interface {
		// Float value of the expression.
		Float() *big.Float
	}

	// Simplifier is an expression that can be simplified.
	Simplifier interface {
		Canonical
		Simplify() Simplifier
	}

	exprStr struct {
		expr Canonical
		str  string
	}

	pExpr struct {
		exprs []Canonical
		str   string
	}

	pExprOp interface {
		Evaluable
		args() []Canonical
		op() token.Token
	}

	add struct {
		*pExpr
	}

	sub struct {
		*pExpr
	}

	mul struct {
		*pExpr
	}

	quo struct {
		*pExpr
	}

	unknown struct {
		tk token.Token
		*pExpr
	}
)

// FromBinary returns a canonical expression from a binary expression.
func FromBinary(op token.Token, x, y Canonical) Simplifier {
	switch op {
	case token.MUL:
		return mul{pExpr: prefixedExpr(x, y)}
	case token.QUO:
		return mul{pExpr: prefixedExpr(x, quo{prefixedExpr(y)})}
	case token.ADD:
		return add{pExpr: prefixedExpr(x, y)}
	case token.SUB:
		return add{pExpr: prefixedExpr(x, sub{prefixedExpr(y)})}
	default:
		return unknown{tk: op, pExpr: prefixedExpr(x, y)}
	}
}

// NewExpr returns a new canonical expression using a prefixed form.
func NewExpr(op token.Token, xs ...Canonical) Simplifier {
	return newExpr(op, xs...)
}

func newExpr(op token.Token, xs ...Canonical) Simplifier {
	switch op {
	case token.MUL:
		return mul{pExpr: prefixedExpr(xs...)}
	case token.QUO:
		return quo{pExpr: prefixedExpr(xs...)}
	case token.ADD:
		return add{pExpr: prefixedExpr(xs...)}
	case token.SUB:
		return sub{pExpr: prefixedExpr(xs...)}
	default:
		return unknown{tk: op, pExpr: prefixedExpr(xs...)}
	}
}

func prefixedExpr(xs ...Canonical) *pExpr {
	cexprs := make([]*exprStr, len(xs))
	for i, x := range xs {
		cexprs[i] = &exprStr{
			expr: x,
			str:  x.String(),
		}
	}
	sort.Slice(cexprs, func(i, j int) bool {
		return cexprs[i].str < cexprs[j].str
	})
	strs := make([]string, len(xs))
	exprs := make([]Canonical, len(xs))
	for i, cexpr := range cexprs {
		exprs[i] = cexpr.expr
		strs[i] = cexpr.str
	}
	return &pExpr{
		exprs: exprs,
		str:   strings.Join(strs, " "),
	}
}

func (m *pExpr) compare(op token.Token, other Comparable) bool {
	otherT, ok := other.(pExprOp)
	if !ok {
		return false
	}
	if op != otherT.op() {
		return false
	}
	oExprs := otherT.args()
	if len(m.exprs) != len(oExprs) {
		return false
	}
	for i, mi := range m.exprs {
		if !mi.Compare(oExprs[i]) {
			return false
		}
	}
	return true
}

func (m *pExpr) string(pe pExprOp) string {
	return fmt.Sprintf("(%s %s)", pe.op(), m.str)
}

func (m *pExpr) args() []Canonical {
	return m.exprs
}

func apply(f func(Canonical) Canonical, xs []Canonical) []Canonical {
	rs := make([]Canonical, len(xs))
	for i, x := range xs {
		rs[i] = f(x)
	}
	return rs
}

func (pe add) op() token.Token {
	return token.ADD
}

func (pe add) Compare(other Comparable) bool {
	return pe.compare(pe.op(), other)
}

func (pe add) Float() *big.Float {
	floats, cans := floatArgs(pe)
	if len(cans) > 0 {
		return nil
	}
	r := new(big.Float)
	for _, f := range floats {
		r.Add(r, f)
	}
	return r
}

func (pe add) Simplify() Simplifier {
	return add{pExpr: prefixedExpr(simplifyArgs(pe)...)}
}

func (pe add) String() string {
	return pe.pExpr.string(pe)
}

func (pe sub) op() token.Token {
	return token.SUB
}

func (pe sub) Compare(other Comparable) bool {
	return pe.compare(pe.op(), other)
}

func (pe sub) Float() *big.Float {
	floats, cans := floatArgs(pe)
	if len(cans) > 0 {
		return nil
	}
	r := new(big.Float)
	for _, f := range floats {
		r.Add(r, new(big.Float).Neg(f))
	}
	return r
}

func (pe sub) Simplify() Simplifier {
	return sub{pExpr: prefixedExpr(simplifyArgs(pe)...)}
}

func (pe sub) String() string {
	return pe.pExpr.string(pe)
}

func (pe mul) op() token.Token {
	return token.MUL
}

func (pe mul) Compare(other Comparable) bool {
	return pe.compare(pe.op(), other)
}

func (pe mul) Float() *big.Float {
	floats, cans := floatArgs(pe)
	if len(cans) > 0 {
		return nil
	}
	r := big.NewFloat(1)
	for _, f := range floats {
		r.Mul(r, f)
	}
	return r
}

func (pe mul) Simplify() Simplifier {
	var args []Canonical
	for _, arg := range simplifyArgs(pe) {
		argF := toFloat(arg)
		if argF != nil && argF.Cmp(one) == 0 {
			continue
		}
		argT, ok := arg.(mul)
		if !ok {
			args = append(args, arg)
			continue
		}
		args = append(args, argT.args()...)
	}
	return mul{pExpr: prefixedExpr(args...)}
}

func (pe mul) String() string {
	return pe.pExpr.string(pe)
}

func (pe quo) op() token.Token {
	return token.QUO
}

func (pe quo) Compare(other Comparable) bool {
	return pe.compare(pe.op(), other)
}

var one = big.NewFloat(1)

func (pe quo) Float() *big.Float {
	floats, cans := floatArgs(pe)
	if len(cans) > 0 {
		return nil
	}
	r := big.NewFloat(1)
	for _, f := range floats {
		r.Mul(r, new(big.Float).Quo(one, f))
	}
	return r
}

func (pe quo) Simplify() Simplifier {
	return quo{pExpr: prefixedExpr(simplifyArgs(pe)...)}
}

func (pe quo) String() string {
	return pe.pExpr.string(pe)
}

func (pe unknown) op() token.Token {
	return pe.tk
}

func (pe unknown) Compare(other Comparable) bool {
	return pe.compare(pe.op(), other)
}

func (pe unknown) Float() *big.Float {
	return nil
}

func (pe unknown) Simplify() Simplifier {
	return pe
}

func (pe unknown) String() string {
	return pe.pExpr.string(pe)
}

func toFloat(x Canonical) *big.Float {
	eval, isEval := x.(Evaluable)
	if !isEval {
		return nil
	}
	return eval.Float()
}

func floatArgs(op pExprOp) (floats []*big.Float, vars []Canonical) {
	for _, arg := range op.args() {
		val := toFloat(arg)
		if val != nil {
			floats = append(floats, val)
		} else {
			vars = append(vars, arg)
		}
	}
	return
}

func simplifyArgs(op pExprOp) []Canonical {
	args := op.args()
	r := make([]Canonical, len(args))
	for i, arg := range args {
		simplifier, ok := arg.(Simplifier)
		if !ok {
			r[i] = arg
		} else {
			r[i] = simplifier.Simplify()
		}
	}
	return r
}

// ToValue returns a float value if the canonical expression is evaluable,
// that is if it contains only known values and no symbol.
func ToValue(el any) *big.Float {
	eval, ok := el.(Evaluable)
	if !ok {
		return nil
	}
	return eval.Float()
}
