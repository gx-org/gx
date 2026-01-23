// Copyright 2025 Google LLC
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

type varargs struct {
	src *ast.Ellipsis

	elt typeExprNode
}

func processVarArgsType(pscope typeProcScope, src *ast.Ellipsis) (typeExprNode, bool) {
	elt, ok := processTypeExpr(pscope, src.Elt)
	return &varargs{
		src: src,
		elt: elt,
	}, ok
}

func (n *varargs) source() ast.Node {
	return n.src
}

func (n *varargs) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	elt, ok := n.elt.buildTypeExpr(rscope)
	typ := &ir.VarArgsType{
		Src: n.src,
		Elt: &ir.SliceType{
			BaseType: ir.BaseType[ast.Expr]{Src: n.src},
			DType:    elt,
			Rank:     1,
		},
	}
	return ir.TypeExpr(&ir.VarArgsExpr{
		Elt: typ,
	}, typ), ok
}

func (n *varargs) String() string {
	return "..." + n.elt.String()
}
