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

// basicLit is the literal of a basic type.
type basicLit struct {
	ext ir.Atomic
	src *ast.BasicLit
	typ typeNode
}

var (
	_ exprNumber = (*basicLit)(nil)
	_ exprScalar = (*basicLit)(nil)
)

func processBasicLit(owner owner, expr *ast.BasicLit) (exprNode, bool) {
	n := &basicLit{
		src: expr,
		typ: numberType,
		ext: &ir.Number{Src: expr},
	}
	return n, true
}

func (n *basicLit) expr() ast.Expr {
	return n.src
}

func (n *basicLit) buildExpr() ir.Expr {
	return n.ext
}

func (n *basicLit) castTo(eval evaluator) (exprScalar, []*ir.ValueRef, bool) {
	n.ext = eval.parse(n.src)
	var ok bool
	n.typ, ok = toTypeNode(eval.scoper(), eval.want())
	return n, nil, ok
}

func (n *basicLit) scalar() ir.Atomic {
	return n.ext
}

// Pos returns the position of the literal in the code.
func (n *basicLit) source() ast.Node {
	return n.src
}

// Type returns the GX type of the literal.
func (n *basicLit) resolveType(scope scoper) (typeNode, bool) {
	return typeNodeOk(n.typ)
}

func (n *basicLit) String() string {
	return n.typ.String()
}
