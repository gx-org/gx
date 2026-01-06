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

// ident is a reference to a value by an identifier.
type ident struct {
	src *ast.Ident
}

var _ exprNode = (*ident)(nil)

func processIdent(pscope procScope, src *ast.Ident) (*ident, bool) {
	return &ident{src: src}, true
}

func (n *ident) source() ast.Node {
	return n.src
}

func (n *ident) buildIdent(rscope resolveScope) (*ir.Ident, bool) {
	storage, ok := findStorage(rscope, n.src)
	if !ok {
		return nil, false
	}
	return &ir.Ident{
		Src:  n.src,
		Stor: storage,
	}, true
}

func (n *ident) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	return n.buildIdent(rscope)
}

func (n *ident) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	idt, ok := n.buildIdent(rscope)
	if !ok {
		return nil, false
	}
	return typeFromStorage(rscope, idt, idt.Stor)
}

func (n *ident) String() string {
	return n.src.Name
}
