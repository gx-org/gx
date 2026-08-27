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

package constants

import (
	"fmt"
	"go/ast"
	"math/big"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/internal/interp/csteager"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/interp/engine"
)

// Scalar is an integer or a float scalar number.
type Scalar interface {
	Constant
	engine.ScalarConstant
	csteager.Eval
	togo.WithGoValue
}

type scalar struct {
	typ ir.Type
	nb  engine.ScalarNumber
}

// NewScalar returns a new scalar element given a typed expression and a number.
func NewScalar(typ ir.Type, nb engine.ScalarNumber) Scalar {
	return &scalar{typ: typ, nb: nb}
}

// Type of the element.
func (n *scalar) Type() ir.Type {
	return n.typ
}

// Build the constant into a given graph.
func (n *scalar) BuildNode(g ops.Graph, kind irkind.Kind) (ops.Node, error) {
	val, err := n.Atom(kind)
	if err != nil {
		return nil, err
	}
	return g.Core().NewAtomLiteral(val)
}

// Number representing the constant scalar.
func (n *scalar) Number() engine.ScalarNumber {
	return n.nb
}

// Expr returns the IR expression represented by the variable.
func (n *scalar) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{n.BuildIR()}, nil
}

func (n *scalar) Simplify(ir.SourceFile) (cmp.Comparable, error) {
	return n, nil
}

func (n *scalar) Equal(other cmp.Comparable) bool {
	otherT, isScalar := other.(*scalar)
	if !isScalar {
		return false
	}
	return n.nb.Float().Cmp(otherT.nb.Float()) == 0
}

func (n *scalar) algExpr() cmp.Expr {
	return n
}

// AlgExpr returns an algebraic expression.
func (n *scalar) AlgExpr(eva ir.Evaluator) (cmp.Expr, error) {
	return n.algExpr(), nil
}

// EvalUnary greedily evaluates an unary expression.
func (n *scalar) EvalUnary(env *engine.Env, expr *ir.UnaryExpr) (engine.Constant, error) {
	nb, err := n.nb.UnaryOp(env, expr)
	return NewScalar(expr.Type(), nb), err
}

func (n *scalar) BuildIR() ir.Expr {
	return &ir.NumberCastExpr{
		Typ: n.Type(),
		X:   n.nb.BuildIR(),
	}
}

// EvalBinary greedily evaluates a binary expression.
func (n *scalar) EvalBinary(env *engine.Env, expr *ir.BinaryExpr, y engine.Constant) (engine.Constant, error) {
	coreY, err := coreiface.ToCore(y)
	if err != nil {
		return nil, err
	}
	yScl, yIsScalar := coreY.(*scalar)
	if !yIsScalar {
		// Evaluation not possible.
		return nil, nil
	}
	cmpEl := n.nb.CmpOp(env, expr, yScl.nb)
	if cmpEl != nil {
		return cmpEl, nil
	}
	nb, err := n.nb.BinaryOp(env, expr, yScl.nb)
	return NewScalar(expr.Type(), nb), err
}

// EvalCast greedily casts an expression.
func (n *scalar) EvalCast(env *engine.Env, expr ir.Expr, tp ir.Type) (engine.Constant, error) {
	return NewScalar(expr.Type(), n.nb), nil
}

// AxisLengths returns the lengths of the axes of the constant.
func (n *scalar) AxisLengths() ([]int, error) {
	return nil, nil
}

// Atom returns the Go value of the scalar.
func (n *scalar) Atom(kind irkind.Kind) (any, error) {
	cvr, err := ConverterFromKind(kind)
	if err != nil {
		return nil, err
	}
	return cvr.Convert(n)
}

func (n *scalar) ShortString(from *ir.File) string {
	return n.String()
}

func (n *scalar) SourceString(from *ir.File) string {
	return fmt.Sprintf("%s(%s)", n.Type().ReferString(from), n.nb.Float().String())
}

func (n *scalar) GoValue() (any, error) {
	cvt := findConverter(n.typ.Kind())
	if cvt != nil {
		return cvt.Convert(n)
	}
	floatVal := n.nb.Float()
	intVal, acc := floatVal.Int(nil)
	if acc == big.Exact {
		return intVal, nil
	}
	return floatVal, nil
}

func (n *scalar) String() string {
	return n.nb.String()
}
