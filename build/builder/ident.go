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

// valueRef is a reference to a value by an identifier.
type valueRef struct {
	src *ast.Ident
}

var _ exprNode = (*valueRef)(nil)

func processIdentExpr(pscope procScope, src *ast.Ident) (*valueRef, bool) {
	return &valueRef{src: src}, pscope.checkIdent(src)
}

func (n *valueRef) source() ast.Node {
	return n.src
}

func (n *valueRef) buildValueRef(rscope resolveScope) (*ir.ValueRef, bool) {
	name := n.src.Name
	node, ok := rscope.ns().Find(name)
	if !ok {
		return nil, rscope.err().Appendf(n.src, "%s undefined", name)
	}
	procOk := rscope.fileScope().process(node)
	if !procOk {
		return nil, false
	}
	irNode := node.ir()
	if irNode == nil {
		return nil, rscope.err().AppendInternalf(n.src, "%s:%s is defined but has not been built", name, node.token())
	}
	return &ir.ValueRef{
		Src:  n.src,
		Stor: node.ir(),
	}, true
}

func (n *valueRef) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	return n.buildValueRef(rscope)
}

func (n *valueRef) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	valueRef, ok := n.buildValueRef(rscope)
	if !ok {
		return nil, false
	}
	return typeFromStorage(rscope, valueRef, valueRef.Stor)
}

func (n *valueRef) String() string {
	return n.src.Name
}
