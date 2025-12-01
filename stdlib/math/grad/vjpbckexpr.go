// Copyright 2024 Google LLC
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

package grad

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/gx-org/gx/base/uname"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/math/grad/special"
)

// exprBackwardVJP decomposes expressions into elementary assignment statements.
type exprBackwardVJP struct {
	*stmtVJP
	wrt   withRespectTo
	stmts []ast.Stmt
}

func (m *stmtVJP) newExprBackwardVJP(wrt withRespectTo) *exprBackwardVJP {
	return &exprBackwardVJP{stmtVJP: m, wrt: wrt}
}

func (m *exprBackwardVJP) singleForwardValue(expr ir.Expr) (forwardValues, bool) {
	fv, ok := m.macro.exprToName[expr]
	if !ok {
		return nil, m.fetcher.Err().AppendInternalf(expr.Source(), "forward expression %T:%s has no name", expr, expr.String())
	}
	if fv.numVars() != 1 {
		return nil, m.fetcher.Err().AppendInternalf(expr.Source(), "forward expression %T:%s has multiple names in a single value context", expr, expr.String())
	}
	return fv, true
}

func (m *exprBackwardVJP) singleForwardName(expr ir.Expr) (_ uname.Name, ok bool) {
	fv, ok := m.singleForwardValue(expr)
	if !ok {
		return
	}
	alloc, ok := fv.(*allocForwardValues)
	if !ok {
		m.fetcher.Err().AppendInternalf(expr.Source(), "forward expression %T:%s has no allocated name", expr, expr.String())
		return
	}
	return alloc.forwards[0], true
}

func (m *exprBackwardVJP) singleForwardIdent(expr ir.Expr) (ast.Expr, bool) {
	if paren, isParen := expr.(*ir.ParenExpr); isParen {
		return m.singleForwardIdent(paren.X)
	}
	fv, ok := m.singleForwardValue(expr)
	if !ok {
		return nil, false
	}
	return fv.exprs()[0], true
}

func (m *exprBackwardVJP) appendVJPStmt(stmt ast.Stmt) {
	m.stmts = append(m.stmts, stmt)
}

func (m *exprBackwardVJP) assign(fwdExpr ir.Expr, expr *special.Expr, suffix string) (*special.Expr, bool) {
	fwdName, ok := m.singleForwardName(fwdExpr)
	if !ok {
		return nil, false
	}
	bckNameRoot := fwdName.NameFor(m.macro.bckRoot)
	bckName := bckNameRoot.String()
	if suffix != "" {
		bckName = m.macro.unames.Name(bckName + suffix)
	}
	idName := &ast.Ident{Name: bckName}
	m.appendVJPStmt(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{idName},
		Rhs: []ast.Expr{expr.Expr},
	})
	return &special.Expr{Expr: idName}, true
}

func (m *exprBackwardVJP) backward(bck *special.Expr, expr ir.Expr) (vjp *special.Expr, ok bool) {
	switch exprT := expr.(type) {
	case *ir.FuncCallExpr:
		return m.callExpr(bck, exprT)
	case *ir.NumberCastExpr:
		return m.numberCastExpr(bck, exprT)
	case *ir.ValueRef:
		return m.valueRef(bck, exprT)
	case *ir.UnaryExpr:
		return m.unaryExpr(bck, exprT)
	case *ir.BinaryExpr:
		return m.binaryExpr(bck, exprT)
	case *ir.ParenExpr:
		return m.parenExpr(bck, exprT)
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of %T expression not supported", exprT)
	}
}

func (m *exprBackwardVJP) parenExpr(bck *special.Expr, expr *ir.ParenExpr) (*special.Expr, bool) {
	xBack, xOk := m.backward(bck, expr.X)
	if !xOk {
		return nil, false
	}
	if xBack.Value != special.Any {
		return xBack, true
	}
	return &special.Expr{
		Expr: &ast.ParenExpr{X: xBack.Expr},
	}, true
}

func (m *exprBackwardVJP) unaryExpr(bck *special.Expr, expr *ir.UnaryExpr) (*special.Expr, bool) {
	xBack, xOk := m.backward(bck, expr.X)
	if !xOk {
		return nil, false
	}
	switch expr.Src.Op {
	case token.SUB:
		return special.UnarySub(xBack), true
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of binary operator %s not supported", expr.Src.Op)
	}
}

