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
	"reflect"
	"strings"

	"github.com/gx-org/gx/build/ir"
)

type (
	iMethod struct {
		name  *ast.Ident
		fType *funcType
	}

	iface struct {
		src *ast.InterfaceType

		typs    []typeExprNode
		methods []*iMethod
	}
)

var _ typeExprNode = (*iface)(nil)

func toStringList(names []*ast.Ident) []string {
	ss := make([]string, len(names))
	for i, name := range names {
		ss[i] = name.Name
	}
	return ss
}

func processInterfaceType(pscope procScope, src *ast.InterfaceType) (*iface, bool) {
	s := &iface{src: src}
	ok := true
	for _, elem := range src.Methods.List {
		if len(elem.Names) > 1 {
			ok = pscope.Err().Appendf(elem, "more than one interface elements not supported: got %v", toStringList(elem.Names))
			continue
		}
		if len(elem.Names) == 1 {
			method, methodOk := processMethodSignature(pscope, elem.Names[0], elem.Type)
			if !methodOk {
				ok = false
				continue
			}
			s.methods = append(s.methods, method)
			continue
		}
		var elemOk bool
		s.typs, elemOk = flattenTypeList(pscope, s.typs, elem.Type)
		ok = ok && elemOk
	}
	return s, ok
}

func processMethodSignature(pscope procScope, name *ast.Ident, tp ast.Expr) (*iMethod, bool) {
	astFType, isFType := tp.(*ast.FuncType)
	if !isFType {
		return nil, pscope.Err().Appendf(name, "cannot process method definition: expected type %s but got %T", reflect.TypeFor[*ast.FuncType]().String(), tp)
	}
	ftype, ok := processFuncType(pscope, astFType, nil, false)
	if !ok {
		return nil, false
	}
	return &iMethod{
		name:  name,
		fType: ftype,
	}, true
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

func (s *iface) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	ok := true
	types := make([]ir.Type, 0, len(s.typs)) // Do not use s.typs length because we exclude types with errors.
	rtypeNames := map[string]ir.Type{}
	for i, typ := range s.typs {
		typeExpr, typOk := typ.buildTypeExpr(rscope)
		if !typOk {
			ok = false
			continue
		}
		types = append(types, typeExpr.Val())
		if prev, exists := rtypeNames[s.typs[i].String()]; exists {
			ok = rscope.Err().Appendf(s.source(), "overlapping terms %s and %s", prev, typeExpr)
		}
		rtypeNames[typeExpr.SourceString(rscope.fileScope().irFile())] = typeExpr.Val()
	}
	var imeths []*ir.IMethod
	for _, method := range s.methods {
		methType, _, methOk := method.fType.buildFuncType(rscope)
		if !methOk {
			ok = false
			continue
		}
		imeths = append(imeths, &ir.IMethod{
			Name:  method.name,
			FType: methType,
		})
	}
	return ir.TypeExpr(nil, ir.NewInterface(s.src, types, imeths)), ok
}

func (s *iface) source() ast.Node {
	return s.src
}

func (s *iface) String() string {
	all := make([]string, len(s.typs))
	for i, typ := range s.typs {
		all[i] = typ.String()
	}
	return fmt.Sprintf("interface { %s }", strings.Join(all, "|"))
}
