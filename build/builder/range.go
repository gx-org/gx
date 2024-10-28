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
	ext        ir.RangeStmt
	key, value *identAssignable
	x          exprNode
	body       *blockStmt
}

var _ stmtNode = (*rangeStmt)(nil)

func processLoopAssignable(owner owner, expr ast.Expr) (*identAssignable, bool) {
	if expr == nil {
		return nil, true
	}
	switch exprT := expr.(type) {
	case *ast.Ident:
		return &identAssignable{target: processIdentExpr(exprT)}, true
	default:
		owner.err().Appendf(expr, "%T not supported", expr)
		return nil, false
	}
}

func processRangeStmt(owner owner, fn *funcDecl, stmt *ast.RangeStmt) (*rangeStmt, bool) {
	n := &rangeStmt{
		ext: ir.RangeStmt{
			Src: stmt,
		},
	}
	var keyOk bool
	n.key, keyOk = processLoopAssignable(owner, stmt.Key)
	var valueOk bool
	n.value, valueOk = processLoopAssignable(owner, stmt.Value)
	var rangeOk bool
	n.x, rangeOk = processExpr(owner, stmt.X)
	var bodyOk bool
	n.body, bodyOk = processBlockStmt(owner, fn, stmt.Body)
	return n, keyOk && valueOk && rangeOk && bodyOk
}

func (n *rangeStmt) source() ast.Node {
	return n.ext.Src
}

func (n *rangeStmt) buildStmt() ir.Stmt {
	n.ext.Key = n.key.buildAssignable()
	if n.value != nil {
		n.ext.Value = n.value.buildAssignable()
	}
	n.ext.X = n.x.buildExpr()
	n.ext.Body = n.body.buildBlockStmt()
	return &n.ext
}

func (n *rangeStmt) resolveTypeOverScalar(scope *scopeBlock, xType typeNode) bool {
	bodyScope := scope.scopeBlock()
	if _, ok := n.key.canAssign(bodyScope, xType); !ok {
		return false
	}
	return n.body.resolveType(bodyScope)
}

func (n *rangeStmt) resolveTypeOverArray(scope *scopeBlock, xType typeNode) bool {
	bodyScope := scope.scopeBlock()
	// xType is an array. So, the key is an integer and the value is the type of the sub-array.
	if _, ok := n.key.canAssign(bodyScope, defaultIntType); !ok {
		return false
	}
	if n.value != nil {
		xArrayType, ok := xType.(*arrayType)
		if !ok {
			scope.err().Appendf(n.ext.Src, "cannot cast %T to *arrayType", xType)
			return false
		}
		valueType, _ := xArrayType.elementType()
		if _, ok := n.value.canAssign(bodyScope, valueType); !ok {
			return false
		}
	}
	return n.body.resolveType(bodyScope)
}

func (n *rangeStmt) resolveType(scope *scopeBlock) bool {
	xType, ok := n.x.resolveType(scope)
	if !ok {
		return false
	}
	if xType.kind() == ir.NumberKind {
		n.x, xType, ok = buildNumberNode(scope, n.x, axisLengthType.buildType())
		if !ok {
			return false
		}
	}
	if ir.IsRangeOk(xType.kind()) {
		return n.resolveTypeOverScalar(scope, xType)
	}
	if xType.kind() == ir.TensorKind {
		return n.resolveTypeOverArray(scope, xType)
	}
	scope.err().Appendf(n.ext.Src, "cannot range over %s", xType.kind().String())
	return false
}