func (m *exprBackwardVJP) binaryExpr(bck *special.Expr, expr *ir.BinaryExpr) (*special.Expr, bool) {
	xFwd, xOk := m.singleForwardIdent(expr.X)
	yFwd, yOk := m.singleForwardIdent(expr.Y)
	if !xOk || !yOk {
		return nil, false
	}
	x := &special.Expr{Expr: xFwd}
	y := &special.Expr{Expr: yFwd}
	xBack, xOk := m.backward(bck, expr.X)
	yBack, yOk := m.backward(bck, expr.Y)
	if !xOk || !yOk {
		return nil, false
	}
	switch expr.Src.Op {
	case token.ADD:
		return special.Add(xBack, yBack), true
	case token.SUB:
		return special.Sub(xBack, yBack), true
	case token.MUL:
		return special.Add(
			special.Mul(xBack, y),
			special.Mul(x, yBack),
		), true
	case token.QUO:
		return special.Quo(
			special.Sub(
				special.Mul(xBack, y),
				special.Mul(x, yBack),
			),
			special.Mul(y, y),
		), true
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of binary operator %s not supported", expr.Src.Op)
	}
}

func (m *exprBackwardVJP) numberCastExpr(bck *special.Expr, expr *ir.NumberCastExpr) (*special.Expr, bool) {
	return special.ZeroExpr(), true
}

func (m *exprBackwardVJP) gradFieldStorage(bck *special.Expr, expr *ir.ValueRef, stor *ir.FieldStorage) (*special.Expr, bool) {
	if m.wrt.same(stor.Field) {
		return bck, true
	}
	return special.ZeroExpr(), true
}

func (m *exprBackwardVJP) valueRef(bck *special.Expr, expr *ir.ValueRef) (*special.Expr, bool) {
	fieldStorage, isField := expr.Stor.(*ir.FieldStorage)
	if isField {
		return m.gradFieldStorage(bck, expr, fieldStorage)
	}
	return &special.Expr{Expr: gradIdent(expr.Stor.NameDef())}, true
}

func (m *exprBackwardVJP) callTrace(backwardIdents []*special.Expr) {
	if !traceAll {
		return
	}
	var traceArgs []ast.Expr
	for i, bckExpr := range backwardIdents {
		ident, ok := bckExpr.Expr.(*ast.Ident)
		if !ok {
			continue
		}
		traceArgs = append(traceArgs,
			&ast.BasicLit{Value: strconv.Quote(ident.Name)},
			backwardIdents[i].Expr,
		)

	}
	m.appendVJPStmt(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  &ast.Ident{Name: "trace"},
			Args: traceArgs,
		},
	})
}

func (m *exprBackwardVJP) callExpr(bck *special.Expr, expr *ir.FuncCallExpr) (*special.Expr, bool) {
	if len(expr.Args) == 0 {
		return special.ZeroExpr(), true
	}
	forwardValues := m.macro.exprToName[expr].(*allocForwardValues)
	calleeParams := expr.Callee.FuncType().Params.Fields()
	backwardIdents := make([]*special.Expr, len(calleeParams))
	for i, param := range calleeParams {
		vjpCall := &ast.CallExpr{
			Fun:  &ast.Ident{Name: forwardValues.vjps[i]},
			Args: []ast.Expr{bck.Expr},
		}
		bckSuffix := ""
		if len(calleeParams) > 1 {
			bckSuffix = param.Name.Name
		}
		var ok bool
		backwardIdents[i], ok = m.assign(expr, &special.Expr{Expr: vjpCall}, bckSuffix)
		if !ok {
			return nil, false
		}
	}
	m.callTrace(backwardIdents)
	argsGrad := make([]*special.Expr, len(expr.Args))
	for i := len(expr.Args) - 1; i >= 0; i-- {
		var ok bool
		argsGrad[i], ok = m.backward(backwardIdents[i], expr.Args[i])
		if !ok {
			return nil, false
		}
	}
	if len(argsGrad) == 1 {
		return argsGrad[0], true
	}
	return special.Add(argsGrad...), true
}
