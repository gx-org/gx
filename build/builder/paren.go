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

	"github.com/gx-org/gx/build/ir"
)

type parenExpr struct {
	src *ast.ParenExpr
	x   exprNode
}

func processParenExpr(pscope procScope, src *ast.ParenExpr) (exprNode, bool) {
	x, ok := processExpr(pscope, src.X)
	return &parenExpr{src: src, x: x}, ok
}

func (n *parenExpr) source() ast.Node {
	return n.src
}

func (n *parenExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	x, ok := n.x.buildExpr(rscope)
	return &ir.ParenExpr{Src: n.src, X: x}, ok
}

func (n *parenExpr) String() string {
	return fmt.Sprintf("(%s)", n.x.String())
}
