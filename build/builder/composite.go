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
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

type compositeLit struct {
	src  *ast.CompositeLit
	elts []exprNode
}

func processUntypedCompositeLit(pscope procScope, src *ast.CompositeLit) (*compositeLit, bool) {
	lit := &compositeLit{src: src, elts: make([]exprNode, len(src.Elts))}
	ok := true
	for i, elt := range src.Elts {
		var eltOk bool
		lit.elts[i], eltOk = processExpr(pscope, elt)
		ok = ok && eltOk
	}
	return lit, ok
}

func (n *compositeLit) source() ast.Node {
	return n.src
}

func (n *compositeLit) buildElements(ascope compositeLitResolveScope) ([]ir.Expr, bool) {
	elts := make([]ir.Expr, len(n.elts))
	if len(n.elts) == 0 {
		return nil, true
	}

	ok := true
	subScope, scopeOk := ascope.sub(n.src)
	if !scopeOk {
		return nil, false
	}
	for i, elt := range n.elts {
		var eltOk bool
		elts[i], eltOk = buildAExpr(subScope, elt)
		if eltOk && irkind.IsNumber(elts[i].Type().Kind()) {
			elts[i], eltOk = castNumber(subScope, elts[i], subScope.dtype())
		}
		if !eltOk {
			ok = false
			continue
		}
		eltOk = assignableToAt(subScope, elt.source(), elts[i].Type(), subScope.want())
		ok = ok && eltOk
	}
	return elts, ok
}

func (n *compositeLit) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	ascope, ok := rscope.(compositeLitResolveScope)
	if !ok {
		return nil, rscope.Err().AppendInternalf(n.src, "cannot build a composite expression without a composite scope")
	}
	elts, ok := n.buildElements(ascope)
	if !ok {
		return nil, false
	}
	return ascope.newInferCompositeType(n.src, elts)
}

func (n *compositeLit) String() string {
	s := make([]string, len(n.elts))
	return fmt.Sprintf("{%s}", strings.Join(s, ","))
}

type arraySliceExprType interface {
	exprNode
	typeExprNode
	processLiteralExpr(procScope, *ast.CompositeLit) (exprNode, bool)
}

func processArraySliceType(pscope typeProcScope, typ *ast.ArrayType) (arraySliceExprType, bool) {
	if typ.Len != nil {
		return processArrayType(pscope, typ)
	}
	return processSliceType(pscope, typ)
}

func processCompositeLit(pscope procScope, src *ast.CompositeLit) (exprNode, bool) {
	if src.Type == nil {
		return processUntypedCompositeLit(pscope, src)
	}
	typScope := defaultTypeProcScope(pscope)
	switch srcTypeT := src.Type.(type) {
	case *ast.ArrayType:
		typ, typeOk := processArraySliceType(typScope, srcTypeT)
		literal, literalOk := typ.processLiteralExpr(pscope, src)
		return literal, typeOk && literalOk
	case *ast.CompositeLit:
		return processCompositeLit(pscope, srcTypeT)
	}
	return processCompositeLitStruct(pscope, src, src.Type)
}
