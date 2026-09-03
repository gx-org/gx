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

// Package unroll unrolls IR for-loops into AST bodies.
package unroll

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/interp/elements"
)

type unroller struct {
	stmt *ir.RangeStmt
	call *ir.FuncCallExpr
	idx  *ast.BasicLit
	el   ir.Element
	elIR ir.Expr
}

func newUnroller(ev ir.Fetcher, stmt *ir.RangeStmt, call *ir.FuncCallExpr, i int, el ir.Element) (*unroller, error) {
	urlr := &unroller{
		stmt: stmt,
		call: call,
		idx: &ast.BasicLit{
			Kind:  token.INT,
			Value: strconv.Itoa(i),
		},
		el: el,
	}
	var err error
	urlr.elIR, err = ir.ToSingleExpr(ev, call.Expr(), el)
	if err != nil {
		return nil, err
	}
	return urlr, nil
}

func (urlr *unroller) Substitute(ev ir.Fetcher, id *ir.Ident) (ast.Expr, bool) {
	if urlr.stmt.Key != nil && urlr.stmt.Key.Same(id.Store()) {
		return urlr.idx, true
	}
	return id.Src, true
}

// Unroll a for-loop into multiple AST body blocks.
func Unroll(ev ir.Fetcher, stmt *ir.RangeStmt, call *ir.FuncCallExpr) ([]*ast.BlockStmt, bool) {
	list, err := ev.EvalExpr(call)
	if err != nil {
		return nil, ev.Err().AppendAt(call.Node(), err)
	}
	slice, err := cast.To[*elements.Slice](list)
	if err != nil {
		return nil, ev.Err().AppendAt(call.Node(), err)
	}
	bodies := make([]*ast.BlockStmt, slice.Len())
	ok := true
	for i, el := range slice.Elements() {
		urlr, err := newUnroller(ev, stmt, call, i, el)
		if err != nil {
			return nil, ev.Err().AppendAt(call.Node(), err)
		}
		var stmtOk bool
		bodies[i], stmtOk = stmt.Body.UnrollBlock(ev, urlr)
		ok = ok && stmtOk
	}
	return bodies, ok
}
