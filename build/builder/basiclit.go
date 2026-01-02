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
	"math/big"

	"github.com/gx-org/gx/build/ir"
)

func processBasicLit(pscope procScope, expr *ast.BasicLit) (exprNode, bool) {
	switch expr.Kind {
	case token.FLOAT:
		val, _, err := new(big.Float).Parse(expr.Value, 0)
		if err != nil {
			pscope.Err().Appendf(expr, "cannot parse float number literal %q: %v", expr.Value, err)
		}
		return &numberLit{
			ext: &ir.NumberFloat{Src: expr, Val: val},
		}, err == nil
	case token.INT:
		val, ok := new(big.Int).SetString(expr.Value, 0)
		if !ok {
			pscope.Err().Appendf(expr, "cannot parse int number literal %q", expr.Value)
		}
		return &numberLit{
			ext: &ir.NumberInt{Src: expr, Val: val},
		}, ok
	case token.STRING:
		return &stringLit{
			ext: ir.StringLiteral{Src: expr},
		}, true
	default:
		return nil, pscope.Err().Appendf(expr, "%s basic literal are not supported", expr.Kind)
	}
}

// stringLit is the literal of a string.
type stringLit struct {
	ext ir.StringLiteral
}

var _ exprNode = (*stringLit)(nil)

func (n *stringLit) buildExpr(resolveScope) (ir.Expr, bool) {
	return &n.ext, true
}

// Pos returns the position of the literal in the code.
func (n *stringLit) source() ast.Node {
	return n.ext.Node()
}

func (n *stringLit) String() string {
	return n.ext.String()
}
