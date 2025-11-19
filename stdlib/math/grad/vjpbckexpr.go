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

func (m *exprBackwardVJP) assign(fwdExpr ir.Expr, expr *gradExprResult, suffix string) (*gradExprResult, bool) {
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
		Rhs: []ast.Expr{expr.expr},
	})
	return &gradExprResult{expr: idName}, true
}

func (m *exprBackwardVJP) backward(bck *gradExprResult, expr ir.Expr) (vjp *gradExprResult, ok bool) {
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

func (m *exprBackwardVJP) parenExpr(bck *gradExprResult, expr *ir.ParenExpr) (*gradExprResult, bool) {
	xBack, xOk := m.backward(bck, expr.X)
	if !xOk {
		return nil, false
	}
	if xBack.kind != notSpecial {
		return xBack, true
	}
	return &gradExprResult{
		expr: &ast.ParenExpr{X: xBack.expr},
	}, true
}

func (m *exprBackwardVJP) unaryExpr(bck *gradExprResult, expr *ir.UnaryExpr) (*gradExprResult, bool) {
	xBack, xOk := m.backward(bck, expr.X)
	if !xOk {
		return nil, false
	}
	switch expr.Src.Op {
	case token.SUB:
		return buildUnarySub(xBack), true
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of binary operator %s not supported", expr.Src.Op)
	}
}

func (m *exprBackwardVJP) binaryExpr(bck *gradExprResult, expr *ir.BinaryExpr) (*gradExprResult, bool) {
	xFwd, xOk := m.singleForwardIdent(expr.X)
	yFwd, yOk := m.singleForwardIdent(expr.Y)
	if !xOk || !yOk {
		return nil, false
	}
	x := &gradExprResult{expr: xFwd}
	y := &gradExprResult{expr: yFwd}
	xBack, xOk := m.backward(bck, expr.X)
	yBack, yOk := m.backward(bck, expr.Y)
	if !xOk || !yOk {
		return nil, false
	}
	switch expr.Src.Op {
	case token.ADD:
		return buildAdd(xBack, yBack), true
	case token.SUB:
		return buildSub(xBack, yBack), true
	case token.MUL:
		return buildAdd(
			buildMul(xBack, y),
			buildMul(x, yBack),
		), true
	case token.QUO:
		return buildQuo(
			buildSub(
				buildMul(xBack, y),
				buildMul(x, yBack),
			),
			buildMul(y, y),
		), true
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of binary operator %s not supported", expr.Src.Op)
	}
}

func (m *exprBackwardVJP) numberCastExpr(bck *gradExprResult, expr *ir.NumberCastExpr) (*gradExprResult, bool) {
	src := expr.X.Source().(ast.Expr)
	return zeroValueOf(src), true
}

func (m *exprBackwardVJP) gradFieldStorage(bck *gradExprResult, expr *ir.ValueRef, stor *ir.FieldStorage) (*gradExprResult, bool) {
	if m.wrt.same(stor.Field) {
		return bck, true
	}
	return zeroValueOf(expr.Source()), true
}

func (m *exprBackwardVJP) valueRef(bck *gradExprResult, expr *ir.ValueRef) (*gradExprResult, bool) {
	fieldStorage, isField := expr.Stor.(*ir.FieldStorage)
	if isField {
		return m.gradFieldStorage(bck, expr, fieldStorage)
	}
	return &gradExprResult{expr: gradIdent(expr.Stor.NameDef())}, true
}

func (m *exprBackwardVJP) callTrace(backwardIdents []*gradExprResult) {
	if !traceAll {
		return
	}
	var traceArgs []ast.Expr
	for i, bckExpr := range backwardIdents {
		ident, ok := bckExpr.expr.(*ast.Ident)
		if !ok {
			continue
		}
		traceArgs = append(traceArgs,
			&ast.BasicLit{Value: strconv.Quote(ident.Name)},
			backwardIdents[i].expr,
		)

	}
	m.appendVJPStmt(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  &ast.Ident{Name: "trace"},
			Args: traceArgs,
		},
	})
}

func (m *exprBackwardVJP) callExpr(bck *gradExprResult, expr *ir.FuncCallExpr) (*gradExprResult, bool) {
	if len(expr.Args) == 0 {
		return zeroValueOf(expr.Source()), true
	}
	forwardValues := m.macro.exprToName[expr].(*allocForwardValues)
	calleeParams := expr.Callee.FuncType().Params.Fields()
	backwardIdents := make([]*gradExprResult, len(calleeParams))
	for i, param := range calleeParams {
		vjpCall := &ast.CallExpr{
			Fun:  &ast.Ident{Name: forwardValues.vjps[i]},
			Args: []ast.Expr{bck.expr},
		}
		bckSuffix := ""
		if len(calleeParams) > 1 {
			bckSuffix = param.Name.Name
		}
		var ok bool
		backwardIdents[i], ok = m.assign(expr, &gradExprResult{expr: vjpCall}, bckSuffix)
		if !ok {
			return nil, false
		}
	}
	m.callTrace(backwardIdents)
	argsGrad := make([]*gradExprResult, len(expr.Args))
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
	return buildAdd(argsGrad...), true
}
