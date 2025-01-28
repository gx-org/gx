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
	"go/token"

	"github.com/gx-org/gx/build/ir"
)

type baseLit struct {
	src *ast.BasicLit
	typ typeNode
}

func (n *baseLit) expr() ast.Expr {
	return n.src
}

// Pos returns the position of the literal in the code.
func (n *baseLit) source() ast.Node {
	return n.src
}

// Type returns the GX type of the literal.
func (n *baseLit) resolveType(scope scoper) (typeNode, bool) {
	return typeNodeOk(n.typ)
}

func (n *baseLit) String() string {
	return n.src.Value
}

func (n *baseLit) buildExpr() ir.Expr {
	return nil
}

func processBasicLit(owner owner, expr *ast.BasicLit) (exprNode, bool) {
	switch expr.Kind {
	case token.FLOAT:
		return &numberLit{
			baseLit: baseLit{
				src: expr,
				typ: numberFloatType,
			},
			ext: &ir.NumberFloat{Src: expr},
		}, true
	case token.INT:
		return &numberLit{
			baseLit: baseLit{
				src: expr,
				typ: numberIntType,
			},
			ext: &ir.NumberInt{Src: expr},
		}, true
	case token.STRING:
		return &stringLit{
			baseLit: baseLit{
				src: expr,
				typ: stringType,
			},
			ext: ir.StringLiteral{Src: expr},
		}, true
	default:
		return &baseLit{
			src: expr,
			typ: invalid,
		}, owner.err().Appendf(expr, "%s basic literal are not supported", expr.Kind)
	}
}

// numberLit is the literal of a number.
type numberLit struct {
	baseLit
	ext ir.Number
}

var _ exprNode = (*numberLit)(nil)

func (n *numberLit) buildExpr() ir.Expr {
	return n.ext
}

// stringLit is the literal of a string.
type stringLit struct {
	baseLit
	ext ir.StringLiteral
}

var _ exprNode = (*stringLit)(nil)

func (n *stringLit) buildExpr() ir.Expr {
	return &n.ext
}
