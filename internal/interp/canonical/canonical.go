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

func (m *pExpr) simplifiedArgs() []Canonical {
	args := m.args()
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
	r := new(big.Float)
	for _, arg := range pe.args() {
		argF := toFloat(arg)
		if argF == nil {
			return nil
		}
		r.Add(r, argF)
	}
	return r
}

func (pe add) Simplify() Simplifier {
	return pe
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
	r := new(big.Float)
	for _, val := range pe.args() {
		evalX, ok := val.(Evaluable)
		if !ok {
			return nil
		}
		x := evalX.Float()
		r.Add(r, new(big.Float).Neg(x))
	}
	return r
}

func (pe sub) Simplify() Simplifier {
	return pe
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
	r := big.NewFloat(1)
	for _, val := range pe.args() {
		valF := toFloat(val)
		if valF == nil {
			return nil
		}
		r.Mul(r, valF)
	}
	return r
}

func toFloat(c Canonical) *big.Float {
	f, ok := c.(Evaluable)
	if !ok {
		return nil
	}
	return f.Float()
}

func (pe mul) Simplify() Simplifier {
	var args []Canonical
	for _, arg := range pe.simplifiedArgs() {
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
	r := big.NewFloat(1)
	for _, arg := range pe.args() {
		argF := toFloat(arg)
		if argF == nil {
			return nil
		}
		r.Mul(r, new(big.Float).Quo(one, argF))
	}
	return r
}

func (pe quo) Simplify() Simplifier {
	return pe
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

// ToValue returns a float value if the canonical expression is evaluable,
// that is if it contains only known values and no symbol.
func ToValue(can Canonical) *big.Float {
	eval, ok := can.(Evaluable)
	if !ok {
		return nil
	}
	return eval.Float()
}
