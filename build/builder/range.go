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
	"go/ast"

	"github.com/gx-org/gx/build/ir"
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

func (n *rangeStmt) buildBodyOverArray(rscope resolveScope, x ir.Expr) (ir.Storage, ir.Storage, bool) {
	key, _, keyOk := n.key.buildStorage(rscope, ir.DefaultIntType)
	if n.value == nil {
		return key, nil, keyOk
	}
	xUnder := ir.Underlying(x.Type())
	xArrayType, ok := xUnder.(ir.ArrayType)
	if !ok {
		return key, nil, rscope.Err().Appendf(n.x.source(), "%s is not an array type", x.Type().String())
	}
	valueType, ok := xArrayType.ElementType()
	if !ok {
		return key, nil, rscope.Err().Appendf(n.x.source(), "cannot range over array %s with 0 axis", x.Type().String())
	}
	value, _, valueOk := n.value.buildStorage(rscope, valueType)
	return key, value, keyOk && valueOk
}

func (n *rangeStmt) buildStmt(parent fnResolveScope) (ir.Stmt, bool, bool) {
	ext := &ir.RangeStmt{Src: n.src}
	rscope, ok := newBlockScope(parent, n)
	if !ok {
		return ext, false, false
	}
	ext.X, ok = buildAExpr(rscope, n.x)
	if !ok {
		return ext, false, false
	}
	if ir.IsNumber(ext.X.Type().Kind()) {
		ext.X, ok = castNumber(rscope, ext.X, ir.IntLenType())
	}
	if !ok {
		return ext, false, false
	}
	if ir.IsRangeOk(ext.X.Type().Kind()) {
		ext.Key, ext.Value, ok = n.buildBodyOverScalar(rscope, ext.X)
	} else if ext.X.Type().Kind() == ir.ArrayKind {
		ext.Key, ext.Value, ok = n.buildBodyOverArray(rscope, ext.X)
	} else {
		return ext, false, rscope.Err().Appendf(n.src, "cannot range over %s", ext.X.Type().String())
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
