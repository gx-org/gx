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

// numberLit is the literal of a number.
type numberLit struct {
	ext ir.Number
}

var _ exprNode = (*numberLit)(nil)

func (n *numberLit) buildExpr(resolveScope) (ir.Expr, bool) {
	return n.ext, true
}

// Pos returns the position of the literal in the code.
func (n *numberLit) source() ast.Node {
	return n.ext.Node()
}

func (n *numberLit) String() string {
	return n.ext.SourceString(nil)
}
