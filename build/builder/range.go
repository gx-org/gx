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

package builder

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/build/ir/unroll"
)

type rangeStmt struct {
	src        *ast.RangeStmt
	key, value *identStorage
	x          exprNode
	body       *blockStmt
}

var _ stmtNode = (*rangeStmt)(nil)

func processRangeStmt(pscope procScope, src *ast.RangeStmt) (*rangeStmt, bool) {
	n := &rangeStmt{src: src}
	var keyOk bool
	n.key, keyOk = processLoopAssignable(pscope, src.Key)
	var valueOk bool
	n.value, valueOk = processLoopAssignable(pscope, src.Value)
	var rangeOk bool
	n.x, rangeOk = processExpr(pscope, src.X)
	var bodyOk bool
	n.body, bodyOk = processBlockStmt(pscope, src.Body)
	return n, keyOk && valueOk && rangeOk && bodyOk
}

func processLoopAssignable(pscope procScope, expr ast.Expr) (*identStorage, bool) {
	if expr == nil {
		return nil, true
	}
	switch exprT := expr.(type) {
	case *ast.Ident:
		target, targetOk := processIdent(pscope, exprT)
		return &identStorage{target: target}, targetOk
	default:
		pscope.Err().Appendf(expr, "%T not supported", expr)
		return nil, false
	}
}

func (n *rangeStmt) source() ast.Node {
	return n.src
}

func (n *rangeStmt) buildBodyOverScalar(rscope resolveScope, x ir.Expr) (ir.Storage, ir.Storage, bool) {
	key, _, keyOk := n.key.buildStorage(rscope, x.Type())
	return key, nil, keyOk
}

func (n *rangeStmt) buildBodyOverSlicer(rscope resolveScope, x ir.Expr) (ir.Storage, ir.Storage, bool) {
	key, _, keyOk := n.key.buildStorage(rscope, ir.IntType())
	if n.value == nil {
		return key, nil, keyOk
	}
	xUnder := ir.Underlying(x.Type())
	slicerType, ok := xUnder.(ir.SlicerType)
	if !ok {
		return key, nil, rscope.Err().AppendInternalf(n.x.source(), "%s is not an array type", x.Type().ReferString(rscope.fileScope().irFile()))
	}
	valueType, ok := slicerType.ElementType()
	if !ok {
		return key, nil, rscope.Err().Appendf(n.x.source(), "cannot range over %s", x.Type().ReferString(rscope.fileScope().irFile()))
	}
	value, _, valueOk := n.value.buildStorage(rscope, valueType)
	return key, value, keyOk && valueOk
}

func (n *rangeStmt) buildStmt(rscope stmtResolveScope) (ir.Stmt, bool, bool) {
	stmt, stop, ok := n.buildRangeStmt(rscope)
	if !ok {
		return stmt, stop, ok
	}
	// Check if range calls a function that requires the loop to be unrolled.
	callExpr, isCall := stmt.X.(*ir.FuncCallExpr)
	if !isCall {
		return stmt, stop, ok
	}
	if !callExpr.Callee.FuncType().Nature.Unroll {
		return stmt, stop, ok
	}
	compEval, ok := rscope.compEval()
	if !ok {
		return stmt, stop, ok
	}
	ext := &ir.UnrollStmt{
		Range: stmt,
	}
	bodiesSrc, ok := unroll.Unroll(compEval, stmt, callExpr)
	if !ok {
		return ext, stop, ok
	}
	fset := token.NewFileSet()
	w := &strings.Builder{}
	for _, body := range bodiesSrc {
		if err := format.Node(w, fset, body); err != nil {
			return ext, stop, rscope.Err().AppendAt(ext.Node(), err)
		}
		fmt.Fprintln(w)
	}
	ext.Source = w.String()
	return ext, stop, ok
}

func (n *rangeStmt) buildRangeStmt(parent stmtResolveScope) (*ir.RangeStmt, bool, bool) {
	ext := &ir.RangeStmt{Src: n.src}
	rscope, ok := newBlockScope(parent, n)
	if !ok {
		return ext, false, false
	}
	ext.X, ok = buildExpr(rscope, n.x)
	if !ok {
		return ext, false, false
	}
	ext.X, ok = castNilAndNumber(rscope, ext.X, ir.IntType())
	if !ok {
		return ext, false, false
	}
	kind := ext.X.Type().Kind()
	switch kind {
	case irkind.Array:
		ext.Key, ext.Value, ok = n.buildBodyOverSlicer(rscope, ext.X)
	case irkind.Slice:
		ext.Key, ext.Value, ok = n.buildBodyOverSlicer(rscope, ext.X)
	default:
		if irkind.IsInteger(kind) {
			ext.Key, ext.Value, ok = n.buildBodyOverScalar(rscope, ext.X)
			break
		}
		return ext, false, rscope.Err().Appendf(n.src, "cannot range over %s", ext.X.Type().ReferString(parent.fileScope().irFile()))
	}
	if !ok {
		return ext, false, false
	}
	if ok = defineLocalVar(rscope, ext.Key); !ok {
		return ext, false, false
	}
	if ext.Value != nil {
		if ok = defineLocalVar(rscope, ext.Value); !ok {
			return ext, false, false
		}
	}
	var stop bool
	ext.Body, stop, ok = n.body.buildBlockStmt(rscope)
	return ext, stop, ok
}
