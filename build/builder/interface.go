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
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/gx-org/gx/build/ir"
)

type typeSet struct {
	src *ast.InterfaceType

	typs []typeExprNode
}

var _ typeExprNode = (*typeSet)(nil)

func processInterfaceType(pscope procScope, src *ast.InterfaceType) (*typeSet, bool) {
	s := &typeSet{src: src}
	ok := true
	for _, elem := range src.Methods.List {
		if len(elem.Names) > 0 {
			ok = pscope.Err().Appendf(elem, "interface element not supported")
			continue
		}
		var elemOk bool
		s.typs, elemOk = flattenTypeList(pscope, s.typs, elem.Type)
		ok = ok && elemOk
	}
	return s, ok
}

func flattenTypeList(pscope procScope, list []typeExprNode, expr ast.Expr) ([]typeExprNode, bool) {
	result := list
	if bExpr, ok := expr.(*ast.BinaryExpr); ok && bExpr.Op == token.OR {
		result, ok = flattenTypeList(pscope, list, bExpr.X)
		if !ok {
			return nil, false
		}
		expr = bExpr.Y
	}
	typScope := defaultTypeProcScope(pscope)
	typ, ok := processTypeExpr(typScope, expr)
	return append(result, typ), ok
}

func (s *typeSet) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	ok := true
	ext := &ir.TypeSet{
		BaseType: ir.BaseType[*ast.InterfaceType]{Src: s.src},
		Typs:     make([]ir.Type, len(s.typs)),
	}
	rtypeNames := map[string]ir.Type{}
	for i, typ := range s.typs {
		typeExpr, typOk := typ.buildTypeExpr(rscope)
		if !typOk {
			ok = false
			continue
		}
		ext.Typs[i] = typeExpr.Typ
		if prev, exists := rtypeNames[s.typs[i].String()]; exists {
			ok = rscope.Err().Appendf(s.source(), "overlapping terms %s and %s", prev, typeExpr)
		}
		rtypeNames[typeExpr.String()] = typeExpr.Typ
	}
	return &ir.TypeValExpr{X: ext, Typ: ext}, ok
}

func (s *typeSet) source() ast.Node {
	return s.src
}

func (s *typeSet) String() string {
	all := make([]string, len(s.typs))
	for i, typ := range s.typs {
		all[i] = typ.String()
	}
	return fmt.Sprintf("interface { %s }", strings.Join(all, "|"))
}
